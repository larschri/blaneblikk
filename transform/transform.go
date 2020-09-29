// Package transform provides a functionality to compute the pixel data for the image from elevation data.
package transform

import (
	"github.com/larschri/blaner/dataset"
	"math"
)

const (
	// maxBlaneDistance is the distance to iterate through in meters
	maxBlaneDistance  = 200_000.0

	// earthRadius is the radius of the earth in meters. Assuming earth is a perfect sphere.
	earthRadius       = 6_371_000.0

	// bottomHeightAngle is the angle between the straight horizontal line and the bottom of the image
	bottomHeightAngle = -0.06

	// totalHeightAngle is the angle between bottomHeightAngle and the top of the image
	totalHeightAngle  = 0.08
	TotalHeightAngle  = 0.08
)

// earthCurvatureDecline contains "elevation penalty" by distance caused by earth curvature. This is an optimisation
// to avoid slow atan operations.
var earthCurvatureDecline [10000 + maxBlaneDistance/dataset.Unit]float64

func init() {
	for i := 0; i < len(earthCurvatureDecline); i++ {
		earthCurvatureDecline[i] = float64(i) * dataset.Unit * math.Atan2(float64(dataset.Unit*i)/2, earthRadius)
	}
}

type GeoPixel struct {
	Distance float64
	Incline  float64
}

type Transform struct {
	Easting     float64
	Northing    float64
	ElevMap     dataset.ElevationMap
	GeoPixelLen int
	geoPixelTan []float64
}

func (t *Transform) init() {
	if t.geoPixelTan == nil {
		t.geoPixelTan = make([]float64, t.GeoPixelLen, t.GeoPixelLen)

		angleStep := totalHeightAngle / float64(t.GeoPixelLen)
		for i := 0; i < t.GeoPixelLen; i++ {
			t.geoPixelTan[i] = math.Tan(bottomHeightAngle + float64(i) * angleStep)
		}
	}
}

// intStepper is used to compute the position in the "forward" direction. The length of one step is either 1 or -1.
type intStepper struct {
	start   dataset.IntStep
	stepLen dataset.IntStep
}

func (s intStepper) step(i dataset.IntStep) dataset.IntStep {
	return s.start + i*s.stepLen
}

// floatStepper is used to compute the position in the "sideways" direction.
// The length of one step is in the open interval <-1, 1>
type floatStepper struct {
	start   float64
	stepLen float64
}

func (s floatStepper) step(i dataset.IntStep) float64 {
	return s.start + float64(i)*s.stepLen
}

type geoPixelBuilder struct {
	stepLength float64

	geopixels       []GeoPixel
	prevElevation   float64
	geopixelLen     int
	geopixelTan     []float64
}

func sign(i float64) dataset.IntStep {
	if i < 0 {
		return -1
	} else {
		return 1
	}
}

// elevationLimit calculates the lowest elevation that would be visible when traversing the next ElevationMap.
// The next ElevationMap can be skipped if the maximum elevation is lower than this.
func (bld *geoPixelBuilder) elevationLimit(i dataset.IntStep) float64 {
	dist1 := float64(i) * bld.stepLength
	elevationLimit1 := earthCurvatureDecline[int(dist1/dataset.Unit)] + bld.geopixelTan[len(bld.geopixels)] * dist1

	dist2 := float64(i+dataset.ElevationMapletSize) * bld.stepLength
	elevationLimit2 :=  earthCurvatureDecline[int(dist2/dataset.Unit)] + bld.geopixelTan[len(bld.geopixels)] * dist2

	return math.Min(elevationLimit1, elevationLimit2)
}

// weightElevation computes the weighted average of two elevations. This is used to compute the elevation for
// a point on a straight line between two points with known elevations.
func weightElevation(elevation1 dataset.Elevation16, elevation2 dataset.Elevation16, elevation1Weight float64) float64 {
	return (float64(elevation2)*elevation1Weight +
		float64(elevation1)*(1-elevation1Weight)) * dataset.Elevation16Unit
}

// updateState updates the elevation and pixels for each step during the iteration
func (bld *geoPixelBuilder) updateState(elevation float64, i dataset.IntStep) {
	dist := float64(i) * bld.stepLength

	elevationX := elevation - earthCurvatureDecline[int(dist/dataset.Unit)]
	tanX := elevationX / dist

	if tanX > bld.geopixelTan[len(bld.geopixels)] {
		pix := GeoPixel{
			Distance: dist,
			Incline:  (elevation - bld.prevElevation) * dataset.Unit / bld.stepLength,
		}
		for tanX > bld.geopixelTan[len(bld.geopixels)] {
			bld.geopixels = append(bld.geopixels, pix)
		}
	}

	bld.prevElevation = elevation
}

