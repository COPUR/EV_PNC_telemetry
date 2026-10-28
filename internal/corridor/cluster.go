package corridor

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

// ClusterRule assigns a cluster label to EVSEs by operator, site id or a name fragment.
// Rules are data (corridor/clusters.json), not code, so the pilot team can correct them.
type ClusterRule struct {
	Label        string   `json:"label"`
	Operators    []string `json:"operators,omitempty"`
	SiteIDs      []string `json:"site_ids,omitempty"`
	NameContains []string `json:"name_contains,omitempty"`
	// Confidence uses the project labels: F sourced, K user input to verify, I inferred, U unknown.
	Confidence string `json:"confidence"`
	Note       string `json:"note,omitempty"`
}

// LoadClusterRules reads the rule list.
func LoadClusterRules(path string) ([]ClusterRule, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var doc struct {
		Rules []ClusterRule `json:"rules"`
	}
	if err := json.Unmarshal(raw, &doc); err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	for _, r := range doc.Rules {
		if r.Label == "" {
			return nil, fmt.Errorf("%s: a rule has no label", path)
		}
		if len(r.Operators)+len(r.SiteIDs)+len(r.NameContains) == 0 {
			return nil, fmt.Errorf("%s: rule %q matches nothing", path, r.Label)
		}
	}
	return doc.Rules, nil
}

// Match returns the label of the first rule that applies, or "".
func Match(rules []ClusterRule, e EVSE) string {
	for _, r := range rules {
		if contains(r.Operators, e.Operator) || contains(r.SiteIDs, e.SiteID) || nameHas(r.NameContains, e.Name) {
			return r.Label
		}
	}
	return ""
}

func contains(list []string, v string) bool {
	if v == "" {
		return false
	}
	for _, s := range list {
		if strings.EqualFold(s, v) {
			return true
		}
	}
	return false
}

func nameHas(fragments []string, name string) bool {
	n := strings.ToLower(name)
	for _, f := range fragments {
		if f != "" && strings.Contains(n, strings.ToLower(f)) {
			return true
		}
	}
	return false
}
