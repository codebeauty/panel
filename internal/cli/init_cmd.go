package cli

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/codebeauty/horde/internal/adapter"
	"github.com/codebeauty/horde/internal/config"
	"github.com/codebeauty/horde/internal/raider"
)

var knownTools = []struct {
	id       string
	commands []string
	adapter  string
}{
	{"claude", []string{"claude"}, "claude"},
	{"codex", []string{"codex"}, "codex"},
	{"gemini", []string{"gemini"}, "gemini"},
	{"amp", []string{"amp"}, "amp"},
	{"cursor-agent", []string{"cursor-agent"}, "cursor-agent"},
}

var searchPaths = []string{
	"/opt/homebrew/bin",
	"/usr/local/bin",
}

func init() {
	if home := os.Getenv("HOME"); home != "" {
		searchPaths = append(searchPaths, filepath.Join(home, ".local/bin"))
	}
}

func newInitCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "wake",
		Aliases: []string{"init"},
		Short:   "Discover and configure installed AI CLI agents",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return fmt.Errorf("loading config: %w", err)
			}

			discovered := 0
			for _, known := range knownTools {
				for _, cmdName := range known.commands {
					binPath := findBinary(cmdName)
					if binPath == "" {
						continue
					}

					models := adapter.AdapterModels[known.adapter]
					if len(models) > 0 {
						for _, m := range models {
							if _, exists := cfg.Tools[m.CompoundID]; exists {
								fmt.Fprintf(os.Stderr, "  already configured: %s (%s)\n", m.CompoundID, binPath)
								continue
							}
							cfg.Tools[m.CompoundID] = config.ToolConfig{
								Binary:     binPath,
								Adapter:    known.adapter,
								ExtraFlags: m.ExtraFlags,
								Enabled:    m.Recommended,
							}
							label := "discovered"
							if !m.Recommended {
								label = "available"
							}
							fmt.Fprintf(os.Stderr, "  %s: %s — %s\n", label, m.CompoundID, m.DisplayName)
							discovered++
						}
					} else {
						if _, exists := cfg.Tools[known.id]; exists {
							fmt.Fprintf(os.Stderr, "  already configured: %s (%s)\n", known.id, binPath)
							continue
						}
						cfg.Tools[known.id] = config.ToolConfig{
							Binary:  binPath,
							Adapter: known.adapter,
							Enabled: true,
						}
						fmt.Fprintf(os.Stderr, "  discovered: %s -> %s\n", known.id, binPath)
						discovered++
					}
					break
				}
			}

			if discovered == 0 && len(cfg.Tools) == 0 {
				return fmt.Errorf("no AI CLI agents found — install claude, codex, gemini, or amp first")
			}

			cfgPath := config.GlobalConfigPath()
			if err := config.Save(cfg, cfgPath); err != nil {
				return fmt.Errorf("saving config: %w", err)
			}

			fmt.Fprintf(os.Stderr, "\nConfig written to: %s\n", cfgPath)
			fmt.Fprintf(os.Stderr, "%d tool(s) configured\n", len(cfg.Tools))

			// Sync built-in raiders
			expertDir := raider.Dir()
			var diffFn raider.DiffFunc
			auto, _ := cmd.Flags().GetBool("auto")
			if !auto {
				diffFn = syncDiffPrompt
			}
			written, pErr := raider.SyncBuiltins(expertDir, diffFn)
			if pErr != nil {
				fmt.Fprintf(os.Stderr, "warning: failed to sync raiders: %v\n", pErr)
			} else if written > 0 {
				fmt.Fprintf(os.Stderr, "%d raider(s) installed to %s\n", written, expertDir)
			}

			return nil
		},
	}

	cmd.Flags().Bool("auto", false, "Auto-discover without prompting")
	return cmd
}

func findBinary(name string) string {
	if path, err := exec.LookPath(name); err == nil {
		return path
	}
	for _, dir := range searchPaths {
		path := filepath.Join(dir, name)
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}
	return ""
}
