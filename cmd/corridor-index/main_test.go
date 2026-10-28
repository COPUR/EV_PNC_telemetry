package main

import (
	"bytes"
	"flag"
	"os"
	"testing"
)

var update = flag.Bool("update", false, "rewrite the golden file")

// The sample EVSEs are fictional and sit near the draft v0 geometry; the golden file pins the output format.
func TestIndexAgainstGolden(t *testing.T) {
	var out bytes.Buffer
	args := []string{
		"--evse", "../../testdata/evse-sample.json",
		"--geometries", "../../corridor/geometries/v0",
		"--clusters", "../../corridor/clusters.json",
	}
	if err := run(args, &out); err != nil {
		t.Fatal(err)
	}
	golden := "../../testdata/index-sample.golden.json"
	if *update {
		if err := os.WriteFile(golden, out.Bytes(), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	want, err := os.ReadFile(golden)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(out.Bytes(), want) {
		t.Fatalf("output differs from %s; run go test ./cmd/... -update after checking the change.\n%s", golden, out.String())
	}
}

func TestEvseFlagIsRequired(t *testing.T) {
	if err := run(nil, &bytes.Buffer{}); err == nil {
		t.Fatal("want an error without --evse")
	}
}
