package helper

import (
	"math"
	"testing"
)

func TestDistanceMeters(t *testing.T) {
	// Monas to Bundaran HI, Jakarta: about 2.2 km.
	d := DistanceMeters(-6.1753924, 106.8271528, -6.1950217, 106.8229886)
	if d < 2150 || d > 2300 {
		t.Fatalf("distance %.0f m, want about 2200 m", d)
	}

	if DistanceMeters(-6.2, 106.8, -6.2, 106.8) != 0 {
		t.Fatal("same point must be 0 m")
	}

	// 0.0009 degrees of latitude is about 100 m.
	if d := DistanceMeters(-6.2, 106.8, -6.2009, 106.8); math.Abs(d-100) > 1 {
		t.Fatalf("distance %.2f m, want about 100 m", d)
	}
}
