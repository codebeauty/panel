package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"
	"golang.org/x/term"

	"github.com/codebeauty/horde/internal/output"
)

type jsonCandidate struct {
	Name  string    `json:"name"`
	Path  string    `json:"path"`
	Mtime time.Time `json:"mtime"`
}

func toJSONCandidates(candidates []output.Candidate) []jsonCandidate {
	jc := make([]jsonCandidate, len(candidates))
	for i, c := range candidates {
		jc[i] = jsonCandidate{Name: c.Name, Path: c.Path, Mtime: c.Mtime}
	}
	return jc
}

func newCleanupCmd() *cobra.Command {
	var (
		olderThan string
		outputDir string
		dryRun    bool
		yes       bool
		jsonOut   bool
	)

	cmd := &cobra.Command{
		Use:   "cleanup",
		Short: "Remove old output directories",
		RunE: func(cmd *cobra.Command, args []string) error {
			dur, err := output.ParseDuration(olderThan)
			if err != nil {
				return fmt.Errorf("invalid --older-than: %w", err)
			}

			baseDir, err := resolveOutputDir(outputDir)
			if err != nil {
				return err
			}

			cutoff := time.Now().Add(-dur)
			candidates, err := output.ScanCandidates(baseDir, cutoff)
			if err != nil {
				return err
			}

			if len(candidates) == 0 {
				if jsonOut {
					fmt.Println("[]")
				} else {
					fmt.Fprintln(os.Stderr, "No directories to clean up.")
				}
				return nil
			}

			if dryRun {
				if jsonOut {
					enc := json.NewEncoder(os.Stdout)
					enc.SetIndent("", "  ")
					return enc.Encode(toJSONCandidates(candidates))
				}
				fmt.Fprintf(os.Stderr, "Would remove %d director(ies):\n", len(candidates))
				for _, c := range candidates {
					fmt.Fprintf(os.Stderr, "  %s (modified %s)\n", c.Name, c.Mtime.Format(time.RFC3339))
				}
				return nil
			}

			if !yes {
				if !term.IsTerminal(int(os.Stdin.Fd())) {
					return fmt.Errorf("refusing to delete without --yes in non-interactive mode")
				}
				fmt.Fprintf(os.Stderr, "Will remove %d director(ies):\n", len(candidates))
				for _, c := range candidates {
					fmt.Fprintf(os.Stderr, "  %s\n", c.Name)
				}
				fmt.Fprintf(os.Stderr, "\nProceed? [y/N] ")
				var answer string
				fmt.Scanln(&answer)
				if answer != "y" && answer != "Y" {
					fmt.Fprintln(os.Stderr, "Cancelled.")
					return nil
				}
			}

			removed := 0
			for _, c := range candidates {
				if err := os.RemoveAll(c.Path); err != nil {
					fmt.Fprintf(os.Stderr, "  error removing %s: %v\n", c.Name, err)
				} else {
					removed++
					if !jsonOut {
						fmt.Fprintf(os.Stderr, "  removed: %s\n", c.Name)
					}
				}
			}
			fmt.Fprintf(os.Stderr, "Removed %d director(ies)\n", removed)

			if jsonOut {
				enc := json.NewEncoder(os.Stdout)
				enc.SetIndent("", "  ")
				return enc.Encode(toJSONCandidates(candidates))
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&olderThan, "older-than", "1d", "Age threshold (e.g., 1d, 2w, 30m)")
	cmd.Flags().StringVarP(&outputDir, "output-dir", "o", "", "Output directory (default: from config)")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Show what would be removed without deleting")
	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "Skip confirmation prompt")
	cmd.Flags().BoolVar(&jsonOut, "json", false, "Output in JSON format")

	return cmd
}
