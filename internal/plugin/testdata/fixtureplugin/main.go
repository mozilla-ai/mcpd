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
//   - anything else behaves like a healthy plugin
package main

import (
	"context"
	"fmt"
	"log"
	"os"
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
	name := filepath.Base(os.Args[0])

	plugin := &fixturePlugin{
		failConfigure:  strings.Contains(name, "configure-fail"),
		failCheckReady: strings.Contains(name, "checkready-fail"),
	}

	if err := mcpdpluginsv1.Serve(plugin); err != nil {
		log.Fatal(err)
	}
}
