// Package updater handles self-updating the ocmgr binary.
package updater

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"time"
)

const (
	githubAPIURL = "https://api.github.com/repos/acchapm1/ocmgr"
	githubRepo   = "acchapm1/ocmgr"
	binaryName   = "ocmgr"
)

// Release represents a GitHub release.
type Release struct {
	TagName string  `json:"tag_name"`
	Name    string  `json:"name"`
	Assets  []Asset `json:"assets"`
	HTMLURL string  `json:"html_url"`
}

// Asset represents a release asset.
type Asset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
}

// Updater handles self-updating.
type Updater struct {
	currentVersion string
	installDir     string
}

// New creates a new Updater.
func New(currentVersion string) *Updater {
	return &Updater{
		currentVersion: currentVersion,
	}
}

// CheckForUpdate checks if a newer version is available.
// Returns the latest release if an update is available, nil otherwise.
func (u *Updater) CheckForUpdate() (*Release, error) {
	latest, err := u.getLatestRelease()
	if err != nil {
		return nil, fmt.Errorf("checking for updates: %w", err)
	}

	// Compare versions
	if u.isNewerVersion(latest.TagName) {
		return latest, nil
	}

	return nil, nil
}

// GetRelease gets a specific release by tag name.
func (u *Updater) GetRelease(tag string) (*Release, error) {
	url := fmt.Sprintf("%s/releases/tags/%s", githubAPIURL, tag)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("fetching release %s: %w", tag, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == 404 {
		return nil, fmt.Errorf("release %s not found", tag)
	}
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("GitHub API returned status %d", resp.StatusCode)
	}

	var release Release
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return nil, fmt.Errorf("parsing release: %w", err)
	}

	return &release, nil
}

// Update downloads and installs the specified release.
func (u *Updater) Update(release *Release) error {
	// Find the current executable
	execPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("finding executable: %w", err)
	}

	// Resolve symlinks
	execPath, err = filepath.EvalSymlinks(execPath)
	if err != nil {
		return fmt.Errorf("resolving executable path: %w", err)
	}

	u.installDir = filepath.Dir(execPath)

	// Detect platform
	platform := fmt.Sprintf("%s_%s", runtime.GOOS, runtime.GOARCH)

	// Find the matching asset
	asset := u.findAsset(release, platform)
	if asset == nil {
		return fmt.Errorf("no binary found for platform %s in release %s", platform, release.TagName)
	}

	fmt.Printf("Downloading %s...\n", asset.Name)

	// Download to temp file
	tmpDir, err := os.MkdirTemp("", "ocmgr-update")
	if err != nil {
		return fmt.Errorf("creating temp directory: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	tmpFile := filepath.Join(tmpDir, asset.Name)
	if err := u.downloadFile(asset.BrowserDownloadURL, tmpFile); err != nil {
		return fmt.Errorf("downloading: %w", err)
	}

	// Extract the binary
	binaryPath, err := u.extractBinary(tmpFile, tmpDir)
	if err != nil {
		return fmt.Errorf("extracting: %w", err)
	}

	// Replace the current binary
	if err := u.replaceBinary(execPath, binaryPath); err != nil {
		return fmt.Errorf("replacing binary: %w", err)
	}

	fmt.Printf("✓ Updated ocmgr to %s\n", release.TagName)

	return nil
}

// getLatestRelease fetches the latest release from GitHub.
func (u *Updater) getLatestRelease() (*Release, error) {
	url := fmt.Sprintf("%s/releases/latest", githubAPIURL)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("GitHub API returned status %d", resp.StatusCode)
	}

	var release Release
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return nil, err
	}

	return &release, nil
}

// isNewerVersion compares version strings.
// Returns true if the new version is newer than current.
func (u *Updater) isNewerVersion(newVersion string) bool {
	current := strings.TrimPrefix(u.currentVersion, "v")
	new := strings.TrimPrefix(newVersion, "v")

	// Handle dev/dirty versions
	if strings.Contains(current, "-") || strings.Contains(current, "dirty") {
		// Development build, always consider updates available
		return true
	}

	// Simple semver comparison
	re := regexp.MustCompile(`^(\d+)\.(\d+)\.(\d+)`)
	currentMatch := re.FindStringSubmatch(current)
	newMatch := re.FindStringSubmatch(new)

	if currentMatch == nil || newMatch == nil {
		return new != current
	}

	// Compare major.minor.patch
	for i := 1; i <= 3; i++ {
		var c, n int
		fmt.Sscanf(currentMatch[i], "%d", &c)
		fmt.Sscanf(newMatch[i], "%d", &n)
		if n > c {
			return true
		}
		if n < c {
			return false
		}
	}

	return false // Same version
}

