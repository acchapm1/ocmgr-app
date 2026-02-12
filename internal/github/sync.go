package github

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/acchapm1/ocmgr-app/internal/config"
	"github.com/acchapm1/ocmgr-app/internal/copier"
	"github.com/acchapm1/ocmgr-app/internal/profile"
)

// cacheDir returns the path to the local sync cache (~/.ocmgr/.sync-cache).
func cacheDir() string {
	return filepath.Join(config.ConfigDir(), ".sync-cache")
}

// cacheProfilesDir returns the profiles/ subdirectory inside the cache.
func cacheProfilesDir() string {
	return filepath.Join(cacheDir(), "profiles")
}

// EnsureCache clones the remote repository into the local sync cache
// if it has not been cloned yet, or pulls the latest changes if a
// cached clone already exists.
//
// The cache lives at ~/.ocmgr/.sync-cache/.
func EnsureCache(repo, authMethod string) (string, error) {
	if _, err := exec.LookPath("git"); err != nil {
		return "", fmt.Errorf("git is required for sync operations but was not found in PATH")
	}

	dir := cacheDir()
	remoteURL, err := ResolveRemoteURL(repo, authMethod)
	if err != nil {
		return "", err
	}
	token := ResolveToken(authMethod)

	if isGitRepo(dir) {
		// Cache exists — pull latest.
		if err := gitPull(dir, token); err != nil {
			return "", fmt.Errorf("pulling latest changes: %w", err)
		}
		return dir, nil
	}

	// No cache — clone.
	if err := os.RemoveAll(dir); err != nil {
		return "", fmt.Errorf("cleaning cache directory: %w", err)
	}

	if err := gitClone(remoteURL, dir, token); err != nil {
		return "", fmt.Errorf("cloning %s: %w", repo, err)
	}

	// Ensure the profiles/ subdirectory exists in the cache so that
	// new pushes don't fail on an empty repo.
	if err := os.MkdirAll(cacheProfilesDir(), 0o755); err != nil {
		return "", err
	}

	return dir, nil
}

// PushProfile copies a local profile into the sync cache and pushes
// the changes to the remote repository.
func PushProfile(name, localProfileDir, repo, authMethod string) error {
	cache, err := EnsureCache(repo, authMethod)
	if err != nil {
		return err
	}

	dst := filepath.Join(cache, "profiles", name)

	// Remove old version in cache so deleted files don't linger.
	_ = os.RemoveAll(dst)

	// Copy profile into cache.
	if err := CopyDirRecursive(localProfileDir, dst); err != nil {
		return fmt.Errorf("copying profile to cache: %w", err)
	}

	// Stage, commit and push.
	token := ResolveToken(authMethod)
	rel := filepath.Join("profiles", name)
	if err := gitAddCommitPush(cache, rel, fmt.Sprintf("sync: update %s", name), token); err != nil {
		return err
	}

	return nil
}

// PullProfile downloads a single profile from the remote repository
// into the local store directory.
func PullProfile(name, targetStoreDir, repo, authMethod string) error {
	if _, err := EnsureCache(repo, authMethod); err != nil {
		return err
	}

	return pullProfileFromCache(name, targetStoreDir)
}

// PullAll downloads every profile from the remote repository into the
// local store directory and returns the names of the profiles that
// were pulled.
func PullAll(targetStoreDir, repo, authMethod string) ([]string, error) {
	if _, err := EnsureCache(repo, authMethod); err != nil {
		return nil, err
	}

	entries, err := os.ReadDir(cacheProfilesDir())
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("reading remote profiles: %w", err)
	}

	var pulled []string
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		name := entry.Name()
		if err := pullProfileFromCache(name, targetStoreDir); err != nil {
			return pulled, fmt.Errorf("pulling %q: %w", name, err)
		}
		pulled = append(pulled, name)
	}

	return pulled, nil
}

// pullProfileFromCache copies a profile from the already-ensured
// cache to the local store.  Avoids redundant EnsureCache calls.
func pullProfileFromCache(name, targetStoreDir string) error {
	src := filepath.Join(cacheProfilesDir(), name)
	if _, err := os.Stat(src); os.IsNotExist(err) {
		return fmt.Errorf("profile %q not found in remote repository", name)
	}

	dst := filepath.Join(targetStoreDir, name)

	// Remove local version so we get a clean copy.
	_ = os.RemoveAll(dst)

	if err := CopyDirRecursive(src, dst); err != nil {
		return fmt.Errorf("copying profile from cache: %w", err)
	}

	return nil
}

// SyncStatus describes the synchronisation state between local and
// remote profiles.
type SyncStatus struct {
	LocalOnly  []string // exist locally but not remotely
	RemoteOnly []string // exist remotely but not locally
	Modified   []string // exist in both but differ
	InSync     []string // exist in both and are identical
}

