//go:build unix

package plugin

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// This file is the graceful-shutdown counterpart of
// manager_startplugin_cleanup_unix_test.go. That file covers startPlugin's
// failure paths; this one covers stop(), which is the path a normal daemon
// shutdown takes via StopPlugins. It shares that file's helpers and its
// constraints: integration tests, skipped under -short, Unix-only.

// fixtureModeHealthyWithDescendant and fixtureModeHealthyWithEscapedDescendant
// mirror modeHealthyWithDescendant and modeHealthyWithEscapedDescendant in
// testdata/fixtureplugin/main.go.
const (
	fixtureModeHealthyWithDescendant        = "healthy-with-descendant"
	fixtureModeHealthyWithEscapedDescendant = "healthy-with-escaped-descendant"
)

func TestManager_stop_KillsDescendantsAndReturnsWithinDeadline(t *testing.T) {
	skipIfShort(t)
	t.Parallel()

	// A plugin that starts successfully and forks a descendant is the case
	// startPlugin's cleanup defer never sees: it only runs when startup
	// fails. Tearing this plugin down has to kill the descendant too, and
	// has to return, since StopPlugins calls stop() for every plugin on the
	// daemon's shutdown path.
	binaryPath := newFixturePlugin(t, fixtureModeHealthyWithDescendant)
	m := newTestManager(t, binaryPath)

	plg, err := m.startPlugin(context.Background(), fixtureModeHealthyWithDescendant, binaryPath)
	require.NoError(t, err)
	require.NotNil(t, plg)
	t.Cleanup(func() { _ = os.Remove(plg.address) })

	require.True(t, anyProcessRunningFor(t, binaryPath),
		"the plugin and its descendant must be running before stop is called")

	// The fixture does not exit on the Stop RPC, so stop() always reaches
	// its force-kill branch. That makes the worst case the graceful RPC
	// timeout plus the wait before the kill plus the wait to reap after it.
	deadline := pluginGracefulStopTimeout + pluginExitGraceTimeout + pluginReapTimeout + 3*time.Second

	start := time.Now()
	stopErr := plg.stop()
	elapsed := time.Since(start)

	require.NoError(t, stopErr)
	require.Less(t, elapsed, deadline,
		"stop must not block indefinitely on a descendant holding the plugin's stdout/stderr open")

	require.Eventually(t, func() bool {
		return !anyProcessRunningFor(t, binaryPath)
	}, 3*time.Second, 20*time.Millisecond,
		"neither the plugin process nor the descendant it forked may survive stop")
}

// TestManager_stop_ReportsIncompleteCleanupWhenReapTimesOut is the
// regression guard for stop()'s post-kill reap timeout: it must report that
// cleanup did not complete rather than treat giving up on the reap as a
// successful stop.
//
// fixtureModeHealthyWithEscapedDescendant's descendant calls setsid, so it
// leaves the plugin's process group entirely while still inheriting and
// holding open the plugin's stdout/stderr. killProcessGroup only signals
// what remains in that group (the PID-reuse and setsid/setpgid limits are
// documented on killProcessGroup in process_unix.go as a follow-up), so the
// force kill in stop() cannot reach this descendant. cmd.Wait then never
// returns, and the post-kill reap wait always times out.
func TestManager_stop_ReportsIncompleteCleanupWhenReapTimesOut(t *testing.T) {
	skipIfShort(t)
	t.Parallel()

	binaryPath := newFixturePlugin(t, fixtureModeHealthyWithEscapedDescendant)
	m := newTestManager(t, binaryPath)

	plg, err := m.startPlugin(context.Background(), fixtureModeHealthyWithEscapedDescendant, binaryPath)
	require.NoError(t, err)
	require.NotNil(t, plg)
	t.Cleanup(func() {
		_ = os.Remove(plg.address)
		// The escaped descendant is expected to outlive stop() (that's the
		// point of this test), so it needs its own cleanup rather than
		// relying on the group kill inside stop().
		killAllRunningFor(t, binaryPath)
	})

	require.True(t, anyProcessRunningFor(t, binaryPath),
		"the plugin and its escaped descendant must be running before stop is called")

	// stop() runs on its own goroutine, bounded by a select with its own
	// timeout: if the post-kill wait were ever to become unbounded again,
	// this test must fail on the assertion below rather than on the
	// package's -timeout panic, which would say nothing about why.
	stopDone := make(chan error, 1)
	go func() { stopDone <- plg.stop() }()

	select {
	case stopErr := <-stopDone:
		require.ErrorIs(t, stopErr, errPluginCleanupIncomplete)
	case <-time.After(pluginGracefulStopTimeout + pluginExitGraceTimeout + pluginReapTimeout + 3*time.Second):
		t.Fatal("stop did not return: the post-kill reap wait is unbounded")
	}

	require.True(t, anyProcessRunningFor(t, binaryPath),
		"the escaped descendant left the process group stop() signals, so it must still be running - "+
			"a false pass here would mean the fixture isn't exercising the regression this test guards")
}
