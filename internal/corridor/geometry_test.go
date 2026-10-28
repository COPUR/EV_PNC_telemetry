package corridor

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const geomRoot = "../../corridor/geometries"

func TestShippedVersionsLoadAndMatchTheirPins(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join(geomRoot, "versions.json"))
	if err != nil {
		t.Fatal(err)
	}
	var doc struct {
		Versions []struct {
			Version        string `json:"version"`
			ManifestSHA256 string `json:"manifest_sha256"`
			Quality        string `json:"quality"`
		} `json:"versions"`
		Current string `json:"current"`
	}
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatal(err)
	}
	found := false
	for _, v := range doc.Versions {
		dir := filepath.Join(geomRoot, v.Version)
		body, err := os.ReadFile(filepath.Join(dir, "manifest.json"))
		if err != nil {
			t.Fatal(err)
		}
		sum := sha256.Sum256(body)
		if hex.EncodeToString(sum[:]) != v.ManifestSHA256 {
			t.Errorf("%s: manifest changed after it was published; add a new version instead", v.Version)
		}
		geoms, m, err := LoadGeometries(dir)
		if err != nil {
			t.Fatalf("%s: %v", v.Version, err)
		}
		if m.Quality != v.Quality || len(geoms) != len(Families) {
			t.Errorf("%s: quality %q or %d families do not match versions.json", v.Version, m.Quality, len(geoms))
		}
		found = found || v.Version == doc.Current
	}
	if !found {
		t.Errorf("current version %q is not listed", doc.Current)
	}
}

func TestEditedGeometryFileIsRejected(t *testing.T) {
	dir := t.TempDir()
	copyDir(t, filepath.Join(geomRoot, "v0"), dir)
	p := filepath.Join(dir, "M.geojson")
	b, _ := os.ReadFile(p)
	if err := os.WriteFile(p, []byte(strings.Replace(string(b), "28.979", "28.980", 1)), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, _, err := LoadGeometries(dir); err == nil || !strings.Contains(err.Error(), "immutable") {
		t.Fatalf("want an immutability error, got %v", err)
	}
}

func TestMissingFamilyIsRejected(t *testing.T) {
	dir := t.TempDir()
	copyDir(t, filepath.Join(geomRoot, "v0"), dir)
	var m Manifest
	b, _ := os.ReadFile(filepath.Join(dir, "manifest.json"))
	_ = json.Unmarshal(b, &m)
	m.Files = m.Files[:len(m.Files)-1]
	out, _ := json.Marshal(m)
	_ = os.WriteFile(filepath.Join(dir, "manifest.json"), out, 0o644)
	if _, _, err := LoadGeometries(dir); err == nil || !strings.Contains(err.Error(), "missing") {
		t.Fatalf("want a missing-family error, got %v", err)
	}
}

func copyDir(t *testing.T, from, to string) {
	t.Helper()
	entries, err := os.ReadDir(from)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		b, err := os.ReadFile(filepath.Join(from, e.Name()))
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(to, e.Name()), b, 0o644); err != nil {
			t.Fatal(err)
		}
	}
}
