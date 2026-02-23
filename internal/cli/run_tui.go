package cli

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/codebeauty/panel/internal/adapter"
	"github.com/codebeauty/panel/internal/config"
	"github.com/codebeauty/panel/internal/output"
	"github.com/codebeauty/panel/internal/persona"
	"github.com/codebeauty/panel/internal/runner"
	"github.com/codebeauty/panel/internal/tui"
)

// runTUI launches the BubbleTea TUI for `panel run`.
func runTUI(cfg *config.Config, prompt string, toolIDs []string, ro config.ReadOnlyMode, personaFlag string, preSelected bool) error {
	// Build adapter name map for display
	adapters := make(map[string]string, len(toolIDs))
	for _, id := range toolIDs {
		if tc, ok := cfg.Tools[id]; ok {
			adapters[id] = tc.Adapter
		}
	}

	// Determine if we should skip phases
	skipSelect := preSelected // tools were pre-resolved via --tools/--group
	skipPersona := personaFlag != ""

	// Load available personas
	var personaIDs []string
	builtinSet := make(map[string]bool)
	personaDir := persona.PersonasDir()
	if pids, err := persona.List(personaDir); err == nil {
		personaIDs = pids
		for _, id := range pids {
			if _, ok := persona.Builtins[id]; ok {
				builtinSet[id] = true
			}
		}
	}
	if len(personaIDs) == 0 {
		skipPersona = true
	}

	tuiCfg := tui.RunConfig{
		AllToolIDs:  toolIDs,
		Adapters:    adapters,
		PersonaIDs:  personaIDs,
		BuiltinSet:  builtinSet,
		Prompt:      prompt,
		SkipSelect:  skipSelect,
		SkipPersona: skipPersona,
		PrePersona:  personaFlag,
	}

	var program *tea.Program

	dispatch := func(ctx context.Context, selectedToolIDs []string, selectedPersona string, _ *tea.Program) {
		err := executeTUIRun(ctx, program, cfg, prompt, selectedToolIDs, ro, selectedPersona)
		if err != nil {
			program.Send(tui.ErrorMsg{Err: err})
		}
	}

	model := tui.NewModel(tuiCfg, dispatch)
	program = tea.NewProgram(model, tea.WithAltScreen())

	finalModel, err := program.Run()
	if err != nil {
		return fmt.Errorf("TUI error: %w", err)
	}

	// Check if the final model has an error
	if m, ok := finalModel.(tui.Model); ok {
		if m.Err != nil {
			return m.Err
		}
	}

	return nil
}

// executeTUIRun handles the actual tool dispatch, sending progress messages to BubbleTea.
func executeTUIRun(ctx context.Context, program *tea.Program, cfg *config.Config, prompt string, toolIDs []string, ro config.ReadOnlyMode, personaFlag string) error {
	toolIDs = expandDuplicateToolIDs(toolIDs, cfg)

	tools, err := buildTools(cfg, toolIDs)
	if err != nil {
		return err
	}

	runDir, err := output.RunDir(cfg.Defaults.OutputDir, prompt)
	if err != nil {
		return err
	}
	promptFilePath := filepath.Join(runDir, "prompt.md")
	if err := output.WritePrompt(runDir, prompt); err != nil {
		return fmt.Errorf("writing prompt: %w", err)
	}

	startedAt := time.Now()
	r := runner.New(cfg.Defaults.MaxParallel)

	// Bridge runner progress to BubbleTea
	r.SetProgressFunc(func(toolID, event string, result *runner.Result) {
		switch event {
		case "started":
			program.Send(tui.ToolStartedMsg{ToolID: toolID})
		case "completed":
			if result != nil {
				program.Send(tui.ToolCompletedMsg{ToolID: toolID, Result: *result})
			}
		}
	})

	// Resolve personas
	personaIDs, personaContents, err := resolveToolPersonas(tools, cfg, personaFlag)
	if err != nil {
		return err
	}

	// Build per-tool params
	baseParams := adapter.RunParams{
		Prompt:     prompt,
		PromptFile: promptFilePath,
		WorkDir:    mustGetwd(),
		ReadOnly:   adapter.ReadOnlyMode(ro),
		Timeout:    time.Duration(cfg.Defaults.Timeout) * time.Second,
	}

	hasPersona := false
	for _, c := range personaContents {
		if c != "" {
			hasPersona = true
			break
		}
	}

	var results []runner.Result
	if !hasPersona {
		results = r.Run(ctx, tools, baseParams, runDir)
	} else {
		perToolParams := make([]adapter.RunParams, len(tools))
		for i, tool := range tools {
			p := baseParams
			if personaContents[i] != "" {
				p.Prompt = persona.InjectPersona(personaContents[i], prompt)
				toolPromptPath := filepath.Join(runDir, tool.ID+".prompt.md")
				if err := os.WriteFile(toolPromptPath, []byte(p.Prompt), 0o600); err != nil {
					return fmt.Errorf("writing persona prompt for %s: %w", tool.ID, err)
				}
				p.PromptFile = toolPromptPath
			}
			perToolParams[i] = p
		}
		results = r.RunWithParams(ctx, tools, perToolParams, runDir)
	}

	// Write manifest and summary
	manifest := output.BuildManifest(prompt, startedAt, results, output.ManifestConfig{
		ReadOnly:    string(ro),
		Timeout:     cfg.Defaults.Timeout,
		MaxParallel: cfg.Defaults.MaxParallel,
	})
	for i, pid := range personaIDs {
		if pid != "" && i < len(manifest.Results) {
			manifest.Results[i].Persona = pid
		}
	}
	if err := output.WriteManifest(runDir, manifest); err != nil {
		fmt.Fprintf(os.Stderr, "warning: failed to write manifest: %v\n", err)
	}
	summary := output.BuildSummary(manifest, runDir)
	if err := output.WriteSummary(runDir, summary); err != nil {
		fmt.Fprintf(os.Stderr, "warning: failed to write summary: %v\n", err)
	}

	program.Send(tui.AllCompletedMsg{
		Results: results,
		RunDir:  runDir,
	})
	return nil
}

// isTTYRun returns true when both stdin and stderr are terminals (interactive mode).
func isTTYRun() bool {
	return tui.IsTTY() && isStdinTerminal()
}

func isStdinTerminal() bool {
	stat, _ := os.Stdin.Stat()
	return (stat.Mode() & os.ModeCharDevice) != 0
}

// shouldUseTUI returns true when the TUI should be used for `panel run`.
func shouldUseTUI(jsonOutput, dryRun bool) bool {
	if jsonOutput || dryRun {
		return false
	}
	return isTTYRun()
}

// resolveToolIDsForTUI resolves tool IDs but defers interactive selection to the TUI.
func resolveToolIDsForTUI(cfg *config.Config, toolsFlag, groupFlag string) (ids []string, preSelected bool, err error) {
	if toolsFlag != "" {
		return strings.Split(toolsFlag, ","), true, nil
	}
	if groupFlag != "" {
		ids, ok := cfg.Groups[groupFlag]
		if !ok {
			return nil, false, fmt.Errorf("unknown group: %q", groupFlag)
		}
		return ids, true, nil
	}
	// Return all enabled tools — TUI will handle selection
	var allIDs []string
	for id, tool := range cfg.Tools {
		if tool.Enabled {
			allIDs = append(allIDs, id)
		}
	}
	return allIDs, false, nil
}
