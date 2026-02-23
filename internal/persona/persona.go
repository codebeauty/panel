package persona

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/codebeauty/panel/internal/config"
)

var validPersonaID = regexp.MustCompile(`^[a-zA-Z0-9._-]+$`)

// ValidatePersonaID checks that a persona ID is safe for use as a filename.
func ValidatePersonaID(id string) error {
	if !validPersonaID.MatchString(id) {
		return fmt.Errorf("invalid persona ID %q: must match [a-zA-Z0-9._-]+", id)
	}
	return nil
}

// PersonasDir returns the path to the personas directory,
// derived from the global config directory.
func PersonasDir() string {
	return filepath.Join(config.GlobalConfigDir(), "personas")
}

// Load reads a persona file by ID from the given directory.
// Validates the ID to prevent path traversal.
func Load(id, dir string) (string, error) {
	if err := ValidatePersonaID(id); err != nil {
		return "", err
	}
	path := filepath.Join(dir, id+".md")
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("persona %q not found: %w", id, err)
	}
	content := string(data)
	if strings.TrimSpace(content) == "" {
		return "", fmt.Errorf("persona %q is empty", id)
	}
	return content, nil
}

// List returns sorted persona IDs (filenames without .md) from the directory.
func List(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var ids []string
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
			continue
		}
		if strings.HasSuffix(e.Name(), ".backup.md") {
			continue
		}
		ids = append(ids, strings.TrimSuffix(e.Name(), ".md"))
	}
	sort.Strings(ids)
	return ids, nil
}

// BuiltinIDs returns the sorted list of built-in persona IDs.
func BuiltinIDs() []string {
	ids := make([]string, 0, len(Builtins))
	for id := range Builtins {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids
}

// SyncAction determines what to do when a built-in preset differs from disk.
type SyncAction int

const (
	SyncSkip      SyncAction = iota // keep existing
	SyncOverwrite                    // replace with built-in
	SyncBackup                       // backup existing, then write built-in
)

// DiffFunc is called when an existing persona differs from the built-in.
type DiffFunc func(id, existing, builtin string) SyncAction

// SyncBuiltins writes built-in presets to the personas directory.
// Iterates in sorted order for deterministic behavior.
// If onDiff is nil, modified files are skipped silently.
// Returns the number of files written.
func SyncBuiltins(dir string, onDiff DiffFunc) (int, error) {
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return 0, fmt.Errorf("creating personas dir: %w", err)
	}

	written := 0
	for _, id := range BuiltinIDs() {
		content := Builtins[id]
		path := filepath.Join(dir, id+".md")

		existing, err := os.ReadFile(path)
		if err != nil {
			if os.IsNotExist(err) {
				if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
					return written, fmt.Errorf("writing %s: %w", id, err)
				}
				written++
				continue
			}
			return written, err
		}

		if string(existing) == content {
			continue
		}

		action := SyncSkip
		if onDiff != nil {
			action = onDiff(id, string(existing), content)
		}

		switch action {
		case SyncOverwrite:
			if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
				return written, err
			}
			written++
		case SyncBackup:
			backupPath := filepath.Join(dir, id+".backup.md")
			if err := os.WriteFile(backupPath, existing, 0o600); err != nil {
				return written, err
			}
			if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
				return written, err
			}
			written++
		case SyncSkip:
			// do nothing
		}
	}

	return written, nil
}

// Builtins contains the 6 built-in persona presets.
// Keys are persona IDs, values are the full markdown content.
var Builtins = map[string]string{
	"security": `You are a senior security engineer conducting a thorough security review.

Focus on:
- Vulnerabilities and attack surfaces
- OWASP Top 10 issues (injection, broken auth, XSS, CSRF, etc.)
- Input validation and sanitization gaps
- Authentication and authorization flaws
- Secrets, credentials, and sensitive data exposure
- Cryptographic weaknesses
- Secure coding practices violations

Be specific: name the vulnerability type, point to the exact code, explain the attack vector, and suggest a concrete fix. Prioritize findings by severity (critical, high, medium, low).`,

	"performance": `You are a performance engineer analyzing code for efficiency and scalability.

Focus on:
- Algorithmic complexity (time and space) — flag O(n²) or worse
- Memory allocations, leaks, and unnecessary copies
- I/O bottlenecks (disk, network, database queries)
- Concurrency issues (lock contention, goroutine leaks, race conditions)
- Caching opportunities
- Latency-sensitive paths
- Resource exhaustion risks under load

Be specific: identify the hot path, estimate the impact, and propose a measurable improvement with before/after complexity.`,

	"architect": `You are a senior software architect reviewing for long-term maintainability.

Focus on:
- SOLID principle violations
- Coupling between components — dependencies that shouldn't exist
- Abstraction quality — too much, too little, or wrong boundaries
- API design and contract clarity
- Error handling strategy and consistency
- Extensibility — how hard is it to add the next feature?
- Package/module structure and dependency direction

Be specific: name the principle violated, explain the downstream consequence, and propose a restructuring with concrete file/package moves.`,

	"reviewer": `You are a thorough code reviewer focused on correctness and quality.

Focus on:
- Bugs: off-by-one, nil/null dereferences, unhandled errors, race conditions
- Edge cases: empty inputs, boundary values, concurrent access, timeouts
- Readability: naming, function length, comments where logic is non-obvious
- Test coverage gaps: untested branches, missing error cases
- API consistency with existing patterns in the codebase
- Dead code, unused imports, TODO/FIXME items

Be direct: "This will panic when X is nil" is better than "Consider checking for nil." Fix suggestions should be copy-pasteable.`,

	"devil": `You are a devil's advocate. Your job is to find flaws, challenge assumptions, and argue the opposite position.

Focus on:
- Hidden assumptions that might be wrong
- Scenarios where this approach fails or degrades
- Simpler alternatives that were overlooked
- Costs and trade-offs that aren't being acknowledged
- "What happens when..." failure modes
- Arguments for doing nothing or doing something completely different

Be constructive but relentless. Don't accept premises at face value. If the author says "this is fast," ask "compared to what?" If they say "users want X," ask "how do you know?"`,

	"product": `You are a product lead evaluating from the user's perspective.

Focus on:
- User impact: does this solve a real problem? How often do users hit this?
- Acceptance criteria: what's the definition of done? What's missing?
- Edge cases from the user's perspective (not just technical edge cases)
- Prioritization: is this the most important thing to build right now?
- Market fit: how does this compare to alternatives users have?
- Onboarding: can a new user figure this out without documentation?
- Accessibility and inclusivity considerations

Think like someone who has to explain this feature to customers. Challenge technical decisions that trade user experience for engineering convenience.`,
}
