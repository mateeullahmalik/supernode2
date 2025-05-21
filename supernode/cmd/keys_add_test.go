package cmd_test

import (
	"bytes"
	"testing"

	"github.com/LumeraProtocol/supernode/pkg/keyring"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
)

func TestKeysAddCmd_RunE(t *testing.T) {
	cmd := getTestKeysAddCmd(t)

	tests := []struct {
		name    string
		args    []string
		wantErr bool
	}{
		{"no_name_arg", []string{}, false},
		{"with_name_arg", []string{"testkey"}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf := new(bytes.Buffer)
			cmd.SetOut(buf)
			cmd.SetArgs(tt.args)

			err := cmd.Execute()
			if (err != nil) != tt.wantErr {
				t.Errorf("unexpected error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// getTestKeysAddCmd initializes and returns a test instance of the cobra.Command for keys add.
func getTestKeysAddCmd(t *testing.T) *cobra.Command {
	return &cobra.Command{
		Use: "add",
		RunE: func(cmd *cobra.Command, args []string) error {
			name := "testkey"
			if len(args) > 0 {
				name = args[0]
			}
			kr, err := keyring.InitKeyring("test", "/tmp")
			require.NoError(t, err)
			_, _, err = keyring.CreateNewAccount(kr, name)
			return err
		},
	}
}
