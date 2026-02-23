package persona

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBuiltinPresets(t *testing.T) {
	assert.Len(t, Builtins, 6)

	expected := []string{"security", "performance", "architect", "reviewer", "devil", "product"}
	for _, id := range expected {
		content, ok := Builtins[id]
		assert.True(t, ok, "missing builtin: %s", id)
		assert.NotEmpty(t, content, "empty builtin: %s", id)
	}
}

func TestBuiltinPresetsHaveRole(t *testing.T) {
	for id, content := range Builtins {
		assert.Contains(t, content, "You are", "builtin %s should define a role", id)
	}
}

func TestBuiltinIDs(t *testing.T) {
	ids := BuiltinIDs()
	assert.Len(t, ids, 6)
	// Should be sorted
	for i := 1; i < len(ids); i++ {
		assert.Less(t, ids[i-1], ids[i], "BuiltinIDs should be sorted")
	}
}

func TestValidatePersonaID(t *testing.T) {
	tests := []struct {
		id    string
		valid bool
	}{
		{"security", true},
		{"my-expert", true},
		{"go_reviewer", true},
		{"expert.v2", true},
		{"../escape", false},
		{"path/traversal", false},
		{"has space", false},
		{"", false},
		{"semi;colon", false},
	}
	for _, tt := range tests {
		err := ValidatePersonaID(tt.id)
		if tt.valid {
			assert.NoError(t, err, "expected %q to be valid", tt.id)
		} else {
			assert.Error(t, err, "expected %q to be invalid", tt.id)
		}
	}
}
