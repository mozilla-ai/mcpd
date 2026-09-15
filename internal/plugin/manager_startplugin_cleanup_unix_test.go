//go:build unix

package plugin

import (
	"context"
	"errors"
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

// The tests in this file are integration tests: each one compiles the
// fixture plugin under testdata/fixtureplugin, spawns it as a real process,
// and talks to it over a real gRPC unix socket. They are skipped under
// -short so the default unit-test path stays fast, and they are Unix-only
// (process groups, the null signal, pgrep/pkill) to match the
// process_unix.go / process_windows.go split of the code under test.

// fixtureModeHealthy and friends mirror the mode constants in
// testdata/fixtureplugin/main.go; the value is baked into each fixture
// binary at build time via -ldflags "-X main.mode=...".
const (
	fixtureModeHealthy                      = "healthy"
	fixtureModeConfigureFail                = "configure-fail"
	fixtureModeCheckReadyFail               = "checkready-fail"
	fixtureModeCheckReadyFailWithDescendant = "checkready-fail-with-descendant"
)

func TestManager_startPlugin_CleansUpDescendantWithinDeadline(t *testing.T) {
	skipIfShort(t)
	t.Parallel()

	// This is the regression guard for the process-group cleanup: a plugin
	// that forks a descendant inheriting its stdout/stderr must not leave
	// startPlugin blocked indefinitely, and the descendant must not survive
	// as an orphan.
	binaryPath := newFixturePlugin(t, fixtureModeCheckReadyFailWithDescendant)
	m := newTestManager(t, binaryPath)

	start := time.Now()
	plg, err := m.startPlugin(context.Background(), fixtureModeCheckReadyFailWithDescendant, binaryPath)
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
		return !anyProcessRunningFor(t, binaryPath)
	}, 3*time.Second, 20*time.Millisecond,
		"neither the plugin process nor the descendant it forked may survive startPlugin's cleanup")
}

func TestManager_startPlugin_CleansUpOnCheckReadyFailure(t *testing.T) {
	skipIfShort(t)
	t.Parallel()

	binaryPath := newFixturePlugin(t, fixtureModeCheckReadyFail)
	m := newTestManager(t, binaryPath)

	plg, err := m.startPlugin(context.Background(), fixtureModeCheckReadyFail, binaryPath)
	require.Error(t, err)
	require.Nil(t, plg)
	require.Contains(t, err.Error(), "plugin not ready")

	// The process was definitely running a moment ago (CheckReady is an RPC
	// to it), so this is a real assertion, not a vacuous one.
	require.Eventually(t, func() bool {
		return !anyProcessRunningFor(t, binaryPath)
	}, 2*time.Second, 20*time.Millisecond,
		"a plugin that fails CheckReady must not survive startPlugin - "+
			"it was never added to m.plugins, so nothing will ever kill it later")

	require.Eventually(t, func() bool {
		return !anyPluginSocketExists(t, binaryPath)
	}, 2*time.Second, 20*time.Millisecond,
		"the plugin's unix socket file must be removed when startPlugin cleans up after a CheckReady failure")
}

func TestManager_startPlugin_CleansUpOnConfigureFailure(t *testing.T) {
	skipIfShort(t)
	t.Parallel()

	binaryPath := newFixturePlugin(t, fixtureModeConfigureFail)
	m := newTestManager(t, binaryPath)

	plg, err := m.startPlugin(context.Background(), fixtureModeConfigureFail, binaryPath)
	require.Error(t, err)
	require.Nil(t, plg)
	require.Contains(t, err.Error(), "error configuring plugin")

	require.Eventually(t, func() bool {
		return !anyProcessRunningFor(t, binaryPath)
	}, 2*time.Second, 20*time.Millisecond,
		"a plugin that fails Configure must not survive startPlugin - "+
			"it was never added to m.plugins, so nothing will ever kill it later")

	require.Eventually(t, func() bool {
		return !anyPluginSocketExists(t, binaryPath)
	}, 2*time.Second, 20*time.Millisecond,
		"the plugin's unix socket file must be removed when startPlugin cleans up after a Configure failure")
}

func TestManager_startPlugin_HealthyPluginSurvivesAndIsReturned(t *testing.T) {
	skipIfShort(t)
	t.Parallel()

	binaryPath := newFixturePlugin(t, fixtureModeHealthy)
	m := newTestManager(t, binaryPath)

	plg, err := m.startPlugin(context.Background(), fixtureModeHealthy, binaryPath)
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

	require.True(t, processAlive(t, plg.cmd.Process.Pid),
		"a healthy plugin must still be running after startPlugin returns successfully")

	_, statErr := os.Stat(plg.address)
	require.NoError(t, statErr, "the plugin's unix socket should exist while it is running")
}

