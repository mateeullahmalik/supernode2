package lumera

import (
	"context"
	"fmt"
)

// TODO: Implement the LumeraClient interface
// This is a mock client that implements the LumeraClient interface just for demo purposes for use in p2p service

type Client interface {
	MasterNodesExtra(ctx context.Context) (MasterNodes, error)
	MasterNodesTop(ctx context.Context) (MasterNodes, error)
	Sign(ctx context.Context, data []byte, pastelID string, passphrase string, algorithm string) (signature string, err error)
	Verify(ctx context.Context, data []byte, signature string, pastelID string, algorithm string) (bool, error)
}

type MasterNode struct {
	ExtKey     string
	ExtAddress string
	ExtP2P     string
}

type MasterNodes []MasterNode

type LumeraClient struct {
	nodes []MasterNode // list of nodes in our demo network
}

// NewLumeraClient creates a new mock client with pre-configured nodes
func NewLumeraClient(nodeAddresses []string) *LumeraClient {
	nodes := make([]MasterNode, len(nodeAddresses))
	for i, addr := range nodeAddresses {
		// Create a unique PastelID for each node
		pastelID := fmt.Sprintf("jXY%d", i+1)

		nodes[i] = MasterNode{
			ExtKey:     pastelID, // PastelID of the node
			ExtAddress: addr,     // IP:Port for general communication
			ExtP2P:     addr,     // Same address for P2P - in real network this might be different
		}
	}

	return &LumeraClient{
		nodes: nodes,
	}
}

func (m *LumeraClient) MasterNodesExtra(ctx context.Context) (MasterNodes, error) {
	return m.nodes, nil
}

func (m *LumeraClient) MasterNodesTop(ctx context.Context) (MasterNodes, error) {
	return m.nodes, nil
}

func (m *LumeraClient) Sign(_ context.Context, _ []byte, _ string, _ string, _ string) (string, error) {
	// For demo, just return a dummy signature
	return "dummy-signature", nil
}

func (m *LumeraClient) Verify(_ context.Context, _ []byte, _ string, _ string, _ string) (bool, error) {
	// For demo, always verify as true
	return true, nil
}
