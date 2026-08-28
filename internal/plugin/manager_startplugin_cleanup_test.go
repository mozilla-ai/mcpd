package plugin

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/hashicorp/go-hclog"
	"github.com/stretchr/testify/require"

	"github.com/mozilla-ai/mcpd/internal/config"
)

// fixturePluginDir holds the binaries built by TestMain for the
// TestManager_startPlugin_* tests in this file, keyed by the failure mode
// encoded in each binary's name (see testdata/fixtureplugin).
var fixturePluginDir string

func TestMain(m *testing.M) {
	dir, cleanup, err := buildFixturePlugins()
	if err != nil {
		fmt.Fprintln(os.Stderr, "building fixture plugins:", err)
		os.Exit(1)
	}
	fixturePluginDir = dir

	code := m.Run()

	// Belt and braces: nothing in this package's tests should still be
	// running a fixture plugin by the time TestMain exits, but a failed
	// assertion partway through a test could in principle skip a deferred
	// kill. Make sure a flaky run doesn't leave real processes behind on
	// whatever machine ran it.
	killAllFixturePlugins(dir)
	cleanup()

	os.Exit(code)
}

// buildFixturePlugins compiles internal/plugin/testdata/fixtureplugin three
// times under different names, since the fixture selects its failure mode
// from its own binary name (see that package's doc comment). Building once
// in TestMain keeps every test from paying the compile cost, and keeps the
// failure mode a function of which binary is launched rather than of shared
// environment/process state, so nothing here needs t.Parallel restrictions.
func buildFixturePlugins() (dir string, cleanup func(), err error) {
	dir, err = os.MkdirTemp("", "mcpd-fixtureplugin-")
	if err != nil {
		return "", nil, fmt.Errorf("creating temp dir: %w", err)
	}
	cleanup = func() { _ = os.RemoveAll(dir) }

	srcDir, err := filepath.Abs("testdata/fixtureplugin")
	if err != nil {
		cleanup()
		return "", nil, fmt.Errorf("resolving fixture plugin source dir: %w", err)
	}

	for _, name := range []string{
		"healthy-plugin",
		"configure-fail-plugin",
		"checkready-fail-plugin",
		"checkready-fail-with-descendant-plugin",
	} {
		out := filepath.Join(dir, name)
		cmd := exec.Command("go", "build", "-o", out, srcDir)
		if output, buildErr := cmd.CombinedOutput(); buildErr != nil {
			cleanup()
			return "", nil, fmt.Errorf("building fixture plugin %q: %w\n%s", name, buildErr, output)
		}
	}

	return dir, cleanup, nil
}

func fixturePluginPath(t *testing.T, name string) string {
	t.Helper()
	path := filepath.Join(fixturePluginDir, name)
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("fixture plugin %q not built: %v", name, err)
	}
	return path
}

// processAlive reports whether pid refers to a running process, by sending
// it the null signal (syscall.Signal(0)): this checks for existence and
// permission without actually signalling the process.
func processAlive(pid int) bool {
	if pid <= 0 {
		return false
	}
	proc, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	return proc.Signal(syscall.Signal(0)) == nil
}

// anyProcessRunningFor reports whether any live process was launched from
// binaryPath, using pgrep against the full command line. It's how these
// tests confirm "no orphan survives" for the failure paths, where
// startPlugin returns nil on error and gives the test no pid to check
// directly.
func anyProcessRunningFor(binaryPath string) bool {
	out, err := exec.Command("pgrep", "-f", regexp.QuoteMeta(binaryPath)).Output()
	if err != nil {
		// pgrep exits 1 (no error output) when nothing matches.
		return false
	}
	return len(strings.TrimSpace(string(out))) > 0
}

// killAllRunningFor is the cleanup safety net for a single test: it force
// kills any process still running from binaryPath, so a test that fails
// its assertions before things settle can't leave a real process behind.
func killAllRunningFor(binaryPath string) {
	_ = exec.Command("pkill", "-9", "-f", regexp.QuoteMeta(binaryPath)).Run()
}

// killAllFixturePlugins is TestMain's final safety net across every fixture
// binary this file builds.
func killAllFixturePlugins(dir string) {
	_ = exec.Command("pkill", "-9", "-f", regexp.QuoteMeta(dir)).Run()
}

func TestManager_startPlugin_CleansUpOnCheckReadyFailure(t *testing.T) {
	logger := hclog.NewNullLogger()
	cfg := &config.PluginConfig{Dir: fixturePluginDir}
	m, err := NewManager(logger, cfg)
	require.NoError(t, err)

	binaryPath := fixturePluginPath(t, "checkready-fail-plugin")
	t.Cleanup(func() { killAllRunningFor(binaryPath) })

	ctx := context.Background()
	plg, err := m.startPlugin(ctx, "checkready-fail-plugin", binaryPath)
	require.Error(t, err)
	require.Nil(t, plg)
	require.Contains(t, err.Error(), "plugin not ready")

	// The process was definitely running a moment ago (CheckReady is an RPC
	// to it), so this is a real assertion, not a vacuous one.
	require.Eventually(t, func() bool {
		return !anyProcessRunningFor(binaryPath)
	}, 2*time.Second, 20*time.Millisecond,
		"a plugin that fails CheckReady must not survive startPlugin - "+
			"it was never added to m.plugins, so nothing will ever kill it later")

	require.Eventually(t, func() bool {
		return !anyPluginSocketExists(t, binaryPath)
	}, 2*time.Second, 20*time.Millisecond,
		"the plugin's unix socket file must be removed when startPlugin cleans up after a CheckReady failure")
}

