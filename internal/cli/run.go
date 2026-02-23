package cli

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
	"golang.org/x/term"

	"github.com/codebeauty/panel/internal/adapter"
	"github.com/codebeauty/panel/internal/config"
	"github.com/codebeauty/panel/internal/expert"
	"github.com/codebeauty/panel/internal/gather"
	"github.com/codebeauty/panel/internal/output"
	"github.com/codebeauty/panel/internal/runner"
	"github.com/codebeauty/panel/internal/tui"
	"github.com/codebeauty/panel/internal/ui"
)

func newRunCmd() *cobra.Command {
	var (
		toolsFlag   string
		groupFlag   string
		readOnly    string
		timeout     int
		outputDir   string
		jsonOutput  bool
		fileFlag    string
		dryRun      bool
		contextFlag string
		expertFlag  string
		teamFlag    string
		yesFlag     bool
	)

	cmd := &cobra.Command{
		Use:   "run [prompt]",
		Short: "Dispatch a prompt to AI tools in parallel",
		Long:  "Sends the same prompt to multiple AI coding agents simultaneously and collects their responses.",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.LoadMerged(mustGetwd())
			if err != nil {
				return fmt.Errorf("loading config: %w", err)
			}

			if teamFlag != "" && expertFlag != "" {
				return fmt.Errorf("--team and --expert are mutually exclusive")
			}

			prompt, err := resolvePrompt(fileFlag, args)
			if err != nil {
				return err
			}

			if contextFlag != "" && fileFlag == "" {
				var patterns []string
				if contextFlag != "." {
					patterns = strings.Split(contextFlag, ",")
				}
				ctx, err := gather.Gather(patterns, 50, mustGetwd())
				if err != nil {
					return fmt.Errorf("gathering context: %w", err)
				}
				prompt = gather.BuildPrompt(prompt, ctx)
			}

			if outputDir != "" {
				cfg.Defaults.OutputDir = outputDir
			}
			if timeout > 0 {
				cfg.Defaults.Timeout = timeout
			}

			ro := config.ReadOnlyMode(cfg.Defaults.ReadOnly)
			if readOnly != "" {
				validated, err := config.ValidateReadOnlyMode(readOnly)
				if err != nil {
					return err
				}
				ro = validated
			}

			// --- TUI path: interactive terminal with alt-screen ---
			if shouldUseTUI(jsonOutput, dryRun) {
				toolIDs, preSelected, err := resolveToolIDsForTUI(cfg, toolsFlag, groupFlag)
				if err != nil {
					return err
				}
				if len(toolIDs) == 0 {
					return fmt.Errorf("no tools configured — run 'panel init' to set up tools")
				}
				if teamFlag != "" {
					teamExperts, err := lookupTeam(cfg, teamFlag)
					if err != nil {
						return err
					}
					toolIDs, err = expandTeamCrossProduct(toolIDs, teamExperts, cfg)
					if err != nil {
						return err
					}
					preSelected = true // skip TUI tool selection, team defines the set
				} else {
					toolIDs = expandDuplicateToolIDs(toolIDs, cfg)
				}
				return runTUI(cfg, prompt, toolIDs, ro, expertFlag, teamFlag, preSelected)
			}

			// --- Non-TUI path: JSON, dry-run, piped, or non-interactive ---
			toolIDs, err := resolveTools(cfg, toolsFlag, groupFlag)
			if err != nil {
				return err
			}
			if len(toolIDs) == 0 {
				return fmt.Errorf("no tools configured — run 'panel init' to set up tools")
			}

			if toolsFlag == "" && groupFlag == "" && len(toolIDs) > 1 &&
				term.IsTerminal(int(os.Stderr.Fd())) && term.IsTerminal(int(os.Stdin.Fd())) {
				toolIDs, err = selectToolsInteractive(toolIDs)
				if err != nil {
					return err
				}
			}

			if teamFlag != "" {
				teamExperts, err := lookupTeam(cfg, teamFlag)
				if err != nil {
					return err
				}
				toolIDs, err = expandTeamCrossProduct(toolIDs, teamExperts, cfg)
				if err != nil {
					return err
				}

				// Confirmation prompt for large cross-products
				if len(toolIDs) > 8 && !yesFlag &&
					term.IsTerminal(int(os.Stderr.Fd())) && term.IsTerminal(int(os.Stdin.Fd())) {
					fmt.Fprintf(os.Stderr, "This will dispatch %d runs (%d experts × %d tools). Continue? [y/N] ",
						len(toolIDs), len(teamExperts), len(toolIDs)/len(teamExperts))
					scanner := bufio.NewScanner(os.Stdin)
					if !scanner.Scan() || strings.ToLower(strings.TrimSpace(scanner.Text())) != "y" {
						return fmt.Errorf("aborted")
					}
				}
			} else {
				toolIDs = expandDuplicateToolIDs(toolIDs, cfg)
			}

			tools, err := buildTools(cfg, toolIDs)
			if err != nil {
				return err
			}

			if dryRun {
				var dryExpertIDs []string
				if teamFlag != "" {
					eids, _, dryErr := resolveTeamExperts(toolIDs, expert.Dir())
					if dryErr != nil {
						fmt.Fprintf(os.Stderr, "warning: %v\n", dryErr)
					}
					dryExpertIDs = eids
				} else {
					eids, _, dryErr := resolveToolExperts(tools, cfg, expertFlag)
					if dryErr != nil {
						fmt.Fprintf(os.Stderr, "warning: %v\n", dryErr)
					}
					dryExpertIDs = eids
				}
				params := adapter.RunParams{
					Prompt:     prompt,
					PromptFile: "<output>/prompt.md",
					WorkDir:    mustGetwd(),
					ReadOnly:   adapter.ReadOnlyMode(ro),
					Timeout:    time.Duration(cfg.Defaults.Timeout) * time.Second,
				}
				for i, tool := range tools {
					inv := tool.Adapter.BuildInvocation(params)
					fmt.Fprintf(os.Stderr, "%s:\n  %s %s\n", tool.ID, inv.Binary, strings.Join(inv.Args, " "))
					if inv.Stdin != "" {
						fmt.Fprintf(os.Stderr, "  stdin: %d bytes\n", len(inv.Stdin))
					}
					if i < len(dryExpertIDs) && dryExpertIDs[i] != "" {
						fmt.Fprintf(os.Stderr, "  expert: %s\n", dryExpertIDs[i])
					}
				}
				return nil
			}

			runDir, err := output.RunDir(cfg.Defaults.OutputDir, prompt)
			if err != nil {
				return err
			}
			promptFilePath := filepath.Join(runDir, "prompt.md")
			if err := output.WritePrompt(runDir, prompt); err != nil {
				return fmt.Errorf("writing prompt: %w", err)
			}

			fmt.Fprintf(os.Stderr, "Dispatching to %d tool(s): %s\n", len(tools), strings.Join(toolIDs, ", "))
			fmt.Fprintf(os.Stderr, "Output: %s\n", runDir)

			ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
			defer stop()

			startedAt := time.Now()
			r := runner.New(cfg.Defaults.MaxParallel)

			prog := ui.NewProgress(toolIDs)
			r.SetProgressFunc(func(toolID, event string, result *runner.Result) {
				switch event {
				case "started":
					prog.MarkRunning(toolID)
				case "completed":
					if result != nil {
						words := len(strings.Fields(string(result.Stdout)))
						prog.MarkDone(toolID, string(result.Status), words)
					}
				}
			})
			prog.Start()
			defer prog.Stop()

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

			manifest := writeManifestAndSummary(runDir, prompt, startedAt, results, expertIDs, cfg, ro)

			if jsonOutput {
				enc := json.NewEncoder(os.Stdout)
				enc.SetIndent("", "  ")
				return enc.Encode(manifest)
			}

			// Rich summary for TTY, plain for non-TTY
			if tui.IsTTY() {
				printRichSummary(results, runDir)
			} else {
				printSummary(results, runDir)
			}
			return nil
		},
	}

	cmd.Flags().StringVarP(&toolsFlag, "tools", "t", "", "Comma-separated tool IDs")
	cmd.Flags().StringVarP(&groupFlag, "group", "g", "", "Named group from config")
	cmd.Flags().StringVarP(&readOnly, "read-only", "r", "", "Read-only mode: enforced, bestEffort, none")
	cmd.Flags().IntVar(&timeout, "timeout", 0, "Per-tool timeout in seconds")
	cmd.Flags().StringVarP(&outputDir, "output", "o", "", "Output directory override")
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output manifest as JSON")
	cmd.Flags().StringVarP(&fileFlag, "file", "f", "", "Read prompt from file")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Show invocations without executing")
	cmd.Flags().StringVarP(&contextFlag, "context", "c", "", "Gather context from paths (comma-separated, or \".\" for git diff)")
	cmd.Flags().StringVarP(&expertFlag, "expert", "E", "", "Expert ID to apply to all tools")
	cmd.Flags().StringVarP(&teamFlag, "team", "T", "", "Named team of experts from config")
	cmd.Flags().BoolVar(&yesFlag, "yes", false, "Skip confirmation prompts")

	return cmd
}

