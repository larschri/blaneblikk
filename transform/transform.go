package transform

import (
	"github.com/larschri/blaner/dataset/dataset5000"
	"math"
)

const (
	maxBlaneDistance  = 200_000.0
	earthRadius       = 6_371_000.0
	bottomHeightAngle = -0.08
	totalHeightAngle  = 0.16
)

var atanPrecalc [10 + maxBlaneDistance/dataset5000.Unit]float64

func init() {
	for i := 0; i < len(atanPrecalc); i++ {
		atanPrecalc[i] = math.Atan2(float64(dataset5000.Unit*i)/2, earthRadius)
	}
}

type Geopixel struct {
	Distance float64
	Incline  float64
}

type Transform struct {
	Easting     float64
	Northing    float64
	ElevMap     dataset5000.ElevationMap
	GeopixelLen int
}

type intStepper struct {
	start   dataset5000.IntStep
	stepLen dataset5000.IntStep
}

func (s intStepper) step(i dataset5000.IntStep) dataset5000.IntStep {
	return s.start + i*s.stepLen
}

type floatStepper struct {
	start   float64
	stepLen float64
}

func (s floatStepper) step(i dataset5000.IntStep) float64 {
	return s.start + float64(i)*s.stepLen
}

type geoPixelBuilder struct {
	stepLength float64

	//
	geopixels       []Geopixel
	currHeightAngle float64
	prevElevation   float64
	geopixelLen     int
}

func sign(i float64) dataset5000.IntStep {
	if i < 0 {
		return -1
	} else {
		return 1
	}
}

func (bld *geoPixelBuilder) elevationLimit(i dataset5000.IntStep) float64 {
	dist := float64(i) * bld.stepLength
	earthCurvatureAngle := atanPrecalc[int(dist/dataset5000.Unit)]
	elevationLimit1 := dist * math.Tan(bld.currHeightAngle+earthCurvatureAngle)
	elevationLimit2 := float64(i+dataset5000.SmallSquareSize) * bld.stepLength * math.Tan(bld.currHeightAngle+earthCurvatureAngle)
	return math.Min(elevationLimit1, elevationLimit2)
}

func weightElevation(elevation1 dataset5000.Elevation16, elevation2 dataset5000.Elevation16, elevation1Weight float64) float64 {
	return (float64(elevation2)*elevation1Weight +
		float64(elevation1)*(1-elevation1Weight)) * dataset5000.Elevation16Unit
}

func (bld *geoPixelBuilder) updateState(elevation float64, i dataset5000.IntStep) {

	dist := float64(i) * bld.stepLength
	earthCurvatureAngle := atanPrecalc[int(dist/dataset5000.Unit)]

	heightAngle := math.Atan2(elevation, dist)

	for bld.currHeightAngle+earthCurvatureAngle <= heightAngle {
		bld.geopixels = append(bld.geopixels, Geopixel{
			Distance: dist,
			Incline:  (elevation - bld.prevElevation) * dataset5000.Unit / bld.stepLength,
		})
		bld.currHeightAngle = float64(len(bld.geopixels))*totalHeightAngle/float64(bld.geopixelLen) + bottomHeightAngle
	}
	bld.prevElevation = elevation
}

func atBorder(a dataset5000.IntStep, b dataset5000.IntStep) bool {
	if a > b {
		return a-b > 1 || b < 0
	}

	return b-a > 1 || a < 0
}

type smallSquareIter struct {
	front dataset5000.IntStep
	side  dataset5000.IntStep
	side2 dataset5000.IntStep
}

func (i *smallSquareIter) init(front dataset5000.IntStep, side dataset5000.IntStep) {
	i.front = front % dataset5000.SmallSquareSize
	i.side = side % dataset5000.SmallSquareSize
	i.side2 = (side + 1) % dataset5000.SmallSquareSize
}