// findAsset finds the matching asset for the platform.
func (u *Updater) findAsset(release *Release, platform string) *Asset {
	// Common naming patterns
	patterns := []string{
		fmt.Sprintf("%s_%s.tar.gz", binaryName, platform),
		fmt.Sprintf("%s-%s.tar.gz", binaryName, platform),
		fmt.Sprintf("%s_%s.tar.gz", release.TagName, platform),
	}

	for _, asset := range release.Assets {
		for _, pattern := range patterns {
			if asset.Name == pattern {
				return &asset
			}
		}
		// Also check if platform is in the name
		if strings.Contains(asset.Name, platform) && strings.HasSuffix(asset.Name, ".tar.gz") {
			return &asset
		}
	}

	return nil
}

// downloadFile downloads a file from URL to path.
func (u *Updater) downloadFile(url, path string) error {
	client := &http.Client{Timeout: 5 * time.Minute}

	resp, err := client.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("download returned status %d", resp.StatusCode)
	}

	out, err := os.Create(path)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	return err
}

// extractBinary extracts the binary from a tar.gz file.
func (u *Updater) extractBinary(archivePath, destDir string) (string, error) {
	// Use tar command for cross-platform compatibility
	cmd := exec.Command("tar", "-xzf", archivePath, "-C", destDir)
	if output, err := cmd.CombinedOutput(); err != nil {
		return "", fmt.Errorf("tar extraction failed: %s: %w", string(output), err)
	}

	// Find the extracted binary
	binaryPath := filepath.Join(destDir, binaryName)
	if _, err := os.Stat(binaryPath); err != nil {
		// Maybe it's in a subdirectory
		entries, _ := os.ReadDir(destDir)
		for _, entry := range entries {
			if entry.Name() == binaryName {
				binaryPath = filepath.Join(destDir, entry.Name())
				break
			}
			if entry.IsDir() {
				subPath := filepath.Join(destDir, entry.Name(), binaryName)
				if _, err := os.Stat(subPath); err == nil {
					binaryPath = subPath
					break
				}
			}
		}
	}

	if _, err := os.Stat(binaryPath); os.IsNotExist(err) {
		return "", fmt.Errorf("binary not found in archive")
	}

	// Make it executable
	if err := os.Chmod(binaryPath, 0o755); err != nil {
		return "", fmt.Errorf("making binary executable: %w", err)
	}

	return binaryPath, nil
}

// replaceBinary replaces the current binary with the new one.
func (u *Updater) replaceBinary(currentPath, newBinaryPath string) error {
	// Create backup of current binary
	backupPath := currentPath + ".old"
	if err := os.Rename(currentPath, backupPath); err != nil {
		// On Windows, we might need to delete first
		if runtime.GOOS == "windows" {
			os.Remove(backupPath)
			if err := os.Rename(currentPath, backupPath); err != nil {
				return fmt.Errorf("creating backup: %w", err)
			}
		} else {
			return fmt.Errorf("creating backup: %w", err)
		}
	}

	// Move new binary to current location
	if err := os.Rename(newBinaryPath, currentPath); err != nil {
		// Restore backup on failure
		os.Rename(backupPath, currentPath)
		return fmt.Errorf("installing new binary: %w", err)
	}

	// Remove backup
	os.Remove(backupPath)

	return nil
}

// DetectInstallMethod returns how ocmgr was installed.
func DetectInstallMethod() string {
	execPath, err := os.Executable()
	if err != nil {
		return "unknown"
	}

	execPath, _ = filepath.EvalSymlinks(execPath)

	// Check for Homebrew
	if strings.Contains(execPath, "/homebrew/") ||
		strings.Contains(execPath, "/Cellar/") ||
		strings.Contains(execPath, "/opt/homebrew/") {
		return "homebrew"
	}

	// Check for curl installer default location
	if strings.Contains(execPath, ".local/bin") {
		return "curl"
	}

	// Check if go install was used
	if strings.Contains(execPath, "go/bin") {
		return "go"
	}

	return "manual"
}

// GetUpdateInstructions returns instructions for updating based on install method.
func GetUpdateInstructions(method string) string {
	switch method {
	case "homebrew":
		return "Run: brew upgrade ocmgr"
	case "go":
		return "Run: go install github.com/acchapm1/ocmgr/cmd/ocmgr@latest"
	case "curl":
		return "Run: ocmgr update"
	default:
		return "Run: curl -sSL https://raw.githubusercontent.com/acchapm1/ocmgr/main/install.sh | bash"
	}
}
