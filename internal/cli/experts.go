package cli

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/spf13/cobra"

	"github.com/codebeauty/horde/internal/config"
	"github.com/codebeauty/horde/internal/raider"
)

func newExpertsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "raiders",
		Aliases: []string{"experts"},
		Short:   "Manage raider presets",
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
		Short: "List all available raiders",
		RunE: func(cmd *cobra.Command, args []string) error {
			dir := raider.Dir()
			ids, err := raider.List(dir)
			if err != nil {
				return err
			}
			if len(ids) == 0 {
				fmt.Println("No raiders found. Run 'horde wake' or 'horde raiders reset' to install built-in presets.")
				return nil
			}
			for _, id := range ids {
				label := id
				if _, ok := raider.Builtins[id]; ok {
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
		Short: "Print raider file contents",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			dir := raider.Dir()
			content, err := raider.Load(args[0], dir)
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
		Short: "Create a new raider (opens $EDITOR)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id := args[0]
			if err := raider.ValidateID(id); err != nil {
				return err
			}

			dir := raider.Dir()
			if err := os.MkdirAll(dir, 0o700); err != nil {
				return err
			}

			path := filepath.Join(dir, id+".md")
			if _, err := os.Stat(path); err == nil {
				return fmt.Errorf("raider %q already exists — use 'horde raiders edit %s'", id, id)
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
		Short: "Edit an existing raider (opens $EDITOR)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := raider.ValidateID(args[0]); err != nil {
				return err
			}
			dir := raider.Dir()
			path := filepath.Join(dir, args[0]+".md")
			if _, err := os.Stat(path); os.IsNotExist(err) {
				return fmt.Errorf("raider %q not found", args[0])
			}
			return openEditor(path)
		},
	}
}

func newExpertsResetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "reset",
		Short: "Re-sync built-in raider presets",
		RunE: func(cmd *cobra.Command, args []string) error {
			dir := raider.Dir()
			written, err := raider.SyncBuiltins(dir, syncDiffPrompt)
			if err != nil {
				return err
			}
			fmt.Fprintf(os.Stderr, "%d raider(s) written to %s\n", written, dir)
			return nil
		},
	}
}

func newExpertsDeleteCmd() *cobra.Command {
	var force bool

	cmd := &cobra.Command{
		Use:   "delete <id>",
		Short: "Delete a raider",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id := args[0]
			if err := raider.ValidateID(id); err != nil {
				return err
			}
			dir := raider.Dir()
			path := filepath.Join(dir, id+".md")
			if _, err := os.Stat(path); os.IsNotExist(err) {
				return fmt.Errorf("raider %q not found", id)
			}

			if !force {
				cfg, err := config.Load()
				if err != nil {
					return err
				}
				refs := findExpertTeamRefs(id, cfg)
				if len(refs) > 0 {
					return fmt.Errorf("raider %q is used in squads: %s. Use --force to delete anyway",
						id, strings.Join(refs, ", "))
				}
			}

			if err := os.Remove(path); err != nil {
				return err
			}
			fmt.Fprintf(os.Stderr, "Deleted raider %q\n", id)
			return nil
		},
	}

	cmd.Flags().BoolVar(&force, "force", false, "Delete even if referenced by squads")
	return cmd
}

func findExpertTeamRefs(expertID string, cfg *config.Config) []string {
	var refs []string
	for name, members := range cfg.Teams {
		for _, m := range members {
			if m == expertID {
				refs = append(refs, name)
				break
			}
		}
	}
	sort.Strings(refs)
	return refs
}

func syncDiffPrompt(id, existing, builtin string) raider.SyncAction {
	fmt.Fprintf(os.Stderr, "\nRaider %q has been modified.\n", id)
	fmt.Fprintf(os.Stderr, "  [o]verwrite  [s]kip  [b]ackup & overwrite\n")
	fmt.Fprintf(os.Stderr, "  Choice: ")

	var choice string
	fmt.Scanln(&choice)
	switch choice {
	case "o", "overwrite":
		return raider.SyncOverwrite
	case "b", "backup":
		return raider.SyncBackup
	default:
		return raider.SyncSkip
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
