package cli

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/spf13/cobra"

	"github.com/codebeauty/horde/internal/config"
	"github.com/codebeauty/horde/internal/tui"
)

func newGroupsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "loadouts",
		Aliases: []string{"groups"},
		Short:   "Manage agent loadouts",
	}

	cmd.AddCommand(newGroupsListCmd())
	cmd.AddCommand(newGroupsCreateCmd())
	cmd.AddCommand(newGroupsDeleteCmd())
	return cmd
}

func newGroupsListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "Show all loadouts",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return err
			}
			if len(cfg.Groups) == 0 {
				fmt.Println("No loadouts configured.")
				return nil
			}

			names := make([]string, 0, len(cfg.Groups))
			for name := range cfg.Groups {
				names = append(names, name)
			}
			sort.Strings(names)

			if !tui.IsTTY() {
				for _, name := range names {
					fmt.Printf("%-15s %s\n", name, strings.Join(cfg.Groups[name], ", "))
				}
				return nil
			}

			var rows [][]string
			for _, name := range names {
				rows = append(rows, []string{
					name,
					tui.StyleMuted.Render(strings.Join(cfg.Groups[name], ", ")),
				})
			}
			t := tui.Table{
				Headers: []string{"LOADOUT", "MEMBERS"},
				Rows:    rows,
			}
			fmt.Print(t.Render())
			return nil
		},
	}
}

func newGroupsCreateCmd() *cobra.Command {
	var toolsFlag string

	cmd := &cobra.Command{
		Use:   "create <name>",
		Short: "Create a loadout",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			// Merge hidden backward-compat flag
			if v, _ := cmd.Flags().GetString("tools"); v != "" && toolsFlag == "" {
				toolsFlag = v
			}

			cfg, err := config.Load()
			if err != nil {
				return err
			}
			name := args[0]
			if err := config.ValidateName(name); err != nil {
				return fmt.Errorf("invalid loadout name: %w", err)
			}
			if toolsFlag == "" {
				return fmt.Errorf("--agents is required")
			}
			tools := strings.Split(toolsFlag, ",")
			for _, t := range tools {
				if _, ok := cfg.Tools[t]; !ok {
					return fmt.Errorf("unknown agent: %q", t)
				}
			}
			cfg.Groups[name] = tools
			if err := config.Save(cfg, config.GlobalConfigPath()); err != nil {
				return err
			}
			fmt.Fprintf(os.Stderr, "Created loadout %q: %s\n", name, strings.Join(tools, ", "))
			return nil
		},
	}

	cmd.Flags().StringVar(&toolsFlag, "agents", "", "Comma-separated agent IDs (required)")
	cmd.Flags().String("tools", "", "")
	cmd.Flags().MarkHidden("tools")
	return cmd
}

func newGroupsDeleteCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "delete <name>",
		Short: "Delete a loadout",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return err
			}
			name := args[0]
			if _, ok := cfg.Groups[name]; !ok {
				return fmt.Errorf("loadout %q not found", name)
			}
			delete(cfg.Groups, name)
			if err := config.Save(cfg, config.GlobalConfigPath()); err != nil {
				return err
			}
			fmt.Fprintf(os.Stderr, "Deleted loadout %q\n", name)
			return nil
		},
	}
}
