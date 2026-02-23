package tui

import (
	"context"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
)

// RunConfig holds the inputs needed to drive the TUI run flow.
type RunConfig struct {
	AllToolIDs  []string          // all enabled tool IDs for selection
	Adapters    map[string]string // toolID -> adapter name
	PersonaIDs  []string          // available personas
	BuiltinSet  map[string]bool   // which personas are built-in
	Prompt      string
	SkipSelect  bool   // --tools or --group provided
	SkipPersona bool   // -P flag or no personas
	PrePersona  string // persona from -P flag
}

// DispatchFunc is called when the user confirms. It runs the tools and sends
// messages back via the tea.Program.
type DispatchFunc func(ctx context.Context, toolIDs []string, persona string, program *tea.Program)

// Model is the top-level BubbleTea model for `panel run`.
type Model struct {
	phase    Phase
	width    int
	height   int
	Err      error
	quitting bool

	// Phase models
	selectModel   SelectModel
	personaModel  PersonaModel
	confirmModel  ConfirmModel
	progressModel ProgressModel
	summaryModel  SummaryModel

	// Config
	cfg      RunConfig
	dispatch DispatchFunc
	cancel   context.CancelFunc

	// Selected state
	selectedTools   []string
	selectedPersona string
}

// NewModel creates the TUI model. The dispatch function is called when execution starts.
func NewModel(cfg RunConfig, dispatch DispatchFunc) Model {
	m := Model{
		cfg:      cfg,
		dispatch: dispatch,
	}

	if cfg.SkipSelect {
		m.selectedTools = cfg.AllToolIDs
		if cfg.SkipPersona {
			m.selectedPersona = cfg.PrePersona
			// Skip directly to progress for single tool with no persona, else confirm
			if len(cfg.AllToolIDs) == 1 && cfg.PrePersona == "" {
				m.phase = PhaseProgress
				m.progressModel = NewProgressModel(cfg.AllToolIDs)
			} else {
				m.phase = PhaseConfirm
				m.confirmModel = ConfirmModel{
					ToolIDs: cfg.AllToolIDs,
					Persona: cfg.PrePersona,
					Prompt:  cfg.Prompt,
				}
			}
		} else {
			m.phase = PhasePersona
		}
	} else {
		m.phase = PhaseSelect
	}

	m.selectModel = NewSelectModel(cfg.AllToolIDs, cfg.Adapters)
	if !cfg.SkipPersona {
		m.personaModel = NewPersonaModel(cfg.PersonaIDs, cfg.BuiltinSet)
	}

	return m
}

func (m Model) Init() tea.Cmd {
	if m.phase == PhaseProgress {
		return tea.Batch(m.startDispatch(), m.progressModel.Spinner.Tick)
	}
	return nil
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
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
			s.Status = "running"
			s.Started = time.Now()
		}
		return m, nil

	case ToolCompletedMsg:
		if s, ok := m.progressModel.Statuses[msg.ToolID]; ok {
			s.Status = string(msg.Result.Status)
			s.Duration = msg.Result.Duration
			s.Words = len(strings.Fields(string(msg.Result.Stdout)))
		}
		return m, nil

	case AllCompletedMsg:
		m.summaryModel = NewSummaryModel(msg.Results, msg.RunDir)
		m.phase = PhaseSummary
		return m, nil

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
	case PhasePersona:
		return m.updatePersona(msg)
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
	case PhasePersona:
		return m.personaModel.View()
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
			if m.cfg.SkipPersona {
				m.selectedPersona = m.cfg.PrePersona
				m.confirmModel = ConfirmModel{
					ToolIDs: m.selectedTools,
					Persona: m.selectedPersona,
					Prompt:  m.cfg.Prompt,
				}
				m.phase = PhaseConfirm
			} else {
				m.phase = PhasePersona
			}
			return m, nil
		}
	}

	var cmd tea.Cmd
	m.selectModel, cmd = m.selectModel.Update(msg)
	return m, cmd
}

func (m Model) updatePersona(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, Keys.Confirm):
			m.selectedPersona = m.personaModel.SelectedPersona()
			m.confirmModel = ConfirmModel{
				ToolIDs: m.selectedTools,
				Persona: m.selectedPersona,
				Prompt:  m.cfg.Prompt,
			}
			m.phase = PhaseConfirm
			return m, nil
		case key.Matches(msg, Keys.Back):
			if !m.cfg.SkipSelect {
				m.phase = PhaseSelect
			}
			return m, nil
		}
	}

	var cmd tea.Cmd
	m.personaModel, cmd = m.personaModel.Update(msg)
	return m, cmd
}

func (m Model) updateConfirm(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, Keys.Confirm):
			m.progressModel = NewProgressModel(m.selectedTools)
			m.phase = PhaseProgress
			return m, tea.Batch(m.startDispatch(), m.progressModel.Spinner.Tick)
		case key.Matches(msg, Keys.Back):
			if !m.cfg.SkipPersona {
				m.phase = PhasePersona
			} else if !m.cfg.SkipSelect {
				m.phase = PhaseSelect
			}
			return m, nil
		}
	}
	return m, nil
}

func (m Model) updateSummary(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Summary is view-only, q/ctrl+c already handled
	return m, nil
}

func (m *Model) startDispatch() tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithCancel(context.Background())
		m.cancel = cancel
		// dispatch runs in a goroutine and sends messages via program.Send()
		// It's called from Init or Confirm phase. The DispatchFunc is responsible
		// for sending ToolStartedMsg, ToolCompletedMsg, and AllCompletedMsg.
		go m.dispatch(ctx, m.selectedTools, m.selectedPersona, nil)
		return nil
	}
}
