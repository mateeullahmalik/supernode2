//go:generate mockery --name=KeyValue

package storage

import (
	"time"

	"github.com/LumeraProtocol/supernode/v2/pkg/errors"
)

var (
	// ErrKeyValueNotFound is returned when key isn't found.
	ErrKeyValueNotFound = errors.New("key not found")
)

// KeyValue represents database that uses a simple key-value method to store data.
type KeyValue interface {
	// Get looks for key and returns corresponding Item.
	// If key is not found, ErrKeyValueNotFound is returned.
	Get(key string) (value []byte, err error)

	// Set adds a key-value pair to the database
	Set(key string, value []byte) (err error)

	// Set adds a key-value pair to the database with expiry time
	SetWithExpiry(key string, value []byte, expiry time.Duration) (err error)

	// Delete deletes a key.
	Delete(key string) error
}
