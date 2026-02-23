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
	"github.com/codebeauty/panel/internal/expert"
	"github.com/codebeauty/panel/internal/output"
	"github.com/codebeauty/panel/internal/runner"
	"github.com/codebeauty/panel/internal/tui"
)

// runTUI launches the BubbleTea TUI for `panel run`.
func runTUI(cfg *config.Config, prompt string, toolIDs []string, ro config.ReadOnlyMode, expertFlag string, teamFlag string, preSelected bool) error {
	// Build adapter name map for display
	adapters := make(map[string]string, len(toolIDs))
	for _, id := range toolIDs {
		if tc, ok := cfg.Tools[id]; ok {
			adapters[id] = tc.Adapter
		}
	}

	skipExpert := expertFlag != "" || teamFlag != ""

	// Load available experts
	var expertIDs []string
	builtinSet := make(map[string]bool)
	expertDir := expert.Dir()
	if eids, err := expert.List(expertDir); err == nil {
		expertIDs = eids
		for _, id := range eids {
			if _, ok := expert.Builtins[id]; ok {
				builtinSet[id] = true
			}
		}
	}
	if len(expertIDs) == 0 {
		skipExpert = true
	}

	tuiCfg := tui.RunConfig{
		AllToolIDs: toolIDs,
		Adapters:   adapters,
		ExpertIDs:  expertIDs,
		BuiltinSet: builtinSet,
		Prompt:     prompt,
		SkipSelect: preSelected,
		SkipExpert: skipExpert,
		PreExpert:  expertFlag,
	}

	var program *tea.Program

	dispatch := func(ctx context.Context, selectedToolIDs []string, selectedExpert string) {
		err := executeTUIRun(ctx, program, cfg, prompt, selectedToolIDs, ro, selectedExpert, teamFlag)
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

	if m, ok := finalModel.(tui.Model); ok && m.Err != nil {
		return m.Err
	}
	return nil
}

// executeTUIRun handles the actual tool dispatch, sending progress messages to BubbleTea.
// Tool expansion (duplicates or team cross-product) is done in run.go before calling runTUI.
func executeTUIRun(ctx context.Context, program *tea.Program, cfg *config.Config, prompt string, toolIDs []string, ro config.ReadOnlyMode, expertFlag string, teamFlag string) error {
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

	// Resolve experts
	expertIDs, expertContents, err := resolveExperts(tools, toolIDs, cfg, expertFlag, teamFlag)
	if err != nil {
		return err
	}

	baseParams := adapter.RunParams{
		Prompt:     prompt,
		PromptFile: promptFilePath,
		WorkDir:    mustGetwd(),
		ReadOnly:   adapter.ReadOnlyMode(ro),
		Timeout:    time.Duration(cfg.Defaults.Timeout) * time.Second,
	}

	perToolParams, err := buildExpertParams(tools, expertContents, baseParams, prompt, runDir)
	if err != nil {
		return err
	}

	var results []runner.Result
	if perToolParams == nil {
		results = r.Run(ctx, tools, baseParams, runDir)
	} else {
		results = r.RunWithParams(ctx, tools, perToolParams, runDir)
	}

	writeManifestAndSummary(runDir, prompt, startedAt, results, expertIDs, cfg, ro)

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
	sortToolIDsByAdapter(allIDs, cfg)
	return allIDs, false, nil
}
