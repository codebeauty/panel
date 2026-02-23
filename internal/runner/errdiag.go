package runner

import (
	"fmt"
	"regexp"
	"strings"
)

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

type Diagnosis struct {
	Category   DiagCategory
	Message    string
	Suggestion string
}

func (d *Diagnosis) String() string {
	return fmt.Sprintf("%s\n%s", d.Message, d.Suggestion)
}

func Diagnose(toolID string, stderr []byte, exitCode int) *Diagnosis {
	s := string(stderr)
	for _, check := range diagChecks {
		if d := check(toolID, s, exitCode); d != nil {
			return d
		}
	}
	return nil
}

func containsAny(s string, patterns []string) bool {
	for _, p := range patterns {
		if strings.Contains(s, p) {
			return true
		}
	}
	return false
}

func hasHTTPStatus(s string, code int) bool {
	codeStr := fmt.Sprintf("%d", code)
	return strings.Contains(s, "code: "+codeStr) ||
		strings.Contains(s, "status: "+codeStr) ||
		strings.Contains(s, `"code":`+codeStr) ||
		strings.Contains(s, "status_code: "+codeStr)
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

func checkModelNotFound(toolID, stderr string, _ int) *Diagnosis {
	patterns := []string{
		"ModelNotFoundError",
		"model_not_found",
		"Model not found",
		"Requested entity was not found",
		"The model `",
		"does not exist",
	}

	matched := containsAny(stderr, patterns)
	if !matched && hasHTTPStatus(stderr, 404) {
		matched = containsAny(stderr, []string{"model", "Model", "entity"})
	}
	if !matched {
		return nil
	}

	return &Diagnosis{
		Category:   DiagModelNotFound,
		Message:    fmt.Sprintf("The requested model was not found by %s.", toolBaseName(toolID)),
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

	if !containsAny(stderr, patterns) && !hasHTTPStatus(stderr, 401) {
		return nil
	}

	tool := toolBaseName(toolID)
	suggestion := "Check that your API key is set and valid."
	if envHint := apiKeyEnvVar(tool); envHint != "" {
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

	if !containsAny(stderr, patterns) && !hasHTTPStatus(stderr, 429) {
		return nil
	}

	return &Diagnosis{
		Category:   DiagRateLimit,
		Message:    fmt.Sprintf("Rate limited by %s.", toolBaseName(toolID)),
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
	if !containsAny(stderr, patterns) {
		return nil
	}
	return &Diagnosis{
		Category:   DiagOverloaded,
		Message:    "The API is temporarily overloaded.",
		Suggestion: "Wait a moment and try again.",
	}
}

func checkPermission(toolID, stderr string, _ int) *Diagnosis {
	patterns := []string{
		"PERMISSION_DENIED",
		"permission_denied",
		"Forbidden",
		"Access denied",
		"access denied",
	}

	if !containsAny(stderr, patterns) && !hasHTTPStatus(stderr, 403) {
		return nil
	}

	return &Diagnosis{
		Category:   DiagPermission,
		Message:    fmt.Sprintf("Permission denied by %s.", toolBaseName(toolID)),
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
	if !containsAny(stderr, patterns) {
		return nil
	}
	return &Diagnosis{
		Category:   DiagNetwork,
		Message:    "A network error occurred.",
		Suggestion: "Check your internet connection and try again.",
	}
}

func toolBaseName(toolID string) string {
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
