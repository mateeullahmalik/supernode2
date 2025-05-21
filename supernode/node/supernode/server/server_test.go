package server

import (
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health/grpc_health_v1"
	"testing"
)

// --- Mock service implementing server.service ---
type mockService struct{}

func (m *mockService) Desc() *grpc.ServiceDesc {
	return &grpc.ServiceDesc{
		ServiceName: "test.Service",
		HandlerType: (*interface{})(nil),
		Methods:     []grpc.MethodDesc{},
		Streams:     []grpc.StreamDesc{},
	}
}

func TestNewServer_WithValidConfig(t *testing.T) {
	ctl := gomock.NewController(t)
	mockKeyring := NewMockKeyring(ctl)

	cfg := NewConfig()
	s, err := New(cfg, "supernode-test", mockKeyring, &mockService{})
	assert.NoError(t, err)
	assert.NotNil(t, s)
}

func TestNewServer_WithNilConfig(t *testing.T) {
	ctl := gomock.NewController(t)
	mockKeyring := NewMockKeyring(ctl)

	s, err := New(nil, "supernode-test", mockKeyring)
	assert.Nil(t, s)
	assert.EqualError(t, err, "config is nil")
}

func TestSetServiceStatusAndClose(t *testing.T) {
	ctl := gomock.NewController(t)
	mockKeyring := NewMockKeyring(ctl)

	cfg := NewConfig()
	s, _ := New(cfg, "test", mockKeyring, &mockService{})
	_ = s.setupGRPCServer()

	s.SetServiceStatus("test.Service", grpc_health_v1.HealthCheckResponse_SERVING)
	s.Close()

	// No assertion — success is no panic / crash on shutdown
}
