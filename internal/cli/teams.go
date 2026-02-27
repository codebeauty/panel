package cli

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/spf13/cobra"

	"github.com/codebeauty/horde/internal/config"
	"github.com/codebeauty/horde/internal/raider"
	"github.com/codebeauty/horde/internal/tui"
)

func newTeamsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "squads",
		Aliases: []string{"teams"},
		Short:   "Manage raider squads",
	}
	cmd.AddCommand(newTeamsListCmd())
	cmd.AddCommand(newTeamsCreateCmd())
	cmd.AddCommand(newTeamsDeleteCmd())
	return cmd
}

func newTeamsListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "Show all squads",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return err
			}
			if len(cfg.Teams) == 0 {
				fmt.Println("No squads configured.")
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
				Headers: []string{"SQUAD", "RAIDERS"},
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
		Short: "Create a squad of raiders",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			// Merge hidden backward-compat flag
			if v, _ := cmd.Flags().GetString("experts"); v != "" && expertsFlag == "" {
				expertsFlag = v
			}

			name := args[0]
			if err := config.ValidateName(name); err != nil {
				return fmt.Errorf("invalid squad name: %w", err)
			}

			expertIDs := strings.Split(expertsFlag, ",")
			if len(expertIDs) == 0 || (len(expertIDs) == 1 && expertIDs[0] == "") {
				return fmt.Errorf("squad must have at least one raider")
			}

			// Check for duplicates
			seen := make(map[string]bool)
			for _, id := range expertIDs {
				if seen[id] {
					return fmt.Errorf("duplicate raider %q in squad", id)
				}
				seen[id] = true
			}

			// Validate all raiders exist on disk
			expertDir := raider.Dir()
			for _, id := range expertIDs {
				if _, err := raider.Load(id, expertDir); err != nil {
					return fmt.Errorf("unknown raider: %w", err)
				}
			}

			cfg, err := config.Load()
			if err != nil {
				return err
			}

			// Warn about name collisions
			if _, ok := cfg.Groups[name]; ok {
				fmt.Fprintf(os.Stderr, "Warning: %q also exists as a loadout name\n", name)
			}

			verb := "Created"
			if _, ok := cfg.Teams[name]; ok {
				verb = "Updated"
			}

			cfg.Teams[name] = expertIDs
			if err := config.Save(cfg, config.GlobalConfigPath()); err != nil {
				return err
			}
			fmt.Fprintf(os.Stderr, "%s squad %q: %s\n", verb, name, strings.Join(expertIDs, ", "))
			return nil
		},
	}

	cmd.Flags().StringVar(&expertsFlag, "raiders", "", "Comma-separated raider IDs (required)")
	cmd.Flags().String("experts", "", "")
	cmd.Flags().MarkHidden("experts")
	return cmd
}

func newTeamsDeleteCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "delete <name>",
		Short: "Delete a squad",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return err
			}
			name := args[0]
			if _, ok := cfg.Teams[name]; !ok {
				return fmt.Errorf("squad %q not found", name)
			}
			delete(cfg.Teams, name)
			if err := config.Save(cfg, config.GlobalConfigPath()); err != nil {
				return err
			}
			fmt.Fprintf(os.Stderr, "Deleted squad %q\n", name)
			return nil
		},
	}
}
