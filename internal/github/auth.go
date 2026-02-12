// Package github implements profile synchronisation with a remote
// GitHub repository using the git CLI.
package github

import (
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"
)

// repoPattern validates owner/repo format.
var repoPattern = regexp.MustCompile(`^[a-zA-Z0-9._-]+/[a-zA-Z0-9._-]+$`)

// ResolveRemoteURL returns a plain git remote URL for the given
// repository.  Authentication tokens are NEVER embedded in the URL;
// they are injected via git http.extraHeader at command execution
// time (see gitCommandWithAuth).
//
// Supported auth methods:
//
//	"ssh"   → git@github.com:<repo>.git
//	"gh"    → https://github.com/<repo>.git  (token injected via extraHeader)
//	"env"   → https://github.com/<repo>.git  (token injected via extraHeader)
//	"token" → https://github.com/<repo>.git  (token injected via extraHeader)
func ResolveRemoteURL(repo, authMethod string) (string, error) {
	if repo == "" {
		return "", fmt.Errorf("no GitHub repository configured; run: ocmgr config set github.repo <owner/repo>")
	}
	if !repoPattern.MatchString(repo) {
		return "", fmt.Errorf("invalid repository slug %q; expected format: owner/repo", repo)
	}

	if authMethod == "ssh" {
		return fmt.Sprintf("git@github.com:%s.git", repo), nil
	}

	return fmt.Sprintf("https://github.com/%s.git", repo), nil
}

// ResolveToken extracts an authentication token using the configured
// auth method.  Returns an empty string (not an error) if no token is
// available — this allows public repos to work without credentials.
func ResolveToken(authMethod string) string {
	switch authMethod {
	case "gh":
		t, _ := resolveGHToken()
		return t
	case "env":
		return resolveEnvToken()
	case "token":
		t, _ := resolveStoredToken()
		return t
	default:
		return ""
	}
}

// resolveGHToken obtains a token from the GitHub CLI (`gh auth token`).
func resolveGHToken() (string, error) {
	out, err := exec.Command("gh", "auth", "token").Output()
	if err != nil {
		return "", fmt.Errorf("gh auth token failed: %w", err)
	}
	token := strings.TrimSpace(string(out))
	if token == "" {
		return "", fmt.Errorf("gh auth token returned empty string")
	}
	return token, nil
}

// resolveEnvToken reads OCMGR_GITHUB_TOKEN or GITHUB_TOKEN from the
// environment.
func resolveEnvToken() string {
	if t := os.Getenv("OCMGR_GITHUB_TOKEN"); t != "" {
		return t
	}
	return os.Getenv("GITHUB_TOKEN")
}

// resolveStoredToken reads a personal access token from ~/.ocmgr/.token.
// It verifies the file has safe permissions (owner-only).
func resolveStoredToken() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	tokenPath := home + "/.ocmgr/.token"

	info, err := os.Stat(tokenPath)
	if err != nil {
		return "", err
	}
	// Warn if permissions are too open (group or world readable).
	if info.Mode().Perm()&0o077 != 0 {
		return "", fmt.Errorf("token file %s has insecure permissions %04o; run: chmod 600 %s",
			tokenPath, info.Mode().Perm(), tokenPath)
	}

	data, err := os.ReadFile(tokenPath)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(data)), nil
}
