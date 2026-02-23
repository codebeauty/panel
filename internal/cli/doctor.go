package cli

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"

	"github.com/codebeauty/panel/internal/config"
	"github.com/codebeauty/panel/internal/tui"
)

func newDoctorCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "doctor",
		Short: "Check configuration and tool availability",
		RunE: func(cmd *cobra.Command, args []string) error {
			rich := tui.IsTTY()
			failed := false

			pass := func(msg string) {
				if rich {
					fmt.Fprintf(os.Stderr, "  %s %s\n", tui.IconSuccess, msg)
				} else {
					fmt.Fprintf(os.Stderr, "✓ %s\n", msg)
				}
			}
			fail := func(msg string) {
				if rich {
					fmt.Fprintf(os.Stderr, "  %s %s\n", tui.IconError, msg)
				} else {
					fmt.Fprintf(os.Stderr, "✗ %s\n", msg)
				}
			}
			warn := func(msg string) {
				if rich {
					fmt.Fprintf(os.Stderr, "  %s %s\n", tui.IconWarning, msg)
				} else {
					fmt.Fprintf(os.Stderr, "⚠ %s\n", msg)
				}
			}
			boldName := func(name string) string {
				if rich {
					return lipgloss.NewStyle().Bold(true).Render(name)
				}
				return name
			}

			// 1. Config file existence
			cfgPath := config.GlobalConfigPath()
			if _, err := os.Stat(cfgPath); err != nil {
				warn(fmt.Sprintf("Config file not found: %s (using defaults)", cfgPath))
			} else {
				pass(fmt.Sprintf("Config file: %s", cfgPath))
			}

			// 2. Config parseable
			cfg, err := config.Load()
			if err != nil {
				fail(fmt.Sprintf("Config invalid: %s", err))
				return fmt.Errorf("config validation failed")
			}
			pass("Config loaded successfully")

			// 3. Tool count
			if len(cfg.Tools) == 0 {
				warn("No tools configured — run 'panel init'")
			} else {
				pass(fmt.Sprintf("%d tool(s) configured", len(cfg.Tools)))
			}

			// 4. Per-tool checks (sorted for deterministic output)
			toolNames := make([]string, 0, len(cfg.Tools))
			for name := range cfg.Tools {
				toolNames = append(toolNames, name)
			}
			sort.Strings(toolNames)

			for _, toolID := range toolNames {
				tc := cfg.Tools[toolID]
				if rich {
					fmt.Fprintf(os.Stderr, "\n  %s\n", boldName(toolID))
				}

				// Binary exists
				binPath := findBinary(tc.Binary)
				if binPath == "" {
					if rich {
						fail(fmt.Sprintf("  Binary: %s (not found)", tc.Binary))
					} else {
						fail(fmt.Sprintf("%s: binary not found (%s)", toolID, tc.Binary))
					}
					failed = true
				} else {
					if rich {
						pass(fmt.Sprintf("  Binary: %s", binPath))
					} else {
						pass(fmt.Sprintf("%s: binary found at %s", toolID, binPath))
					}

					// Version check (only if binary found)
					ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
					out, err := exec.CommandContext(ctx, binPath, "--version").CombinedOutput()
					cancel()
					if err != nil {
						if rich {
							warn("  Could not determine version")
						} else {
							warn(fmt.Sprintf("%s: could not determine version", toolID))
						}
					} else {
						firstLine := strings.TrimSpace(strings.SplitN(string(out), "\n", 2)[0])
						if rich {
							pass(fmt.Sprintf("  Version: %s", firstLine))
						} else {
							pass(fmt.Sprintf("%s: version %s", toolID, firstLine))
						}
					}
				}

				// Read-only info
				if rich {
					fmt.Fprintf(os.Stderr, "      Read-only: %s\n", cfg.Defaults.ReadOnly)
				} else {
					fmt.Fprintf(os.Stderr, "  read-only: %s\n", cfg.Defaults.ReadOnly)
				}
			}

			// 5. Group validation
			groupNames := make([]string, 0, len(cfg.Groups))
			for name := range cfg.Groups {
				groupNames = append(groupNames, name)
			}
			sort.Strings(groupNames)

			if rich && len(groupNames) > 0 {
				fmt.Fprintln(os.Stderr)
			}

			for _, groupName := range groupNames {
				members := cfg.Groups[groupName]
				allValid := true
				for _, member := range members {
					if _, exists := cfg.Tools[member]; !exists {
						fail(fmt.Sprintf("group %s: unknown tool %s", groupName, member))
						failed = true
						allValid = false
					}
				}
				if allValid {
					pass(fmt.Sprintf("group %s: %d tool(s)", groupName, len(members)))
				}
			}

			if failed {
				return fmt.Errorf("doctor found problems")
			}
			return nil
		},
	}
}