func resolvePrompt(fileFlag string, args []string) (string, error) {
	if fileFlag != "" {
		data, err := os.ReadFile(fileFlag)
		if err != nil {
			return "", fmt.Errorf("reading prompt file: %w", err)
		}
		return string(data), nil
	}
	if len(args) > 0 {
		return strings.Join(args, " "), nil
	}
	stat, _ := os.Stdin.Stat()
	if (stat.Mode() & os.ModeCharDevice) == 0 {
		data, err := io.ReadAll(os.Stdin)
		if err != nil {
			return "", fmt.Errorf("reading stdin: %w", err)
		}
		prompt := string(data)
		if strings.TrimSpace(prompt) != "" {
			return prompt, nil
		}
	}
	return "", fmt.Errorf("no prompt provided (pass as argument, use -f, or pipe to stdin)")
}

func resolveTools(cfg *config.Config, toolsFlag, groupFlag string) ([]string, error) {
	if toolsFlag != "" {
		return strings.Split(toolsFlag, ","), nil
	}
	if groupFlag != "" {
		ids, ok := cfg.Groups[groupFlag]
		if !ok {
			return nil, fmt.Errorf("unknown group: %q", groupFlag)
		}
		return ids, nil
	}
	var ids []string
	for id, tool := range cfg.Tools {
		if tool.Enabled {
			ids = append(ids, id)
		}
	}
	sortToolIDsByAdapter(ids, cfg)
	return ids, nil
}

