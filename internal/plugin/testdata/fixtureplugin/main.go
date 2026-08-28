// Command fixtureplugin is a test-only plugin binary used by
// internal/plugin's manager tests to exercise startPlugin's post-spawn
// failure paths against a real process and a real gRPC socket.
//
// It is never built by `go build ./...` (the "testdata" directory is
// excluded from wildcard package patterns by the go tool), only by the
// tests themselves via an explicit path.
//
// Its behaviour is selected by the basename of the compiled binary, so
// tests can run several failure modes in parallel without racing on
// shared environment variables:
//
//   - a binary name containing "configure-fail" fails Configure
//   - a binary name containing "checkready-fail" fails CheckReady
//   - a binary name additionally containing "with-descendant" first forks a
//     copy of itself that inherits this process's stdout/stderr and blocks
//     forever, simulating a descendant that keeps those file descriptors
//     open after the plugin process itself has been killed
//   - anything else behaves like a healthy plugin
//
// Setting the FIXTURE_DESCENDANT_BLOCK env var makes the binary skip
// plugin serving entirely and just block forever: this is how the
// "with-descendant" mode's forked copy of itself behaves.
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	mcpdpluginsv1 "github.com/mozilla-ai/mcpd-plugins-sdk-go/pkg/plugins/v1"
	"google.golang.org/protobuf/types/known/emptypb"
)

type fixturePlugin struct {
	mcpdpluginsv1.BasePlugin

	failConfigure  bool
	failCheckReady bool
}

func (p *fixturePlugin) Configure(
	ctx context.Context,
	cfg *mcpdpluginsv1.PluginConfig,
) (*emptypb.Empty, error) {
	if p.failConfigure {
		return nil, fmt.Errorf("fixture plugin: configure failed on purpose")
	}
	return p.BasePlugin.Configure(ctx, cfg)
}

func (p *fixturePlugin) CheckReady(ctx context.Context, e *emptypb.Empty) (*emptypb.Empty, error) {
	if p.failCheckReady {
		return nil, fmt.Errorf("fixture plugin: checkready failed on purpose")
	}
	return p.BasePlugin.CheckReady(ctx, e)
}

func (p *fixturePlugin) GetMetadata(ctx context.Context, e *emptypb.Empty) (*mcpdpluginsv1.Metadata, error) {
	return &mcpdpluginsv1.Metadata{Name: "fixture-plugin", Version: "0.0.1"}, nil
}

func main() {
	// A forked descendant lands here: it never serves the plugin protocol,
	// it just holds this process's inherited stdout/stderr open forever.
	if os.Getenv("FIXTURE_DESCENDANT_BLOCK") == "1" {
		select {}
	}

	name := filepath.Base(os.Args[0])

	if strings.Contains(name, "with-descendant") {
		spawnBlockingDescendant()
	}

	plugin := &fixturePlugin{
		failConfigure:  strings.Contains(name, "configure-fail"),
		failCheckReady: strings.Contains(name, "checkready-fail"),
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
