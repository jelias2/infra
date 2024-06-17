package integration_tests

import (
	"context"
	"net/http"
	"os"
	"path"
	"testing"
	"time"

	"github.com/ethereum-optimism/optimism/proxyd"
	sw "github.com/ethereum-optimism/optimism/proxyd/pkg/avg-sliding-window"
	ms "github.com/ethereum-optimism/optimism/proxyd/tools/mockserver/handler"
	"github.com/stretchr/testify/require"
)

// ts is a convenient method that must parse a time.Time from a string in format `"2006-01-02 15:04:05"`
func ts(s string) time.Time {
	t, err := time.Parse(time.DateTime, s)
	if err != nil {
		panic(err)
	}
	return t
}

func setupBlockHeightZero(t *testing.T) (map[string]*nodeContext, *proxyd.BackendGroup, *ProxydHTTPClient, func(), *sw.AdjustableClock) {
	// setup mock servers
	node1 := NewMockBackend(nil)
	node2 := NewMockBackend(nil)

	dir, err := os.Getwd()
	require.NoError(t, err)

	responses := path.Join(dir, "testdata/block_height_zero_responses.yml")

	h1 := ms.MockedHandler{
		Overrides:    []*ms.MethodTemplate{},
		Autoload:     true,
		AutoloadFile: responses,
	}
	h2 := ms.MockedHandler{
		Overrides:    []*ms.MethodTemplate{},
		Autoload:     true,
		AutoloadFile: responses,
	}

	require.NoError(t, os.Setenv("NODE1_URL", node1.URL()))
	require.NoError(t, os.Setenv("NODE2_URL", node2.URL()))

	node1.SetHandler(http.HandlerFunc(h1.Handler))
	node2.SetHandler(http.HandlerFunc(h2.Handler))

	// setup proxyd
	config := ReadConfig("block_height_zero")
	svr, shutdown, err := proxyd.Start(config)
	require.NoError(t, err)

	// expose the proxyd client
	client := NewProxydClient("http://127.0.0.1:8545")

	// expose the backend group
	bg := svr.BackendGroups["node"]
	require.NotNil(t, bg)
	require.NotNil(t, bg.Consensus)
	require.Equal(t, 2, len(bg.Backends))

	// Check backend default values are non zero
	require.Equal(t, 60*time.Second, bg.Backends[0].GetBlockHeightZeroSlidingWindow().WindowLength())
	require.Equal(t, float64(0.1), bg.Backends[0].GetBlockHeightZeroThreshold())

	// Check custom values can be applied
	require.Equal(t, 70*time.Second, bg.Backends[1].GetBlockHeightZeroSlidingWindow().WindowLength())
	require.Equal(t, float64(0.5), bg.Backends[1].GetBlockHeightZeroThreshold())

	// Create a New Sliding Window and maintain pointer to clock
	clock := sw.NewAdjustableClock(ts("2023-04-21 15:00:00"))
	sw1 := sw.NewSlidingWindow(
		sw.WithWindowLength(bg.Backends[0].GetBlockHeightZeroSlidingWindow().WindowLength()),
		sw.WithClock(clock))

	sw2 := sw.NewSlidingWindow(
		sw.WithWindowLength(bg.Backends[1].GetBlockHeightZeroSlidingWindow().WindowLength()),
		sw.WithClock(clock))

	bg.Backends[0].Override(proxyd.WithBlockHeightZeroSlidingWindow(sw1))
	bg.Backends[1].Override(proxyd.WithBlockHeightZeroSlidingWindow(sw2))

	// convenient mapping to access the nodes
	nodes := map[string]*nodeContext{
		"node1": {
			mockBackend: node1,
			backend:     bg.Backends[0],
			handler:     &h1,
		},
		"node2": {
			mockBackend: node2,
			backend:     bg.Backends[1],
			handler:     &h2,
		},
	}

	return nodes, bg, client, shutdown, clock
}

