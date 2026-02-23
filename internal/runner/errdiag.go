package runner

import (
	"fmt"
	"regexp"
	"strings"
)

// DiagCategory classifies the kind of error encountered.
type DiagCategory string

const (
	DiagModelNotFound DiagCategory = "model_not_found"
	DiagAuthFailure   DiagCategory = "auth_failure"
	DiagRateLimit     DiagCategory = "rate_limit"
	DiagBinaryMissing DiagCategory = "binary_missing"
	DiagNetwork       DiagCategory = "network_error"
	DiagPermission    DiagCategory = "permission_denied"
	DiagOverloaded    DiagCategory = "overloaded"
)

// Diagnosis holds a human-readable explanation of a tool failure.
type Diagnosis struct {
	Category   DiagCategory
	Message    string
	Suggestion string
}

func (d *Diagnosis) String() string {
	return fmt.Sprintf("%s\n%s", d.Message, d.Suggestion)
}

// Diagnose inspects stderr output from a failed tool and returns a structured
// diagnosis if a known error pattern is matched. Returns nil if the error
// is not recognized.
func Diagnose(toolID string, stderr []byte, exitCode int) *Diagnosis {
	s := string(stderr)

	// Check patterns in priority order (most specific first).
	for _, check := range diagChecks {
		if d := check(toolID, s, exitCode); d != nil {
			return d
		}
	}
	return nil
}

var diagChecks = []func(toolID, stderr string, exitCode int) *Diagnosis{
	checkBinaryMissing,
	checkModelNotFound,
	checkAuthFailure,
	checkRateLimit,
	checkOverloaded,
	checkPermission,
	checkNetwork,
}

var binaryNotFoundRe = regexp.MustCompile(`exec: "([^"]+)": executable file not found`)

func checkBinaryMissing(toolID, stderr string, _ int) *Diagnosis {
	m := binaryNotFoundRe.FindStringSubmatch(stderr)
	if m != nil {
		return &Diagnosis{
			Category:   DiagBinaryMissing,
			Message:    fmt.Sprintf("Binary %q not found on PATH.", m[1]),
			Suggestion: fmt.Sprintf("Install %s or check that it is in your PATH.", m[1]),
		}
	}
	return nil
}

func checkModelNotFound(toolID, stderr string, exitCode int) *Diagnosis {
	patterns := []string{
		"ModelNotFoundError",
		"model_not_found",
		"Model not found",
		"Requested entity was not found",
		"The model `",
		"does not exist",
	}
	// Also match HTTP 404 with model-related context.
	has404 := strings.Contains(stderr, "code: 404") || strings.Contains(stderr, "status: 404") || strings.Contains(stderr, "\"code\":404")

	matched := false
	for _, p := range patterns {
		if strings.Contains(stderr, p) {
			matched = true
			break
		}
	}
	if !matched && has404 {
		// 404 alongside model-related terms
		modelTerms := []string{"model", "Model", "entity"}
		for _, t := range modelTerms {
			if strings.Contains(stderr, t) {
				matched = true
				break
			}
		}
	}
	if !matched {
		return nil
	}

	tool := toolBaseName(toolID)
	return &Diagnosis{
		Category:   DiagModelNotFound,
		Message:    fmt.Sprintf("The requested model was not found by %s.", tool),
		Suggestion: "Check that the model name is valid and available for your account.",
	}
}

func checkAuthFailure(toolID, stderr string, _ int) *Diagnosis {
	patterns := []string{
		"UNAUTHENTICATED",
		"authentication_error",
		"Invalid API Key",
		"invalid_api_key",
		"invalid x-api-key",
		"API key not valid",
		"API key is invalid",
		"Unauthorized",
	}
	has401 := strings.Contains(stderr, "code: 401") || strings.Contains(stderr, "status: 401") || strings.Contains(stderr, "\"code\":401") || strings.Contains(stderr, "status_code: 401")

	matched := has401
	for _, p := range patterns {
		if strings.Contains(stderr, p) {
			matched = true
			break
		}
	}
	if !matched {
		return nil
	}

	tool := toolBaseName(toolID)
	envHint := apiKeyEnvVar(tool)
	suggestion := "Check that your API key is set and valid."
	if envHint != "" {
		suggestion = fmt.Sprintf("Check that %s is set and valid.", envHint)
	}

	return &Diagnosis{
		Category:   DiagAuthFailure,
		Message:    fmt.Sprintf("Authentication failed for %s.", tool),
		Suggestion: suggestion,
	}
}

