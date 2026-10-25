// Command corridor-index builds the corridor EVSE index: it tags every EVSE within the buffer of a
// route geometry with its family codes and cluster label, and reports which records count as K1 evidence.
//
//	corridor-index --evse evse.json --geometries corridor/geometries/v0 --clusters corridor/clusters.json --out index.json
//
// The output is deterministic: the same inputs give the same bytes, so a nightly run only changes
// the file when the data changed.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/copur/ev_pnc_telemetry/internal/corridor"
)

func main() {
	if err := run(os.Args[1:], os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, "corridor-index:", err)
		os.Exit(1)
	}
}

func run(args []string, stdout io.Writer) error {
	fs := flag.NewFlagSet("corridor-index", flag.ContinueOnError)
	evsePath := fs.String("evse", "", "JSON array of EVSE records (required)")
	geomDir := fs.String("geometries", "corridor/geometries/v0", "geometry version directory")
	clusterPath := fs.String("clusters", "corridor/clusters.json", "cluster rules")
	radius := fs.Float64("radius-m", corridor.DefaultRadiusM, "corridor buffer in metres")
	out := fs.String("out", "-", "output file, or - for stdout")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *evsePath == "" {
		return fmt.Errorf("--evse is required")
	}
	raw, err := os.ReadFile(*evsePath)
	if err != nil {
		return err
	}
	var evses []corridor.EVSE
	if err := json.Unmarshal(raw, &evses); err != nil {
		return fmt.Errorf("%s: %w", *evsePath, err)
	}
	geoms, _, err := corridor.LoadGeometries(*geomDir)
	if err != nil {
		return err
	}
	rules, err := corridor.LoadClusterRules(*clusterPath)
	if err != nil {
		return err
	}
	idx, err := corridor.Build(evses, geoms, rules, *radius)
	if err != nil {
		return err
	}
	body, err := json.MarshalIndent(idx, "", "  ")
	if err != nil {
		return err
	}
	body = append(body, '\n')
	if *out == "-" {
		_, err = stdout.Write(body)
		return err
	}
	return os.WriteFile(*out, body, 0o644)
}
