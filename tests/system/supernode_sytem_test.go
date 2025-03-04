//go:build system_test

package system

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestSuperNode(t *testing.T) {
	// Initialize and reset chain
	sut.ResetChain(t)

	// Start the chain
	sut.StartChain(t)

	// Create CLI helper
	cli := NewLumeradCLI(t, sut, true)

	// Log node addresses
	t.Log("Retrieving addresses for all nodes:")
	for i := 0; i < sut.nodesCount; i++ {
		nodeName := fmt.Sprintf("node%d", i)
		address := cli.GetKeyAddr(nodeName)
		t.Logf("Node %d (%s) address: %s", i, nodeName, address)
	}

	// Create context for P2P operations
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Setup P2P services for all Lumera nodes
	t.Log("Setting up P2P services...")
	services, rqStores, err := SetupP2PServicesForNodes(t, sut, verbose, ctx)
	require.NoError(t, err)
	t.Logf("Successfully created %d P2P services", len(services))

	defer func() {
		// Cleanup RQ stores
		t.Log("Closing RQ stores...")
		for i, store := range rqStores {
			t.Logf("Closing store %d", i)
			store.Close()
		}

		// Cleanup data directories
		t.Log("Cleaning up test data directories...")
		if err := os.RemoveAll("./data"); err != nil {
			t.Logf("Warning: Failed to cleanup test data directory: %v", err)
		} else {
			t.Log("Successfully cleaned up test data directories")
		}
	}()

	// Wait for P2P network to stabilize
	t.Log("Waiting for P2P network to stabilize...")
	time.Sleep(10 * time.Second)

	// Very basic check - just log that services were created
	t.Logf("Test complete: Successfully created %d P2P services", len(services))
	require.Equal(t, sut.nodesCount, len(services), "Should have one P2P service per node")
}
