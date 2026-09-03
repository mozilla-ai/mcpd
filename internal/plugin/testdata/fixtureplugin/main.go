// Command fixtureplugin is a test-only plugin binary used by
// internal/plugin's manager tests to exercise startPlugin's post-spawn
// failure paths against a real process and a real gRPC socket.
//
// It is never built by `go build ./...` (the "testdata" directory is
// excluded from wildcard package patterns by the go tool), only by the
// tests themselves via an explicit path.
//
// Its behaviour is fixed at build time through the mode variable, which the
// tests set with -ldflags "-X main.mode=<mode>". Baking the mode into the
// binary keeps each test's fixture independent of environment variables and
// of the binary's file name, so tests can run in parallel without sharing
// any process state. Recognised modes:
//
//   - modeHealthy: behaves like a well-formed plugin
//   - modeConfigureFail: fails Configure
//   - modeCheckReadyFail: fails CheckReady
//   - modeCheckReadyFailWithDescendant: first forks a copy of itself that
//     inherits this process's stdout/stderr and blocks forever, simulating a
//     descendant that keeps those file descriptors open after the plugin
//     process itself has been killed, then fails CheckReady
//
// Setting the FIXTURE_DESCENDANT_BLOCK env var makes the binary skip plugin
// serving entirely and just block forever: this is how the forked descendant
// behaves, and it is set only on that child's own environment.
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"

	mcpdpluginsv1 "github.com/mozilla-ai/mcpd-plugins-sdk-go/pkg/plugins/v1"
	"google.golang.org/protobuf/types/known/emptypb"
)

const (
	modeHealthy                      = "healthy"
	modeConfigureFail                = "configure-fail"
	modeCheckReadyFail               = "checkready-fail"
	modeCheckReadyFailWithDescendant = "checkready-fail-with-descendant"
)

// mode selects the fixture's behaviour. It is set at build time via
// -ldflags "-X main.mode=<mode>"; an empty or unknown value is a test bug
// and fails fast rather than silently behaving like a healthy plugin.
var mode string

// fixturePlugin is a minimal plugin whose Configure and CheckReady RPCs can be
// made to fail on demand, so the manager's post-spawn failure paths can be
// exercised against a real process.
type fixturePlugin struct {
	mcpdpluginsv1.BasePlugin

	failConfigure  bool
	failCheckReady bool
}

// Configure fails when the fixture was built to fail configuration, and
// otherwise defers to BasePlugin.
func (p *fixturePlugin) Configure(
	ctx context.Context,
	cfg *mcpdpluginsv1.PluginConfig,
) (*emptypb.Empty, error) {
	if p.failConfigure {
		return nil, fmt.Errorf("fixture plugin: configure failed on purpose")
	}
	return p.BasePlugin.Configure(ctx, cfg)
}

// CheckReady fails when the fixture was built to fail readiness, and
// otherwise defers to BasePlugin.
func (p *fixturePlugin) CheckReady(ctx context.Context, e *emptypb.Empty) (*emptypb.Empty, error) {
	if p.failCheckReady {
		return nil, fmt.Errorf("fixture plugin: checkready failed on purpose")
	}
	return p.BasePlugin.CheckReady(ctx, e)
}

// GetMetadata returns a fixed name and version; the tests never inspect it.
func (p *fixturePlugin) GetMetadata(ctx context.Context, e *emptypb.Empty) (*mcpdpluginsv1.Metadata, error) {
	return &mcpdpluginsv1.Metadata{Name: "fixture-plugin", Version: "0.0.1"}, nil
}

// main serves the fixture plugin in the behaviour selected by mode, or blocks
// forever when running as a forked descendant.
func main() {
	// A forked descendant lands here: it never serves the plugin protocol,
	// it just holds this process's inherited stdout/stderr open forever.
	if os.Getenv("FIXTURE_DESCENDANT_BLOCK") == "1" {
		select {}
	}

	plugin := &fixturePlugin{}

	switch mode {
	case modeHealthy:
	case modeConfigureFail:
		plugin.failConfigure = true
	case modeCheckReadyFail:
		plugin.failCheckReady = true
	case modeCheckReadyFailWithDescendant:
		spawnBlockingDescendant()
		plugin.failCheckReady = true
	default:
		log.Fatalf("fixtureplugin: unknown mode %q (build with -ldflags \"-X main.mode=<mode>\")", mode)
	}

	if err := mcpdpluginsv1.Serve(plugin); err != nil {
		log.Fatal(err)
	}
}

// spawnBlockingDescendant forks a copy of this binary that inherits the
// current process's stdout/stderr and never exits on its own. It is started
// and deliberately not waited on, so it outlives this process's own
// lifecycle exactly like a real plugin's runaway grandchild would.
func spawnBlockingDescendant() {
	self, err := os.Executable()
	if err != nil {
		log.Fatal(err)
	}

	cmd := exec.Command(self)
	cmd.Env = append(os.Environ(), "FIXTURE_DESCENDANT_BLOCK=1")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		log.Fatal(err)
	}
	// Deliberately not waited on: this process's stdout/stderr stay open
	// via the descendant even after this process itself is killed.
}
