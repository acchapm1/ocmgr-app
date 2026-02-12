// Package copier handles copying profile contents into a target .opencode/
// directory with configurable conflict resolution.
package copier

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// Strategy controls how file conflicts are resolved when copying a profile
// into an existing .opencode/ directory.
type Strategy string

const (
	// StrategyPrompt asks the user for each conflicting file via the
	// OnConflict callback.
	StrategyPrompt Strategy = "prompt"
	// StrategyOverwrite replaces every conflicting file unconditionally.
	StrategyOverwrite Strategy = "overwrite"
	// StrategyMerge copies only new files and leaves existing files untouched.
	StrategyMerge Strategy = "merge"
	// StrategySkip skips all existing files (alias-like behaviour to Merge but
	// semantically indicates the user chose to skip).
	StrategySkip Strategy = "skip"
)

// ConflictChoice represents a per-file decision returned by the OnConflict
// callback when the strategy is StrategyPrompt.
type ConflictChoice int

const (
	// ChoiceOverwrite replaces the existing file with the profile version.
	ChoiceOverwrite ConflictChoice = iota
	// ChoiceSkip leaves the existing file in place.
	ChoiceSkip
	// ChoiceCompare signals that the caller should show a diff and then
	// re-prompt for a final decision.
	ChoiceCompare
	// ChoiceCancel aborts the entire copy operation.
	ChoiceCancel
)

// Options configures the behaviour of CopyProfile.
type Options struct {
	// Strategy determines how conflicting files are handled.
	Strategy Strategy
	// DryRun, when true, populates the Result without writing anything to disk.
	DryRun bool
	// Force is a convenience flag that behaves identically to
	// StrategyOverwrite.
	Force bool
	// OnConflict is called for every conflicting file when the strategy is
	// StrategyPrompt. It receives the source and destination paths and must
	// return a ConflictChoice. If OnConflict is nil and the strategy is
	// StrategyPrompt, conflicting files are skipped.
	OnConflict func(src, dst string) (ConflictChoice, error)
	// IncludeDirs, when non-empty, restricts copying to only the listed
	// content directories (e.g. ["agents", "skills"]).  It is mutually
	// exclusive with ExcludeDirs.
	IncludeDirs []string
	// ExcludeDirs, when non-empty, skips the listed content directories
	// during copying (e.g. ["plugins"]).  It is mutually exclusive with
	// IncludeDirs.
	ExcludeDirs []string
}

// Result summarises the outcome of a CopyProfile invocation.
type Result struct {
	// Copied lists the destination paths of files that were (or would be)
	// written.
	Copied []string
	// Skipped lists the destination paths of files that already existed and
	// were not overwritten.
	Skipped []string
	// Errors lists human-readable descriptions of files that could not be
	// processed.
	Errors []string
}

// profileDirs is the set of top-level directories inside a profile that are
// copied into .opencode/. Everything else (notably profile.toml) is ignored.
var profileDirs = map[string]bool{
	"agents":   true,
	"commands": true,
	"skills":   true,
	"plugins":  true,
}

// errCancelled is returned when the user chooses ChoiceCancel during an
// interactive prompt.
var errCancelled = errors.New("copy operation cancelled by user")

// CopyProfile walks profileDir and copies the relevant subdirectories
// (agents/, commands/, skills/, plugins/) into targetDir, applying the
// conflict resolution strategy described in opts.
//
// profileDir is typically ~/.ocmgr/profiles/<name> and targetDir is the
// project's .opencode/ directory.
func CopyProfile(profileDir, targetDir string, opts Options) (*Result, error) {
	// Normalise the force shorthand.
	if opts.Force {
		opts.Strategy = StrategyOverwrite
	}

	// Build lookup sets for include/exclude filtering.
	includeSet := toSet(opts.IncludeDirs)
	excludeSet := toSet(opts.ExcludeDirs)

	result := &Result{}

	err := filepath.WalkDir(profileDir, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("%s: %v", path, walkErr))
			return nil // continue walking
		}

		// Compute the path relative to the profile root.
		rel, err := filepath.Rel(profileDir, path)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("%s: %v", path, err))
			return nil
		}

		// Skip the profile root itself.
		if rel == "." {
			return nil
		}

		// Determine the top-level component (e.g. "skills" from
		// "skills/analyzing-projects/SKILL.md").
		topLevel := strings.SplitN(rel, string(filepath.Separator), 2)[0]

		// Only descend into recognised profile directories.
		if !profileDirs[topLevel] {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil // skip loose files like profile.toml
		}

		// Apply --only / --exclude filtering.
		if len(includeSet) > 0 && !includeSet[topLevel] {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if len(excludeSet) > 0 && excludeSet[topLevel] {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		// Nothing to copy for directories themselves; they are created
		// implicitly by CopyFile.
		if d.IsDir() {
			return nil
		}

		src := path
		dst := filepath.Join(targetDir, rel)

		// Check whether the destination already exists.
		_, statErr := os.Stat(dst)
		exists := statErr == nil

		if !exists {
			// New file — always copy.
			if !opts.DryRun {
				if err := CopyFile(src, dst); err != nil {
					result.Errors = append(result.Errors, fmt.Sprintf("%s: %v", rel, err))
					return nil
				}
			}
			result.Copied = append(result.Copied, rel)
			return nil
		}

		// File exists — apply conflict strategy.
		switch opts.Strategy {
		case StrategyOverwrite:
			if !opts.DryRun {
				if err := CopyFile(src, dst); err != nil {
					result.Errors = append(result.Errors, fmt.Sprintf("%s: %v", rel, err))
					return nil
				}
			}
			result.Copied = append(result.Copied, rel)

		case StrategyMerge, StrategySkip:
			result.Skipped = append(result.Skipped, rel)

		case StrategyPrompt:
			choice, err := resolveConflict(src, dst, opts.OnConflict)
			if err != nil {
				if errors.Is(err, errCancelled) {
					return err
				}
				result.Errors = append(result.Errors, fmt.Sprintf("%s: %v", rel, err))
				return nil
			}

			switch choice {
			case ChoiceOverwrite:
				if !opts.DryRun {
					if err := CopyFile(src, dst); err != nil {
						result.Errors = append(result.Errors, fmt.Sprintf("%s: %v", rel, err))
						return nil
					}
				}
				result.Copied = append(result.Copied, rel)
			case ChoiceSkip:
				result.Skipped = append(result.Skipped, rel)
			case ChoiceCancel:
				return errCancelled
			}

		default:
			// Unknown strategy — treat as skip to be safe.
			result.Skipped = append(result.Skipped, rel)
		}

		return nil
	})

	if err != nil && errors.Is(err, errCancelled) {
		return result, err
	}

	return result, err
}

