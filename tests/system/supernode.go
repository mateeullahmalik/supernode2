package system

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/LumeraProtocol/supernode/common/storage/rqstore"
	"github.com/LumeraProtocol/supernode/p2p"
	"github.com/LumeraProtocol/supernode/pkg/lumera"
)

// SetupP2PServicesForNodes creates a P2P service for each Lumera node in the blockchain
// Each service uses the node's blockchain address as its Lumera ID
// Ports start at 8000 and increment sequentially
func SetupP2PServicesForNodes(t *testing.T, sut *SystemUnderTest, verbose bool, ctx context.Context) ([]p2p.Client, []*rqstore.SQLiteRQStore, error) {
	t.Helper()

	// Get all node addresses
	cli := NewLumeradCLI(t, sut, verbose)
	var nodeAddresses []string

	t.Log("Retrieving addresses for all nodes:")
	for i := 0; i < sut.nodesCount; i++ {
		nodeName := fmt.Sprintf("node%d", i)
		address := cli.GetKeyAddr(nodeName)
		nodeAddresses = append(nodeAddresses, address)
		t.Logf("Node %d (%s) address: %s", i, nodeName, address)
	}

	// Setup P2P services using the node addresses
	var services []p2p.Client
	var rqStores []*rqstore.SQLiteRQStore

	t.Log("Setting up P2P services for each node...")

	// Create a Lumera client config
	var nodeConfigs lumera.LumeraClientConfig

	// First pass: populate the node configs
	for i, address := range nodeAddresses {
		port := 8000 + i
		nodeConfigs = append(nodeConfigs, struct {
			Address  string
			LumeraID string
		}{
			Address:  fmt.Sprintf("127.0.0.1:%d", port),
			LumeraID: address,
		})
	}

	// Create the mock Lumera client with all node configs
	mockClient := lumera.NewLumeraClient(nodeConfigs)

	// Second pass: create and start the actual P2P services
	for i, address := range nodeAddresses {
		port := 8000 + i

		// Create data directory for the node
		dataDir := fmt.Sprintf("./data/p2p_node%d", i)
		if err := os.MkdirAll(dataDir, 0755); err != nil {
			return nil, nil, fmt.Errorf("failed to create data directory for node %d: %v", i, err)
		}

		// Collect addresses from previous nodes as bootstrap addresses
		bootstrapAddresses := make([]string, i)
		for j := 0; j < i; j++ {
			bootstrapAddresses[j] = nodeConfigs[j].Address
		}

		p2pConfig := &p2p.Config{
			ListenAddress: "127.0.0.1",
			Port:          port,
			DataDir:       dataDir,
			ID:            address,
			BootstrapIPs:  strings.Join(bootstrapAddresses, ","),
		}

		// Initialize SQLite RQ store for each node
		rqStoreFile := filepath.Join(dataDir, "rqstore.db")
		if err := os.MkdirAll(filepath.Dir(rqStoreFile), 0755); err != nil {
			return nil, nil, fmt.Errorf("failed to create rqstore directory for node %d: %v", i, err)
		}

		rqStore, err := rqstore.NewSQLiteRQStore(rqStoreFile)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to create rqstore for node %d: %v", i, err)
		}
		rqStores = append(rqStores, rqStore)

		t.Logf("Creating P2P service for node %d with address %s on port %d", i, address, port)
		service, err := p2p.New(ctx, p2pConfig, mockClient, nil, rqStore, nil, nil)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to create p2p service for node %d: %v", i, err)
		}

		// Start P2P service in a goroutine
		go func(nodeID int, rqStore *rqstore.SQLiteRQStore) {
			defer rqStore.Close()
			t.Logf("Starting P2P service for node %d", nodeID)
			if err := service.Run(ctx); err != nil && err != context.Canceled {
				t.Logf("Node %d P2P service failed: %v", nodeID, err)
			}
		}(i, rqStore)

		services = append(services, service)

		// Give node time to start up
		time.Sleep(1 * time.Second)
	}

	t.Log("All P2P services created and started")

	// Give extra time for all nodes to connect
	time.Sleep(2 * time.Second)

	return services, rqStores, nil
}
