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

type loopValues struct {
	earthCurvatureAngle float64
	elevationLimit float64
	elevation float64
}

func (transform Transform) TraceDirection(rad float64, elevation0 float64) []Geopixel{
	geopixels := make([]Geopixel, 0)
	loopValueSlice := make([]loopValues, 0)
	sin := math.Sin(rad)
	cos := math.Cos(rad)
	currHeightAngle := bottomHeightAngle
	prevElevation := elevation0
	for dist := step; dist < 200000; dist = dist + step {
		loopVals := loopValues{}
		if len(loopValueSlice) > 1 {
			loopVals = loopValueSlice[len(loopValueSlice) - 1]
			loopValueSlice = loopValueSlice[:len(loopValueSlice)-1]
		} else {
			loopVals.earthCurvatureAngle = math.Atan2(dist/2, 6371000.0)
			loopVals.elevationLimit = elevation0 + dist*math.Tan(currHeightAngle+loopVals.earthCurvatureAngle)
			loopVals.elevation = transform.ElevMap.GetElevation(transform.Easting+sin*dist, transform.Northing+cos*dist, loopVals.elevationLimit)
			if loopVals.elevation < loopVals.elevationLimit {
				if loopVals.elevation == -1 {
					dist = dist + step*15
				}
				prevElevation = loopVals.elevation
				continue
			}
		}

		if prevElevation == -1 {
			for loopVals.elevation >= loopVals.elevationLimit {
				loopValueSlice = append(loopValueSlice, loopVals)
				dist = dist - step
				loopVals.earthCurvatureAngle = math.Atan2(dist / 2, 6371000.0)
				loopVals.elevationLimit = elevation0 + dist * math.Tan(currHeightAngle + loopVals.earthCurvatureAngle)
				loopVals.elevation = transform.ElevMap.GetElevation(transform.Easting + sin * dist, transform.Northing + cos * dist, 0)
			}
			prevElevation = loopVals.elevation
			continue
		}
		heightAngle := math.Atan2(loopVals.elevation - elevation0, dist)

		for currHeightAngle + loopVals.earthCurvatureAngle <= heightAngle {
			geopixels = append(geopixels, Geopixel{
				Distance: dist,
				Incline:  (loopVals.elevation - prevElevation),
			})
			currHeightAngle = float64(len(geopixels)) * totalHeightAngle / float64(transform.GeopixelLen) + bottomHeightAngle
		}
		prevElevation = loopVals.elevation
	}
	return geopixels
}

