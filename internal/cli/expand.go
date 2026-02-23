package cli

import (
	"fmt"

	"github.com/codebeauty/panel/internal/config"
)

// expandDuplicateToolIDs handles repeated tool IDs by creating __N aliases.
// ["claude", "claude", "claude"] -> ["claude", "claude__2", "claude__3"]
func expandDuplicateToolIDs(toolIDs []string, cfg *config.Config) []string {
	counts := make(map[string]int)
	var expanded []string
	for _, id := range toolIDs {
		counts[id]++
		if counts[id] == 1 {
			expanded = append(expanded, id)
		} else {
			aliasID := fmt.Sprintf("%s__%d", id, counts[id])
			if tc, ok := cfg.Tools[id]; ok {
				cfg.Tools[aliasID] = tc
			}
			expanded = append(expanded, aliasID)
		}
	}
	return expanded
}
