package corridor

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/copur/ev_pnc_telemetry/internal/geo"
)

// Synthetic, well separated geometries so distances are exact: each family is a west-east line at its own latitude.
func testGeoms() []Geometry {
	lat := map[string]float64{FamilyM: 40.0, FamilyA1: 41.0, FamilyA2: 37.5, FamilyA3: 38.5, FamilyA4: 36.0}
	var out []Geometry
	for _, f := range Families {
		out = append(out, Geometry{Family: f, Version: "t", Quality: QualityRouted,
			Line: []geo.Point{{Lat: lat[f], Lon: 28.0}, {Lat: lat[f], Lon: 29.0}}})
	}
	return out
}

func f64(v float64) *float64 { return &v }

// 1.5 km is 0.01349 degrees of latitude.
const km15 = 1.5 / 111.195

var oksijen = []ClusterRule{{Label: "Oksijen", NameContains: []string{"oksijen"}, Confidence: "U"}}

// Scenario "EVSEs are tagged with family and cluster" (corridor-planning.feature, CP-GEO-002).
func TestEvseNearTheMainRouteInAnOksijenClusterIsTagged(t *testing.T) {
	idx, err := Build([]EVSE{{UID: "E1", Name: "Oksijen Dinlenme Tesisi", Lat: f64(40.0 + km15), Lon: f64(28.5), Source: SourceOCPIFeed}},
		testGeoms(), oksijen, DefaultRadiusM)
	if err != nil {
		t.Fatal(err)
	}
	if len(idx.Entries) != 1 {
		t.Fatalf("want 1 entry, got %+v", idx)
	}
	e := idx.Entries[0]
	if !reflect.DeepEqual(e.Families, []string{FamilyM}) || e.Cluster != "Oksijen" {
		t.Fatalf("families %v cluster %q", e.Families, e.Cluster)
	}
	if d := e.DistanceM[FamilyM]; d < 1490 || d > 1510 {
		t.Fatalf("distance %.1f m, want about 1500", d)
	}
}

func TestEvseBeyondTheBufferIsUnplaced(t *testing.T) {
	idx, _ := Build([]EVSE{{UID: "E1", Lat: f64(40.0 + 2.5/111.195), Lon: f64(28.5), Source: SourceOCPIFeed}}, testGeoms(), nil, DefaultRadiusM)
	if len(idx.Entries) != 0 || len(idx.Unplaced) != 1 || idx.Unplaced[0].Reason != "outside every family buffer" {
		t.Fatalf("%+v", idx)
	}
}

func TestEvseBetweenTwoFamiliesCarriesBoth(t *testing.T) {
	geoms := testGeoms()
	geoms[2].Line = []geo.Point{{Lat: 40.01, Lon: 28.0}, {Lat: 40.01, Lon: 29.0}} // A2 right next to M
	idx, _ := Build([]EVSE{{UID: "E1", Lat: f64(40.005), Lon: f64(28.5), Source: SourceHubFeed}}, geoms, nil, DefaultRadiusM)
	if got := idx.Entries[0].Families; !reflect.DeepEqual(got, []string{FamilyM, FamilyA2}) {
		t.Fatalf("families %v", got)
	}
	if idx.Summary.ByFamily[FamilyM] != 1 || idx.Summary.ByFamily[FamilyA2] != 1 || idx.Summary.InCorridor != 1 {
		t.Fatalf("summary %+v", idx.Summary)
	}
}

// AC: stations the user listed are marked K and do not count as K1 evidence.
func TestUserListedStationsAreIndexedButNeverK1Evidence(t *testing.T) {
	evs := []EVSE{
		{UID: "U1", Name: "Soke Shell", Lat: f64(37.5), Lon: f64(28.5), Source: SourceUserListed},
		{UID: "V1", Name: "Soke ZES", Lat: f64(37.5), Lon: f64(28.6), Source: SourceOCPIFeed},
	}
	idx, err := Build(evs, testGeoms(), nil, DefaultRadiusM)
	if err != nil {
		t.Fatal(err)
	}
	if idx.Summary.InCorridor != 2 || idx.Summary.K1Evidence != 1 {
		t.Fatalf("summary %+v", idx.Summary)
	}
	for _, e := range idx.Entries {
		if (e.UID == "U1") == e.K1Evidence {
			t.Errorf("%s: k1_evidence=%v", e.UID, e.K1Evidence)
		}
		if e.UID == "U1" && e.K1Withheld == "" {
			t.Error("a withheld K1 record must say why")
		}
	}
}

