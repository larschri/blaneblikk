package transform

import (
	"github.com/larschri/blaner/elevationmap"
	"math"
)

type Geopixel struct {
	Distance float64
	Incline float64
}

type Transform struct {
	Easting float64
	Northing float64
	ElevMap elevationmap.ElevationMap
	GeopixelLen int
}

const step float64 = 10
const bottomHeightAngle float64 = -0.08
const totalHeightAngle float64 = 0.16

func (transform Transform) TraceDirection(rad float64, elevation0 float64) []Geopixel{
	geopixels := make([]Geopixel, 0)
	sin := math.Sin(rad)
	cos := math.Cos(rad)
	prevElevation := elevation0
	currHeightAngle := bottomHeightAngle
	for dist := step; dist < 200000; dist = dist + step {
		earthCurvatureAngle := math.Atan2(dist / 2, 6371000.0)
		elevationLimit := elevation0 + dist * math.Tan(currHeightAngle + earthCurvatureAngle)
		elevation := transform.ElevMap.GetElevation(transform.Easting + sin * dist, transform.Northing + cos * dist, elevationLimit)
		if elevation < elevationLimit {
			if elevation == -1 {
				dist = dist + 3*step
			}
			continue
		}
		heightAngle := math.Atan2(elevation - elevation0, dist)

		for currHeightAngle + earthCurvatureAngle <= heightAngle {
			geopixels = append(geopixels, Geopixel{
				Distance: dist,
				Incline:  (elevation - prevElevation),
			})
			currHeightAngle = float64(len(geopixels)) * totalHeightAngle / float64(transform.GeopixelLen) + bottomHeightAngle
		}
		prevElevation = elevation
	}
	return geopixels
}

