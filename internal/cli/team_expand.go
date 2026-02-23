package cli

import (
	"fmt"
	"strings"

	"github.com/codebeauty/panel/internal/config"
	"github.com/codebeauty/panel/internal/expert"
)

// expandTeamCrossProduct builds tool×expert composite IDs.
// Deduplicates tool IDs. Mutates cfg.Tools by registering composite entries
// (e.g., "claude@security") so buildTools can resolve them later.
// Composite IDs intentionally use "@" which is outside ValidateToolName's
// charset — these are runtime-generated, not user-created tool names.
func expandTeamCrossProduct(toolIDs []string, teamExperts []string, cfg *config.Config) ([]string, error) {
	seen := make(map[string]bool)
	var deduped []string
	for _, id := range toolIDs {
		if !seen[id] {
			seen[id] = true
			deduped = append(deduped, id)
		}
	}

	var crossIDs []string
	for _, toolID := range deduped {
		baseTool, ok := cfg.Tools[toolID]
		if !ok {
			return nil, fmt.Errorf("unknown tool %q in team expansion", toolID)
		}
		for _, expertID := range teamExperts {
			compositeID := fmt.Sprintf("%s@%s", toolID, expertID)
			cfg.Tools[compositeID] = baseTool
			crossIDs = append(crossIDs, compositeID)
		}
	}
	return crossIDs, nil
}

// resolveTeamExperts extracts expert IDs from composite tool IDs (tool@expert)
// and loads their content. Caches loaded experts to avoid redundant reads.
// For non-composite IDs (no @), returns empty strings.
func resolveTeamExperts(compositeIDs []string, expertDir string) (ids []string, contents []string, err error) {
	cache := make(map[string]string)
	ids = make([]string, len(compositeIDs))
	contents = make([]string, len(compositeIDs))

	for i, cid := range compositeIDs {
		parts := strings.SplitN(cid, "@", 2)
		if len(parts) != 2 {
			continue
		}
		eid := parts[1]
		if _, ok := cache[eid]; !ok {
			content, loadErr := expert.Load(eid, expertDir)
			if loadErr != nil {
				return nil, nil, fmt.Errorf("expert %q: %w", eid, loadErr)
			}
			cache[eid] = content
		}
		ids[i] = eid
		contents[i] = cache[eid]
	}
	return ids, contents, nil
}
