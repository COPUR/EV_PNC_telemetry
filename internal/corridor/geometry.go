// Package corridor models the Istanbul-Bodrum pilot corridor: versioned route geometries for the
// main route and the four alternative families, cluster rules, and the EVSE index built from them.
package corridor

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/copur/ev_pnc_telemetry/internal/geo"
)

// Family codes, as in corridor-pilot/scope.md.
const (
	FamilyM  = "M"
	FamilyA1 = "A1"
	FamilyA2 = "A2"
	FamilyA3 = "A3"
	FamilyA4 = "A4"
)

// Families lists every valid family code in reporting order.
var Families = []string{FamilyM, FamilyA1, FamilyA2, FamilyA3, FamilyA4}

// Quality says how far a geometry can be trusted.
const (
	// QualityDraft marks hand-placed waypoints at city level. Good for wiring and tests, not for
	// deciding whether a station is within 2 km of the route.
	QualityDraft = "draft-waypoints"
	// QualityRouted marks a geometry exported from a routing engine run over the real road network.
	QualityRouted = "routed"
)

// Geometry is one family's route at one version.
type Geometry struct {
	Family  string
	Version string
	Quality string
	Line    []geo.Point
}

// Manifest describes a geometry version directory (corridor/geometries/<version>/manifest.json).
type Manifest struct {
	Version string         `json:"version"`
	Quality string         `json:"quality"`
	Source  string         `json:"source"`
	Files   []ManifestFile `json:"files"`
}

// ManifestFile pins one GeoJSON file by hash, so a published version cannot change unnoticed.
type ManifestFile struct {
	Family string `json:"family"`
	File   string `json:"file"`
	SHA256 string `json:"sha256"`
}

type geoJSONFeature struct {
	Type     string `json:"type"`
	Geometry struct {
		Type        string       `json:"type"`
		Coordinates [][2]float64 `json:"coordinates"`
	} `json:"geometry"`
	Properties struct {
		Family string `json:"family"`
	} `json:"properties"`
}

// LoadGeometries reads and verifies one version directory. Every family must be present exactly
// once and every file must match the hash in the manifest.
func LoadGeometries(dir string) ([]Geometry, Manifest, error) {
	var m Manifest
	raw, err := os.ReadFile(filepath.Join(dir, "manifest.json"))
	if err != nil {
		return nil, m, err
	}
	if err := json.Unmarshal(raw, &m); err != nil {
		return nil, m, fmt.Errorf("%s/manifest.json: %w", dir, err)
	}
	if m.Quality != QualityDraft && m.Quality != QualityRouted {
		return nil, m, fmt.Errorf("%s: unknown quality %q", dir, m.Quality)
	}
	seen := map[string]bool{}
	var out []Geometry
	for _, f := range m.Files {
		if !validFamily(f.Family) {
			return nil, m, fmt.Errorf("%s: unknown family %q", dir, f.Family)
		}
		if seen[f.Family] {
			return nil, m, fmt.Errorf("%s: family %s listed twice", dir, f.Family)
		}
		seen[f.Family] = true
		body, err := os.ReadFile(filepath.Join(dir, f.File))
		if err != nil {
			return nil, m, err
		}
		sum := sha256.Sum256(body)
		if hex.EncodeToString(sum[:]) != f.SHA256 {
			return nil, m, fmt.Errorf("%s/%s: hash differs from manifest; published versions are immutable, add a new version instead", dir, f.File)
		}
		var feat geoJSONFeature
		if err := json.Unmarshal(body, &feat); err != nil {
			return nil, m, fmt.Errorf("%s/%s: %w", dir, f.File, err)
		}
		if feat.Geometry.Type != "LineString" || len(feat.Geometry.Coordinates) < 2 {
			return nil, m, fmt.Errorf("%s/%s: want a LineString with at least two points", dir, f.File)
		}
		if feat.Properties.Family != f.Family {
			return nil, m, fmt.Errorf("%s/%s: properties.family %q does not match manifest %q", dir, f.File, feat.Properties.Family, f.Family)
		}
		line := make([]geo.Point, len(feat.Geometry.Coordinates))
		for i, c := range feat.Geometry.Coordinates { // GeoJSON is [lon, lat]
			line[i] = geo.Point{Lat: c[1], Lon: c[0]}
		}
		out = append(out, Geometry{Family: f.Family, Version: m.Version, Quality: m.Quality, Line: line})
	}
	for _, fam := range Families {
		if !seen[fam] {
			return nil, m, fmt.Errorf("%s: family %s is missing", dir, fam)
		}
	}
	sort.Slice(out, func(i, j int) bool { return familyOrder(out[i].Family) < familyOrder(out[j].Family) })
	return out, m, nil
}

func validFamily(f string) bool { return familyOrder(f) >= 0 }

func familyOrder(f string) int {
	for i, v := range Families {
		if v == f {
			return i
		}
	}
	return -1
}