func sortToolIDsByAdapter(ids []string, cfg *config.Config) {
	sort.Slice(ids, func(i, j int) bool {
		ai := cfg.Tools[ids[i]].Adapter
		aj := cfg.Tools[ids[j]].Adapter
		if ai != aj {
			return ai < aj
		}
		return ids[i] < ids[j]
	})
}

func buildTools(cfg *config.Config, toolIDs []string) ([]runner.Tool, error) {
	var tools []runner.Tool
	for _, id := range toolIDs {
		tc, ok := cfg.Tools[id]
		if !ok {
			return nil, fmt.Errorf("unknown tool: %q", id)
		}

		adapterName := tc.Adapter
		if adapterName == "" {
			adapterName = id
		}

		var a adapter.Adapter
		switch adapterName {
		case "claude":
			a = adapter.NewClaudeAdapter(tc.Binary, tc.ExtraFlags)
		case "codex":
			a = adapter.NewCodexAdapter(tc.Binary, tc.ExtraFlags)
		case "gemini":
			a = adapter.NewGeminiAdapter(tc.Binary, tc.ExtraFlags)
		case "amp":
			a = adapter.NewAmpAdapter(tc.Binary, tc.ExtraFlags)
		case "cursor-agent":
			a = adapter.NewCursorAdapter(tc.Binary, tc.ExtraFlags)
		default:
			a = adapter.NewCustomAdapter(id, tc.Binary, tc.ExtraFlags, tc.Stdin)
		}

		tools = append(tools, runner.Tool{ID: id, Adapter: a})
	}
	return tools, nil
}

