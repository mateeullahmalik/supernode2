package client

import "time"

const (
	defaultTimeout = 10 * time.Second
)

// Options holds configuration options for the client
type Options struct {
	Timeout time.Duration
}

// Option defines the functional option signature
type Option func(options *Options)

func NewDefaultOptions() *Options {
	return &Options{
		Timeout: defaultTimeout,
	}
}

// WithTimeout sets a custom timeout for gRPC connection
func WithTimeout(d time.Duration) Option {
	return func(o *Options) {
		o.Timeout = d
	}
}
