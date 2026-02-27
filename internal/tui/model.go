package tui

import (
	"context"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
)

type RunConfig struct {
	AllToolIDs []string          // all enabled tool IDs for selection
	Adapters   map[string]string // toolID -> adapter name
	ExpertIDs  []string          // available experts
	BuiltinSet map[string]bool   // which experts are built-in
	Prompt     string
	SkipSelect bool   // --tools or --group provided
	SkipExpert bool   // -E flag or no experts
	PreExpert  string // expert from -E flag
}

type DeployFunc func(ctx context.Context, toolIDs []string, expert string)

type Model struct {
	phase    Phase
	width    int
	height   int
	Err      error
	quitting bool

	// Phase models
	selectModel   SelectModel
	raiderModel   RaiderModel
	confirmModel  ConfirmModel
	progressModel ProgressModel
	summaryModel  SummaryModel

	// Config
	cfg      RunConfig
	dispatch DeployFunc
	cancel   context.CancelFunc

	// Selected state
	selectedTools  []string
	selectedExpert string
}

func NewModel(cfg RunConfig, dispatch DeployFunc) Model {
	m := Model{
		cfg:      cfg,
		dispatch: dispatch,
	}

	switch {
	case cfg.SkipSelect && cfg.SkipExpert:
		m.selectedTools = cfg.AllToolIDs
		m.selectedExpert = cfg.PreExpert
		// Skip directly to progress for single tool with no expert, else confirm
		if len(cfg.AllToolIDs) == 1 && cfg.PreExpert == "" {
			m.phase = PhaseProgress
			m.progressModel = NewProgressModel(cfg.AllToolIDs)
		} else {
			m = m.withConfirmPhase(false)
		}
	case cfg.SkipSelect:
		m.selectedTools = cfg.AllToolIDs
		m.phase = PhaseRaider
	default:
		m.phase = PhaseSelect
	}

	m.selectModel = NewSelectModel(cfg.AllToolIDs, cfg.Adapters)
	if !cfg.SkipExpert {
		m.raiderModel = NewRaiderModel(cfg.ExpertIDs, cfg.BuiltinSet)
	}

	return m
}

func (m Model) Init() tea.Cmd {
	if m.phase == PhaseProgress {
		return tea.Batch(
			func() tea.Msg { return doDeployMsg{} },
			m.progressModel.Spinner.Tick,
		)
	}
	return nil
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		if m.phase == PhaseSummary {
			var cmd tea.Cmd
			m.summaryModel, cmd = m.summaryModel.Update(msg)
			return m, cmd
		}
		return m, nil

	case tea.KeyMsg:
		if key.Matches(msg, Keys.Quit) {
			m.quitting = true
			if m.cancel != nil {
				m.cancel()
			}
			return m, tea.Quit
		}

	case ErrorMsg:
		m.Err = msg.Err
		m.quitting = true
		return m, tea.Quit

	case ToolStartedMsg:
		if s, ok := m.progressModel.Statuses[msg.ToolID]; ok {
			updated := *s
			updated.Status = "running"
			updated.Started = time.Now()
			m.progressModel.Statuses[msg.ToolID] = &updated
		}
		return m, nil

	case ToolCompletedMsg:
		if s, ok := m.progressModel.Statuses[msg.ToolID]; ok {
			updated := *s
			updated.Status = string(msg.Result.Status)
			updated.Duration = msg.Result.Duration
			updated.Words = len(strings.Fields(string(msg.Result.Stdout)))
			m.progressModel.Statuses[msg.ToolID] = &updated
		}
		return m, nil

	case AllCompletedMsg:
		m.summaryModel = NewSummaryModel(msg.Results, msg.RunDir, m.width, m.height)
		m.phase = PhaseSummary
		return m, nil

	case doDeployMsg:
		ctx, cancel := context.WithCancel(context.Background())
		m.cancel = cancel
		return m, m.startDeploy(ctx)

	case spinner.TickMsg:
		if m.phase == PhaseProgress {
			var cmd tea.Cmd
			m.progressModel.Spinner, cmd = m.progressModel.Spinner.Update(msg)
			return m, cmd
		}
		return m, nil
	}

	switch m.phase {
	case PhaseSelect:
		return m.updateSelect(msg)
	case PhaseRaider:
		return m.updateExpert(msg)
	case PhaseConfirm:
		return m.updateConfirm(msg)
	case PhaseProgress:
		return m, nil // progress is driven by messages
	case PhaseSummary:
		return m.updateSummary(msg)
	}

	return m, nil
}

func (m Model) View() string {
	if m.Err != nil {
		return StyleError.Render("  Error: " + m.Err.Error()) + "\n"
	}

	switch m.phase {
	case PhaseSelect:
		return m.selectModel.View()
	case PhaseRaider:
		return m.raiderModel.View()
	case PhaseConfirm:
		return m.confirmModel.View()
	case PhaseProgress:
		return m.progressModel.View()
	case PhaseSummary:
		return m.summaryModel.View()
	}

	return ""
}

func (m Model) updateSelect(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if key.Matches(msg, Keys.Confirm) {
			m.selectedTools = m.selectModel.SelectedIDs()
			if len(m.selectedTools) == 0 {
				return m, nil // require at least one
			}
			if m.cfg.SkipExpert {
				m.selectedExpert = m.cfg.PreExpert
				m = m.withConfirmPhase(true)
			} else {
				m.phase = PhaseRaider
			}
			return m, nil
		}
	}

	var cmd tea.Cmd
	m.selectModel, cmd = m.selectModel.Update(msg)
	return m, cmd
}

func (m Model) updateExpert(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, Keys.Confirm):
			m.selectedExpert = m.raiderModel.SelectedExpert()
			m = m.withConfirmPhase(true)
			return m, nil
		case key.Matches(msg, Keys.Back):
			if !m.cfg.SkipSelect {
				m.phase = PhaseSelect
			}
			return m, nil
		}
	}

	var cmd tea.Cmd
	m.raiderModel, cmd = m.raiderModel.Update(msg)
	return m, cmd
}

func (m Model) updateConfirm(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, Keys.Confirm):
			m.progressModel = NewProgressModel(m.selectedTools)
			m.phase = PhaseProgress
			ctx, cancel := context.WithCancel(context.Background())
			m.cancel = cancel
			return m, tea.Batch(m.startDeploy(ctx), m.progressModel.Spinner.Tick)
		case key.Matches(msg, Keys.Back):
			if !m.cfg.SkipExpert {
				m.phase = PhaseRaider
			} else if !m.cfg.SkipSelect {
				m.phase = PhaseSelect
			}
			return m, nil
		}
	}
	return m, nil
}

func (m Model) updateSummary(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if key.Matches(msg, Keys.QuitSummary) {
			m.quitting = true
			return m, tea.Quit
		}
	}

	var cmd tea.Cmd
	m.summaryModel, cmd = m.summaryModel.Update(msg)
	return m, cmd
}

func (m Model) withConfirmPhase(canGoBack bool) Model {
	m.phase = PhaseConfirm
	m.confirmModel = ConfirmModel{
		ToolIDs:   m.selectedTools,
		Expert:    m.selectedExpert,
		Prompt:    m.cfg.Prompt,
		CanGoBack: canGoBack,
	}
	return m
}

func (m Model) startDeploy(ctx context.Context) tea.Cmd {
	return func() tea.Msg {
		go func() {
			defer func() { recover() }() // program.Send may panic after exit
			m.dispatch(ctx, m.selectedTools, m.selectedExpert)
		}()
		return nil
	}
}
