package cli

import (
	"fmt"
	"strings"

	"github.com/codebeauty/horde/internal/config"
	"github.com/codebeauty/horde/internal/raider"
)

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
			return nil, fmt.Errorf("unknown agent %q in squad expansion", toolID)
		}
		for _, expertID := range teamExperts {
			compositeID := fmt.Sprintf("%s@%s", toolID, expertID)
			cfg.Tools[compositeID] = baseTool
			crossIDs = append(crossIDs, compositeID)
		}
	}
	return crossIDs, nil
}

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
			content, loadErr := raider.Load(eid, expertDir)
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