func lookupTeam(cfg *config.Config, teamName string) ([]string, error) {
	experts, ok := cfg.Teams[teamName]
	if !ok {
		return nil, fmt.Errorf("unknown team: %q", teamName)
	}
	if len(experts) == 0 {
		return nil, fmt.Errorf("team %q has no experts", teamName)
	}
	return experts, nil
}

func mustGetwd() string {
	wd, err := os.Getwd()
	if err != nil {
		return "."
	}
	return wd
}

func resolveOutputDir(flagValue string) (string, error) {
	if flagValue != "" {
		return flagValue, nil
	}
	cfg, err := config.LoadMerged(mustGetwd())
	if err != nil {
		return "", fmt.Errorf("loading config: %w", err)
	}
	return cfg.Defaults.OutputDir, nil
}

func selectToolsInteractive(toolIDs []string) ([]string, error) {
	sort.Strings(toolIDs)
	fmt.Fprintln(os.Stderr, "Available tools:")
	for i, id := range toolIDs {
		fmt.Fprintf(os.Stderr, "  %d) %s\n", i+1, id)
	}
	fmt.Fprintf(os.Stderr, "\nSelect tools (comma-separated numbers, or Enter for all): ")

	scanner := bufio.NewScanner(os.Stdin)
	if !scanner.Scan() {
		return toolIDs, nil
	}
	input := strings.TrimSpace(scanner.Text())
	if input == "" {
		return toolIDs, nil
	}

	var selected []string
	for _, part := range strings.Split(input, ",") {
		part = strings.TrimSpace(part)
		n, err := strconv.Atoi(part)
		if err != nil || n < 1 || n > len(toolIDs) {
			return nil, fmt.Errorf("invalid selection: %q", part)
		}
		selected = append(selected, toolIDs[n-1])
	}
	if len(selected) == 0 {
		return toolIDs, nil
	}
	return selected, nil
}

func printRichSummary(results []runner.Result, runDir string) {
	fmt.Fprintf(os.Stderr, "\n%s\n", tui.StyleBold.Render("--- Results ---"))
	maxW := 0
	for _, r := range results {
		if w := lipgloss.Width(tui.FormatToolID(r.ToolID)); w > maxW {
			maxW = w
		}
	}
	for _, r := range results {
		icon := tui.StatusIcon(string(r.Status))
		display := tui.FormatToolID(r.ToolID)
		pad := max(0, maxW-lipgloss.Width(display))
		fmt.Fprintf(os.Stderr, " %s %s%s %s %s %s\n",
			icon, display, strings.Repeat(" ", pad), r.Status,
			tui.StyleMuted.Render(fmt.Sprintf("(exit %d)", r.ExitCode)),
			tui.StyleMuted.Render(r.Duration.Round(time.Millisecond).String()))
		if r.Status != runner.StatusSuccess {
			if snippet := stderrSnippet(r.Stderr); snippet != "" {
				fmt.Fprintf(os.Stderr, "   %s\n", tui.StyleMuted.Render(snippet))
			}
		}
	}
	fmt.Fprintf(os.Stderr, "\n%s %s\n", tui.StyleBold.Render("Output:"), runDir)
}

var plainIcons = map[runner.Status]string{
	runner.StatusSuccess:   "+",
	runner.StatusFailed:    "x",
	runner.StatusTimeout:   "!",
	runner.StatusCancelled: "-",
}