func TestBlockHeightZero(t *testing.T) {
	nodes, bg, _, shutdown, clock := setupBlockHeightZero(t)
	defer nodes["node1"].mockBackend.Close()
	defer nodes["node2"].mockBackend.Close()
	defer shutdown()

	ctx := context.Background()

	// poll for updated consensus
	update := func() {
		for _, be := range bg.Backends {
			bg.Consensus.UpdateBackend(ctx, be)
		}
		bg.Consensus.UpdateBackendGroupConsensus(ctx)
	}

	// ts is a convenient method that must parse a time.Time from a string in format `"2006-01-02 15:04:05"`
	ts := func(s string) time.Time {
		t, err := time.Parse(time.DateTime, s)
		if err != nil {
			panic(err)
		}
		return t
	}

	addTimeToBackend := func(ts time.Duration) {
		clock.Set(clock.Now().Add(ts))
	}
	// convenient methods to manipulate state and mock responses
	reset := func() {
		for _, node := range nodes {
			node.handler.ResetOverrides()
			node.mockBackend.Reset()
		}
		bg.Consensus.ClearListeners()
		bg.Consensus.Reset()

		require.False(t, bg.Consensus.IsBanned(nodes["node1"].backend))
		require.False(t, bg.Consensus.IsBanned(nodes["node2"].backend))

		// Reset and clear the sliding windows
		clock.Set(ts("2023-04-21 15:04:00"))
		sw1 := sw.NewSlidingWindow(
			sw.WithWindowLength(60*time.Second),
			sw.WithBucketSize(time.Second),
			sw.WithClock(clock))

		sw2 := sw.NewSlidingWindow(
			sw.WithWindowLength(60*time.Second),
			sw.WithBucketSize(time.Second),
			sw.WithClock(clock))

		bg.Backends[0].Override(proxyd.WithBlockHeightZeroSlidingWindow(sw1))
		bg.Backends[1].Override(proxyd.WithBlockHeightZeroSlidingWindow(sw2))
	}

	override := func(node string, method string, block string, response string) {
		if _, ok := nodes[node]; !ok {
			t.Fatalf("node %s does not exist in the nodes map", node)
		}
		nodes[node].handler.AddOverride(&ms.MethodTemplate{
			Method:   method,
			Block:    block,
			Response: response,
		})
	}

	overrideBlock := func(node string, blockRequest string, blockResponse string) {
		override(node,
			"eth_getBlockByNumber",
			blockRequest,
			buildResponse(map[string]string{
				"number": blockResponse,
				"hash":   "hash_" + blockResponse,
			}))
	}

	t.Run("Test Backend BlockHeight Zero Does not if the infractions occur outside window", func(t *testing.T) {
		reset()
		overrideBlock("node1", "latest", "0x0")
		for i := 0; i < 6; i++ {
			update()
			require.Equal(t, uint(1), nodes["node1"].backend.GetBlockHeightZeroSlidingWindow().Count())
			require.Equal(t, float64(1), nodes["node1"].backend.GetBlockHeightZeroSlidingWindow().Avg())
			require.False(t, bg.Consensus.IsBanned(nodes["node1"].backend))
			require.False(t, bg.Consensus.IsBanned(nodes["node2"].backend))
			addTimeToBackend(nodes["node1"].backend.GetBlockHeightZeroSlidingWindow().WindowLength() + time.Second)
		}
		require.False(t, bg.Consensus.IsBanned(nodes["node1"].backend))
		require.False(t, bg.Consensus.IsBanned(nodes["node2"].backend))
	})

	t.Run("Test Backend is banned if above threshold", func(t *testing.T) {
		reset()
		overrideBlock("node1", "latest", "0x0")
		for i := 0; i < 10; i++ {
			update()
			if nodes["node1"].backend.BlockHeightZeroAboveThreshold() {
				require.True(t, bg.Consensus.IsBanned(nodes["node1"].backend))
				require.False(t, bg.Consensus.IsBanned(nodes["node2"].backend))
			} else {
				require.False(t, nodes["node2"].backend.BlockHeightZeroAboveThreshold())
				require.False(t, bg.Consensus.IsBanned(nodes["node1"].backend))
				require.False(t, bg.Consensus.IsBanned(nodes["node2"].backend))
			}
			addTimeToBackend(3 * time.Second)
		}
		require.True(t, bg.Consensus.IsBanned(nodes["node1"].backend))
		require.False(t, bg.Consensus.IsBanned(nodes["node2"].backend))
	})

	t.Run("Test Sliding Window Does not increase on non-zero block", func(t *testing.T) {
		reset()
		for i := 1; i < 10; i++ {
			if i%2 == 0 {
				overrideBlock("node1", "latest", "0x0")
			} else {
				overrideBlock("node1", "latest", "0x101")
			}
			update()
			require.Equal(t, uint(i/2), nodes["node1"].backend.GetBlockHeightZeroSlidingWindow().Count())
			require.Equal(t, uint(0), nodes["node2"].backend.GetBlockHeightZeroSlidingWindow().Count())

			addTimeToBackend(3 * time.Second)
		}
	})

	t.Run("Test Backend is Banned -> not banned as long as good blocks come -> ban", func(t *testing.T) {
		reset()
		overrideBlock("node1", "latest", "0x0")
		// Ban Node 1
		for i := 0; i < 20; i++ {
			update()
			if nodes["node1"].backend.BlockHeightZeroAboveThreshold() {
				require.True(t, bg.Consensus.IsBanned(nodes["node1"].backend))
				break
			}
			addTimeToBackend(1 * time.Second)
		}

		// Unban, and start seeing good blocks = no ban
		bg.Consensus.Unban(nodes["node1"].backend)
		overrideBlock("node1", "latest", "0x101")
		for i := 0; i < 5; i++ {
			update()
			require.False(t, bg.Consensus.IsBanned(nodes["node1"].backend))
			addTimeToBackend(1 * time.Second)
		}

		// See a bad block, and sliding window above threshold -> ban
		overrideBlock("node1", "latest", "0x0")
		update()
		if nodes["node1"].backend.BlockHeightZeroAboveThreshold() {
			require.True(t, bg.Consensus.IsBanned(nodes["node1"].backend))
		}
	})

	t.Run("Test that sliding window will return below threshold after time passes", func(t *testing.T) {
		reset()
		overrideBlock("node1", "latest", "0x0")
		for i := 0; i < 20; i++ {
			update()
			if nodes["node1"].backend.BlockHeightZeroAboveThreshold() {
				require.True(t, bg.Consensus.IsBanned(nodes["node1"].backend))
				require.False(t, bg.Consensus.IsBanned(nodes["node2"].backend))
			}
			addTimeToBackend(1 * time.Second)
		}
		addTimeToBackend(50 * time.Second)
		bg.Consensus.Unban(nodes["node1"].backend)

		update()
		require.False(t, nodes["node1"].backend.BlockHeightZeroAboveThreshold())
		require.False(t, bg.Consensus.IsBanned(nodes["node1"].backend))
	})
}
