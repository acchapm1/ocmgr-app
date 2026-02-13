package cli

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/acchapm1/ocmgr/internal/configgen"
	"github.com/acchapm1/ocmgr/internal/copier"
	"github.com/acchapm1/ocmgr/internal/mcps"
	"github.com/acchapm1/ocmgr/internal/plugins"
	"github.com/acchapm1/ocmgr/internal/resolver"
	"github.com/acchapm1/ocmgr/internal/store"
	"github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
	Use:   "init [target-dir]",
	Short: "Initialize .opencode directory from a profile",
	Long: `Initialize a .opencode directory by copying one or more profile
contents into the target directory. If no target directory is
specified, the current working directory is used.

Multiple profiles can be layered by passing --profile more than once;
they are applied in order so later profiles override earlier ones.

If a profile has an "extends" field in its profile.toml, the parent
profile is automatically included before the child. Circular
dependencies are detected and reported as errors.

Use --only or --exclude to limit which content directories are copied
(agents, commands, skills, plugins).`,
	Args: cobra.MaximumNArgs(1),
	RunE: runInit,
}

func init() {
	initCmd.Flags().StringSliceP("profile", "p", nil, "profile name(s) to apply (required, may be repeated)")
	initCmd.Flags().BoolP("force", "f", false, "overwrite existing files without prompting")
	initCmd.Flags().BoolP("merge", "m", false, "only copy new files, skip existing ones")
	initCmd.Flags().BoolP("dry-run", "d", false, "preview changes without copying")
	initCmd.Flags().StringP("only", "o", "", "content dirs to include (comma-separated: agents,commands,skills,plugins)")
	initCmd.Flags().StringP("exclude", "e", "", "content dirs to exclude (comma-separated: agents,commands,skills,plugins)")
	_ = initCmd.MarkFlagRequired("profile")
}

