package lumera

import (
	"context"
	"fmt"
	"net"
	"net/url"
	"strings"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/connectivity"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/keepalive"
)

const (
	defaultLumeraPort = "9090"

	keepaliveTime    = 6 * time.Minute
	keepaliveTimeout = 10 * time.Second
	connectWaitTime  = 10 * time.Second
)

// Connection defines the interface for a client connection.
type Connection interface {
	Close() error
	GetConn() *grpc.ClientConn
}

// grpcConnection wraps a gRPC connection.
type grpcConnection struct {
	conn *grpc.ClientConn
}

// newGRPCConnection creates a new gRPC connection.
// TLS is chosen strictly by scheme (https/grpcs → TLS; http/grpc or no scheme → plaintext).
// When no port is provided, it will try :9090 first, then fall back to :443 using the same TLS setting.
func newGRPCConnection(ctx context.Context, rawAddr string) (Connection, error) {
	hostPort, useTLS, serverName, err := normaliseAddr(rawAddr)
	if err != nil {
		return nil, err
	}

	var creds credentials.TransportCredentials
	if useTLS {
		creds = credentials.NewClientTLSFromCert(nil, serverName)
	} else {
		creds = insecure.NewCredentials()
	}

	// First attempt (normalized address)
	conn, err := createGRPCConnection(ctx, hostPort, creds)
	if err != nil {
		// If user did not provide an explicit port, try fallback :443 with same TLS decision
		if !hasExplicitPort(rawAddr) {
			altHostPort := net.JoinHostPort(serverName, "443")
			conn2, err2 := createGRPCConnection(ctx, altHostPort, creds)
			if err2 == nil {
				return &grpcConnection{conn: conn2}, nil
			}
			return nil, fmt.Errorf("failed to connect to gRPC server (attempts %s then %s): %v | %v", hostPort, altHostPort, err, err2)
		}
		return nil, fmt.Errorf("failed to connect to gRPC server: %w", err)
	}

	return &grpcConnection{conn: conn}, nil
}

// Address handling examples (initial attempt; fallback handled by caller):
//
//	https://grpc.testnet.lumera.io           → TLS, initial host = grpc.testnet.lumera.io:9090 (fallback :443)
//	grpcs://grpc.node9x.com:7443             → TLS, host = grpc.node9x.com:7443
//	grpc.node9x.com:443                      → plaintext (no scheme), host = grpc.node9x.com:443 (no TLS inference)
//	grpc.node9x.com:9090                     → plaintext, host = grpc.node9x.com:9090
//	grpc.testnet.lumera.io                   → plaintext, initial host = grpc.testnet.lumera.io:9090 (fallback :443)
func normaliseAddr(raw string) (hostPort string, useTLS bool, serverName string, err error) {
	// If scheme present, parse as URL first.
	if strings.Contains(raw, "://") {
		u, err := url.Parse(raw)
		if err != nil {
			return "", false, "", fmt.Errorf("parse address %q: %w", raw, err)
		}

		host := u.Hostname()
		port := u.Port()
		switch u.Scheme {
		case "https", "grpcs":
			useTLS = true
			if port == "" {
				// Do not assume 443; prefer 9090 as initial attempt. Fallback to 443 is handled by the caller.
				port = defaultLumeraPort
			}
		case "http", "grpc":
			useTLS = false
			if port == "" {
				port = defaultLumeraPort
			}
		default:
			return "", false, "", fmt.Errorf("unsupported scheme %q in %q", u.Scheme, raw)
		}
		return net.JoinHostPort(host, port), useTLS, host, nil
	}

	// No scheme: split host[:port].
	host, port, splitErr := net.SplitHostPort(raw)
	if splitErr != nil {
		// No port given → prefer :9090, plaintext. Fallback to :443 handled by the caller.
		return net.JoinHostPort(raw, defaultLumeraPort), false, raw, nil
	}

	// Port explicit: do not infer TLS based on port value; no scheme means plaintext.
	return net.JoinHostPort(host, port), false, host, nil
}

// hasExplicitPort reports whether the raw address string contains an explicit port.
func hasExplicitPort(raw string) bool {
	if strings.Contains(raw, "://") {
		if u, err := url.Parse(raw); err == nil {
			return u.Port() != ""
		}
		return false
	}
	if _, _, err := net.SplitHostPort(raw); err == nil {
		return true
	}
	return false
}

// createGRPCConnection creates a gRPC connection with keepalive
func createGRPCConnection(ctx context.Context, hostPort string, creds credentials.TransportCredentials) (*grpc.ClientConn, error) {
	opts := []grpc.DialOption{
		grpc.WithTransportCredentials(creds),
		grpc.WithKeepaliveParams(keepalive.ClientParameters{
			Time:                keepaliveTime,
			Timeout:             keepaliveTimeout,
			PermitWithoutStream: false,
		}),
	}

	// Establish client connection (non-blocking) then wait until READY.
	conn, err := grpc.NewClient(hostPort, opts...)
	if err != nil {
		return nil, err
	}

	// Start connection attempts and wait for readiness with a bounded timeout.
	conn.Connect()

	// Use provided context deadline if present; otherwise apply a default.
	var cancel context.CancelFunc = func() {}
	if _, ok := ctx.Deadline(); !ok {
		ctx, cancel = context.WithTimeout(ctx, connectWaitTime)
	}
	defer cancel()

	for {
		state := conn.GetState()
		switch state {
		case connectivity.Ready:
			return conn, nil
		case connectivity.Shutdown:
			conn.Close()
			return nil, fmt.Errorf("grpc connection is shutdown")
		case connectivity.TransientFailure:
			conn.Close()
			return nil, fmt.Errorf("grpc connection is in transient failure")
		default:
			// Idle or Connecting: wait for a state change or timeout
			if !conn.WaitForStateChange(ctx, state) {
				conn.Close()
				return nil, fmt.Errorf("timeout waiting for grpc connection readiness")
			}
		}
	}
}

// Close closes the gRPC connection.
func (c *grpcConnection) Close() error {
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

// GetConn returns the underlying gRPC connection.
func (c *grpcConnection) GetConn() *grpc.ClientConn {
	return c.conn
}
