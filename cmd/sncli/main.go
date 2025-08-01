package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/BurntSushi/toml"
	"github.com/LumeraProtocol/supernode/pkg/net/credentials"
	"github.com/LumeraProtocol/supernode/pkg/net/credentials/alts/conn"
	grpcclient "github.com/LumeraProtocol/supernode/pkg/net/grpc/client"
	"github.com/LumeraProtocol/supernode/sdk/adapters/lumera"
	sdkcfg "github.com/LumeraProtocol/supernode/sdk/config"
	sdklog "github.com/LumeraProtocol/supernode/sdk/log"
	sdknet "github.com/LumeraProtocol/supernode/sdk/net"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	reflectpb "google.golang.org/grpc/reflection/grpc_reflection_v1"
 	snkeyring "github.com/LumeraProtocol/supernode/pkg/keyring"
)

type CLIConfig struct {
	Lumera struct {
		GRPCAddr string `toml:"grpc_addr"`
		ChainID  string `toml:"chain_id"`
	} `toml:"lumera"`

	Keyring struct {
		Backend      string `toml:"backend"`
		Dir          string `toml:"dir"`
		KeyName      string `toml:"key_name"`
		LocalAddress string `toml:"local_address"`
	} `toml:"keyring"`

	Supernode struct {
		GRPCEndpoint string `toml:"grpc_endpoint"`
		Address      string `toml:"address"`
	} `toml:"supernode"`
}

func loadCLIConfig(path string) (*CLIConfig, error) {
	var cfg CLIConfig
	_, err := toml.DecodeFile(path, &cfg)
	return &cfg, err
}

func listGRPCMethods(cfg *CLIConfig, kr keyring.Keyring, validator lumera.Client) error {
	conn.RegisterALTSRecordProtocols()
	defer conn.UnregisterALTSRecordProtocols()

	clientCreds, err := credentials.NewClientCreds(&credentials.ClientOptions{
		CommonOptions: credentials.CommonOptions{
			Keyring:       kr,
			LocalIdentity: cfg.Keyring.LocalAddress,
			PeerType:      1, // Simplenode
			Validator:     validator,
		},
	})
	if err != nil {
		return fmt.Errorf("failed to create secure credentials: %w", err)
	}

	grpcClient := grpcclient.NewClient(clientCreds)
	target := credentials.FormatAddressWithIdentity(cfg.Supernode.Address, cfg.Supernode.GRPCEndpoint)

	conn, err := grpcClient.Connect(context.Background(), target, grpcclient.DefaultClientOptions())
	if err != nil {
		return fmt.Errorf("secure grpc connect failed: %w", err)
	}
	defer conn.Close()

	rc := reflectpb.NewServerReflectionClient(conn)
	stream, err := rc.ServerReflectionInfo(context.Background())
	if err != nil {
		return fmt.Errorf("failed to start reflection stream: %w", err)
	}

	if err := stream.Send(&reflectpb.ServerReflectionRequest{
		MessageRequest: &reflectpb.ServerReflectionRequest_ListServices{
			ListServices: "*",
		},
	}); err != nil {
		return fmt.Errorf("failed to send list services request: %w", err)
	}

	resp, err := stream.Recv()
	if err != nil {
		return fmt.Errorf("failed to receive reflection response: %w", err)
	}

	svcResp := resp.GetListServicesResponse()
	fmt.Println("\U0001f4e1 Available gRPC Services on Supernode:")
	for _, svc := range svcResp.Service {
		fmt.Println(" -", svc.Name)
	}
	return nil
}

func showHelp() {
	helpText := `Supernode CLI Usage:
  ./sncli <command> [args...]

Available Commands:
  help                         Show this help message
  list                         List available gRPC services on Supernode
  health-check                 Check Supernode health status
  get-supernode-status         Query Supernode's current status (CPU, memory)`
	fmt.Println(helpText)
}

func main() {
	if len(os.Args) < 2 || os.Args[1] == "help" {
		showHelp()
		return
	}

	command := os.Args[1]

	cfg, err := loadCLIConfig("config.toml")
	if err != nil {
		log.Fatalf("Config load failed: %v", err)
	}

	snkeyring.InitSDKConfig()

	kr, err := snkeyring.InitKeyring(cfg.Keyring.Backend, cfg.Keyring.Dir)
	if err != nil {
		log.Fatalf("Keyring init failed: %v", err)
	}

	sdkConfig := sdkcfg.NewConfig(
		sdkcfg.AccountConfig{
			LocalCosmosAddress: cfg.Keyring.LocalAddress,
			KeyName:            cfg.Keyring.KeyName,
			Keyring:            kr,
		},
		sdkcfg.LumeraConfig{
			GRPCAddr: cfg.Lumera.GRPCAddr,
			ChainID:  cfg.Lumera.ChainID,
		},
	)

	lumeraClient, err := lumera.NewAdapter(context.Background(), lumera.ConfigParams{
		GRPCAddr: sdkConfig.Lumera.GRPCAddr,
		ChainID:  sdkConfig.Lumera.ChainID,
		KeyName:  sdkConfig.Account.KeyName,
		Keyring:  kr,
	}, sdklog.NewNoopLogger())
	if err != nil {
		log.Fatalf("Lumera client init failed: %v", err)
	}

	if command == "list" {
		if err := listGRPCMethods(cfg, kr, lumeraClient); err != nil {
			log.Fatalf("List failed: %v", err)
		}
		return
	}

	supernode := lumera.Supernode{
		CosmosAddress: cfg.Supernode.Address,
		GrpcEndpoint:  cfg.Supernode.GRPCEndpoint,
	}

	clientFactory := sdknet.NewClientFactory(
		context.Background(),
		sdklog.NewNoopLogger(),
		kr,
		lumeraClient,
		sdknet.FactoryConfig{
			LocalCosmosAddress: cfg.Keyring.LocalAddress,
			PeerType:           1, // Simplenode
		},
	)

	client, err := clientFactory.CreateClient(context.Background(), supernode)
	if err != nil {
		log.Fatalf("Supernode client init failed: %v", err)
	}
	defer client.Close(context.Background())

	switch command {
	case "health-check":
		resp, err := client.HealthCheck(context.Background())
		if err != nil {
			log.Fatalf("Health check failed: %v", err)
		}
		fmt.Println("✅ Health status:", resp.Status)

	case "get-supernode-status":
		status, err := client.GetSupernodeStatus(context.Background())
		if err != nil {
			log.Fatalf("Status failed: %v", err)
		}
		fmt.Printf("✅ Supernode CPU: %s, Mem: %s\n", status.CPU.Usage, status.Memory.Used)

	default:
		fmt.Println("Unknown command:", command)
	}
}
