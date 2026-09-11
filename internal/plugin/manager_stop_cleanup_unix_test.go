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

// fixtureModeHealthyWithDescendant mirrors modeHealthyWithDescendant in
// testdata/fixtureplugin/main.go.
const fixtureModeHealthyWithDescendant = "healthy-with-descendant"

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
	// Before this fix the last of those three was unbounded, so a descendant
	// holding the plugin's stdout/stderr open kept cmd.Wait from returning
	// and stop() never came back at all.
	deadline := pluginGracefulStopTimeout + 2*pluginForceKillTimeout + 3*time.Second

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
