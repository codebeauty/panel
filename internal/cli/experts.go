package cli

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/codebeauty/panel/internal/expert"
)

func newExpertsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "experts",
		Short: "Manage expert presets",
	}

	cmd.AddCommand(newExpertsListCmd())
	cmd.AddCommand(newExpertsShowCmd())
	cmd.AddCommand(newExpertsCreateCmd())
	cmd.AddCommand(newExpertsEditCmd())
	cmd.AddCommand(newExpertsResetCmd())
	cmd.AddCommand(newExpertsDeleteCmd())
	return cmd
}

func newExpertsListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List all available experts",
		RunE: func(cmd *cobra.Command, args []string) error {
			dir := expert.Dir()
			ids, err := expert.List(dir)
			if err != nil {
				return err
			}
			if len(ids) == 0 {
				fmt.Println("No experts found. Run 'panel init' or 'panel experts reset' to install built-in presets.")
				return nil
			}
			for _, id := range ids {
				label := id
				if _, ok := expert.Builtins[id]; ok {
					label += "  (built-in)"
				}
				fmt.Println(label)
			}
			return nil
		},
	}
}

func newExpertsShowCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "show <id>",
		Short: "Print expert file contents",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			dir := expert.Dir()
			content, err := expert.Load(args[0], dir)
			if err != nil {
				return err
			}
			fmt.Print(content)
			if len(content) > 0 && content[len(content)-1] != '\n' {
				fmt.Println()
			}
			return nil
		},
	}
}

func newExpertsCreateCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "create <id>",
		Short: "Create a new expert (opens $EDITOR)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id := args[0]
			if err := expert.ValidateID(id); err != nil {
				return err
			}

			dir := expert.Dir()
			if err := os.MkdirAll(dir, 0o700); err != nil {
				return err
			}

			path := filepath.Join(dir, id+".md")
			if _, err := os.Stat(path); err == nil {
				return fmt.Errorf("expert %q already exists — use 'panel experts edit %s'", id, id)
			}

			template := fmt.Sprintf("You are a %s.\n\nFocus on:\n- \n", id)
			if err := os.WriteFile(path, []byte(template), 0o600); err != nil {
				return err
			}

			return openEditor(path)
		},
	}
}

func newExpertsEditCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "edit <id>",
		Short: "Edit an existing expert (opens $EDITOR)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := expert.ValidateID(args[0]); err != nil {
				return err
			}
			dir := expert.Dir()
			path := filepath.Join(dir, args[0]+".md")
			if _, err := os.Stat(path); os.IsNotExist(err) {
				return fmt.Errorf("expert %q not found", args[0])
			}
			return openEditor(path)
		},
	}
}

func newExpertsResetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "reset",
		Short: "Re-sync built-in expert presets",
		RunE: func(cmd *cobra.Command, args []string) error {
			dir := expert.Dir()
			written, err := expert.SyncBuiltins(dir, syncDiffPrompt)
			if err != nil {
				return err
			}
			fmt.Fprintf(os.Stderr, "%d expert(s) written to %s\n", written, dir)
			return nil
		},
	}
}

func newExpertsDeleteCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "delete <id>",
		Short: "Delete an expert",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id := args[0]
			if err := expert.ValidateID(id); err != nil {
				return err
			}
			dir := expert.Dir()
			path := filepath.Join(dir, id+".md")
			if _, err := os.Stat(path); os.IsNotExist(err) {
				return fmt.Errorf("expert %q not found", id)
			}
			if err := os.Remove(path); err != nil {
				return err
			}
			fmt.Fprintf(os.Stderr, "Deleted expert %q\n", id)
			return nil
		},
	}
}

// syncDiffPrompt interactively asks the user what to do with a modified preset.
func syncDiffPrompt(id, existing, builtin string) expert.SyncAction {
	fmt.Fprintf(os.Stderr, "\nExpert %q has been modified.\n", id)
	fmt.Fprintf(os.Stderr, "  [o]verwrite  [s]kip  [b]ackup & overwrite\n")
	fmt.Fprintf(os.Stderr, "  Choice: ")

	var choice string
	fmt.Scanln(&choice)
	switch choice {
	case "o", "overwrite":
		return expert.SyncOverwrite
	case "b", "backup":
		return expert.SyncBackup
	default:
		return expert.SyncSkip
	}
}

func openEditor(path string) error {
	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = "vi"
	}
	c := exec.Command(editor, path)
	c.Stdin = os.Stdin
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	return c.Run()
}
