package cmd

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// mockService implements the service interface
type mockService struct {
	name    string
	runFunc func(ctx context.Context) error
}

func (m *mockService) Run(ctx context.Context) error {
	return m.runFunc(ctx)
}

func TestRunServices_AllSuccessful(t *testing.T) {
	s1 := &mockService{name: "s1", runFunc: func(ctx context.Context) error {
		return nil
	}}
	s2 := &mockService{name: "s2", runFunc: func(ctx context.Context) error {
		return nil
	}}

	err := RunServices(context.Background(), s1, s2)
	assert.NoError(t, err)
}

func TestRunServices_OneFails(t *testing.T) {
	s1 := &mockService{name: "s1", runFunc: func(ctx context.Context) error {
		return errors.New("s1 failed")
	}}
	s2 := &mockService{name: "s2", runFunc: func(ctx context.Context) error {
		return nil
	}}

	err := RunServices(context.Background(), s1, s2)
	assert.Error(t, err)
	assert.Equal(t, "s1 failed", err.Error())
}

func TestRunServices_MultipleFail(t *testing.T) {
	s1 := &mockService{name: "s1", runFunc: func(ctx context.Context) error {
		return errors.New("s1 failed")
	}}
	s2 := &mockService{name: "s2", runFunc: func(ctx context.Context) error {
		return errors.New("s2 failed")
	}}

	err := RunServices(context.Background(), s1, s2)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed") // may not be deterministic which one returns
}

func TestRunServices_WithCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	s1 := &mockService{name: "s1", runFunc: func(ctx context.Context) error {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(2 * time.Second):
			return nil
		}
	}}
	s2 := &mockService{name: "s2", runFunc: func(ctx context.Context) error {
		cancel() // cancel context early
		return nil
	}}

	err := RunServices(ctx, s1, s2)
	assert.Error(t, err)
	assert.Equal(t, context.Canceled, err)
}
