package cmd

import (
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"syscall"

	"github.com/AlecAivazis/survey/v2"
	"github.com/LumeraProtocol/supernode/pkg/keyring"
	"github.com/LumeraProtocol/supernode/supernode/config"
	consmoskeyring "github.com/cosmos/cosmos-sdk/crypto/keyring"

	"github.com/spf13/cobra"
)

var (
	forceInit          bool
	skipInteractive    bool
	keyringBackendFlag string
	keyNameFlag        string
	shouldRecoverFlag  bool
	mnemonicFlag       string
	supernodeAddrFlag  string
	supernodePortFlag  int
	lumeraGrpcFlag     string
	chainIDFlag        string
)

// Default configuration values
const (
	DefaultKeyringBackend = "test"
	DefaultKeyName        = ""
	DefaultSupernodeAddr  = "0.0.0.0"
	DefaultSupernodePort  = 4444
	DefaultLumeraGRPC     = "localhost:9090"
	DefaultChainID        = "lumera-mainnet-1"
)

// InitInputs holds all user inputs for initialization
type InitInputs struct {
	KeyringBackend string
	KeyName        string
	ShouldRecover  bool
	Mnemonic       string
	SupernodeAddr  string
	SupernodePort  int
	LumeraGRPC     string
	ChainID        string
}

// initCmd represents the init command
var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize a new supernode",
	Long: `Initialize a new supernode by creating a configuration file and setting up keys.

This command will guide you through an interactive setup process to:
1. Create a config.yml file at ~/.supernode
2. Select keyring backend (test, file, or os)
3. Recover an existing key from mnemonic
4. Configure network settings (GRPC address, port, chain ID)

Example:
  supernode init
  supernode init --force  # Override existing installation
  supernode init -y       # Use default values, skip interactive prompts`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Set up signal handling for graceful exit
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

		// Create a channel to communicate when the command is done
		done := make(chan struct{})

		// Handle signals in a separate goroutine
		go func() {
			select {
			case <-sigCh:
				fmt.Println("\nInterrupted. Exiting...")
				os.Exit(0)
			case <-done:
				return
			}
		}()

		// Setup base directory
		if err := setupBaseDirectory(); err != nil {
			close(done)
			return err
		}

		// Get user inputs through interactive prompts or use defaults
		inputs, err := gatherUserInputs()
		if err != nil {
			close(done)
			return err
		}

		// Create and setup configuration
		if err := createAndSetupConfig(inputs.KeyName, inputs.ChainID, inputs.KeyringBackend); err != nil {
			return err
		}

		// Setup keyring and handle key creation/recovery
		var address string
		var generatedMnemonic string

		// Always setup keyring, even in non-interactive mode
		address, generatedMnemonic, err = setupKeyring(inputs.KeyName, inputs.ShouldRecover, inputs.Mnemonic)
		if err != nil {
			return err
		}

		// Update config with gathered settings and save
		if err := updateAndSaveConfig(address, inputs.SupernodeAddr, inputs.SupernodePort, inputs.LumeraGRPC, inputs.ChainID); err != nil {
			return err
		}

		// Print success message
		printSuccessMessage(generatedMnemonic)
		return nil
	},
}

// setupBaseDirectory handles base directory creation and validation
func setupBaseDirectory() error {
	if baseDir == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("failed to get home directory: %w", err)
		}
		baseDir = filepath.Join(homeDir, DefaultBaseDir)
	}

	// Check if base directory already exists
	if _, err := os.Stat(baseDir); err == nil && !forceInit {
		return fmt.Errorf("supernode directory already exists at %s\nUse --force to overwrite or remove the directory manually", baseDir)
	}

	// If force flag is used, clean up config file and keys directory
	if forceInit {
		cfgFile := filepath.Join(baseDir, DefaultConfigFile)
		if err := os.Remove(cfgFile); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("failed to remove existing config file: %w", err)
		}

		keysDir := filepath.Join(baseDir, "keys")
		if err := os.RemoveAll(keysDir); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("failed to remove existing keys directory: %w", err)
		}

		fmt.Println("Cleaned up existing config file and keys directory")
	}

	// Create base directory if it doesn't exist
	if err := os.MkdirAll(baseDir, 0700); err != nil {
		return fmt.Errorf("failed to create base directory: %w", err)
	}

	fmt.Printf("BaseDirectory: %s\n", baseDir)
	return nil
}

