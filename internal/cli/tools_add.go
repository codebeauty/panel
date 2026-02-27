package cli

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/codebeauty/horde/internal/adapter"
	"github.com/codebeauty/horde/internal/config"
)

func newToolsAddCmd() *cobra.Command {
	var (
		model    string
		name     string
		binary   string
		flags    string
		stdin    bool
		readOnly string
	)

	cmd := &cobra.Command{
		Use:   "add <adapter>",
		Short: "Add a new agent",
		Long:  "Add an agent using a built-in adapter (claude, codex, gemini, amp, cursor-agent) or custom.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			adapterType := args[0]

			cfg, err := config.Load()
			if err != nil {
				return fmt.Errorf("loading config: %w", err)
			}

			if adapterType == "custom" {
				return addCustomTool(cfg, name, binary, flags, stdin, readOnly)
			}
			return addBuiltinTool(cfg, adapterType, model, name, binary)
		},
	}

	cmd.Flags().StringVar(&model, "model", "", "Model ID (for built-in adapters)")
	cmd.Flags().StringVar(&name, "name", "", "Agent name (default: adapter-model compound ID)")
	cmd.Flags().StringVar(&binary, "binary", "", "Path to binary (default: auto-discover)")
	cmd.Flags().StringVar(&flags, "flags", "", "Extra flags (space-separated)")
	cmd.Flags().BoolVar(&stdin, "stdin", false, "Send prompt via stdin (custom adapter only)")
	cmd.Flags().StringVar(&readOnly, "read-only", "", "Read-only mode (custom adapter only)")

	return cmd
}

func addBuiltinTool(cfg *config.Config, adapterType, modelFlag, nameFlag, binaryFlag string) error {
	models, ok := adapter.AdapterModels[adapterType]
	if !ok {
		return fmt.Errorf("unknown adapter %q — available: %s", adapterType, strings.Join(adapter.BuiltinNames(), ", "))
	}

	var chosen *adapter.Model
	if modelFlag != "" {
		for i := range models {
			if models[i].ID == modelFlag {
				chosen = &models[i]
				break
			}
		}
		if chosen == nil {
			var available []string
			for _, m := range models {
				available = append(available, m.ID)
			}
			return fmt.Errorf("unknown model %q for %s — available: %s", modelFlag, adapterType, strings.Join(available, ", "))
		}
	} else {
		chosen = adapter.RecommendedModel(adapterType)
		if chosen == nil {
			return fmt.Errorf("no recommended model for %s — specify one with --model", adapterType)
		}
	}

	toolName := chosen.CompoundID
	if nameFlag != "" {
		toolName = nameFlag
	}

	if err := config.ValidateToolName(toolName); err != nil {
		return err
	}

	if _, exists := cfg.Tools[toolName]; exists {
		return fmt.Errorf("agent %q already exists — remove it first or choose a different --name", toolName)
	}

	binPath := binaryFlag
	if binPath == "" {
		binPath = findBinary(adapterType)
	}
	if binPath == "" {
		return fmt.Errorf("binary for %s not found in PATH — specify with --binary", adapterType)
	}

	cfg.Tools[toolName] = config.ToolConfig{
		Binary:     binPath,
		Adapter:    adapterType,
		ExtraFlags: chosen.ExtraFlags,
		Enabled:    true,
	}

	cfgPath := config.GlobalConfigPath()
	if err := config.Save(cfg, cfgPath); err != nil {
		return fmt.Errorf("saving config: %w", err)
	}

	fmt.Fprintf(os.Stderr, "Added %s (%s — %s)\n", toolName, chosen.DisplayName, binPath)
	return nil
}

func addCustomTool(cfg *config.Config, nameFlag, binaryFlag, flagsStr string, stdinFlag bool, readOnlyFlag string) error {
	if binaryFlag == "" {
		return fmt.Errorf("--binary is required for custom adapter")
	}
	if nameFlag == "" {
		return fmt.Errorf("--name is required for custom adapter")
	}

	if _, err := os.Stat(binaryFlag); err != nil {
		return fmt.Errorf("binary %q not found: %w", binaryFlag, err)
	}

	var extraFlags []string
	if flagsStr != "" {
		extraFlags = strings.Fields(flagsStr)
	}

	if readOnlyFlag != "" {
		if _, err := config.ValidateReadOnlyMode(readOnlyFlag); err != nil {
			return err
		}
	}

	if err := config.ValidateToolName(nameFlag); err != nil {
		return err
	}

	if _, exists := cfg.Tools[nameFlag]; exists {
		return fmt.Errorf("agent %q already exists — remove it first or choose a different --name", nameFlag)
	}

	cfg.Tools[nameFlag] = config.ToolConfig{
		Binary:     binaryFlag,
		Adapter:    "custom",
		ExtraFlags: extraFlags,
		Enabled:    true,
		Stdin:      stdinFlag,
	}

	cfgPath := config.GlobalConfigPath()
	if err := config.Save(cfg, cfgPath); err != nil {
		return fmt.Errorf("saving config: %w", err)
	}

	fmt.Fprintf(os.Stderr, "Added %s (custom — %s)\n", nameFlag, binaryFlag)
	return nil
}