func TestShippedUserListedStationsAreAllWithoutCoordinatesAndCountForNothing(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "..", "corridor", "candidates", "user-listed.json"))
	if err != nil {
		t.Fatal(err)
	}
	var doc struct {
		Stations []struct {
			Name   string `json:"name"`
			Source string `json:"source"`
		} `json:"stations"`
	}
	if err := json.Unmarshal(raw, &doc); err != nil || len(doc.Stations) == 0 {
		t.Fatalf("parse: %v (%d stations)", err, len(doc.Stations))
	}
	var evs []EVSE
	for i, s := range doc.Stations {
		if s.Source != SourceUserListed {
			t.Errorf("%s: source %q, want user_listed", s.Name, s.Source)
		}
		evs = append(evs, EVSE{UID: string(rune('a' + i)), Name: s.Name, Source: s.Source})
	}
	idx, err := Build(evs, testGeoms(), nil, DefaultRadiusM)
	if err != nil {
		t.Fatal(err)
	}
	if idx.Summary.K1Evidence != 0 || len(idx.Unplaced) != len(evs) {
		t.Fatalf("summary %+v, unplaced %d", idx.Summary, len(idx.Unplaced))
	}
}

func TestOutputIsIndependentOfInputOrder(t *testing.T) {
	a := []EVSE{
		{UID: "B", Lat: f64(40.0), Lon: f64(28.2), Source: SourceOCPIFeed},
		{UID: "A", Lat: f64(41.0), Lon: f64(28.4), Source: SourceHubFeed},
		{UID: "C", Source: SourceEPDKList},
	}
	b := []EVSE{a[2], a[0], a[1]}
	i1, _ := Build(a, testGeoms(), nil, DefaultRadiusM)
	i2, _ := Build(b, testGeoms(), nil, DefaultRadiusM)
	j1, _ := json.Marshal(i1)
	j2, _ := json.Marshal(i2)
	if string(j1) != string(j2) {
		t.Fatalf("outputs differ:\n%s\n%s", j1, j2)
	}
}

func TestBadInputIsRejected(t *testing.T) {
	cases := map[string][]EVSE{
		"no uid":         {{Source: SourceOCPIFeed}},
		"duplicate uid":  {{UID: "A", Source: SourceOCPIFeed}, {UID: "A", Source: SourceOCPIFeed}},
		"unknown source": {{UID: "A", Source: "rumour"}},
	}
	for name, evs := range cases {
		if _, err := Build(evs, testGeoms(), nil, DefaultRadiusM); err == nil {
			t.Errorf("%s: want an error", name)
		}
	}
}

func TestOutOfRangeCoordinatesAreUnplacedNotIndexed(t *testing.T) {
	idx, _ := Build([]EVSE{{UID: "A", Lat: f64(95), Lon: f64(10), Source: SourceOCPIFeed}}, testGeoms(), nil, DefaultRadiusM)
	if len(idx.Entries) != 0 || idx.Unplaced[0].Reason != "coordinates out of range" {
		t.Fatalf("%+v", idx)
	}
}

func TestClusterRules(t *testing.T) {
	rules := []ClusterRule{
		{Label: "X", Operators: []string{"acme"}},
		{Label: "Oksijen", SiteIDs: []string{"S-9"}, NameContains: []string{"OKSIJEN"}},
	}
	for _, c := range []struct {
		e    EVSE
		want string
	}{
		{EVSE{Operator: "ACME"}, "X"},
		{EVSE{SiteID: "S-9"}, "Oksijen"},
		{EVSE{Name: "Gebze Oksijen tesisi"}, "Oksijen"},
		{EVSE{Name: "other", Operator: "z"}, ""},
		{EVSE{}, ""},
	} {
		if got := Match(rules, c.e); got != c.want {
			t.Errorf("%+v: got %q want %q", c.e, got, c.want)
		}
	}
}

func TestShippedClusterRulesLoad(t *testing.T) {
	rules, err := LoadClusterRules(filepath.Join("..", "..", "corridor", "clusters.json"))
	if err != nil || len(rules) == 0 {
		t.Fatalf("%v %d", err, len(rules))
	}
}
