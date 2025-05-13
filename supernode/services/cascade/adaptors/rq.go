package adaptors

import (
	"context"

	"github.com/LumeraProtocol/supernode/pkg/codec"
)

// CodecService defines the interface for RaptorQ encoding of input data.
//
//go:generate mockgen -destination=mocks/rq_mock.go -package=cascadeadaptormocks -source=rq.go
type CodecService interface {
	EncodeInput(ctx context.Context, taskID string, data []byte) (EncodeResult, error)
}

// EncodeResult represents the outcome of encoding the input data.
type EncodeResult struct {
	SymbolsDir string
	Metadata   codec.Layout
}

// codecImpl is the default implementation using the real codec service.
type codecImpl struct {
	codec codec.Codec
}

// NewCodecService creates a new production instance of CodecService.
func NewCodecService(codec codec.Codec) CodecService {
	return &codecImpl{codec: codec}
}

// EncodeInput encodes the provided data and returns symbols and metadata.
func (c *codecImpl) EncodeInput(ctx context.Context, taskID string, data []byte) (EncodeResult, error) {
	resp, err := c.codec.Encode(ctx, codec.EncodeRequest{
		TaskID: taskID,
		Data:   data,
	})
	if err != nil {
		return EncodeResult{}, err
	}

	return EncodeResult{
		SymbolsDir: resp.SymbolsDir,
		Metadata:   resp.Metadata,
	}, nil
}
