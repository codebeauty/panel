package runner

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDiagnose_GeminiModelNotFound(t *testing.T) {
	stderr := []byte(`Loaded cached credentials.Error when talking to Gemini API Full report available at:
/var/folders/lp/htnm4yzj5g99fxbh5cz0ttq80000gn/T/gemini-client-error-Turn.run-sendMessageStream-2026-02-23T19-5
ModelNotFoundError: Requested entity was not found.at classifyGoogleError
(file:///opt/homebrew/Cellar/gemini-cli/0.29.5/libexec/lib/node_modules/@google/gemini-cli/node_modules/@google
{ code: 404 }
An unexpected critical error occurred:[object Object]`)

	d := Diagnose("gemini-3.1-pro", stderr, 1)
	assert.NotNil(t, d)
	assert.Equal(t, DiagModelNotFound, d.Category)
	assert.Contains(t, d.Message, "gemini")
	assert.Contains(t, d.Suggestion, "model name")
}

func TestDiagnose_AuthFailure401(t *testing.T) {
	stderr := []byte(`Error: authentication_error - Invalid API Key (status_code: 401)`)

	d := Diagnose("claude", stderr, 1)
	assert.NotNil(t, d)
	assert.Equal(t, DiagAuthFailure, d.Category)
	assert.Contains(t, d.Message, "claude")
	assert.Contains(t, d.Suggestion, "ANTHROPIC_API_KEY")
}

func TestDiagnose_GeminiAuthFailure(t *testing.T) {
	stderr := []byte(`UNAUTHENTICATED: API key not valid. Please pass a valid API key.`)

	d := Diagnose("gemini-2.5-pro", stderr, 1)
	assert.NotNil(t, d)
	assert.Equal(t, DiagAuthFailure, d.Category)
	assert.Contains(t, d.Suggestion, "GEMINI_API_KEY")
}

func TestDiagnose_RateLimit(t *testing.T) {
	stderr := []byte(`Error: rate_limit_error - Too many requests. Please wait and try again. { code: 429 }`)

	d := Diagnose("claude", stderr, 1)
	assert.NotNil(t, d)
	assert.Equal(t, DiagRateLimit, d.Category)
	assert.Contains(t, d.Suggestion, "Wait")
}

func TestDiagnose_BinaryMissing(t *testing.T) {
	stderr := []byte(`exec: "gemini": executable file not found in $PATH`)

	d := Diagnose("gemini-2.5-pro", stderr, -1)
	assert.NotNil(t, d)
	assert.Equal(t, DiagBinaryMissing, d.Category)
	assert.Contains(t, d.Message, `"gemini"`)
	assert.Contains(t, d.Suggestion, "Install")
}

func TestDiagnose_NetworkError(t *testing.T) {
	stderr := []byte(`dial tcp 142.250.185.46:443: connection refused`)

	d := Diagnose("gemini-2.5-pro", stderr, 1)
	assert.NotNil(t, d)
	assert.Equal(t, DiagNetwork, d.Category)
	assert.Contains(t, d.Suggestion, "internet connection")
}

func TestDiagnose_PermissionDenied(t *testing.T) {
	stderr := []byte(`PERMISSION_DENIED: The caller does not have permission { code: 403 }`)

	d := Diagnose("gemini-2.5-pro", stderr, 1)
	assert.NotNil(t, d)
	assert.Equal(t, DiagPermission, d.Category)
	assert.Contains(t, d.Suggestion, "permissions")
}

func TestDiagnose_Overloaded(t *testing.T) {
	stderr := []byte(`Error: overloaded_error - The API is temporarily overloaded. Please try again later.`)

	d := Diagnose("claude", stderr, 1)
	assert.NotNil(t, d)
	assert.Equal(t, DiagOverloaded, d.Category)
}

func TestDiagnose_UnknownError(t *testing.T) {
	stderr := []byte(`some random error that does not match any pattern`)

	d := Diagnose("claude", stderr, 1)
	assert.Nil(t, d, "should return nil for unrecognized errors")
}

func TestDiagnose_EmptyStderr(t *testing.T) {
	d := Diagnose("claude", nil, 1)
	assert.Nil(t, d)
}

func TestToolBaseName(t *testing.T) {
	tests := []struct {
		toolID string
		want   string
	}{
		{"claude", "claude"},
		{"gemini-3.1-pro", "gemini"},
		{"gemini-2.5-pro", "gemini"},
		{"codex-mini", "codex"},
		{"amp", "amp"},
		{"custom-tool", "custom-tool"},
	}
	for _, tt := range tests {
		t.Run(tt.toolID, func(t *testing.T) {
			assert.Equal(t, tt.want, toolBaseName(tt.toolID))
		})
	}
}

func TestApiKeyEnvVar(t *testing.T) {
	assert.Equal(t, "ANTHROPIC_API_KEY", apiKeyEnvVar("claude"))
	assert.Equal(t, "OPENAI_API_KEY", apiKeyEnvVar("codex"))
	assert.Contains(t, apiKeyEnvVar("gemini"), "GEMINI_API_KEY")
	assert.Equal(t, "AMP_API_KEY", apiKeyEnvVar("amp"))
	assert.Equal(t, "", apiKeyEnvVar("unknown"))
}
