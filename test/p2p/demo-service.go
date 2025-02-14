package p2pdemo

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/LumeraProtocol/supernode/common/storage/rqstore"
	"github.com/LumeraProtocol/supernode/p2p"
	"github.com/LumeraProtocol/supernode/pkg/lumera"
	"github.com/gorilla/mux"
)

// P2PDebugService provides basic debug endpoints for P2P testing
type P2PDebugService struct {
	httpServer *http.Server
	p2pClient  p2p.Client
}

func NewP2PDebugService(p2pClient p2p.Client, port int) *P2PDebugService {
	service := &P2PDebugService{
		p2pClient: p2pClient,
	}

	router := mux.NewRouter()

	// Only keep essential endpoints
	router.HandleFunc("/p2p/stats", service.p2pStats).Methods(http.MethodGet)
	router.HandleFunc("/p2p", service.p2pStore).Methods(http.MethodPost)
	router.HandleFunc("/p2p/{key}", service.p2pRetrieve).Methods(http.MethodGet)
	router.HandleFunc("/health", service.p2pHealth).Methods(http.MethodGet)

	service.httpServer = &http.Server{
		Addr:    fmt.Sprintf(":%d", port),
		Handler: router,
	}

	return service
}

func (s *P2PDebugService) Run(ctx context.Context) error {
	go func() {
		if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("HTTP server error: %v", err)
		}
	}()

	<-ctx.Done()
	return s.httpServer.Shutdown(context.Background())
}

// Essential handlers
func (s *P2PDebugService) p2pStats(w http.ResponseWriter, r *http.Request) {
	stats, err := s.p2pClient.Stats(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(stats)
}

func (s *P2PDebugService) p2pStore(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Value []byte `json:"value"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	key, err := s.p2pClient.Store(r.Context(), req.Value, 0)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]string{"key": key})
}

func (s *P2PDebugService) p2pRetrieve(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	key := vars["key"]

	value, err := s.p2pClient.Retrieve(r.Context(), key)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"key":   key,
		"value": value,
	})
}

func (s *P2PDebugService) p2pHealth(w http.ResponseWriter, r *http.Request) {
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})

}

func RunP2PDemo() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Setup node addresses and their corresponding Lumera IDs
	nodeConfigs := []struct {
		address  string
		lumeraID string
	}{
		{
			address:  "127.0.0.1:9000",
			lumeraID: "lumera1xdxm4tytunhhq5hs45nwxds3h73qu4hf9nnjhr",
		},
		{
			address:  "127.0.0.1:9001",
			lumeraID: "lumera1xdxm4tytunhhq5hs45nwxds3h73qu4hf9nnjhr",
		},
		{
			address:  "127.0.0.1:9002",
			lumeraID: "lumera1gx4a82h6fq8jhkn08a5k55k5a3e2ygktujtxca",
		},
	}

	// Get all addresses for the mock client
	var nodeAddresses []string
	for _, config := range nodeConfigs {
		nodeAddresses = append(nodeAddresses, config.address)
	}

	// Create and start nodes
	for i, config := range nodeConfigs {
		mockClient := lumera.NewLumeraClient(nodeAddresses)

		// Create data directory for the node
		dataDir := fmt.Sprintf("./data/node%d", i)
		if err := os.MkdirAll(dataDir, 0755); err != nil {
			log.Fatalf("Failed to create data directory for node %d: %v", i, err)
		}

		p2pConfig := &p2p.Config{
			ListenAddress: "127.0.0.1",
			Port:          9000 + i,
			DataDir:       dataDir,
			ID:            config.lumeraID,
			BootstrapIPs:  strings.Join(nodeAddresses[:i], ","),
		}

		// Initialize SQLite RQ store for each node
		rqStoreFile := filepath.Join(dataDir, "rqstore.db")
		if err := os.MkdirAll(filepath.Dir(rqStoreFile), 0755); err != nil {
			log.Fatalf("Failed to create rqstore directory for node %d: %v", i, err)
		}

		rqStore, err := rqstore.NewSQLiteRQStore(rqStoreFile)
		if err != nil {
			log.Fatalf("Failed to create rqstore for node %d: %v", i, err)
		}

		service, err := p2p.New(ctx, p2pConfig, mockClient, nil, rqStore, nil, nil)
		if err != nil {
			log.Fatalf("Failed to create p2p service for node %d: %v", i, err)
		}

		// Start P2P service
		go func(nodeID int, rqStore *rqstore.SQLiteRQStore) {
			defer rqStore.Close()
			if err := service.Run(ctx); err != nil {
				log.Printf("Node %d P2P service failed: %v", nodeID, err)
			}
		}(i, rqStore)

		// Start debug service
		debugService := NewP2PDebugService(service, 8080+i)
		go func(nodeID int) {
			if err := debugService.Run(ctx); err != nil {
				log.Printf("Node %d Debug service failed: %v", nodeID, err)
			}
		}(i)

		// Give nodes time to start up and connect
		time.Sleep(2 * time.Second)
	}

	// Setup clean shutdown on interrupt
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// Wait for interrupt signal
	select {
	case <-sigChan:
		log.Println("Received interrupt signal")
	case <-ctx.Done():
		log.Println("Context cancelled")
	}

	// Initiate graceful shutdown
	log.Println("Shutting down...")
	cancel()

	// Give services time to clean up
	time.Sleep(2 * time.Second)
}