func runInit(cmd *cobra.Command, args []string) error {
	profileNames, _ := cmd.Flags().GetStringSlice("profile")
	force, _ := cmd.Flags().GetBool("force")
	merge, _ := cmd.Flags().GetBool("merge")
	dryRun, _ := cmd.Flags().GetBool("dry-run")
	onlyRaw, _ := cmd.Flags().GetString("only")
	excludeRaw, _ := cmd.Flags().GetString("exclude")

	// Validate mutually exclusive flags.
	if force && merge {
		return fmt.Errorf("--force and --merge are mutually exclusive")
	}
	if onlyRaw != "" && excludeRaw != "" {
		return fmt.Errorf("--only and --exclude are mutually exclusive")
	}

	// Parse and validate --only / --exclude values.
	includeDirs, err := parseContentDirs(onlyRaw)
	if err != nil {
		return fmt.Errorf("--only: %w", err)
	}
	excludeDirs, err := parseContentDirs(excludeRaw)
	if err != nil {
		return fmt.Errorf("--exclude: %w", err)
	}

	// Resolve target directory.
	targetDir := "."
	if len(args) == 1 {
		targetDir = args[0]
	}
	absTarget, err := filepath.Abs(targetDir)
	if err != nil {
		return fmt.Errorf("cannot resolve target directory: %w", err)
	}
	targetOpencode := filepath.Join(absTarget, ".opencode")

	// Create a single reader for all interactive prompts.
	// This avoids buffering issues when input is piped.
	reader := bufio.NewReader(os.Stdin)

	// Open the profile store.
	s, err := store.NewStore()
	if err != nil {
		return fmt.Errorf("cannot open store: %w", err)
	}

	// Resolve the extends dependency chain for all requested profiles.
	// This expands "go" (extends "base") into ["base", "go"] so parents
	// are applied first.
	resolved, err := resolver.Resolve(profileNames, func(name string) (string, error) {
		p, err := s.Get(name)
		if err != nil {
			return "", err
		}
		return p.Extends, nil
	})
	if err != nil {
		return fmt.Errorf("resolving profile dependencies: %w", err)
	}

	// If the resolved list differs from what the user requested, show
	// the full chain so the user knows what will be applied.
	if len(resolved) != len(profileNames) || !slicesEqual(resolved, profileNames) {
		fmt.Printf("Resolved dependency chain: %s\n", strings.Join(resolved, " → "))
	}

	// Load every resolved profile up-front so we fail fast.
	type loadedProfile struct {
		name string
		path string
	}
	profiles := make([]loadedProfile, 0, len(resolved))
	for _, name := range resolved {
		p, err := s.Get(name)
		if err != nil {
			return fmt.Errorf("profile %q: %w", name, err)
		}
		profiles = append(profiles, loadedProfile{name: name, path: p.Path})
	}

	// Determine copy strategy.
	var strategy copier.Strategy
	switch {
	case force:
		strategy = copier.StrategyOverwrite
	case merge:
		strategy = copier.StrategyMerge
	default:
		strategy = copier.StrategyPrompt
	}

	// Build copy options.
	opts := copier.Options{
		Strategy:    strategy,
		DryRun:      dryRun,
		IncludeDirs: includeDirs,
		ExcludeDirs: excludeDirs,
		OnConflict: func(src, dst string) (copier.ConflictChoice, error) {
			relPath, _ := filepath.Rel(targetOpencode, dst)
			fmt.Fprintf(os.Stderr, "Conflict: %s\n", relPath)
			fmt.Fprintf(os.Stderr, "  [o]verwrite  [s]kip  [c]ompare  [a]bort\n")
			for {
				fmt.Fprintf(os.Stderr, "Choice: ")
				input, _ := reader.ReadString('\n')
				switch strings.TrimSpace(strings.ToLower(input)) {
				case "o":
					return copier.ChoiceOverwrite, nil
				case "s":
					return copier.ChoiceSkip, nil
				case "c":
					diff := exec.Command("diff", "--color=always", src, dst)
					diff.Stdout = os.Stdout
					diff.Stderr = os.Stderr
					if err := diff.Run(); err != nil {
						// diff returns exit code 1 when files differ — that's expected.
						// Only warn if the command itself failed to run.
						if diff.ProcessState == nil || !diff.ProcessState.Exited() {
							fmt.Fprintf(os.Stderr, "  (diff command failed: %v)\n", err)
						}
					}
					return copier.ChoiceCompare, nil
				case "a":
					return copier.ChoiceCancel, nil
				default:
					continue
				}
			}
		},
	}

	prefix := ""
	if dryRun {
		prefix = "[dry run] "
	}

	// Apply each profile in order.
	for _, lp := range profiles {
		fmt.Printf("%sApplying profile %q …\n", prefix, lp.name)

		result, err := copier.CopyProfile(lp.path, targetOpencode, opts)
		if err != nil {
			return fmt.Errorf("copying profile %q: %w", lp.name, err)
		}

		// Summary: copied files.
		if len(result.Copied) > 0 {
			fmt.Printf("%s✓ Copied %d files\n", prefix, len(result.Copied))
			for _, f := range result.Copied {
				fmt.Printf("    %s\n", f)
			}
		}

		// Summary: skipped files.
		if len(result.Skipped) > 0 {
			fmt.Printf("%s→ Skipped %d files\n", prefix, len(result.Skipped))
			for _, f := range result.Skipped {
				fmt.Printf("    %s\n", f)
			}
		}

		// Summary: errors.
		if len(result.Errors) > 0 {
			fmt.Printf("%s✗ %d errors\n", prefix, len(result.Errors))
			for _, e := range result.Errors {
				fmt.Printf("    %s\n", e)
			}
		}
	}

	// Check for plugin dependencies.
	if copier.DetectPluginDeps(targetOpencode) {
		fmt.Fprintf(os.Stderr, "Plugin dependencies detected. Install now? [y/N] ")
		answer, _ := reader.ReadString('\n')
		answer = strings.TrimSpace(strings.ToLower(answer))
		if answer == "y" {
			if dryRun {
				fmt.Printf("[dry run] Would run: bun install in %s\n", targetOpencode)
			} else {
				install := exec.Command("bun", "install")
				install.Dir = targetOpencode
				install.Stdout = os.Stdout
				install.Stderr = os.Stderr
				if err := install.Run(); err != nil {
					return fmt.Errorf("bun install failed: %w", err)
				}
			}
		} else {
			fmt.Printf("To install later, run: cd %s && bun install\n", targetOpencode)
		}
	}

	// Prompt for plugins and MCPs (skip in dry-run mode).
	if !dryRun {
		if err := promptForPluginsAndMCPs(targetOpencode, reader); err != nil {
			return fmt.Errorf("plugin/MCP selection: %w", err)
		}
	} else {
		fmt.Printf("[dry run] Would prompt for plugins and MCP servers\n")
	}

	return nil
}

