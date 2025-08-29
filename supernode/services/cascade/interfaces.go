package cascade

import (
	"context"
)

// CascadeServiceFactory defines an interface to create cascade tasks
//
//go:generate mockgen -destination=mocks/cascade_interfaces_mock.go -package=cascademocks -source=interfaces.go
type CascadeServiceFactory interface {
	NewCascadeRegistrationTask() CascadeTask
}

// CascadeTask interface defines operations for cascade registration and data management
type CascadeTask interface {
	Register(ctx context.Context, req *RegisterRequest, send func(resp *RegisterResponse) error) error
	Download(ctx context.Context, req *DownloadRequest, send func(resp *DownloadResponse) error) error
	CleanupDownload(ctx context.Context, actionID string) error
}