// Status compares local profiles against the remote cache and returns
// a SyncStatus summary.
func Status(localStoreDir, repo, authMethod string) (*SyncStatus, error) {
	if _, err := EnsureCache(repo, authMethod); err != nil {
		return nil, err
	}

	local, err := listProfileNames(localStoreDir)
	if err != nil {
		return nil, fmt.Errorf("listing local profiles: %w", err)
	}
	remote, err := listProfileNames(cacheProfilesDir())
	if err != nil {
		return nil, fmt.Errorf("listing remote profiles: %w", err)
	}

	localSet := make(map[string]bool, len(local))
	for _, n := range local {
		localSet[n] = true
	}
	remoteSet := make(map[string]bool, len(remote))
	for _, n := range remote {
		remoteSet[n] = true
	}

	status := &SyncStatus{}
	for _, n := range local {
		if !remoteSet[n] {
			status.LocalOnly = append(status.LocalOnly, n)
			continue
		}
		eq, err := dirsEqual(
			filepath.Join(localStoreDir, n),
			filepath.Join(cacheProfilesDir(), n),
		)
		if err != nil {
			// Treat errors as "modified" to surface them.
			status.Modified = append(status.Modified, n)
			continue
		}
		if eq {
			status.InSync = append(status.InSync, n)
		} else {
			status.Modified = append(status.Modified, n)
		}
	}
	for _, n := range remote {
		if !localSet[n] {
			status.RemoteOnly = append(status.RemoteOnly, n)
		}
	}

	return status, nil
}

// ──────────────────────────────────────────────────────────────────
// Git helpers — thin wrappers around the git CLI.
//
// Authentication tokens are NEVER embedded in URLs.  For HTTPS
// remotes, tokens are injected via the Authorization header using
// git's -c http.extraHeader option so they are not persisted in
// .git/config or visible in error messages.
// ──────────────────────────────────────────────────────────────────

func isGitRepo(dir string) bool {
	info, err := os.Stat(filepath.Join(dir, ".git"))
	return err == nil && info.IsDir()
}

// gitAuthArgs returns extra git CLI arguments that inject an
// Authorization header when a token is available.
func gitAuthArgs(token string) []string {
	if token == "" {
		return nil
	}
	return []string{"-c", fmt.Sprintf("http.extraHeader=Authorization: Bearer %s", token)}
}

func gitClone(url, dir, token string) error {
	args := append(gitAuthArgs(token), "clone", url, dir)
	cmd := exec.Command("git", args...)
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func gitPull(dir, token string) error {
	args := append(gitAuthArgs(token), "pull", "--ff-only")
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func gitAddCommitPush(repoDir, pathSpec, message, token string) error {
	// git add
	add := exec.Command("git", "add", pathSpec)
	add.Dir = repoDir
	add.Stderr = os.Stderr
	if err := add.Run(); err != nil {
		return fmt.Errorf("git add: %w", err)
	}

	// Check if there are staged changes to commit.
	// Using `git diff --cached --quiet` — exits 1 if there ARE staged changes.
	check := exec.Command("git", "diff", "--cached", "--quiet")
	check.Dir = repoDir
	if err := check.Run(); err == nil {
		// Exit 0 means nothing staged — skip commit and push.
		return nil
	}

	// git commit
	commit := exec.Command("git", "commit", "-m", message)
	commit.Dir = repoDir
	commit.Stderr = os.Stderr
	if err := commit.Run(); err != nil {
		return fmt.Errorf("git commit: %w", err)
	}

	// git push (with auth header)
	pushArgs := append(gitAuthArgs(token), "push")
	push := exec.Command("git", pushArgs...)
	push.Dir = repoDir
	push.Stdout = os.Stderr
	push.Stderr = os.Stderr
	if err := push.Run(); err != nil {
		return fmt.Errorf("git push: %w", err)
	}

	return nil
}

// ──────────────────────────────────────────────────────────────────
// File helpers
// ──────────────────────────────────────────────────────────────────

// CopyDirRecursive copies an entire directory tree from src to dst,
// skipping .git directories.  It is exported so cli/profile.go can
// reuse it for import/export.
func CopyDirRecursive(src, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip .git directory.
		if info.IsDir() && info.Name() == ".git" {
			return filepath.SkipDir
		}

		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}

		target := filepath.Join(dst, rel)

		if info.IsDir() {
			return os.MkdirAll(target, 0o755)
		}

		return copier.CopyFile(path, target)
	})
}

// listProfileNames returns the names of subdirectories in dir that
// contain a valid profile.toml.
func listProfileNames(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var names []string
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		// Only include directories that contain a profile.toml.
		toml := filepath.Join(dir, e.Name(), "profile.toml")
		if _, err := os.Stat(toml); err == nil {
			names = append(names, e.Name())
		}
	}
	return names, nil
}

// dirsEqual reports whether every file under two directory trees is
// identical.  Only regular files are compared.
func dirsEqual(a, b string) (bool, error) {
	aFiles, err := collectFiles(a)
	if err != nil {
		return false, err
	}
	bFiles, err := collectFiles(b)
	if err != nil {
		return false, err
	}

	if len(aFiles) != len(bFiles) {
		return false, nil
	}

	for rel, aPath := range aFiles {
		bPath, ok := bFiles[rel]
		if !ok {
			return false, nil
		}
		eq, err := copier.FilesEqual(aPath, bPath)
		if err != nil {
			return false, err
		}
		if !eq {
			return false, nil
		}
	}

	return true, nil
}

// collectFiles walks a directory and returns a map of relative paths
// to absolute paths for every regular file.
func collectFiles(root string) (map[string]string, error) {
	files := make(map[string]string)

	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}

		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		files[rel] = path
		return nil
	})

	return files, err
}

// ValidateProfileDir checks whether dir is not empty and is a
// loadable profile.  Used by import to validate before copying.
func ValidateProfileDir(dir string) (*profile.Profile, error) {
	p, err := profile.LoadProfile(dir)
	if err != nil {
		return nil, fmt.Errorf("not a valid profile directory: %w", err)
	}
	return p, nil
}
