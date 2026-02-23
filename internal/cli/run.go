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

	"github.com/spf13/cobra"
	"golang.org/x/term"

	"github.com/codebeauty/panel/internal/adapter"
	"github.com/codebeauty/panel/internal/config"
	"github.com/codebeauty/panel/internal/gather"
	"github.com/codebeauty/panel/internal/output"
	"github.com/codebeauty/panel/internal/expert"
	"github.com/codebeauty/panel/internal/runner"
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

			toolIDs = expandDuplicateToolIDs(toolIDs, cfg)

			tools, err := buildTools(cfg, toolIDs)
			if err != nil {
				return err
			}

			if dryRun {
				expertIDs, _, pErr := resolveToolExperts(tools, cfg, expertFlag)
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
					if pErr == nil && expertIDs[i] != "" {
						fmt.Fprintf(os.Stderr, "  expert: %s\n", expertIDs[i])
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

			// Resolve experts
			expertIDs, expertContents, err := resolveToolExperts(tools, cfg, expertFlag)
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

			hasExpert := false
			for _, c := range expertContents {
				if c != "" {
					hasExpert = true
					break
				}
			}

			var results []runner.Result
			if !hasExpert {
				results = r.Run(ctx, tools, baseParams, runDir)
			} else {
				perToolParams := make([]adapter.RunParams, len(tools))
				for i, tool := range tools {
					p := baseParams
					if expertContents[i] != "" {
						p.Prompt = expert.Inject(expertContents[i], prompt)
						toolPromptPath := filepath.Join(runDir, tool.ID+".prompt.md")
						if err := os.WriteFile(toolPromptPath, []byte(p.Prompt), 0o600); err != nil {
							return fmt.Errorf("writing expert prompt for %s: %w", tool.ID, err)
						}
						p.PromptFile = toolPromptPath
					}
					perToolParams[i] = p
				}
				results = r.RunWithParams(ctx, tools, perToolParams, runDir)
			}

			manifest := output.BuildManifest(prompt, startedAt, results, output.ManifestConfig{
				ReadOnly:    string(ro),
				Timeout:     cfg.Defaults.Timeout,
				MaxParallel: cfg.Defaults.MaxParallel,
			})

			// Record experts in manifest
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

			if jsonOutput {
				enc := json.NewEncoder(os.Stdout)
				enc.SetIndent("", "  ")
				return enc.Encode(manifest)
			}

			printSummary(results, runDir)
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
	sort.Strings(ids)
	return ids, nil
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

func mustGetwd() string {
	wd, err := os.Getwd()
	if err != nil {
		return "."
	}
	return wd
}

// resolveOutputDir returns the flag value if non-empty, otherwise loads the
// merged config and returns the configured output directory.
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

func printSummary(results []runner.Result, runDir string) {
	fmt.Fprintf(os.Stderr, "\n--- Results ---\n")
	for _, r := range results {
		var icon string
		switch r.Status {
		case runner.StatusSuccess:
			icon = "+"
		case runner.StatusFailed:
			icon = "x"
		case runner.StatusTimeout:
			icon = "!"
		case runner.StatusCancelled:
			icon = "-"
		default:
			icon = "?"
		}
		fmt.Fprintf(os.Stderr, " %s %-20s %s (exit %d) %s\n",
			icon, r.ToolID, r.Status, r.ExitCode, r.Duration.Round(time.Millisecond))
	}
	fmt.Fprintf(os.Stderr, "\nOutput: %s\n", runDir)
}

// resolveToolExperts resolves expert content for each tool.
// Returns parallel slices of expert IDs and content (empty string = no expert).
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
