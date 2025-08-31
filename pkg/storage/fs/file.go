package fs

import (
	"context"
	"os"
	"path/filepath"

	"github.com/LumeraProtocol/supernode/v2/pkg/errors"
	"github.com/LumeraProtocol/supernode/v2/pkg/logtrace"
	"github.com/LumeraProtocol/supernode/v2/pkg/storage"
)

const (
	logPrefix = "storage-fs"
)

// FS represents file system storage.
type FS struct {
	dir string
}

// Open implements storage.FileStorageInterface.Open
func (fs *FS) Open(filename string) (storage.FileInterface, error) {
	filename = filepath.Join(fs.dir, filename)

	if _, err := os.Stat(filename); os.IsNotExist(err) {
		return nil, storage.ErrFileNotFound
	}

	file, err := os.Open(filename)
	if err != nil {
		return nil, errors.Errorf("open file %q: %w", filename, err)
	}
	return file, nil
}

// Create implements storage.FileStorageInterface.Create
func (fs *FS) Create(filename string) (storage.FileInterface, error) {
	filename = filepath.Join(fs.dir, filename)

	if _, err := os.Stat(filename); !os.IsNotExist(err) {
		logtrace.Info(context.Background(), "Rewrite file", logtrace.Fields{logtrace.FieldModule: logPrefix, "filename": filename})
	} else {
		logtrace.Info(context.Background(), "Create file", logtrace.Fields{logtrace.FieldModule: logPrefix, "filename": filename})
	}

	file, err := os.Create(filename)
	if err != nil {
		return nil, errors.Errorf("create file %q: %w", filename, err)
	}
	return file, nil
}

// Remove implements storage.FileStorageInterface.Remove
func (fs *FS) Remove(filename string) error {
	filename = filepath.Join(fs.dir, filename)

	logtrace.Info(context.Background(), "Remove file", logtrace.Fields{logtrace.FieldModule: logPrefix, "filename": filename})

	if err := os.Remove(filename); err != nil {
		return errors.Errorf("remove file %q: %w", filename, err)
	}
	return nil
}

// Rename renames oldName to newName.
func (fs *FS) Rename(oldname, newname string) error {
	if oldname == newname {
		return nil
	}

	oldname = filepath.Join(fs.dir, oldname)
	newname = filepath.Join(fs.dir, newname)

	logtrace.Info(context.Background(), "Rename file", logtrace.Fields{logtrace.FieldModule: logPrefix, "old_filename": oldname, "new_filename": newname})

	if err := os.Rename(oldname, newname); err != nil {
		return errors.Errorf("rename file %q to %q: %w", oldname, newname, err)
	}
	return nil
}

// NewFileStorage returns new FS instance. Where `dir` is the path for storing files.
func NewFileStorage(dir string) storage.FileStorageInterface {
	return &FS{
		dir: dir,
	}
}
