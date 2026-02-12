package cli

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/acchapm1/ocmgr-app/internal/profile"
	"github.com/acchapm1/ocmgr-app/internal/store"
	"github.com/spf13/cobra"
)

var profileCmd = &cobra.Command{
	Use:   "profile",
	Short: "Manage profiles",
	Long:  "List, show, create, and delete profiles in the local store.",
}

var profileListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all profiles in the local store",
	RunE: func(cmd *cobra.Command, args []string) error {
		s, err := store.NewStore()
		if err != nil {
			return fmt.Errorf("opening store: %w", err)
		}

		profiles, err := s.List()
		if err != nil {
			return fmt.Errorf("listing profiles: %w", err)
		}

		if len(profiles) == 0 {
			fmt.Println("No profiles found. Create one with: ocmgr profile create <name>")
			return nil
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintf(w, "NAME\tVERSION\tDESCRIPTION\tTAGS\n")
		for _, p := range profiles {
			desc := p.Description
			if len(desc) > 42 {
				desc = desc[:42] + "..."
			}
			tags := strings.Join(p.Tags, ", ")
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", p.Name, p.Version, desc, tags)
		}
		w.Flush()

		return nil
	},
}

var profileShowCmd = &cobra.Command{
	Use:   "show <name>",
	Short: "Show details of a profile",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]

		s, err := store.NewStore()
		if err != nil {
			return fmt.Errorf("opening store: %w", err)
		}

		p, err := s.Get(name)
		if err != nil {
			return err
		}

		fmt.Printf("Profile: %s\n", p.Name)
		if p.Description != "" {
			fmt.Printf("Description: %s\n", p.Description)
		}
		if p.Version != "" {
			fmt.Printf("Version: %s\n", p.Version)
		}
		if p.Author != "" {
			fmt.Printf("Author: %s\n", p.Author)
		}
		if len(p.Tags) > 0 {
			fmt.Printf("Tags: %s\n", strings.Join(p.Tags, ", "))
		}
		if p.Extends != "" {
			fmt.Printf("Extends: %s\n", p.Extends)
		}

		contents, err := profile.ListContents(p)
		if err != nil {
			return fmt.Errorf("listing contents: %w", err)
		}

		fmt.Println()
		fmt.Println("Contents:")

		if len(contents.Agents) > 0 {
			fmt.Printf("  agents/ (%d files)\n", len(contents.Agents))
			for _, f := range contents.Agents {
				fmt.Printf("    %s\n", strings.TrimPrefix(f, "agents/"))
			}
		}

		if len(contents.Commands) > 0 {
			fmt.Printf("  commands/ (%d files)\n", len(contents.Commands))
			for _, f := range contents.Commands {
				fmt.Printf("    %s\n", strings.TrimPrefix(f, "commands/"))
			}
		}

		if len(contents.Skills) > 0 {
			fmt.Printf("  skills/ (%d skills)\n", len(contents.Skills))
			for _, f := range contents.Skills {
				fmt.Printf("    %s\n", strings.TrimPrefix(f, "skills/"))
			}
		}

		if len(contents.Plugins) > 0 {
			fmt.Printf("  plugins/ (%d files)\n", len(contents.Plugins))
			for _, f := range contents.Plugins {
				fmt.Printf("    %s\n", strings.TrimPrefix(f, "plugins/"))
			}
		}

		return nil
	},
}

var profileCreateCmd = &cobra.Command{
	Use:   "create <name>",
	Short: "Create a new empty profile",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]

		s, err := store.NewStore()
		if err != nil {
			return fmt.Errorf("opening store: %w", err)
		}

		p, err := profile.ScaffoldProfile(s.Dir, name)
		if err != nil {
			return fmt.Errorf("creating profile: %w", err)
		}

		fmt.Printf("Created profile '%s' at %s\n", name, p.Path)
		fmt.Println("Add files to agents/, commands/, skills/, plugins/ directories.")
		return nil
	},
}

var profileDeleteCmd = &cobra.Command{
	Use:   "delete <name>",
	Short: "Delete a profile from the local store",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		force, _ := cmd.Flags().GetBool("force")

		s, err := store.NewStore()
		if err != nil {
			return fmt.Errorf("opening store: %w", err)
		}

		if !force {
			fmt.Printf("Delete profile '%s'? This cannot be undone. [y/N] ", name)
			reader := bufio.NewReader(os.Stdin)
			answer, _ := reader.ReadString('\n')
			answer = strings.TrimSpace(answer)
			if answer != "y" && answer != "Y" {
				fmt.Println("Aborted.")
				return nil
			}
		}

		if err := s.Delete(name); err != nil {
			return err
		}

		fmt.Printf("Deleted profile '%s'\n", name)
		return nil
	},
}

func init() {
	profileDeleteCmd.Flags().BoolP("force", "f", false, "skip confirmation prompt")

	profileCmd.AddCommand(profileListCmd)
	profileCmd.AddCommand(profileShowCmd)
	profileCmd.AddCommand(profileCreateCmd)
	profileCmd.AddCommand(profileDeleteCmd)
}
