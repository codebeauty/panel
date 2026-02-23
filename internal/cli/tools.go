package cli

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"

	"github.com/codebeauty/panel/internal/adapter"
	"github.com/codebeauty/panel/internal/config"
	"github.com/codebeauty/panel/internal/runner"
)

func newToolsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "tools",
		Short: "Manage configured AI tools",
	}

	cmd.AddCommand(newToolsListCmd())
	cmd.AddCommand(newToolsAddCmd())
	cmd.AddCommand(newToolsRemoveCmd())
	cmd.AddCommand(newToolsTestCmd())
	cmd.AddCommand(newToolsDiscoverCmd())
	cmd.AddCommand(newToolsRenameCmd())
	return cmd
}

func newToolsListCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "Show all configured tools",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return err
			}

			if len(cfg.Tools) == 0 {
				fmt.Println("No tools configured. Run 'panel init' to discover tools.")
				return nil
			}

			w := tabwriter.NewWriter(os.Stdout, 0, 4, 2, ' ', 0)
			fmt.Fprintln(w, "ID\tADAPTER\tBINARY\tENABLED")
			for id, t := range cfg.Tools {
				fmt.Fprintf(w, "%s\t%s\t%s\t%v\n", id, t.Adapter, t.Binary, t.Enabled)
			}
			w.Flush()
			return nil
		},
	}
}

func newToolsRemoveCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "remove <id>",
		Short: "Remove a tool",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return err
			}
			id := args[0]
			if _, ok := cfg.Tools[id]; !ok {
				return fmt.Errorf("tool %q not found", id)
			}
			delete(cfg.Tools, id)
			for name, members := range cfg.Groups {
				var filtered []string
				for _, m := range members {
					if m != id {
						filtered = append(filtered, m)
					}
				}
				cfg.Groups[name] = filtered
			}
			if err := config.Save(cfg, config.GlobalConfigPath()); err != nil {
				return err
			}
			fmt.Fprintf(os.Stderr, "Removed %s\n", id)
			return nil
		},
	}
}

func newToolsTestCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "test [id]",
		Short: "Test tool(s) by running with a trivial prompt",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return err
			}

			var toolIDs []string
			if len(args) > 0 {
				toolIDs = args
			} else {
				for id, t := range cfg.Tools {
					if t.Enabled {
						toolIDs = append(toolIDs, id)
					}
				}
			}

			failed := 0
			for _, id := range toolIDs {
				if err := testTool(cfg, id); err != nil {
					fmt.Fprintf(os.Stderr, "  x %s: %v\n", id, err)
					failed++
				}
			}
			if failed > 0 {
				return fmt.Errorf("%d tool(s) failed", failed)
			}
			return nil
		},
	}
}

func testTool(cfg *config.Config, id string) error {
	tools, err := buildTools(cfg, []string{id})
	if err != nil {
		return err
	}

	outDir, err := os.MkdirTemp("", "panel-test-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(outDir)

	promptFile := filepath.Join(outDir, "prompt.md")
	if err := os.WriteFile(promptFile, []byte("Reply OK"), 0o600); err != nil {
		return fmt.Errorf("writing test prompt: %w", err)
	}

	r := runner.New(1)
	results := r.Run(context.Background(), tools, adapter.RunParams{
		Prompt:     "Reply OK",
		PromptFile: promptFile,
		WorkDir:    mustGetwd(),
		ReadOnly:   adapter.ReadOnlyEnforced,
		Timeout:    30 * time.Second,
	}, outDir)

	if len(results) == 0 {
		return fmt.Errorf("no result")
	}
	result := results[0]
	if result.Status != runner.StatusSuccess {
		return fmt.Errorf("%s (exit %d)", result.Status, result.ExitCode)
	}
	fmt.Fprintf(os.Stderr, "  + %s: OK (%s, %s)\n", id, cfg.Tools[id].Binary, result.Duration.Round(time.Millisecond))
	return nil
}

func newToolsDiscoverCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "discover",
		Short: "Scan for installed AI tools not yet configured",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return err
			}

			found := 0
			for _, known := range knownTools {
				alreadyConfigured := false
				if _, exists := cfg.Tools[known.id]; exists {
					alreadyConfigured = true
				}
				for _, m := range adapter.AdapterModels[known.adapter] {
					if _, exists := cfg.Tools[m.CompoundID]; exists {
						alreadyConfigured = true
						break
					}
				}
				if alreadyConfigured {
					continue
				}
				for _, cmdName := range known.commands {
					if path := findBinary(cmdName); path != "" {
						fmt.Fprintf(os.Stderr, "  found: %s -> %s\n", known.id, path)
						found++
						break
					}
				}
			}
			if found == 0 {
				fmt.Fprintln(os.Stderr, "No new tools found.")
			} else {
				fmt.Fprintf(os.Stderr, "\nRun 'panel init --auto' to add them.\n")
			}
			return nil
		},
	}
}

func newToolsRenameCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "rename <old> <new>",
		Short: "Rename a tool",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			oldName := args[0]
			newName := args[1]

			// Validate new name format
			if err := config.ValidateToolName(newName); err != nil {
				return err
			}

			cfg, err := config.Load()
			if err != nil {
				return err
			}

			// Validate old exists
			tc, ok := cfg.Tools[oldName]
			if !ok {
				return fmt.Errorf("tool %q not found", oldName)
			}

			// Validate new doesn't exist
			if _, exists := cfg.Tools[newName]; exists {
				return fmt.Errorf("tool %q already exists", newName)
			}

			// Move config entry
			cfg.Tools[newName] = tc
			delete(cfg.Tools, oldName)

			// Update all group references
			for name, members := range cfg.Groups {
				for i, m := range members {
					if m == oldName {
						members[i] = newName
					}
				}
				cfg.Groups[name] = members
			}

			if err := config.Save(cfg, config.GlobalConfigPath()); err != nil {
				return err
			}
			fmt.Fprintf(os.Stderr, "Renamed %s → %s\n", oldName, newName)
			return nil
		},
	}
}