// gatherUserInputs collects all user inputs through interactive prompts or uses defaults/flags
func gatherUserInputs() (InitInputs, error) {
	// Check if all required parameters are provided via flags
	allFlagsProvided := keyNameFlag != "" &&
		(supernodeAddrFlag != "" || supernodePortFlag != 0 || lumeraGrpcFlag != "" || chainIDFlag != "")

	// If -y flag is set or all flags are provided, use flags or defaults
	if skipInteractive || allFlagsProvided {
		fmt.Println("Using provided flags or default configuration values...")

		// Use flags if provided, otherwise use defaults
		backend := DefaultKeyringBackend
		if keyringBackendFlag != "" {
			backend = keyringBackendFlag
		}

		keyName := DefaultKeyName
		if keyNameFlag != "" {
			keyName = keyNameFlag
		}

		supernodeAddr := DefaultSupernodeAddr
		if supernodeAddrFlag != "" {
			supernodeAddr = supernodeAddrFlag
		}

		supernodePort := DefaultSupernodePort
		if supernodePortFlag != 0 {
			supernodePort = supernodePortFlag
		}

		lumeraGRPC := DefaultLumeraGRPC
		if lumeraGrpcFlag != "" {
			lumeraGRPC = lumeraGrpcFlag
		}

		chainID := DefaultChainID
		if chainIDFlag != "" {
			chainID = chainIDFlag
		}

		// Check if mnemonic is provided when recover flag is set
		if shouldRecoverFlag && mnemonicFlag == "" {
			return InitInputs{}, fmt.Errorf("--mnemonic flag is required when --recover flag is set")
		}

		return InitInputs{
			KeyringBackend: backend,
			KeyName:        keyName,
			ShouldRecover:  shouldRecoverFlag,
			Mnemonic:       mnemonicFlag,
			SupernodeAddr:  supernodeAddr,
			SupernodePort:  supernodePort,
			LumeraGRPC:     lumeraGRPC,
			ChainID:        chainID,
		}, nil
	}

	var inputs InitInputs
	var err error

	// Interactive setup
	inputs.KeyringBackend, err = promptKeyringBackend()
	if err != nil {
		return InitInputs{}, fmt.Errorf("failed to select keyring backend: %w", err)
	}

	inputs.KeyName, inputs.ShouldRecover, inputs.Mnemonic, err = promptKeyManagement()
	if err != nil {
		return InitInputs{}, fmt.Errorf("failed to configure key management: %w", err)
	}

	inputs.SupernodeAddr, inputs.SupernodePort, inputs.LumeraGRPC, inputs.ChainID, err = promptNetworkConfig()
	if err != nil {
		return InitInputs{}, fmt.Errorf("failed to configure network settings: %w", err)
	}

	return inputs, nil
}

// createAndSetupConfig creates default configuration and necessary directories
func createAndSetupConfig(keyName, chainID, keyringBackend string) error {
	// Set config file path
	cfgFile := filepath.Join(baseDir, DefaultConfigFile)

	fmt.Printf("Using config file: %s\n", cfgFile)

	// Create default configuration
	appConfig = config.CreateDefaultConfig(keyName, "", chainID, keyringBackend, "")
	appConfig.BaseDir = baseDir

	// Create directories
	if err := appConfig.EnsureDirs(); err != nil {
		return fmt.Errorf("failed to create directories: %w", err)
	}

	return nil
}

// setupKeyring initializes keyring and handles key creation or recovery
// Returns address and mnemonic (if a new key was created)
func setupKeyring(keyName string, shouldRecover bool, mnemonic string) (string, string, error) {
	kr, err := initKeyringFromConfig(appConfig)
	if err != nil {
		return "", "", fmt.Errorf("failed to initialize keyring: %w", err)
	}

	var address string
	var generatedMnemonic string

	if shouldRecover {
		address, err = recoverExistingKey(kr, keyName, mnemonic)
		if err != nil {
			return "", "", err
		}
	} else {
		address, generatedMnemonic, err = createNewKey(kr, keyName)
		if err != nil {
			return "", "", err

		}
	}

	return address, generatedMnemonic, nil
}