func printSummary(results []runner.Result, runDir string) {
	fmt.Fprintf(os.Stderr, "\n--- Results ---\n")
	maxLen := 0
	for _, r := range results {
		if len(r.ToolID) > maxLen {
			maxLen = len(r.ToolID)
		}
	}
	fmtStr := fmt.Sprintf(" %%s %%-%ds %%s (exit %%d) %%s\n", maxLen)
	for _, r := range results {
		icon := plainIcons[r.Status]
		if icon == "" {
			icon = "?"
		}
		fmt.Fprintf(os.Stderr, fmtStr,
			icon, r.ToolID, r.Status, r.ExitCode, r.Duration.Round(time.Millisecond))
		if r.Status != runner.StatusSuccess {
			if snippet := stderrSnippet(r.Stderr); snippet != "" {
				fmt.Fprintf(os.Stderr, "   %s\n", snippet)
			}
		}
	}
	fmt.Fprintf(os.Stderr, "\nOutput: %s\n", runDir)
}

func stderrSnippet(stderr []byte) string {
	s := strings.TrimSpace(string(stderr))
	if s == "" {
		return ""
	}
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		s = s[:i]
	}
	if len(s) > 120 {
		s = s[:120] + "..."
	}
	return s
}

func resolveToolExperts(tools []runner.Tool, cfg *config.Config, expertFlag string) (ids []string, contents []string, err error) {
	expertDir := expert.Dir()
	ids = make([]string, len(tools))
	contents = make([]string, len(tools))

	for i, tool := range tools {
		eid := expertFlag // CLI flag wins
		if eid == "" {
			if tc, ok := cfg.Tools[tool.ID]; ok {
				eid = tc.Expert
			}
		}
		if eid == "" {
			continue
		}
		content, err := expert.Load(eid, expertDir)
		if err != nil {
			return nil, nil, fmt.Errorf("loading expert %q for %s: %w", eid, tool.ID, err)
		}
		ids[i] = eid
		contents[i] = content
	}
	return ids, contents, nil
}

func resolveExperts(tools []runner.Tool, toolIDs []string, cfg *config.Config, expertFlag, teamFlag string) (ids []string, contents []string, err error) {
	if teamFlag != "" {
		return resolveTeamExperts(toolIDs, expert.Dir())
	}
	return resolveToolExperts(tools, cfg, expertFlag)
}

func buildExpertParams(tools []runner.Tool, expertContents []string, baseParams adapter.RunParams, prompt, runDir string) ([]adapter.RunParams, error) {
	hasExpert := false
	for _, c := range expertContents {
		if c != "" {
			hasExpert = true
			break
		}
	}
	if !hasExpert {
		return nil, nil
	}

	params := make([]adapter.RunParams, len(tools))
	for i, tool := range tools {
		p := baseParams
		if expertContents[i] != "" {
			p.Prompt = expert.Inject(expertContents[i], prompt)
			toolPromptPath := filepath.Join(runDir, tool.ID+".prompt.md")
			if err := os.WriteFile(toolPromptPath, []byte(p.Prompt), 0o600); err != nil {
				return nil, fmt.Errorf("writing expert prompt for %s: %w", tool.ID, err)
			}
			p.PromptFile = toolPromptPath
		}
		params[i] = p
	}
	return params, nil
}

func writeManifestAndSummary(runDir, prompt string, startedAt time.Time, results []runner.Result, expertIDs []string, cfg *config.Config, ro config.ReadOnlyMode) *output.Manifest {
	manifest := output.BuildManifest(prompt, startedAt, results, output.ManifestConfig{
		ReadOnly:    string(ro),
		Timeout:     cfg.Defaults.Timeout,
		MaxParallel: cfg.Defaults.MaxParallel,
	})
	for i, eid := range expertIDs {
		if eid != "" && i < len(manifest.Results) {
			manifest.Results[i].Expert = eid
		}
	}
	if err := output.WriteManifest(runDir, manifest); err != nil {
		fmt.Fprintf(os.Stderr, "warning: failed to write manifest: %v\n", err)
	}
	summary := output.BuildSummary(manifest, runDir)
	if err := output.WriteSummary(runDir, summary); err != nil {
		fmt.Fprintf(os.Stderr, "warning: failed to write summary: %v\n", err)
	}
	return manifest
}
