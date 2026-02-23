package cli

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/codebeauty/panel/internal/config"
)

func newGroupsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "groups",
		Short: "Manage tool groups",
	}

	cmd.AddCommand(newGroupsListCmd())
	cmd.AddCommand(newGroupsCreateCmd())
	cmd.AddCommand(newGroupsDeleteCmd())
	return cmd
}

func newGroupsListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "Show all groups",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return err
			}
			if len(cfg.Groups) == 0 {
				fmt.Println("No groups configured.")
				return nil
			}
			for name, members := range cfg.Groups {
				fmt.Printf("%-15s %s\n", name, strings.Join(members, ", "))
			}
			return nil
		},
	}
}

func newGroupsCreateCmd() *cobra.Command {
	var toolsFlag string

	cmd := &cobra.Command{
		Use:   "create <name>",
		Short: "Create a group",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return err
			}
			name := args[0]
			tools := strings.Split(toolsFlag, ",")
			for _, t := range tools {
				if _, ok := cfg.Tools[t]; !ok {
					return fmt.Errorf("unknown tool: %q", t)
				}
			}
			cfg.Groups[name] = tools
			if err := config.Save(cfg, config.GlobalConfigPath()); err != nil {
				return err
			}
			fmt.Fprintf(os.Stderr, "Created group %q: %s\n", name, strings.Join(tools, ", "))
			return nil
		},
	}

	cmd.Flags().StringVar(&toolsFlag, "tools", "", "Comma-separated tool IDs (required)")
	cmd.MarkFlagRequired("tools")
	return cmd
}

func newGroupsDeleteCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "delete <name>",
		Short: "Delete a group",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return err
			}
			name := args[0]
			if _, ok := cfg.Groups[name]; !ok {
				return fmt.Errorf("group %q not found", name)
			}
			delete(cfg.Groups, name)
			if err := config.Save(cfg, config.GlobalConfigPath()); err != nil {
				return err
			}
			fmt.Fprintf(os.Stderr, "Deleted group %q\n", name)
			return nil
		},
	}
}