// recoverExistingKey handles the recovery of an existing key from mnemonic
func recoverExistingKey(kr consmoskeyring.Keyring, keyName, mnemonic string) (string, error) {
	// Process and validate mnemonic using helper function
	processedMnemonic, err := processAndValidateMnemonic(mnemonic)
	if err != nil {
		fmt.Printf("Warning: %v\n", err)
		// Continue with original mnemonic if validation fails
		processedMnemonic = mnemonic
	}

	info, err := keyring.RecoverAccountFromMnemonic(kr, keyName, processedMnemonic)
	if err != nil {
		return "", fmt.Errorf("failed to recover account: %w", err)
	}

	addr, err := getAddressFromKeyName(kr, keyName)
	if err != nil {
		return "", fmt.Errorf("failed to get address: %w", err)
	}
	address := addr.String()

	fmt.Printf("Key recovered successfully! Name: %s, Address: %s\n", info.Name, address)
	return address, nil
}

// createNewKey handles the creation of a new key
func createNewKey(kr consmoskeyring.Keyring, keyName string) (string, string, error) {
	// Generate mnemonic and create new account
	keyMnemonic, info, err := keyring.CreateNewAccount(kr, keyName)
	if err != nil {
		return "", "", fmt.Errorf("failed to create new account: %w", err)
	}

	addr, err := getAddressFromKeyName(kr, keyName)
	if err != nil {
		return "", "", fmt.Errorf("failed to get address: %w", err)
	}
	address := addr.String()

	fmt.Printf("Key generated successfully! Name: %s, Address: %s\n", info.Name, address)
	fmt.Println("\nIMPORTANT: Write down the mnemonic and keep it in a safe place.")
	fmt.Println("The mnemonic is the only way to recover your account if you forget your password.")
	fmt.Printf("Mnemonic: %s\n", keyMnemonic)

	return address, keyMnemonic, nil
}

// updateAndSaveConfig updates the configuration with network settings and saves it
func updateAndSaveConfig(address, supernodeAddr string, supernodePort int, lumeraGrpcAddr string, chainID string) error {
	// Update config with address and network settings
	appConfig.SupernodeConfig.Identity = address
	appConfig.SupernodeConfig.IpAddress = supernodeAddr
	appConfig.SupernodeConfig.Port = uint16(supernodePort)
	appConfig.LumeraClientConfig.GRPCAddr = lumeraGrpcAddr
	appConfig.LumeraClientConfig.ChainID = chainID

	// Save config
	cfgFile := filepath.Join(baseDir, DefaultConfigFile)
	if err := config.SaveConfig(appConfig, cfgFile); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	fmt.Printf("\nConfiguration saved to %s\n", cfgFile)
	return nil
}

// printSuccessMessage displays the final success message
func printSuccessMessage(mnemonic string) {
	fmt.Println("\nYour supernode has been initialized successfully!")

	// If a mnemonic was generated, display it again
	if mnemonic != "" {
		fmt.Println("\nIMPORTANT: Make sure you have saved your mnemonic:")
		fmt.Printf("Mnemonic: %s\n", mnemonic)
	}

	fmt.Println("\nYou can now start your supernode with:")
	fmt.Println("  supernode start")
}

// Interactive prompt functions
func promptKeyringBackend() (string, error) {
	var backend string
	prompt := &survey.Select{
		Message: "Choose keyring backend:",
		Options: []string{"os", "file", "test"},
		Default: "os",
		Help:    "os: OS keyring (most secure), file: encrypted file, test: unencrypted (dev only), Ctrl-C for exit",
	}
	return backend, survey.AskOne(prompt, &backend)
}