// atBorder is true if a and be is not subsequent ordinates in the same ElevationMaplet
func atBorder(a dataset.IntStep, b dataset.IntStep) bool {
	if a > b {
		return a-b > 1 || b < 0
	}

	return b-a > 1 || a < 0
}

// elevationMapletIter holds the indices of the ElevationMap during iteration
type elevationMapletIter struct {
	// front is the iteration in the forward direction. This is incremented by one for each step.
	front dataset.IntStep

	// side is the "sideways" position.
	side  dataset.IntStep

	// side2 is the side + 1, but will wrap around and start from zero in the next ElevationMaplet at the boundary.
	side2 dataset.IntStep
}

// init sets the iteration values for the elevationMapletIter and is invoked at before each step during the iteration
func (i *elevationMapletIter) init(front dataset.IntStep, side dataset.IntStep) {
	i.front = front % dataset.ElevationMapletSize
	i.side = side % dataset.ElevationMapletSize
	i.side2 = (side + 1) % dataset.ElevationMapletSize
}

// traceEastWest iterates through the ElevationMap by incrementing easting step-by-step while adjusting northing accordingly
func (bld *geoPixelBuilder) traceEastWest(elevationMap dataset.ElevationMap, eastStepper intStepper, northStepper floatStepper) {
	totalSteps := dataset.IntStep(maxBlaneDistance / bld.stepLength)
	elevation0 := bld.prevElevation + 9
	prevIter := elevationMapletIter{
		front: 10000,
		side:  10000,
		side2: 10000,
	}
	var sIter elevationMapletIter
	var sq0 = elevationMap.LookupElevationMaplet(eastStepper.start, dataset.IntStep(math.Floor(northStepper.start)))
	var sq1 = elevationMap.LookupElevationMaplet(eastStepper.start, dataset.IntStep(math.Floor(northStepper.start))+1)
	for i := dataset.IntStep(1); i < totalSteps; i++ {
		eastStep := eastStepper.step(i)
		northFloat := northStepper.step(i)
		northStep := dataset.IntStep(math.Floor(northFloat))

		sIter.init(eastStep, northStep)

		if atBorder(prevIter.front, sIter.front) {
			elevationLimit := elevation0 + bld.elevationLimit(i)

			if elevationMap.MaxElevation(eastStep, northStep) < elevationLimit &&
				elevationMap.MaxElevation(eastStep, northStep+dataset.IntStep(northStepper.stepLen*dataset.ElevationMapletSize)) < elevationLimit {
				i += dataset.ElevationMapletSize - 1
				eastingIndex := eastStepper.step(i)
				northFloat = northStepper.step(i)
				northStep = dataset.IntStep(math.Floor(northFloat))
				prevIter.init(eastingIndex, northStep)
				continue
			}
			sq0 = elevationMap.LookupElevationMaplet(eastStep, northStep)
			if sq0 == nil {
				break
			}
			if sIter.side2 == 0 {
				sq1 = elevationMap.LookupElevationMaplet(eastStep, northStep+1)
				if sq1 == nil {
					break
				}
			} else {
				sq1 = sq0
			}
		} else {
			if atBorder(prevIter.side, sIter.side) {
				if sIter.side == 0 {
					sq0 = sq1
				} else {
					sq0 = elevationMap.LookupElevationMaplet(eastStep, northStep)
					if sq0 == nil {
						break
					}
				}
			}

			if atBorder(prevIter.side2, sIter.side2) {
				if sIter.side2 == 0 {
					sq1 = elevationMap.LookupElevationMaplet(eastStep, northStep+1)
					if sq1 == nil {
						break
					}
				} else {
					sq1 = sq0
				}
			}
		}
		elevation := weightElevation(sq0[sIter.side][sIter.front],
			sq1[sIter.side2][sIter.front],
			northFloat-float64(northStep))
		bld.updateState(elevation-elevation0, i)

		prevIter = sIter
	}
}