// parseContentDirs splits a comma-separated string of content directory
// names, validates each one, and returns the list. An empty input returns
// nil.
func parseContentDirs(raw string) ([]string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}

	var dirs []string
	for _, part := range strings.Split(raw, ",") {
		d := strings.TrimSpace(part)
		if d == "" {
			continue
		}
		if !copier.ValidContentDirs[d] {
			return nil, fmt.Errorf("invalid content directory %q; must be one of: agents, commands, skills, plugins", d)
		}
		dirs = append(dirs, d)
	}
	return dirs, nil
}

// slicesEqual reports whether two string slices have the same elements
// in the same order.
func slicesEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// promptForPluginsAndMCPs prompts the user to select plugins and MCP servers.
func promptForPluginsAndMCPs(targetDir string, reader *bufio.Reader) error {
	// Load plugin registry
	pluginRegistry, err := plugins.Load()
	if err != nil {
		return fmt.Errorf("loading plugins: %w", err)
	}

	// Load MCP registry
	mcpRegistry, err := mcps.Load()
	if err != nil {
		return fmt.Errorf("loading MCPs: %w", err)
	}

	// Skip if nothing to configure
	if pluginRegistry.IsEmpty() && mcpRegistry.IsEmpty() {
		return nil
	}

	// Collect selected plugins and MCPs
	selectedPlugins := []string{}
	selectedMCPs := map[string]configgen.MCPEntry{}

	// Prompt for plugins
	if !pluginRegistry.IsEmpty() {
		selected, err := promptForPlugins(pluginRegistry, reader)
		if err != nil {
			return err
		}
		selectedPlugins = selected
	}

	// Prompt for MCPs
	if !mcpRegistry.IsEmpty() {
		selected, err := promptForMCPs(mcpRegistry, reader)
		if err != nil {
			return err
		}
		selectedMCPs = selected
	}

	// Generate opencode.json if there's anything to write
	if len(selectedPlugins) > 0 || len(selectedMCPs) > 0 {
		opts := configgen.Options{
			Plugins: selectedPlugins,
			MCPs:    selectedMCPs,
		}
		if err := configgen.Generate(targetDir, opts); err != nil {
			return fmt.Errorf("generating opencode.json: %w", err)
		}
		fmt.Printf("✓ Created opencode.json with %d plugin(s) and %d MCP server(s)\n",
			len(selectedPlugins), len(selectedMCPs))
	}

	return nil
}

// promptForPlugins prompts the user to select plugins from the registry.
func promptForPlugins(registry *plugins.Registry, reader *bufio.Reader) ([]string, error) {
	fmt.Printf("\nWould you like to add plugins to this project? [y/N] ")
	answer, _ := reader.ReadString('\n')
	answer = strings.TrimSpace(strings.ToLower(answer))
	if answer != "y" && answer != "yes" {
		return nil, nil
	}

	// Display available plugins
	fmt.Println("\nAvailable plugins:")
	for i, p := range registry.List() {
		fmt.Printf("  %d. %s\n", i+1, p.Name)
		if p.Description != "" {
			fmt.Printf("     %s\n", p.Description)
		}
	}
	fmt.Printf("\nSelect plugins (comma-separated numbers, 'all', or 'none'): ")

	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(strings.ToLower(input))

	if input == "none" || input == "" {
		return nil, nil
	}

	if input == "all" {
		return registry.Names(), nil
	}

	// Parse selection
	return parseSelection(input, registry.List(), func(p plugins.Plugin) string {
		return p.Name
	})
}

