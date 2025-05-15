package task

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/LumeraProtocol/supernode/pkg/logtrace"
	"github.com/LumeraProtocol/supernode/sdk/adapters/lumera"
	"github.com/LumeraProtocol/supernode/sdk/adapters/supernodeservice"
	"github.com/LumeraProtocol/supernode/sdk/event"
	"github.com/LumeraProtocol/supernode/sdk/net"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc/health/grpc_health_v1"
)

const (
	registrationTimeout = 120 * time.Second // Timeout for registration requests
	connectionTimeout   = 10 * time.Second  // Timeout for connection requests
)

type CascadeTask struct {
	BaseTask
	filePath string
	actionId string
}

// NewCascadeTask creates a new CascadeTask using a BaseTask plus cascade-specific parameters
func NewCascadeTask(base BaseTask, filePath string, actionId string) *CascadeTask {
	return &CascadeTask{
		BaseTask: base,
		filePath: filePath,
		actionId: actionId,
	}
}

// Run executes the full cascade‐task lifecycle.
func (t *CascadeTask) Run(ctx context.Context) error {
	t.LogEvent(ctx, event.TaskStarted, "Running cascade task", nil)

	// Use the already validated Action directly to get the height
	supernodes, err := t.fetchSupernodes(ctx, t.Action.Height)
	if err != nil {
		t.logger.Error(ctx, "Task failed", "taskID", t.TaskID, "actionID", t.ActionID, "error", err)
		t.EmitEvent(ctx, event.TaskProgressSupernodesUnavailable, event.EventData{
			event.KeyError: err.Error(),
		})
		t.EmitEvent(ctx, event.TaskFailed, event.EventData{
			event.KeyError: err.Error(),
		})
		return err
	}
	t.LogEvent(ctx, event.TaskProgressSupernodesFound, "Supernodes found.", event.EventData{
		event.KeyCount: len(supernodes),
	})

	if err := t.registerWithSupernodes(ctx, supernodes); err != nil {
		t.logger.Error(ctx, "Task failed", "taskID", t.TaskID, "actionID", t.ActionID, "error", err)
		t.EmitEvent(ctx, event.TaskProgressRegistrationFailure, event.EventData{
			event.KeyError: err.Error(),
		})
		t.EmitEvent(ctx, event.TaskFailed, event.EventData{
			event.KeyError: err.Error(),
		})
		return err
	}

	t.LogEvent(ctx, event.TaskCompleted, "Cascade task completed successfully", nil)

	return nil
}

func (t *CascadeTask) fetchSupernodes(ctx context.Context, height int64) (lumera.Supernodes, error) {
	sns, err := t.client.GetSupernodes(ctx, height)
	if err != nil {
		return nil, fmt.Errorf("fetch supernodes: %w", err)
	}
	t.logger.Info(ctx, "Supernodes fetched", "count", len(sns))

	if len(sns) == 0 {
		return nil, errors.New("no supernodes found")
	}

	if len(sns) > 10 {
		sns = sns[:10]
	}

	// Keep only SERVING nodes (done in parallel – keeps latency flat)
	healthy := make(lumera.Supernodes, 0, len(sns))
	eg, ctx := errgroup.WithContext(ctx)
	mu := sync.Mutex{}

	for _, sn := range sns {
		sn := sn
		eg.Go(func() error {
			if t.isServing(ctx, sn) {
				mu.Lock()
				healthy = append(healthy, sn)
				mu.Unlock()
			}
			return nil
		})
	}
	if err := eg.Wait(); err != nil {
		return nil, fmt.Errorf("health-check goroutines: %w", err)
	}

	if len(healthy) == 0 {
		return nil, errors.New("no healthy supernodes found")
	}
	t.logger.Info(ctx, "Healthy supernodes", "count", len(healthy))

	return healthy, nil
}

// isServing pings the super-node once with a short timeout.
func (t *CascadeTask) isServing(parent context.Context, sn lumera.Supernode) bool {
	ctx, cancel := context.WithTimeout(parent, connectionTimeout)
	defer cancel()

	client, err := net.NewClientFactory(ctx, t.logger, t.keyring, net.FactoryConfig{
		LocalCosmosAddress: t.config.Account.LocalCosmosAddress,
	}).CreateClient(ctx, sn)
	if err != nil {
		logtrace.Info(ctx, "Failed to create client for supernode", logtrace.Fields{
			logtrace.FieldMethod: "isServing"})
		return false
	}
	defer client.Close(ctx)

	resp, err := client.HealthCheck(ctx)
	return err == nil && resp.Status == grpc_health_v1.HealthCheckResponse_SERVING
}

func (t *CascadeTask) registerWithSupernodes(ctx context.Context, supernodes lumera.Supernodes) error {
	factoryCfg := net.FactoryConfig{
		LocalCosmosAddress: t.config.Account.LocalCosmosAddress,
	}
	clientFactory := net.NewClientFactory(ctx, t.logger, t.keyring, factoryCfg)

	req := &supernodeservice.CascadeSupernodeRegisterRequest{
		FilePath: t.filePath,
		ActionID: t.ActionID,
		TaskId:   t.TaskID,
	}

	var lastErr error
	for idx, sn := range supernodes {
		if err := t.attemptRegistration(ctx, idx, sn, clientFactory, req); err != nil {
			lastErr = err
			continue
		}
		return nil // success
	}

	return fmt.Errorf("failed to upload to all supernodes: %w", lastErr)
}

func (t *CascadeTask) attemptRegistration(ctx context.Context, index int, sn lumera.Supernode, factory *net.ClientFactory, req *supernodeservice.CascadeSupernodeRegisterRequest) error {
	t.LogEvent(ctx, event.TaskProgressRegistrationInProgress, "attempting registration with supernode", event.EventData{
		event.KeySupernode:        sn.GrpcEndpoint,
		event.KeySupernodeAddress: sn.CosmosAddress,
		event.KeyIteration:        index + 1,
	})

	client, err := factory.CreateClient(ctx, sn)
	if err != nil {
		return fmt.Errorf("create client %s: %w", sn.CosmosAddress, err)
	}
	defer client.Close(ctx)

	uploadCtx, cancel := context.WithTimeout(ctx, registrationTimeout)
	defer cancel()

	req.EventLogger = func(ctx context.Context, evt event.EventType, msg string, data event.EventData) {
		t.LogEvent(ctx, evt, msg, data)
	}
	resp, err := client.RegisterCascade(uploadCtx, req)
	if err != nil {
		return fmt.Errorf("upload to %s: %w", sn.CosmosAddress, err)
	}
	if !resp.Success {
		return fmt.Errorf("upload rejected by %s: %s", sn.CosmosAddress, resp.Message)
	}

	// Use txhash directly without cleaning
	t.LogEvent(ctx, event.TxhasReceived, "txhash received", event.EventData{
		event.KeyTxHash:    resp.TxHash,
		event.KeySupernode: sn.CosmosAddress,
	})

	t.logger.Info(ctx, "upload OK", "taskID", t.TaskID, "address", sn.CosmosAddress)
	return nil
}

// logEvent writes a structured log entry **and** emits the SDK event.