// traceNorthSouth iterates through the ElevationMap by incrementing northing step-by-step while adjusting easting accordingly
func (bld *geoPixelBuilder) traceNorthSouth(elevationMap dataset.ElevationMap, eastStepper floatStepper, northStepper intStepper) {
	totalSteps := dataset.IntStep(maxBlaneDistance / bld.stepLength)
	elevation0 := bld.prevElevation + 9
	prevIter := elevationMapletIter{
		front: 10000,
		side:  10000,
		side2: 10000,
	}
	var sIter elevationMapletIter
	var sq0 = elevationMap.LookupElevationMaplet(dataset.IntStep(math.Floor(eastStepper.start)), northStepper.start)
	var sq1 = elevationMap.LookupElevationMaplet(dataset.IntStep(math.Floor(eastStepper.start))+1, northStepper.start)
	for i := dataset.IntStep(1); i < totalSteps; i++ {
		northStep := northStepper.step(i)
		eastFloat := eastStepper.step(i)
		eastStep := dataset.IntStep(math.Floor(eastFloat))

		sIter.init(northStep, eastStep)

		if atBorder(prevIter.front, sIter.front) {
			//elevationLimit := elevation0 + bld.elevationLimit(i)
			elevationLimit := elevation0 + bld.elevationLimit(i)

			if elevationMap.MaxElevation(eastStep, northStep) < elevationLimit &&
				elevationMap.MaxElevation(eastStep+dataset.IntStep(eastStepper.stepLen*dataset.ElevationMapletSize), northStep) < elevationLimit { //?
				i += dataset.ElevationMapletSize - 1
				northStep = northStepper.step(i)
				eastFloat = eastStepper.step(i)
				eastStep = dataset.IntStep(math.Floor(eastFloat))
				prevIter.init(northStep, eastStep)
				continue
			}
			sq0 = elevationMap.LookupElevationMaplet(eastStep, northStep)
			if sq0 == nil {
				break
			}
			if sIter.side2 == 0 {
				sq1 = elevationMap.LookupElevationMaplet(eastStep+1, northStep)
				if sq1 == nil {
					break
				}
			} else {
				sq1 = sq0
			}
		} else {
			if atBorder(prevIter.side, sIter.side) {
				if sIter.side == 0 {
					sq0 = sq1
				} else {
					sq0 = elevationMap.LookupElevationMaplet(eastStep, northStep)
					if sq0 == nil {
						break
					}
				}
			}

			if atBorder(prevIter.side2, sIter.side2) {
				if sIter.side2 == 0 {
					sq1 = elevationMap.LookupElevationMaplet(eastStep+1, northStep)
					if sq1 == nil {
						break
					}
				} else {
					sq1 = sq0
				}
			}
		}
		elevation := weightElevation(sq0[sIter.front][sIter.side],
			sq1[sIter.front][sIter.side2],
			eastFloat-float64(eastStep))

		bld.updateState(elevation-elevation0, i)
		prevIter = sIter
	}
}

//TraceDirection iterates through the ElevationMap to build a column of GeoPixel that can be used to render an image.
//The iteration starts at [t.Northing, t.Easting] and the direction is given by rad.
func (t *Transform) TraceDirection(rad float64, pixels []GeoPixel) []GeoPixel {
	t.init()
	northing0 := math.Round(t.Northing/dataset.Unit) * dataset.Unit
	easting0 := math.Round(t.Easting/dataset.Unit) * dataset.Unit

	minEasting, maxNorthing := t.ElevMap.Offsets()
	var eastingStart = dataset.IntStep(easting0-minEasting) / dataset.Unit
	var northingStart = dataset.IntStep(maxNorthing-northing0) / dataset.Unit

	bld := geoPixelBuilder{
		geopixels:       pixels,
		prevElevation:   t.ElevMap.Elevation(eastingStart, northingStart),
		geopixelLen:     t.GeoPixelLen,
		geopixelTan:     t.geoPixelTan,
	}

	sin := math.Sin(rad) // east
	cos := math.Cos(rad) // north

	if math.Abs(sin) > math.Abs(cos) {
		bld.stepLength = dataset.Unit / math.Abs(sin)
		bld.traceEastWest(t.ElevMap,
			intStepper{
				start:   eastingStart,
				stepLen: sign(sin),
			},
			floatStepper{
				start:   float64(northingStart),
				stepLen: -cos / math.Abs(sin),
			})
	} else {
		bld.stepLength = dataset.Unit / math.Abs(cos)
		bld.traceNorthSouth(t.ElevMap,
			floatStepper{
				start:   float64(eastingStart),
				stepLen: sin / math.Abs(cos),
			},
			intStepper{
				start:   northingStart,
				stepLen: -sign(cos),
			})
	}

	return bld.geopixels
}