// promptForMCPs prompts the user to select MCP servers from the registry.
func promptForMCPs(registry *mcps.Registry, reader *bufio.Reader) (map[string]configgen.MCPEntry, error) {
	fmt.Printf("\nWould you like to add MCP servers to this project? [y/N] ")
	answer, _ := reader.ReadString('\n')
	answer = strings.TrimSpace(strings.ToLower(answer))
	if answer != "y" && answer != "yes" {
		return nil, nil
	}

	// Display available MCPs
	fmt.Println("\nAvailable MCP servers:")
	for i, m := range registry.List() {
		fmt.Printf("  %d. %s\n", i+1, m.Name)
		if m.Description != "" {
			fmt.Printf("     %s\n", m.Description)
		}
	}
	fmt.Printf("\nSelect MCP servers (comma-separated numbers, 'all', or 'none'): ")

	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(strings.ToLower(input))

	if input == "none" || input == "" {
		return nil, nil
	}

	if input == "all" {
		result := make(map[string]configgen.MCPEntry)
		for _, m := range registry.List() {
			result[m.Name] = mcpConfigToEntry(m.Config)
		}
		return result, nil
	}

	// Parse selection
	selected, err := parseSelection(input, registry.List(), func(m mcps.Definition) string {
		return m.Name
	})
	if err != nil {
		return nil, err
	}

	result := make(map[string]configgen.MCPEntry)
	for _, name := range selected {
		if m := registry.GetByName(name); m != nil {
			result[name] = mcpConfigToEntry(m.Config)
		}
	}
	return result, nil
}

// parseSelection parses a comma-separated list of numbers into selected items.
func parseSelection[T any](input string, items []T, getName func(T) string) ([]string, error) {
	var selected []string
	seen := make(map[string]bool)

	parts := strings.Split(input, ",")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		num, err := strconv.Atoi(part)
		if err != nil {
			continue // skip invalid entries
		}

		if num < 1 || num > len(items) {
			continue // skip out of range
		}

		name := getName(items[num-1])
		if !seen[name] {
			seen[name] = true
			selected = append(selected, name)
		}
	}

	// Sort for consistent output
	sort.Strings(selected)
	return selected, nil
}

// mcpConfigToEntry converts an MCP config map to an MCPEntry struct.
func mcpConfigToEntry(config map[string]interface{}) configgen.MCPEntry {
	entry := configgen.MCPEntry{}

	if v, ok := config["type"].(string); ok {
		entry.Type = v
	}
	if v, ok := config["url"].(string); ok {
		entry.URL = v
	}
	if v, ok := config["enabled"].(bool); ok {
		entry.Enabled = v
	}
	if v, ok := config["timeout"].(float64); ok {
		entry.Timeout = int(v)
	}

	// Handle command array
	if cmd, ok := config["command"].([]interface{}); ok {
		for _, c := range cmd {
			if s, ok := c.(string); ok {
				entry.Command = append(entry.Command, s)
			}
		}
	}

	// Handle environment map
	if env, ok := config["environment"].(map[string]interface{}); ok {
		entry.Environment = make(map[string]string)
		for k, v := range env {
			if s, ok := v.(string); ok {
				entry.Environment[k] = s
			}
		}
	}

	// Handle headers map
	if headers, ok := config["headers"].(map[string]interface{}); ok {
		entry.Headers = make(map[string]string)
		for k, v := range headers {
			if s, ok := v.(string); ok {
				entry.Headers[k] = s
			}
		}
	}

	// Handle oauth
	if oauth, ok := config["oauth"]; ok {
		entry.OAuth = oauth
	}

	return entry
}
