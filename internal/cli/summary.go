package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/codebeauty/panel/internal/output"
)

func newSummaryCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "summary",
		Short: "View run summaries",
	}

	cmd.AddCommand(newSummaryLatestCmd())
	cmd.AddCommand(newSummaryListCmd())

	return cmd
}

func newSummaryLatestCmd() *cobra.Command {
	var (
		outputDir string
		showPath  bool
		jsonOut   bool
	)

	cmd := &cobra.Command{
		Use:   "latest",
		Short: "Show the most recent run summary",
		RunE: func(cmd *cobra.Command, args []string) error {
			baseDir, err := resolveOutputDir(outputDir)
			if err != nil {
				return err
			}

			runs, err := output.ScanRuns(baseDir)
			if err != nil {
				return err
			}
			if len(runs) == 0 {
				return fmt.Errorf("no runs found in %s", baseDir)
			}

			latest := runs[0]

			if showPath {
				fmt.Fprintln(cmd.OutOrStdout(), latest.Path)
				return nil
			}

			if jsonOut {
				m, err := output.ReadManifest(latest.Path)
				if err != nil {
					return fmt.Errorf("reading manifest: %w", err)
				}
				enc := json.NewEncoder(cmd.OutOrStdout())
				enc.SetIndent("", "  ")
				return enc.Encode(m)
			}

			data, err := os.ReadFile(filepath.Join(latest.Path, "summary.md"))
			if err != nil {
				return fmt.Errorf("reading summary: %w", err)
			}
			fmt.Fprint(cmd.OutOrStdout(), string(data))
			return nil
		},
	}

	cmd.Flags().StringVarP(&outputDir, "output-dir", "o", "", "Output directory (default: from config)")
	cmd.Flags().BoolVar(&showPath, "path", false, "Print the run directory path instead of summary")
	cmd.Flags().BoolVar(&jsonOut, "json", false, "Print the run manifest (run.json) instead of summary")

	return cmd
}

func newSummaryListCmd() *cobra.Command {
	var (
		outputDir string
		limit     int
		jsonOut   bool
	)

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List recent runs",
		RunE: func(cmd *cobra.Command, args []string) error {
			baseDir, err := resolveOutputDir(outputDir)
			if err != nil {
				return err
			}

			runs, err := output.ScanRuns(baseDir)
			if err != nil {
				return err
			}
			if len(runs) == 0 {
				fmt.Fprintln(cmd.ErrOrStderr(), "No runs found.")
				return nil
			}

			if limit > 0 && limit < len(runs) {
				runs = runs[:limit]
			}

			if jsonOut {
				var manifests []*output.Manifest
				for _, r := range runs {
					m, err := output.ReadManifest(r.Path)
					if err != nil {
						continue
					}
					manifests = append(manifests, m)
				}
				enc := json.NewEncoder(cmd.OutOrStdout())
				enc.SetIndent("", "  ")
				return enc.Encode(manifests)
			}

			w := cmd.OutOrStdout()
			for _, r := range runs {
				m, err := output.ReadManifest(r.Path)
				if err != nil {
					continue
				}

				fmt.Fprintf(w, "─── %s ───\n", r.Mtime.Format("2006-01-02 15:04"))

				prompt := m.Prompt
				if len(prompt) > 60 {
					prompt = prompt[:60] + "..."
				}
				fmt.Fprintf(w, "Prompt: %s\n", prompt)

				if len(m.Results) > 0 {
					toolSummaries := make([]string, len(m.Results))
					for i, res := range m.Results {
						icon := "✓"
						if res.Status != "success" {
							icon = "✗"
						}
						toolSummaries[i] = fmt.Sprintf("%s (%s %s)", res.ToolID, icon, res.Duration)
					}
					fmt.Fprintf(w, "Tools:  %s\n", strings.Join(toolSummaries, ", "))
				}

				fmt.Fprintf(w, "Path:   %s\n\n", r.Path)
			}

			return nil
		},
	}

	cmd.Flags().StringVarP(&outputDir, "output-dir", "o", "", "Output directory (default: from config)")
	cmd.Flags().IntVar(&limit, "limit", 10, "Maximum number of runs to show")
	cmd.Flags().BoolVar(&jsonOut, "json", false, "Output as JSON array of manifests")

	return cmd
}