// resolveConflict invokes the OnConflict callback, handling the ChoiceCompare
// loop (show diff, then re-prompt). If cb is nil the file is skipped.
func resolveConflict(src, dst string, cb func(string, string) (ConflictChoice, error)) (ConflictChoice, error) {
	if cb == nil {
		return ChoiceSkip, nil
	}

	for {
		choice, err := cb(src, dst)
		if err != nil {
			return choice, err
		}

		if choice == ChoiceCancel {
			return choice, errCancelled
		}

		// ChoiceCompare means "show a diff then ask again", so we loop.
		// The callback itself is responsible for displaying the diff; we
		// simply re-invoke it.
		if choice != ChoiceCompare {
			return choice, nil
		}
	}
}

// CopyFile copies the file at src to dst, creating any necessary parent
// directories. The original file permissions are preserved.
func CopyFile(src, dst string) error {
	info, err := os.Stat(src)
	if err != nil {
		return fmt.Errorf("stat source: %w", err)
	}

	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return fmt.Errorf("create parent dirs: %w", err)
	}

	in, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("open source: %w", err)
	}
	defer in.Close()

	out, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, info.Mode().Perm())
	if err != nil {
		return fmt.Errorf("create destination: %w", err)
	}

	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		return fmt.Errorf("copy data: %w", err)
	}

	return out.Close()
}

// FilesEqual reports whether the files at paths a and b have identical
// contents. It performs a byte-by-byte comparison and returns early on the
// first difference. An error is returned if either file cannot be read.
func FilesEqual(a, b string) (bool, error) {
	infoA, err := os.Stat(a)
	if err != nil {
		return false, err
	}
	infoB, err := os.Stat(b)
	if err != nil {
		return false, err
	}

	// Fast path: different sizes means different contents.
	if infoA.Size() != infoB.Size() {
		return false, nil
	}

	fa, err := os.Open(a)
	if err != nil {
		return false, err
	}
	defer fa.Close()

	fb, err := os.Open(b)
	if err != nil {
		return false, err
	}
	defer fb.Close()

	const bufSize = 32 * 1024
	bufA := make([]byte, bufSize)
	bufB := make([]byte, bufSize)

	for {
		nA, errA := fa.Read(bufA)
		nB, errB := fb.Read(bufB)

		if !bytes.Equal(bufA[:nA], bufB[:nB]) {
			return false, nil
		}

		if errA == io.EOF && errB == io.EOF {
			return true, nil
		}
		if errA == io.EOF || errB == io.EOF {
			return false, nil
		}
		if errA != nil {
			return false, errA
		}
		if errB != nil {
			return false, errB
		}
	}
}

// ValidContentDirs is the set of recognised content directory names that may
// be passed to IncludeDirs or ExcludeDirs.
var ValidContentDirs = map[string]bool{
	"agents":   true,
	"commands": true,
	"skills":   true,
	"plugins":  true,
}

// toSet converts a string slice into a lookup map.
func toSet(items []string) map[string]bool {
	if len(items) == 0 {
		return nil
	}
	m := make(map[string]bool, len(items))
	for _, v := range items {
		m[v] = true
	}
	return m
}

// DetectPluginDeps checks whether targetDir contains any .ts files under a
// plugins/ subdirectory, indicating that TypeScript plugin dependencies may
// need to be installed.
func DetectPluginDeps(targetDir string) bool {
	pluginsDir := filepath.Join(targetDir, "plugins")

	found := false
	_ = filepath.WalkDir(pluginsDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() && strings.HasSuffix(d.Name(), ".ts") {
			found = true
			return filepath.SkipAll
		}
		return nil
	})

	return found
}
