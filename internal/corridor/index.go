package corridor

import (
	"fmt"
	"math"
	"sort"

	"github.com/copur/ev_pnc_telemetry/internal/geo"
)

// DefaultRadiusM is the corridor buffer from CP-GEO-002: an EVSE within 2 km of a geometry serves its family.
const DefaultRadiusM = 2000.0

// Source says where an EVSE record came from. Only user_listed records are not evidence for K1.
const (
	SourceUserListed = "user_listed" // K: product owner's scope text, not yet verified
	SourceEPDKList   = "epdk_list"
	SourceOCPIFeed   = "ocpi_feed"
	SourceHubFeed    = "hub_feed"
)

// EVSE is the input record. It is deliberately independent of the OCPI types: an adapter from
// OCPI Locations (ocpi-model) and from the EPDK list produces these.
type EVSE struct {
	UID        string   `json:"evse_uid"`
	EvseID     string   `json:"evse_id,omitempty"`
	LocationID string   `json:"location_id,omitempty"`
	SiteID     string   `json:"site_id,omitempty"`
	Name       string   `json:"name,omitempty"`
	Operator   string   `json:"operator,omitempty"`
	Lat        *float64 `json:"lat,omitempty"`
	Lon        *float64 `json:"lon,omitempty"`
	MaxPowerKW float64  `json:"max_power_kw,omitempty"`
	PowerType  string   `json:"power_type,omitempty"` // AC or DC
	Source     string   `json:"source"`
}

// Entry is one indexed EVSE.
type Entry struct {
	UID        string             `json:"evse_uid"`
	EvseID     string             `json:"evse_id,omitempty"`
	LocationID string             `json:"location_id,omitempty"`
	Operator   string             `json:"operator,omitempty"`
	Lat        float64            `json:"lat"`
	Lon        float64            `json:"lon"`
	MaxPowerKW float64            `json:"max_power_kw,omitempty"`
	PowerType  string             `json:"power_type,omitempty"`
	Source     string             `json:"source"`
	Families   []string           `json:"families"`
	DistanceM  map[string]float64 `json:"distance_m"`
	Cluster    string             `json:"cluster,omitempty"`
	K1Evidence bool               `json:"k1_evidence"`
	K1Withheld string             `json:"k1_withheld_reason,omitempty"`
}

// Unplaced is an input record that could not be indexed, with the reason.
type Unplaced struct {
	UID    string `json:"evse_uid"`
	Name   string `json:"name,omitempty"`
	Source string `json:"source"`
	Reason string `json:"reason"`
}

// Index is the nightly output.
type Index struct {
	GeometryVersion string     `json:"geometry_version"`
	GeometryQuality string     `json:"geometry_quality"`
	RadiusM         float64    `json:"radius_m"`
	Entries         []Entry    `json:"entries"`
	Unplaced        []Unplaced `json:"unplaced"`
	Summary         Summary    `json:"summary"`
}

// Summary counts what the index holds, for a quick read and for the K1 report.
type Summary struct {
	Input      int            `json:"input"`
	InCorridor int            `json:"in_corridor"`
	K1Evidence int            `json:"k1_evidence"`
	ByFamily   map[string]int `json:"by_family"`
	BySource   map[string]int `json:"by_source"`
}

// Build tags every EVSE within radiusM of a family geometry with the family codes it serves and,
// by the cluster rules, a cluster label. Output order is by EVSE uid so reruns are byte-identical.
func Build(evses []EVSE, geoms []Geometry, rules []ClusterRule, radiusM float64) (Index, error) {
	if len(geoms) == 0 {
		return Index{}, fmt.Errorf("no geometries")
	}
	idx := Index{
		GeometryVersion: geoms[0].Version,
		GeometryQuality: geoms[0].Quality,
		RadiusM:         radiusM,
		Entries:         []Entry{},
		Unplaced:        []Unplaced{},
		Summary:         Summary{Input: len(evses), ByFamily: map[string]int{}, BySource: map[string]int{}},
	}
	sorted := append([]EVSE(nil), evses...)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].UID < sorted[j].UID })
	seen := map[string]bool{}
	for _, e := range sorted {
		if e.UID == "" {
			return Index{}, fmt.Errorf("an EVSE record has no evse_uid")
		}
		if seen[e.UID] {
			return Index{}, fmt.Errorf("duplicate evse_uid %q", e.UID)
		}
		seen[e.UID] = true
		if !validSource(e.Source) {
			return Index{}, fmt.Errorf("evse %s: unknown source %q", e.UID, e.Source)
		}
		if e.Lat == nil || e.Lon == nil {
			idx.Unplaced = append(idx.Unplaced, Unplaced{e.UID, e.Name, e.Source, "no coordinates"})
			continue
		}
		p := geo.Point{Lat: *e.Lat, Lon: *e.Lon}
		if math.Abs(p.Lat) > 90 || math.Abs(p.Lon) > 180 {
			idx.Unplaced = append(idx.Unplaced, Unplaced{e.UID, e.Name, e.Source, "coordinates out of range"})
			continue
		}
		dist := map[string]float64{}
		var fams []string
		for _, g := range geoms {
			d := geo.DistanceToPolylineM(p, g.Line)
			if d <= radiusM {
				fams = append(fams, g.Family)
				dist[g.Family] = math.Round(d*10) / 10
			}
		}
		if len(fams) == 0 {
			idx.Unplaced = append(idx.Unplaced, Unplaced{e.UID, e.Name, e.Source, "outside every family buffer"})
			continue
		}
		ent := Entry{
			UID: e.UID, EvseID: e.EvseID, LocationID: e.LocationID, Operator: e.Operator,
			Lat: p.Lat, Lon: p.Lon, MaxPowerKW: e.MaxPowerKW, PowerType: e.PowerType, Source: e.Source,
			Families: fams, DistanceM: dist, Cluster: Match(rules, e),
			K1Evidence: e.Source != SourceUserListed,
		}
		if !ent.K1Evidence {
			ent.K1Withheld = "user-listed station (K): unverified, does not count as K1 evidence"
		}
		idx.Entries = append(idx.Entries, ent)
		idx.Summary.InCorridor++
		idx.Summary.BySource[e.Source]++
		if ent.K1Evidence {
			idx.Summary.K1Evidence++
		}
		for _, f := range fams {
			idx.Summary.ByFamily[f]++
		}
	}
	return idx, nil
}

func validSource(s string) bool {
	switch s {
	case SourceUserListed, SourceEPDKList, SourceOCPIFeed, SourceHubFeed:
		return true
	}
	return false
}
