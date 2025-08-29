package cascade

import (
	"context"
	"errors"
	"testing"

	pb "github.com/LumeraProtocol/supernode/v2/gen/supernode/action/cascade"
	"github.com/LumeraProtocol/supernode/v2/supernode/services/cascade"
	cascademocks "github.com/LumeraProtocol/supernode/v2/supernode/services/cascade/mocks"

	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func TestRegister_Success(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockTask := cascademocks.NewMockCascadeTask(ctrl)
	mockFactory := cascademocks.NewMockCascadeServiceFactory(ctrl)

	// Expect Register to be called with any input, respond via callback
	mockTask.EXPECT().Register(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(
		func(ctx context.Context, req *cascade.RegisterRequest, send func(*cascade.RegisterResponse) error) error {
			return send(&cascade.RegisterResponse{
				EventType: 1,
				Message:   "registration successful",
				TxHash:    "tx123",
			})
		},
	).Times(1)

	mockFactory.EXPECT().NewCascadeRegistrationTask().Return(mockTask).Times(1)

	server := NewCascadeActionServer(mockFactory)

	stream := &mockStream{
		ctx: context.Background(),
		request: []*pb.RegisterRequest{
			{RequestType: &pb.RegisterRequest_Chunk{Chunk: &pb.DataChunk{Data: []byte("abc123")}}},
			{RequestType: &pb.RegisterRequest_Metadata{
				Metadata: &pb.Metadata{TaskId: "t1", ActionId: "a1"},
			}},
		},
	}

	err := server.Register(stream)
	assert.NoError(t, err)
	assert.Len(t, stream.sent, 1)
	assert.Equal(t, "registration successful", stream.sent[0].Message)
	assert.Equal(t, "tx123", stream.sent[0].TxHash)
}

func TestRegister_Error_NoMetadata(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockFactory := cascademocks.NewMockCascadeServiceFactory(ctrl)
	server := NewCascadeActionServer(mockFactory)

	stream := &mockStream{
		ctx: context.Background(),
		request: []*pb.RegisterRequest{
			{RequestType: &pb.RegisterRequest_Chunk{Chunk: &pb.DataChunk{Data: []byte("abc123")}}},
		},
	}

	err := server.Register(stream)
	assert.EqualError(t, err, "no metadata received")
}

func TestRegister_Error_TaskFails(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockTask := cascademocks.NewMockCascadeTask(ctrl)
	mockFactory := cascademocks.NewMockCascadeServiceFactory(ctrl)

	mockTask.EXPECT().Register(gomock.Any(), gomock.Any(), gomock.Any()).Return(errors.New("task failed")).Times(1)
	mockFactory.EXPECT().NewCascadeRegistrationTask().Return(mockTask).Times(1)

	server := NewCascadeActionServer(mockFactory)

	stream := &mockStream{
		ctx: context.Background(),
		request: []*pb.RegisterRequest{
			{RequestType: &pb.RegisterRequest_Chunk{Chunk: &pb.DataChunk{Data: []byte("abc123")}}},
			{RequestType: &pb.RegisterRequest_Metadata{
				Metadata: &pb.Metadata{TaskId: "t1", ActionId: "a1"},
			}},
		},
	}

	err := server.Register(stream)
	assert.EqualError(t, err, "registration failed: task failed")
}
