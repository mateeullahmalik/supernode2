package action

import (
	"context"
	"fmt"

	"google.golang.org/grpc"

	lumeraaction "github.com/LumeraProtocol/lumera/x/action/types"
)

type Client struct {
	conn          *grpc.ClientConn
	actionService lumeraaction.QueryClient
}

type Service interface {
	GetAction(ctx context.Context, r GetActionRequest) (Action, error)
}

func NewClient(serverAddr string) (Service, error) {
	conn, err := grpc.Dial(serverAddr, grpc.WithInsecure(), grpc.WithBlock())
	if err != nil {
		return nil, fmt.Errorf("failed to connect to gRPC server: %w", err)
	}

	return &Client{
		conn:          conn,
		actionService: lumeraaction.NewQueryClient(conn),
	}, nil
}

func (c *Client) Close() {
	c.conn.Close()
}
