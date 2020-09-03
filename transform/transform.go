package transform

import (
	"github.com/larschri/blaner/dataset"
	"math"
)

const (
	maxBlaneDistance  = 200_000.0
	earthRadius       = 6_371_000.0
	bottomHeightAngle = -0.08
	totalHeightAngle  = 0.16
)

var atanPrecalc [10 + maxBlaneDistance/dataset.Unit]float64

func init() {
	for i := 0; i < len(atanPrecalc); i++ {
		atanPrecalc[i] = math.Atan2(float64(dataset.Unit*i)/2, earthRadius)
	}
}

type Geopixel struct {
	Distance float64
	Incline  float64
}

type Transform struct {
	Easting     float64
	Northing    float64
	ElevMap     dataset.ElevationMap
	GeopixelLen int
}

type intStepper struct {
	start   dataset.IntStep
	stepLen dataset.IntStep
}

func (s intStepper) step(i dataset.IntStep) dataset.IntStep {
	return s.start + i*s.stepLen
}

type floatStepper struct {
	start   float64
	stepLen float64
}

func (s floatStepper) step(i dataset.IntStep) float64 {
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

func sign(i float64) dataset.IntStep {
	if i < 0 {
		return -1
	} else {
		return 1
	}
}

func (bld *geoPixelBuilder) elevationLimit(i dataset.IntStep) float64 {
	dist := float64(i) * bld.stepLength
	earthCurvatureAngle := atanPrecalc[int(dist/dataset.Unit)]
	elevationLimit1 := dist * math.Tan(bld.currHeightAngle+earthCurvatureAngle)
	elevationLimit2 := float64(i+dataset.ElevationMapletSize) * bld.stepLength * math.Tan(bld.currHeightAngle+earthCurvatureAngle)
	return math.Min(elevationLimit1, elevationLimit2)
}

func weightElevation(elevation1 dataset.Elevation16, elevation2 dataset.Elevation16, elevation1Weight float64) float64 {
	return (float64(elevation2)*elevation1Weight +
		float64(elevation1)*(1-elevation1Weight)) * dataset.Elevation16Unit
}

func (bld *geoPixelBuilder) updateState(elevation float64, i dataset.IntStep) {

	dist := float64(i) * bld.stepLength
	earthCurvatureAngle := atanPrecalc[int(dist/dataset.Unit)]

	heightAngle := math.Atan2(elevation, dist)

	if bld.currHeightAngle+earthCurvatureAngle <= heightAngle {
		pix := Geopixel{
			Distance: dist,
			Incline:  (elevation - bld.prevElevation) * dataset.Unit / bld.stepLength,
		}
		for bld.currHeightAngle+earthCurvatureAngle <= heightAngle {
			bld.geopixels = append(bld.geopixels, pix)
			bld.currHeightAngle = float64(len(bld.geopixels))*totalHeightAngle/float64(bld.geopixelLen) + bottomHeightAngle
		}
	}
	bld.prevElevation = elevation
}

func atBorder(a dataset.IntStep, b dataset.IntStep) bool {
	if a > b {
		return a-b > 1 || b < 0
	}

	return b-a > 1 || a < 0
}

type smallSquareIter struct {
	front dataset.IntStep
	side  dataset.IntStep
	side2 dataset.IntStep
}

func (i *smallSquareIter) init(front dataset.IntStep, side dataset.IntStep) {
	i.front = front % dataset.ElevationMapletSize
	i.side = side % dataset.ElevationMapletSize
	i.side2 = (side + 1) % dataset.ElevationMapletSize
}

func (bld *geoPixelBuilder) traceEastWest(elevationMap dataset.ElevationMap, eastStepper intStepper, northStepper floatStepper) {
	totalSteps := dataset.IntStep(maxBlaneDistance / bld.stepLength)
	elevation0 := bld.prevElevation + 9
	prevIter := smallSquareIter{
		front: 10000,
		side:  10000,
		side2: 10000,
	}
	var sIter smallSquareIter
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
				i += (dataset.ElevationMapletSize - 1)
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

func (bld *geoPixelBuilder) traceNorthSouth(elevationMap dataset.ElevationMap, eastStepper floatStepper, northStepper intStepper) {
	totalSteps := dataset.IntStep(maxBlaneDistance / bld.stepLength)
	elevation0 := bld.prevElevation + 9
	prevIter := smallSquareIter{
		front: 10000,
		side:  10000,
		side2: 10000,
	}
	var sIter smallSquareIter
	var sq0 = elevationMap.LookupElevationMaplet(dataset.IntStep(math.Floor(eastStepper.start)), northStepper.start)
	var sq1 = elevationMap.LookupElevationMaplet(dataset.IntStep(math.Floor(eastStepper.start))+1, northStepper.start)
	for i := dataset.IntStep(1); i < totalSteps; i++ {
		northStep := northStepper.step(i)
		eastFloat := eastStepper.step(i)
		eastStep := dataset.IntStep(math.Floor(eastFloat))

		sIter.init(northStep, eastStep)

		if atBorder(prevIter.front, sIter.front) {
			elevationLimit := elevation0 + bld.elevationLimit(i)

			if elevationMap.MaxElevation(eastStep, northStep) < elevationLimit &&
				elevationMap.MaxElevation(eastStep+dataset.IntStep(eastStepper.stepLen*dataset.ElevationMapletSize), northStep) < elevationLimit { //?
				i += (dataset.ElevationMapletSize - 1)
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

func (t Transform) TraceDirection(rad float64) []Geopixel {
	northing0 := math.Round(t.Northing/dataset.Unit) * dataset.Unit
	easting0 := math.Round(t.Easting/dataset.Unit) * dataset.Unit

	minEasting, maxNorthing := t.ElevMap.Offsets()
	var eastingStart = dataset.IntStep(easting0-minEasting) / dataset.Unit
	var northingStart = dataset.IntStep(maxNorthing-northing0) / dataset.Unit

	bld := geoPixelBuilder{
		geopixels:       make([]Geopixel, 0, 2000),
		currHeightAngle: bottomHeightAngle,
		prevElevation:   t.ElevMap.Elevation(eastingStart, northingStart),
		geopixelLen:     t.GeopixelLen,
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