func TestManager_startPlugin_CleansUpOnConfigureFailure(t *testing.T) {
	logger := hclog.NewNullLogger()
	cfg := &config.PluginConfig{Dir: fixturePluginDir}
	m, err := NewManager(logger, cfg)
	require.NoError(t, err)

	binaryPath := fixturePluginPath(t, "configure-fail-plugin")
	t.Cleanup(func() { killAllRunningFor(binaryPath) })

	ctx := context.Background()
	plg, err := m.startPlugin(ctx, "configure-fail-plugin", binaryPath)
	require.Error(t, err)
	require.Nil(t, plg)
	require.Contains(t, err.Error(), "error configuring plugin")

	require.Eventually(t, func() bool {
		return !anyProcessRunningFor(binaryPath)
	}, 2*time.Second, 20*time.Millisecond,
		"a plugin that fails Configure must not survive startPlugin - "+
			"it was never added to m.plugins, so nothing will ever kill it later")

	require.Eventually(t, func() bool {
		return !anyPluginSocketExists(t, binaryPath)
	}, 2*time.Second, 20*time.Millisecond,
		"the plugin's unix socket file must be removed when startPlugin cleans up after a Configure failure")
}

// TestManager_startPlugin_CleansUpDescendantWithinDeadline is the regression
// guard for the process-group cleanup: a plugin that forks a descendant
// inheriting its stdout/stderr must not leave startPlugin blocked
// indefinitely, and the descendant must not survive as an orphan.
func TestManager_startPlugin_CleansUpDescendantWithinDeadline(t *testing.T) {
	logger := hclog.NewNullLogger()
	cfg := &config.PluginConfig{Dir: fixturePluginDir}
	m, err := NewManager(logger, cfg)
	require.NoError(t, err)

	binaryPath := fixturePluginPath(t, "checkready-fail-with-descendant-plugin")
	t.Cleanup(func() { killAllRunningFor(binaryPath) })

	ctx := context.Background()

	start := time.Now()
	plg, err := m.startPlugin(ctx, "checkready-fail-with-descendant-plugin", binaryPath)
	elapsed := time.Since(start)

	require.Error(t, err)
	require.Nil(t, plg)
	require.Contains(t, err.Error(), "plugin not ready")

	// startPlugin's cleanup defer must complete before the call returns, so
	// this bounds the whole call: it must not be able to block forever on a
	// descendant that inherited stdout/stderr.
	require.Less(t, elapsed, pluginForceKillTimeout+2*time.Second,
		"startPlugin must not block indefinitely on a descendant holding the plugin's stdout/stderr open")

	require.Eventually(t, func() bool {
		return !anyProcessRunningFor(binaryPath)
	}, 3*time.Second, 20*time.Millisecond,
		"neither the plugin process nor the descendant it forked may survive startPlugin's cleanup")
}

func TestManager_startPlugin_HealthyPluginSurvivesAndIsReturned(t *testing.T) {
	logger := hclog.NewNullLogger()
	cfg := &config.PluginConfig{Dir: fixturePluginDir}
	m, err := NewManager(logger, cfg)
	require.NoError(t, err)

	binaryPath := fixturePluginPath(t, "healthy-plugin")
	t.Cleanup(func() { killAllRunningFor(binaryPath) })

	ctx := context.Background()
	plg, err := m.startPlugin(ctx, "healthy-plugin", binaryPath)
	require.NoError(t, err)
	require.NotNil(t, plg)
	t.Cleanup(func() {
		// The regression guard for this fix: a plugin that starts
		// successfully must NOT be killed by the new cleanup defer. Kill it
		// directly here rather than via the full graceful plg.stop() RPC
		// round trip, which this fixture doesn't implement and which would
		// otherwise cost the pluginForceKillTimeout on every run.
		_ = plg.cmd.Process.Kill()
		_ = plg.conn.Close()
		_ = os.Remove(plg.address)
	})

	require.True(t, processAlive(plg.cmd.Process.Pid),
		"a healthy plugin must still be running after startPlugin returns successfully")

	_, statErr := os.Stat(plg.address)
	require.NoError(t, statErr, "the plugin's unix socket should exist while it is running")
}

// anyPluginSocketExists reports whether a unix socket file for the given
// plugin binary is still sitting under os.TempDir(). It matches on
// generateAddress's naming convention (plugin-<basename>-<id>.sock) rather
// than a captured exact path, since a failed startPlugin call gives the
// test no runningPlugin to read the address off of.
func anyPluginSocketExists(t *testing.T, binaryPath string) bool {
	t.Helper()

	pattern := filepath.Join(os.TempDir(), fmt.Sprintf("plugin-%s-*.sock", filepath.Base(binaryPath)))
	matches, err := filepath.Glob(pattern)
	require.NoError(t, err)
	return len(matches) > 0
}
