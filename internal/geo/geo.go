// Package geo has the few geometry helpers the corridor index needs.
package geo

import "math"

const earthRadiusM = 6371008.8

// Point is a WGS84 position in degrees.
type Point struct {
	Lat, Lon float64
}

// DistanceToPolylineM returns the shortest distance in metres from p to the polyline.
// Each segment is measured in a local equirectangular projection centred on p, which is
// accurate to well under a metre per kilometre at the distances used here (a few km).
func DistanceToPolylineM(p Point, line []Point) float64 {
	switch len(line) {
	case 0:
		return math.Inf(1)
	case 1:
		return haversineM(p, line[0])
	}
	best := math.Inf(1)
	for i := 0; i+1 < len(line); i++ {
		if d := distanceToSegmentM(p, line[i], line[i+1]); d < best {
			best = d
		}
	}
	return best
}

func haversineM(a, b Point) float64 {
	la1, la2 := rad(a.Lat), rad(b.Lat)
	dla, dlo := la2-la1, rad(b.Lon-a.Lon)
	h := math.Sin(dla/2)*math.Sin(dla/2) + math.Cos(la1)*math.Cos(la2)*math.Sin(dlo/2)*math.Sin(dlo/2)
	return 2 * earthRadiusM * math.Asin(math.Sqrt(h))
}

func distanceToSegmentM(p, a, b Point) float64 {
	// project to metres around p
	kx := math.Cos(rad(p.Lat)) * earthRadiusM * math.Pi / 180
	ky := earthRadiusM * math.Pi / 180
	ax, ay := (a.Lon-p.Lon)*kx, (a.Lat-p.Lat)*ky
	bx, by := (b.Lon-p.Lon)*kx, (b.Lat-p.Lat)*ky
	dx, dy := bx-ax, by-ay
	l2 := dx*dx + dy*dy
	t := 0.0
	if l2 > 0 {
		t = math.Max(0, math.Min(1, -(ax*dx+ay*dy)/l2))
	}
	return math.Hypot(ax+t*dx, ay+t*dy)
}

func rad(d float64) float64 { return d * math.Pi / 180 }
