package cli

import "github.com/spf13/cobra"

var version = "dev"

func newRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:     "panel",
		Short:   "Fan out prompts to multiple AI coding agents in parallel",
		Version: version,
	}

	root.AddCommand(newRunCmd())
	root.AddCommand(newInitCmd())
	root.AddCommand(newToolsCmd())
	root.AddCommand(newGroupsCmd())
	root.AddCommand(newSkillCmd())
	root.AddCommand(newDoctorCmd())
	root.AddCommand(newCleanupCmd())
	root.AddCommand(newSummaryCmd())
	root.AddCommand(newConfigCmd())
	root.AddCommand(newAgentCmd())
	root.AddCommand(newExpertsCmd())
	root.AddCommand(newTeamsCmd())

	// Top-level aliases
	addCmd := newToolsAddCmd()
	addCmd.Use = "add <adapter>"
	root.AddCommand(addCmd)

	lsCmd := &cobra.Command{
		Use:   "ls",
		Short: "List configured tools (alias for 'tools list')",
		RunE:  newToolsListCmd().RunE,
	}
	root.AddCommand(lsCmd)

	return root
}

func Execute() error {
	return newRootCmd().Execute()
}
