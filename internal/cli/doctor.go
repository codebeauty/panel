package cli

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"sort"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/codebeauty/panel/internal/config"
)

func newDoctorCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "doctor",
		Short: "Check configuration and tool availability",
		RunE: func(cmd *cobra.Command, args []string) error {
			failed := false

			// 1. Config file existence
			cfgPath := config.GlobalConfigPath()
			if _, err := os.Stat(cfgPath); err != nil {
				fmt.Fprintf(os.Stderr, "⚠ Config file not found: %s (using defaults)\n", cfgPath)
			} else {
				fmt.Fprintf(os.Stderr, "✓ Config file: %s\n", cfgPath)
			}

			// 2. Config parseable
			cfg, err := config.Load()
			if err != nil {
				fmt.Fprintf(os.Stderr, "✗ Config invalid: %s\n", err)
				return fmt.Errorf("config validation failed")
			}
			fmt.Fprintf(os.Stderr, "✓ Config loaded successfully\n")

			// 3. Tool count
			if len(cfg.Tools) == 0 {
				fmt.Fprintf(os.Stderr, "⚠ No tools configured — run 'panel init'\n")
			} else {
				fmt.Fprintf(os.Stderr, "✓ %d tool(s) configured\n", len(cfg.Tools))
			}

			// 4. Per-tool checks (sorted for deterministic output)
			toolNames := make([]string, 0, len(cfg.Tools))
			for name := range cfg.Tools {
				toolNames = append(toolNames, name)
			}
			sort.Strings(toolNames)

			for _, toolID := range toolNames {
				tc := cfg.Tools[toolID]

				// Binary exists
				binPath := findBinary(tc.Binary)
				if binPath == "" {
					fmt.Fprintf(os.Stderr, "✗ %s: binary not found (%s)\n", toolID, tc.Binary)
					failed = true
				} else {
					fmt.Fprintf(os.Stderr, "✓ %s: binary found at %s\n", toolID, binPath)

					// Version check (only if binary found)
					ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
					out, err := exec.CommandContext(ctx, binPath, "--version").CombinedOutput()
					cancel()
					if err != nil {
						fmt.Fprintf(os.Stderr, "⚠ %s: could not determine version\n", toolID)
					} else {
						firstLine := strings.TrimSpace(strings.SplitN(string(out), "\n", 2)[0])
						fmt.Fprintf(os.Stderr, "✓ %s: version %s\n", toolID, firstLine)
					}
				}

				// Read-only info
				fmt.Fprintf(os.Stderr, "  read-only: %s\n", cfg.Defaults.ReadOnly)
			}

			// 5. Group validation
			groupNames := make([]string, 0, len(cfg.Groups))
			for name := range cfg.Groups {
				groupNames = append(groupNames, name)
			}
			sort.Strings(groupNames)

			for _, groupName := range groupNames {
				members := cfg.Groups[groupName]
				allValid := true
				for _, member := range members {
					if _, exists := cfg.Tools[member]; !exists {
						fmt.Fprintf(os.Stderr, "✗ group %s: unknown tool %s\n", groupName, member)
						failed = true
						allValid = false
					}
				}
				if allValid {
					fmt.Fprintf(os.Stderr, "✓ group %s: %d tool(s)\n", groupName, len(members))
				}
			}

			if failed {
				return fmt.Errorf("doctor found problems")
			}
			return nil
		},
	}
}