func (bld *geoPixelBuilder) traceEastWest(elevationMap dataset5000.ElevationMap, eastStepper intStepper, northStepper floatStepper) {
	totalSteps := dataset5000.IntStep(maxBlaneDistance / bld.stepLength)
	elevation0 := bld.prevElevation + 9
	prevIter := smallSquareIter{
		front: 10000,
		side:  10000,
		side2: 10000,
	}
	var sIter smallSquareIter
	var sq0 = elevationMap.LookupSquare(eastStepper.start, dataset5000.IntStep(math.Floor(northStepper.start)))
	var sq1 = elevationMap.LookupSquare(eastStepper.start, dataset5000.IntStep(math.Floor(northStepper.start))+1)
	for i := dataset5000.IntStep(1); i < totalSteps; i++ {
		eastStep := eastStepper.step(i)
		northFloat := northStepper.step(i)
		northStep := dataset5000.IntStep(math.Floor(northFloat))

		sIter.init(eastStep, northStep)

		if atBorder(prevIter.front, sIter.front) {
			elevationLimit := elevation0 + bld.elevationLimit(i)

			if elevationMap.MaxElevation(eastStep, northStep) < elevationLimit &&
				elevationMap.MaxElevation(eastStep, northStep+dataset5000.IntStep(northStepper.stepLen*dataset5000.SmallSquareSize)) < elevationLimit {
				i += (dataset5000.SmallSquareSize - 1)
				eastingIndex := eastStepper.step(i)
				northFloat = northStepper.step(i)
				northStep = dataset5000.IntStep(math.Floor(northFloat))
				prevIter.init(eastingIndex, northStep)
				continue
			}
			sq0 = elevationMap.LookupSquare(eastStep, northStep)
			if sq0 == nil {
				break
			}
			if sIter.side2 == 0 {
				sq1 = elevationMap.LookupSquare(eastStep, northStep+1)
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
					sq0 = elevationMap.LookupSquare(eastStep, northStep)
					if sq0 == nil {
						break
					}
				}
			}

			if atBorder(prevIter.side2, sIter.side2) {
				if sIter.side2 == 0 {
					sq1 = elevationMap.LookupSquare(eastStep, northStep+1)
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

func (bld *geoPixelBuilder) traceNorthSouth(elevationMap dataset5000.ElevationMap, eastStepper floatStepper, northStepper intStepper) {
	totalSteps := dataset5000.IntStep(maxBlaneDistance / bld.stepLength)
	elevation0 := bld.prevElevation + 9
	prevIter := smallSquareIter{
		front: 10000,
		side:  10000,
		side2: 10000,
	}
	var sIter smallSquareIter
	var sq0 = elevationMap.LookupSquare(dataset5000.IntStep(math.Floor(eastStepper.start)), northStepper.start)
	var sq1 = elevationMap.LookupSquare(dataset5000.IntStep(math.Floor(eastStepper.start))+1, northStepper.start)
	for i := dataset5000.IntStep(1); i < totalSteps; i++ {
		northStep := northStepper.step(i)
		eastFloat := eastStepper.step(i)
		eastStep := dataset5000.IntStep(math.Floor(eastFloat))

		sIter.init(northStep, eastStep)

		if atBorder(prevIter.front, sIter.front) {
			elevationLimit := elevation0 + bld.elevationLimit(i)

			if elevationMap.MaxElevation(eastStep, northStep) < elevationLimit &&
				elevationMap.MaxElevation(eastStep+dataset5000.IntStep(eastStepper.stepLen*dataset5000.SmallSquareSize), northStep) < elevationLimit { //?
				i += (dataset5000.SmallSquareSize - 1)
				northStep = northStepper.step(i)
				eastFloat = eastStepper.step(i)
				eastStep = dataset5000.IntStep(math.Floor(eastFloat))
				prevIter.init(northStep, eastStep)
				continue
			}
			sq0 = elevationMap.LookupSquare(eastStep, northStep)
			if sq0 == nil {
				break
			}
			if sIter.side2 == 0 {
				sq1 = elevationMap.LookupSquare(eastStep+1, northStep)
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
					sq0 = elevationMap.LookupSquare(eastStep, northStep)
					if sq0 == nil {
						break
					}
				}
			}

			if atBorder(prevIter.side2, sIter.side2) {
				if sIter.side2 == 0 {
					sq1 = elevationMap.LookupSquare(eastStep+1, northStep)
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

func (t Transform) TraceDirection(rad float64) []Geopixel {
	northing0 := math.Round(t.Northing/dataset5000.Unit) * dataset5000.Unit
	easting0 := math.Round(t.Easting/dataset5000.Unit) * dataset5000.Unit

	minEasting, maxNorthing := t.ElevMap.Offsets()
	var eastingStart = dataset5000.IntStep(easting0-minEasting) / dataset5000.Unit
	var northingStart = dataset5000.IntStep(maxNorthing-northing0) / dataset5000.Unit

	bld := geoPixelBuilder{
		geopixels:       make([]Geopixel, 1000),
		currHeightAngle: bottomHeightAngle,
		prevElevation:   t.ElevMap.Elevation(eastingStart, northingStart),
		geopixelLen:     t.GeopixelLen,
	}

	sin := math.Sin(rad) // east
	cos := math.Cos(rad) // north

	if math.Abs(sin) > math.Abs(cos) {
		bld.stepLength = dataset5000.Unit / math.Abs(sin)
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
		bld.stepLength = dataset5000.Unit / math.Abs(cos)
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
