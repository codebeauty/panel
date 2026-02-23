package cli

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/spf13/cobra"

	"github.com/codebeauty/panel/internal/config"
	"github.com/codebeauty/panel/internal/expert"
	"github.com/codebeauty/panel/internal/tui"
)

func newTeamsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "teams",
		Short: "Manage expert teams",
	}
	cmd.AddCommand(newTeamsListCmd())
	cmd.AddCommand(newTeamsCreateCmd())
	cmd.AddCommand(newTeamsDeleteCmd())
	return cmd
}

func newTeamsListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "Show all teams",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return err
			}
			if len(cfg.Teams) == 0 {
				fmt.Println("No teams configured.")
				return nil
			}

			names := make([]string, 0, len(cfg.Teams))
			for name := range cfg.Teams {
				names = append(names, name)
			}
			sort.Strings(names)

			if !tui.IsTTY() {
				for _, name := range names {
					fmt.Printf("%-15s %s\n", name, strings.Join(cfg.Teams[name], ", "))
				}
				return nil
			}

			var rows [][]string
			for _, name := range names {
				rows = append(rows, []string{
					name,
					tui.StyleMuted.Render(strings.Join(cfg.Teams[name], ", ")),
				})
			}
			t := tui.Table{
				Headers: []string{"TEAM", "EXPERTS"},
				Rows:    rows,
			}
			fmt.Print(t.Render())
			return nil
		},
	}
}

func newTeamsCreateCmd() *cobra.Command {
	var expertsFlag string

	cmd := &cobra.Command{
		Use:   "create <name>",
		Short: "Create a team of experts",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			if err := config.ValidateName(name); err != nil {
				return fmt.Errorf("invalid team name: %w", err)
			}

			expertIDs := strings.Split(expertsFlag, ",")
			if len(expertIDs) == 0 || (len(expertIDs) == 1 && expertIDs[0] == "") {
				return fmt.Errorf("team must have at least one expert")
			}

			// Check for duplicates
			seen := make(map[string]bool)
			for _, id := range expertIDs {
				if seen[id] {
					return fmt.Errorf("duplicate expert %q in team", id)
				}
				seen[id] = true
			}

			// Validate all experts exist on disk
			expertDir := expert.Dir()
			for _, id := range expertIDs {
				if _, err := expert.Load(id, expertDir); err != nil {
					return fmt.Errorf("unknown expert: %w", err)
				}
			}

			cfg, err := config.Load()
			if err != nil {
				return err
			}

			// Warn about name collisions
			if _, ok := cfg.Groups[name]; ok {
				fmt.Fprintf(os.Stderr, "Warning: %q also exists as a group name\n", name)
			}

			verb := "Created"
			if _, ok := cfg.Teams[name]; ok {
				verb = "Updated"
			}

			cfg.Teams[name] = expertIDs
			if err := config.Save(cfg, config.GlobalConfigPath()); err != nil {
				return err
			}
			fmt.Fprintf(os.Stderr, "%s team %q: %s\n", verb, name, strings.Join(expertIDs, ", "))
			return nil
		},
	}

	cmd.Flags().StringVar(&expertsFlag, "experts", "", "Comma-separated expert IDs (required)")
	cmd.MarkFlagRequired("experts")
	return cmd
}

func newTeamsDeleteCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "delete <name>",
		Short: "Delete a team",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return err
			}
			name := args[0]
			if _, ok := cfg.Teams[name]; !ok {
				return fmt.Errorf("team %q not found", name)
			}
			delete(cfg.Teams, name)
			if err := config.Save(cfg, config.GlobalConfigPath()); err != nil {
				return err
			}
			fmt.Fprintf(os.Stderr, "Deleted team %q\n", name)
			return nil
		},
	}
}
