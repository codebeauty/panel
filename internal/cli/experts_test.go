package cli

import (
	"testing"

	"github.com/codebeauty/panel/internal/config"
	"github.com/stretchr/testify/assert"
)

func TestExpertTeamRefs(t *testing.T) {
	cfg := config.NewDefaults()
	cfg.Teams["review"] = []string{"security", "architect"}
	cfg.Teams["deep"] = []string{"security", "devil"}

	refs := findExpertTeamRefs("security", cfg)
	assert.Len(t, refs, 2)
	assert.Contains(t, refs, "review")
	assert.Contains(t, refs, "deep")
}

func TestExpertTeamRefsNone(t *testing.T) {
	cfg := config.NewDefaults()
	cfg.Teams["review"] = []string{"architect"}

	refs := findExpertTeamRefs("security", cfg)
	assert.Empty(t, refs)
}
