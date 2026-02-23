package ui

import (
	"fmt"
	"os"
	"sync"
	"time"

	"golang.org/x/term"
)

var spinnerFrames = []string{"◐", "◓", "◑", "◒"}

type ToolStatus struct {
	Status  string // pending, running, done, failed, timeout
	Started time.Time
	Words   int
}

type Progress struct {
	toolIDs   []string
	states    map[string]*ToolStatus
	mu        sync.Mutex
	isTTY     bool
	startTime time.Time
	done      chan struct{}
	stopOnce  sync.Once
}

func NewProgress(toolIDs []string) *Progress {
	states := make(map[string]*ToolStatus, len(toolIDs))
	for _, id := range toolIDs {
		states[id] = &ToolStatus{Status: "pending"}
	}
	return &Progress{
		toolIDs:   toolIDs,
		states:    states,
		isTTY:     term.IsTerminal(int(os.Stderr.Fd())),
		startTime: time.Now(),
		done:      make(chan struct{}),
	}
}

func (p *Progress) Start() {
	if !p.isTTY {
		fmt.Fprintf(os.Stderr, "Running %d tool(s)...\n", len(p.toolIDs))
		return
	}
	fmt.Fprintf(os.Stderr, "\nThis may take more than 10 minutes.\n\n")
	go p.animate()
}

func (p *Progress) Stop() {
	p.stopOnce.Do(func() { close(p.done) })
	if p.isTTY {
		// Clear spinner lines
		p.mu.Lock()
		for range p.toolIDs {
			fmt.Fprintf(os.Stderr, "\033[A\033[2K")
		}
		p.mu.Unlock()
	}
}

func (p *Progress) MarkRunning(toolID string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if s, ok := p.states[toolID]; ok {
		s.Status = "running"
		s.Started = time.Now()
	}
	if !p.isTTY {
		fmt.Fprintf(os.Stderr, "  started: %s\n", toolID)
	}
}

func (p *Progress) MarkDone(toolID, status string, words int) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if s, ok := p.states[toolID]; ok {
		s.Status = status
		s.Words = words
	}
	if !p.isTTY {
		fmt.Fprintf(os.Stderr, "  %s: %s (%d words)\n", status, toolID, words)
	}
}

func (p *Progress) animate() {
	tick := time.NewTicker(200 * time.Millisecond)
	defer tick.Stop()

	frame := 0
	firstRender := true
	for {
		select {
		case <-p.done:
			return
		case <-tick.C:
			p.mu.Lock()
			if !firstRender {
				// Move cursor up N lines
				for range p.toolIDs {
					fmt.Fprintf(os.Stderr, "\033[A\033[2K")
				}
			}
			firstRender = false
			spinner := spinnerFrames[frame%len(spinnerFrames)]
			for _, id := range p.toolIDs {
				s := p.states[id]
				p.renderLine(id, s, spinner)
			}
			frame++
			p.mu.Unlock()
		}
	}
}

func (p *Progress) renderLine(id string, s *ToolStatus, spinner string) {
	switch s.Status {
	case "pending":
		fmt.Fprintf(os.Stderr, " · %-20s waiting\n", id)
	case "running":
		elapsed := time.Since(s.Started).Round(time.Second)
		fmt.Fprintf(os.Stderr, " %s %-20s running  %s\n", spinner, id, elapsed)
	case "done", "success":
		fmt.Fprintf(os.Stderr, " + %-20s done     %d words\n", id, s.Words)
	case "failed":
		fmt.Fprintf(os.Stderr, " x %-20s failed\n", id)
	case "timeout":
		fmt.Fprintf(os.Stderr, " ! %-20s timeout\n", id)
	default:
		fmt.Fprintf(os.Stderr, " - %-20s %s\n", id, s.Status)
	}
}