func promptKeyManagement() (keyName string, shouldRecover bool, mnemonic string, err error) {
	// Key name input with validation
	keyNamePrompt := &survey.Input{
		Message: "Enter key name:",
		Help:    "Alphanumeric characters and underscores only, Ctrl-C for exit",
	}
	err = survey.AskOne(keyNamePrompt, &keyName, survey.WithValidator(survey.Required))
	if err != nil {
		return "", false, "", err
	}

	// Ask whether to create a new address or recover from mnemonic
	createOrRecoverPrompt := &survey.Select{
		Message: "Would you like to create a new address or recover from mnemonic?",
		Options: []string{"Create new address", "Recover from mnemonic"},
		Default: "Create new address",
		Help:    "Create a new address or recover an existing one from mnemonic, Ctrl-C for exit",
	}
	var createOrRecover string
	err = survey.AskOne(createOrRecoverPrompt, &createOrRecover)
	if err != nil {
		return "", false, "", err
	}

	shouldRecover = createOrRecover == "Recover from mnemonic"

	// If recovering, ask for mnemonic
	if shouldRecover {
		mnemonicPrompt := &survey.Password{
			Message: "Enter your mnemonic phrase:",
			Help:    "Space-separated words (typically 12 or 24 words), Ctrl-C for exit",
		}
		err = survey.AskOne(mnemonicPrompt, &mnemonic, survey.WithValidator(survey.Required))
		if err != nil {
			return "", false, "", err
		}
	}

	return keyName, shouldRecover, mnemonic, nil
}

func promptNetworkConfig() (supernodeAddr string, supernodePort int, lumeraGrpcAddr string, chainID string, err error) {
	// Supernode IP address
	supernodePrompt := &survey.Input{
		Message: "Enter supernode IP address:",
		Default: DefaultSupernodeAddr,
		Help:    "IP address for the supernode to listen on, Ctrl-C for exit",
	}
	err = survey.AskOne(supernodePrompt, &supernodeAddr)
	if err != nil {
		return "", 0, "", "", err
	}

	// Supernode port
	var portStr string
	supernodePortPrompt := &survey.Input{
		Message: "Enter supernode port:",
		Default: fmt.Sprintf("%d", DefaultSupernodePort),
		Help:    "Port for the supernode to listen on (1-65535), Ctrl-C for exit",
	}
	err = survey.AskOne(supernodePortPrompt, &portStr)
	if err != nil {
		return "", 0, "", "", err
	}

	supernodePort, err = strconv.Atoi(portStr)
	if err != nil || supernodePort < 1 || supernodePort > 65535 {
		return "", 0, "", "", fmt.Errorf("invalid supernode port: %s", portStr)
	}

	// Lumera GRPC address (full address with port)
	lumeraPrompt := &survey.Input{
		Message: "Enter Lumera GRPC address:",
		Default: DefaultLumeraGRPC,
		Help:    "GRPC address of the Lumera node (host:port), Ctrl-C for exit",
	}
	err = survey.AskOne(lumeraPrompt, &lumeraGrpcAddr)
	if err != nil {
		return "", 0, "", "", err
	}

	// Chain ID
	chainPrompt := &survey.Input{
		Message: "Enter chain ID:",
		Default: DefaultChainID,
		Help:    "Chain ID of the Lumera network, Ctrl-C for exit",
	}
	err = survey.AskOne(chainPrompt, &chainID, survey.WithValidator(survey.Required))
	if err != nil {
		return "", 0, "", "", err
	}

	return supernodeAddr, supernodePort, lumeraGrpcAddr, chainID, nil
}

func init() {
	rootCmd.AddCommand(initCmd)

	// Add flags
	initCmd.Flags().BoolVar(&forceInit, "force", false, "Force initialization, overwriting existing directory")
	initCmd.Flags().BoolVarP(&skipInteractive, "yes", "y", false, "Skip interactive prompts and use default values")
	initCmd.Flags().StringVar(&keyringBackendFlag, "keyring-backend", "", "Keyring backend to use with -y flag (test, file, os)")
	initCmd.Flags().StringVar(&keyNameFlag, "key-name", "", "Name of the key to create or recover")
	initCmd.Flags().BoolVar(&shouldRecoverFlag, "recover", false, "Recover an existing key from mnemonic")
	initCmd.Flags().StringVar(&mnemonicFlag, "mnemonic", "", "Mnemonic phrase for key recovery (only used with --recover)")
	initCmd.Flags().StringVar(&supernodeAddrFlag, "supernode-addr", "", "IP address for the supernode to listen on")
	initCmd.Flags().IntVar(&supernodePortFlag, "supernode-port", 0, "Port for the supernode to listen on")
	initCmd.Flags().StringVar(&lumeraGrpcFlag, "lumera-grpc", "", "GRPC address of the Lumera node (host:port)")
	initCmd.Flags().StringVar(&chainIDFlag, "chain-id", "", "Chain ID of the Lumera network")
}
