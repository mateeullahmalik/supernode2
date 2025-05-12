package cmd

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestKeysCmdMetadata(t *testing.T) {
	assert.Equal(t, "keys", keysCmd.Use)
	assert.Contains(t, keysCmd.Short, "Manage keys")
	assert.Contains(t, keysCmd.Long, "Manage keys for the Supernode")
}

func TestKeysCmdRegistered(t *testing.T) {
	found := false
	for _, cmd := range rootCmd.Commands() {
		if cmd == keysCmd {
			found = true
			break
		}
	}
	assert.True(t, found, "keysCmd should be registered with rootCmd")
}
