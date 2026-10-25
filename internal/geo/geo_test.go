package geo

import (
	"math"
	"testing"
)

func TestDistanceToPolylineM(t *testing.T) {
	// A due-east line at 40°N; a point 0.01° (about 1111 m) north of its middle.
	line := []Point{{40, 29.0}, {40, 29.2}}
	got := DistanceToPolylineM(Point{40.01, 29.1}, line)
	if math.Abs(got-1111.95) > 2 {
		t.Fatalf("perpendicular distance = %.1f m, want about 1112 m", got)
	}
}

func TestDistanceBeyondTheEndsMeasuresToTheEndpoint(t *testing.T) {
	line := []Point{{40, 29.0}, {40, 29.2}}
	got := DistanceToPolylineM(Point{40, 29.3}, line)
	want := haversineM(Point{40, 29.3}, Point{40, 29.2})
	if math.Abs(got-want) > 5 {
		t.Fatalf("got %.1f m, want %.1f m", got, want)
	}
}

func TestDegenerateLines(t *testing.T) {
	if !math.IsInf(DistanceToPolylineM(Point{1, 1}, nil), 1) {
		t.Fatal("empty polyline should be infinitely far")
	}
	if d := DistanceToPolylineM(Point{40, 29}, []Point{{40, 29}}); d != 0 {
		t.Fatalf("single point at the same place: %v", d)
	}
}
