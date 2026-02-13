// Package store manages the local profile store at ~/.ocmgr/profiles/.
//
// Each subdirectory under the profiles directory represents a single profile
// and is expected to contain a profile.toml file that can be loaded by the
// profile package.
package store

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/acchapm1/ocmgr/internal/config"
	"github.com/acchapm1/ocmgr/internal/profile"
)

// Store provides access to locally stored profiles on disk.
type Store struct {
	// Dir is the absolute path to the profiles directory (e.g. ~/.ocmgr/profiles).
	Dir string
}

// NewStore creates a Store pointing to the configured profiles directory.
// It reads the store path from config.toml, falling back to ~/.ocmgr/profiles
// if the config cannot be loaded. The directory is created if it does not
// already exist.
func NewStore() (*Store, error) {
	cfg, err := config.Load()
	if err != nil {
		// Fall back to default location if config can't be loaded.
		dir := filepath.Join(config.ConfigDir(), "profiles")
		return NewStoreAt(dir)
	}
	dir := config.ExpandPath(cfg.Store.Path)
	return NewStoreAt(dir)
}

// NewStoreAt creates a Store rooted at the given directory. The path is
// expanded (a leading "~" is replaced with the user's home directory) and
// the directory is created if it does not already exist.
func NewStoreAt(dir string) (*Store, error) {
	dir = config.ExpandPath(dir)

	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("creating store directory: %w", err)
	}

	return &Store{Dir: dir}, nil
}

// List returns all valid profiles found in the store, sorted alphabetically
// by name. Subdirectories that do not contain a valid profile.toml are
// silently skipped.
func (s *Store) List() ([]*profile.Profile, error) {
	entries, err := os.ReadDir(s.Dir)
	if err != nil {
		return nil, fmt.Errorf("reading store directory: %w", err)
	}

	var profiles []*profile.Profile
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		dir := filepath.Join(s.Dir, entry.Name())
		p, err := profile.LoadProfile(dir)
		if err != nil {
			// Skip directories without a valid profile.toml.
			continue
		}

		profiles = append(profiles, p)
	}

	sort.Slice(profiles, func(i, j int) bool {
		return profiles[i].Name < profiles[j].Name
	})

	return profiles, nil
}

// Get loads and returns the profile with the given name. An error is returned
// if the profile directory does not exist or cannot be loaded.
func (s *Store) Get(name string) (*profile.Profile, error) {
	if err := profile.ValidateName(name); err != nil {
		return nil, err
	}

	dir := s.ProfileDir(name)

	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return nil, fmt.Errorf("profile %q not found", name)
	}

	p, err := profile.LoadProfile(dir)
	if err != nil {
		return nil, fmt.Errorf("loading profile %q: %w", name, err)
	}

	return p, nil
}

// Exists reports whether a profile with the given name exists in the store.
func (s *Store) Exists(name string) bool {
	info, err := os.Stat(s.ProfileDir(name))
	return err == nil && info.IsDir()
}

// Delete removes the profile directory for the given name. An error is
// returned if the profile does not exist.
func (s *Store) Delete(name string) error {
	if err := profile.ValidateName(name); err != nil {
		return err
	}

	dir := s.ProfileDir(name)

	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return fmt.Errorf("profile %q not found", name)
	}

	if err := os.RemoveAll(dir); err != nil {
		return fmt.Errorf("deleting profile %q: %w", name, err)
	}

	return nil
}

// ProfileDir returns the absolute path to the directory for the named profile.
func (s *Store) ProfileDir(name string) string {
	return filepath.Join(s.Dir, name)
}
