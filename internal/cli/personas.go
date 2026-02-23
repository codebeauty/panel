package cli

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/codebeauty/panel/internal/persona"
)

func newPersonasCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "personas",
		Short: "Manage persona presets",
	}

	cmd.AddCommand(newPersonasListCmd())
	cmd.AddCommand(newPersonasShowCmd())
	cmd.AddCommand(newPersonasCreateCmd())
	cmd.AddCommand(newPersonasEditCmd())
	cmd.AddCommand(newPersonasResetCmd())
	return cmd
}

func newPersonasListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List all available personas",
		RunE: func(cmd *cobra.Command, args []string) error {
			dir := persona.PersonasDir()
			ids, err := persona.List(dir)
			if err != nil {
				return err
			}
			if len(ids) == 0 {
				fmt.Println("No personas found. Run 'panel init' or 'panel personas reset' to install built-in presets.")
				return nil
			}
			for _, id := range ids {
				label := id
				if _, ok := persona.Builtins[id]; ok {
					label += "  (built-in)"
				}
				fmt.Println(label)
			}
			return nil
		},
	}
}

func newPersonasShowCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "show <id>",
		Short: "Print persona file contents",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			dir := persona.PersonasDir()
			content, err := persona.Load(args[0], dir)
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

func newPersonasCreateCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "create <id>",
		Short: "Create a new persona (opens $EDITOR)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id := args[0]
			if err := persona.ValidatePersonaID(id); err != nil {
				return err
			}

			dir := persona.PersonasDir()
			if err := os.MkdirAll(dir, 0o700); err != nil {
				return err
			}

			path := filepath.Join(dir, id+".md")
			if _, err := os.Stat(path); err == nil {
				return fmt.Errorf("persona %q already exists — use 'panel personas edit %s'", id, id)
			}

			template := fmt.Sprintf("You are a %s.\n\nFocus on:\n- \n", id)
			if err := os.WriteFile(path, []byte(template), 0o600); err != nil {
				return err
			}

			return openEditor(path)
		},
	}
}

func newPersonasEditCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "edit <id>",
		Short: "Edit an existing persona (opens $EDITOR)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := persona.ValidatePersonaID(args[0]); err != nil {
				return err
			}
			dir := persona.PersonasDir()
			path := filepath.Join(dir, args[0]+".md")
			if _, err := os.Stat(path); os.IsNotExist(err) {
				return fmt.Errorf("persona %q not found", args[0])
			}
			return openEditor(path)
		},
	}
}

func newPersonasResetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "reset",
		Short: "Re-sync built-in persona presets",
		RunE: func(cmd *cobra.Command, args []string) error {
			dir := persona.PersonasDir()
			written, err := persona.SyncBuiltins(dir, syncDiffPrompt)
			if err != nil {
				return err
			}
			fmt.Fprintf(os.Stderr, "%d persona(s) written to %s\n", written, dir)
			return nil
		},
	}
}

// syncDiffPrompt interactively asks the user what to do with a modified preset.
func syncDiffPrompt(id, existing, builtin string) persona.SyncAction {
	fmt.Fprintf(os.Stderr, "\nPersona %q has been modified.\n", id)
	fmt.Fprintf(os.Stderr, "  [o]verwrite  [s]kip  [b]ackup & overwrite\n")
	fmt.Fprintf(os.Stderr, "  Choice: ")

	var choice string
	fmt.Scanln(&choice)
	switch choice {
	case "o", "overwrite":
		return persona.SyncOverwrite
	case "b", "backup":
		return persona.SyncBackup
	default:
		return persona.SyncSkip
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