func checkRateLimit(toolID, stderr string, _ int) *Diagnosis {
	patterns := []string{
		"RESOURCE_EXHAUSTED",
		"rate_limit_error",
		"Rate limit",
		"rate limit",
		"Too many requests",
		"too many requests",
		"quota exceeded",
		"Quota exceeded",
	}
	has429 := strings.Contains(stderr, "code: 429") || strings.Contains(stderr, "status: 429") || strings.Contains(stderr, "\"code\":429")

	matched := has429
	for _, p := range patterns {
		if strings.Contains(stderr, p) {
			matched = true
			break
		}
	}
	if !matched {
		return nil
	}

	tool := toolBaseName(toolID)
	return &Diagnosis{
		Category:   DiagRateLimit,
		Message:    fmt.Sprintf("Rate limited by %s.", tool),
		Suggestion: "Wait a moment and try again, or check your usage quota.",
	}
}

func checkOverloaded(_, stderr string, _ int) *Diagnosis {
	patterns := []string{
		"overloaded_error",
		"overloaded",
		"server_error",
		"503",
		"Service Unavailable",
	}
	for _, p := range patterns {
		if strings.Contains(stderr, p) {
			return &Diagnosis{
				Category:   DiagOverloaded,
				Message:    "The API is temporarily overloaded.",
				Suggestion: "Wait a moment and try again.",
			}
		}
	}
	return nil
}

func checkPermission(toolID, stderr string, _ int) *Diagnosis {
	patterns := []string{
		"PERMISSION_DENIED",
		"permission_denied",
		"Forbidden",
		"Access denied",
		"access denied",
	}
	has403 := strings.Contains(stderr, "code: 403") || strings.Contains(stderr, "status: 403") || strings.Contains(stderr, "\"code\":403")

	matched := has403
	for _, p := range patterns {
		if strings.Contains(stderr, p) {
			matched = true
			break
		}
	}
	if !matched {
		return nil
	}

	tool := toolBaseName(toolID)
	return &Diagnosis{
		Category:   DiagPermission,
		Message:    fmt.Sprintf("Permission denied by %s.", tool),
		Suggestion: "Check that your API key has the required permissions.",
	}
}

func checkNetwork(_, stderr string, _ int) *Diagnosis {
	patterns := []string{
		"connection refused",
		"Connection refused",
		"no such host",
		"dial tcp",
		"network is unreachable",
		"Network is unreachable",
		"TLS handshake timeout",
		"certificate",
		"ECONNREFUSED",
		"ENOTFOUND",
		"getaddrinfo",
	}
	for _, p := range patterns {
		if strings.Contains(stderr, p) {
			return &Diagnosis{
				Category:   DiagNetwork,
				Message:    "A network error occurred.",
				Suggestion: "Check your internet connection and try again.",
			}
		}
	}
	return nil
}

// toolBaseName extracts the base tool name from a toolID like "gemini-3.1-pro".
// It returns the first segment before any dash-digit boundary that looks like a version.
func toolBaseName(toolID string) string {
	// Known prefixes.
	for _, prefix := range []string{"claude", "codex", "gemini", "amp"} {
		if strings.HasPrefix(strings.ToLower(toolID), prefix) {
			return prefix
		}
	}
	return toolID
}

func apiKeyEnvVar(tool string) string {
	switch strings.ToLower(tool) {
	case "claude":
		return "ANTHROPIC_API_KEY"
	case "codex":
		return "OPENAI_API_KEY"
	case "gemini":
		return "GEMINI_API_KEY or GOOGLE_API_KEY"
	case "amp":
		return "AMP_API_KEY"
	default:
		return ""
	}
}
