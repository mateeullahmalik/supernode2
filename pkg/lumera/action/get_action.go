package action

import (
	"context"
	"fmt"

	lumeraaction "github.com/LumeraProtocol/lumera/x/action/types"
	"github.com/LumeraProtocol/supernode/pkg/logtrace"
	"github.com/LumeraProtocol/supernode/pkg/net"
)

type ActionState int32
type ActionType string

const (
	ActionStateUnspecified ActionState = 0
	ActionStatePending     ActionState = 1
	ActionStateDone        ActionState = 2
	ActionStateApproved    ActionState = 3
	ActionStateRejected    ActionState = 4
	ActionStateFailed      ActionState = 5

	CascadeActionType ActionType = "cascade"
	SenseActionType   ActionType = "sense"
)

type GetActionRequest struct {
	ActionID string
	Type     ActionType
}

type Action struct {
	Creator        string
	ActionID       string
	Metadata       *Metadata
	Price          string
	ExpirationTime string
	BlockHeight    int64
	State          ActionState
}

type Metadata struct {
	SenseMetadata
	CascadeMetadata
}

type SenseMetadata struct {
	DataHash             string
	DdAndFingerprintsIc  int32
	DdAndFingerprintsMax int32
	DdAndFingerprintsIds []string
}

type CascadeMetadata struct {
	DataHash string
	FileName string
	RqIds    []string
	RqMax    int32
	RqIc     int32
	RqOti    []byte
}

func (c *Client) GetAction(ctx context.Context, r GetActionRequest) (Action, error) {
	ctx = net.AddCorrelationID(ctx)

	fields := logtrace.Fields{
		logtrace.FieldMethod:  "GetAction",
		logtrace.FieldModule:  logtrace.ValueLumeraSDK,
		logtrace.FieldRequest: r,
	}
	logtrace.Info(ctx, "fetching action details", fields)

	resp, err := c.actionService.GetAction(ctx, &lumeraaction.QueryGetActionRequest{
		ActionID: r.ActionID,
	})
	if err != nil {
		fields[logtrace.FieldError] = err.Error()
		logtrace.Error(ctx, "failed to fetch action detail", fields)
		return Action{}, fmt.Errorf("failed to fetch action: %w", err)
	}

	logtrace.Info(ctx, "successfully fetched the action details", fields)

	return toAction(resp), nil
}

func toAction(ac *lumeraaction.QueryGetActionResponse) Action {
	var meta *Metadata
	switch x := ac.Action.Metadata.MetadataType.(type) {
	case *lumeraaction.Metadata_SenseMetadata:
		meta = &Metadata{
			SenseMetadata: SenseMetadata{
				DataHash:             x.SenseMetadata.DataHash,
				DdAndFingerprintsIc:  x.SenseMetadata.DdAndFingerprintsIc,
				DdAndFingerprintsMax: x.SenseMetadata.DdAndFingerprintsMax,
				DdAndFingerprintsIds: x.SenseMetadata.DdAndFingerprintsIds,
			},
		}
	case *lumeraaction.Metadata_CascadeMetadata:
		meta = &Metadata{
			CascadeMetadata: CascadeMetadata{
				DataHash: x.CascadeMetadata.DataHash,
				FileName: x.CascadeMetadata.FileName,
				RqIds:    x.CascadeMetadata.RqIds,
				RqMax:    x.CascadeMetadata.RqMax,
				RqIc:     x.CascadeMetadata.RqIc,
				RqOti:    x.CascadeMetadata.RqOti,
			},
		}
	default:
		meta = nil
	}

	return Action{
		Creator:        ac.Action.Creator,
		ActionID:       ac.Action.ActionID,
		Metadata:       meta,
		Price:          ac.Action.Price,
		ExpirationTime: ac.Action.ExpirationTime,
		BlockHeight:    ac.Action.BlockHeight,
		State:          ActionState(ac.Action.State),
	}
}

func (m *Metadata) GetCascadeMetadata() *CascadeMetadata {
	if m != nil {
		return &m.CascadeMetadata
	}

	return nil
}

func (m *Metadata) GetSenseMetadata() *SenseMetadata {
	if m != nil {
		return &m.SenseMetadata
	}

	return nil
}
