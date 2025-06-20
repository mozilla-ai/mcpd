package daemon

import (
	"fmt"
	"sync"
	"testing"

	"github.com/mark3labs/mcp-go/client"
	"github.com/stretchr/testify/require"
)

func TestClientManager_Add_Client_Tools(t *testing.T) {
	t.Parallel()
	cm := NewClientManager()

	c := &client.Client{}
	tools := []string{"tool1", "tool2"}
	name := "server1"

	cm.Add(name, c, tools)

	// Test Client retrieval
	rc, ok := cm.Client(name)
	require.True(t, ok)
	require.Equal(t, c, rc)

	// Test Tools retrieval
	rt, ok := cm.Tools(name)
	require.True(t, ok)
	require.Equal(t, tools, rt)
}

func TestClientManager_List(t *testing.T) {
	t.Parallel()
	cm := NewClientManager()

	cm.Add("server1", &client.Client{}, []string{"a"})
	cm.Add("server2", &client.Client{}, []string{"b"})

	names := cm.List()
	require.Len(t, names, 2)
	require.ElementsMatch(t, []string{"server1", "server2"}, names)
}

func TestClientManager_Remove(t *testing.T) {
	t.Parallel()
	cm := NewClientManager()

	cm.Add("server1", &client.Client{}, []string{"tool"})
	cm.Remove("server1")

	_, ok := cm.Client("server1")
	require.False(t, ok)

	_, ok = cm.Tools("server1")
	require.False(t, ok)

	require.Empty(t, cm.List())
}

func TestClientManager_EmptyManager(t *testing.T) {
	t.Parallel()
	cm := NewClientManager()

	_, ok := cm.Client("missing")
	require.False(t, ok)

	_, ok = cm.Tools("missing")
	require.False(t, ok)

	require.Empty(t, cm.List())
}

// TestClientManager_ConcurrentAccess can be run with: go test -race ./...
func TestClientManager_ConcurrentAccess(t *testing.T) {
	t.Parallel()
	cm := NewClientManager()

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(3)

		name := fmt.Sprintf("server-%d", i)
		go func() {
			defer wg.Done()
			cm.Add(name, &client.Client{}, []string{"tool"})
		}()
		go func() {
			defer wg.Done()
			_, _ = cm.Client(name)
		}()
		go func() {
			defer wg.Done()
			cm.List()
		}()
	}

	wg.Wait()
}