// anyPluginSocketExists reports whether a unix socket file for the given
// plugin binary is still sitting under os.TempDir(). It matches on
// generateAddress's naming convention (plugin-<basename>-<id>.sock) rather
// than a captured exact path, since a failed startPlugin call gives the
// test no runningPlugin to read the address off of. Each test builds its
// fixture under a distinct basename, so the glob cannot see a parallel
// test's socket.
func anyPluginSocketExists(t *testing.T, binaryPath string) bool {
	t.Helper()

	pattern := filepath.Join(os.TempDir(), fmt.Sprintf("plugin-%s-*.sock", filepath.Base(binaryPath)))
	matches, err := filepath.Glob(pattern)
	if err != nil {
		// Reached from require.Eventually's condition goroutine, where FailNow
		// is not safe, so mark the failure and let the condition settle.
		t.Errorf("globbing %q: %v", pattern, err)
		return false
	}
	return len(matches) > 0
}

// anyProcessRunningFor reports whether any live process was launched from
// binaryPath, using pgrep against the full command line. It's how these
// tests confirm "no orphan survives" for the failure paths, where
// startPlugin returns nil on error and gives the test no pid to check
// directly. pgrep exiting 1 means "no match"; any other failure means the
// check itself could not run, which fails the test rather than silently
// reporting "nothing running".
func anyProcessRunningFor(t *testing.T, binaryPath string) bool {
	t.Helper()

	out, err := exec.Command("pgrep", "-f", regexp.QuoteMeta(binaryPath)).Output()
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) && exitErr.ExitCode() == 1 {
			return false
		}
		// Both callers invoke this from require.Eventually's condition, which
		// testify runs on its own goroutine. t.Fatalf there calls FailNow off
		// the test goroutine: the message prints, the goroutine exits without
		// signalling testify, and Eventually then blocks for its full waitFor
		// before adding a misleading "Condition never satisfied". t.Errorf
		// fails the test just as hard and lets the condition return.
		t.Errorf("running pgrep for %q: %v", binaryPath, err)
		return false
	}
	return len(strings.TrimSpace(string(out))) > 0
}

// killAllRunningFor is the cleanup safety net for a single test: it force
// kills any process still running from binaryPath, so a test that fails
// its assertions before things settle can't leave a real process behind.
// pkill exiting 1 means nothing was left to kill; anything else is
// reported, since a silent no-op here would defeat the point.
func killAllRunningFor(t *testing.T, binaryPath string) {
	t.Helper()

	err := exec.Command("pkill", "-9", "-f", regexp.QuoteMeta(binaryPath)).Run()
	if err == nil {
		return
	}
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) && exitErr.ExitCode() == 1 {
		return
	}
	t.Errorf("running pkill for %q: %v", binaryPath, err)
}

// newFixturePlugin compiles testdata/fixtureplugin into t.TempDir() with the
// given behaviour baked in via -ldflags, registers a cleanup that kills any
// process still running from it, and returns the binary's path. Each test
// builds exactly the one binary it uses, so no test pays for another's
// compile, and the basename (the mode) keeps sockets distinguishable
// between parallel tests.
func newFixturePlugin(t *testing.T, mode string) string {
	t.Helper()

	srcDir, err := filepath.Abs("testdata/fixtureplugin")
	require.NoError(t, err, "resolving fixture plugin source dir")

	binaryPath := filepath.Join(t.TempDir(), "fixture-"+mode)
	cmd := exec.Command("go", "build", "-ldflags", "-X main.mode="+mode, "-o", binaryPath, srcDir)
	output, err := cmd.CombinedOutput()
	require.NoError(t, err, "building fixture plugin in mode %q:\n%s", mode, output)

	t.Cleanup(func() { killAllRunningFor(t, binaryPath) })

	return binaryPath
}

// newTestManager returns a Manager whose plugin directory is the one holding
// binaryPath, logging to a null logger.
func newTestManager(t *testing.T, binaryPath string) *Manager {
	t.Helper()

	m, err := NewManager(hclog.NewNullLogger(), &config.PluginConfig{Dir: filepath.Dir(binaryPath)})
	require.NoError(t, err)
	return m
}

// processAlive reports whether pid refers to a running process, by sending
// it the null signal (syscall.Signal(0)): this checks for existence and
// permission without actually signalling the process.
func processAlive(t *testing.T, pid int) bool {
	t.Helper()

	if pid <= 0 {
		return false
	}
	proc, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	return proc.Signal(syscall.Signal(0)) == nil
}

// skipIfShort skips the integration tests in this file under -short, since
// each one compiles and spawns a real plugin binary.
func skipIfShort(t *testing.T) {
	t.Helper()

	if testing.Short() {
		t.Skip("skipping plugin process integration test in -short mode")
	}
}
