package client

import (
	"context"

	"google.golang.org/grpc"

	pb "github.com/LumeraProtocol/supernode/gen/supernode/action"
)

type Client struct {
	conn    *grpc.ClientConn
	service pb.ActionServiceClient
}

func NewClient(address string, opts ...Option) (*Client, error) {
	options := NewDefaultOptions()

	for _, opt := range opts {
		opt(options)
	}

	ctx, cancel := context.WithTimeout(context.Background(), options.Timeout)
	defer cancel()

	conn, err := grpc.DialContext(ctx, address, grpc.WithInsecure()) // Example
	if err != nil {
		return nil, err
	}

	return &Client{
		conn:    conn,
		service: pb.NewActionServiceClient(conn),
	}, nil
}
