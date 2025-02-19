package client

import (
	"context"
	"errors"

	pb "github.com/LumeraProtocol/supernode/gen/supernode/action"
	"github.com/LumeraProtocol/supernode/pkg/logtrace"
	"github.com/LumeraProtocol/supernode/pkg/net"
)

type HealthCheckResponse struct {
	Status string `json:"status"`
}

func (c *Client) CheckHealth(ctx context.Context) (HealthCheckResponse, error) {
	ctx = net.AddCorrelationID(ctx)

	fields := logtrace.Fields{
		logtrace.FieldMethod: "CheckHealth",
		logtrace.FieldModule: logtrace.ValueActionSDK,
	}
	logtrace.Info(ctx, "performing health check", fields)

	res, err := c.service.CheckHealth(ctx, &pb.GetHealthCheckRequest{})
	if err != nil {
		fields[logtrace.FieldError] = err.Error()
		logtrace.Error(ctx, "health check failed", fields)
		return HealthCheckResponse{}, err
	}

	if res == nil {
		logtrace.Warn(ctx, "action server returned nil response", fields)
		return HealthCheckResponse{}, errors.New("received nil response from action-server")
	}
	fields[logtrace.FieldStatus] = res.Status

	logtrace.Info(ctx, "health check successful", fields)
	return HealthCheckResponse{Status: res.Status}, nil
}
