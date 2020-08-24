package dataset5000

import (
	"math"
)

type ElevationMapInterface interface {
	lookupSquare(e intStep, n intStep) *[200][200]elevation16
	maxElevation(e intStep, n intStep) float64
	elevation(easting intStep, northing intStep) float64
	offsets() (float64, float64)
}

type SmallSquare interface {
	elevation(easting intStep, northing intStep) elevation16
}
// type smallSquare *[smallSquareSize][smallSquareSize]elevation16

const (
	unit              = 10
	bigSquareSize     = 5000
	smallSquareSize   = 200
	maxBlaneDistance  = 200_000.0
	earthRadius       = 6_371_000.0
	bottomHeightAngle = -0.08
	totalHeightAngle  = 0.16
)

var atanPrecalc [10 + maxBlaneDistance/unit]float64

func init() {
	for i := 0; i < len(atanPrecalc); i++ {
		atanPrecalc[i] = math.Atan2(float64(unit*i)/2, earthRadius)
	}
}

type Geopixel struct {
	Distance float64
	Incline  float64
}

type Transform struct {
	Easting     float64
	Northing    float64
	ElevMap     ElevationMapInterface
	GeopixelLen int
}

type intStepper struct {
	start   intStep
	stepLen intStep
}

func (s intStepper) step(i intStep) intStep {
	return s.start + i*s.stepLen
}

type floatStepper struct {
	start   float64
	stepLen float64
}

func (s floatStepper) step(i intStep) float64 {
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

func sign(i float64) intStep {
	if i < 0 {
		return -1
	} else {
		return 1
	}
}

func (bld *geoPixelBuilder) elevationLimit(i intStep) float64 {
	dist := float64(i) * bld.stepLength
	earthCurvatureAngle := atanPrecalc[int(dist/unit)]
	elevationLimit1 := dist * math.Tan(bld.currHeightAngle+earthCurvatureAngle)
	elevationLimit2 := float64(i+smallSquareSize) * bld.stepLength * math.Tan(bld.currHeightAngle+earthCurvatureAngle)
	return math.Min(elevationLimit1, elevationLimit2)
}

func weightElevation(elevation1 elevation16, elevation2 elevation16, elevation1Weight float64) float64 {
	return (float64(elevation2)*elevation1Weight +
		float64(elevation1)*(1-elevation1Weight)) * elevation16Unit
}

func (bld *geoPixelBuilder) updateState(elevation float64, i intStep) {

	dist := float64(i) * bld.stepLength
	earthCurvatureAngle := atanPrecalc[int(dist/unit)]

	heightAngle := math.Atan2(elevation, dist)

	for bld.currHeightAngle+earthCurvatureAngle <= heightAngle {
		bld.geopixels = append(bld.geopixels, Geopixel{
			Distance: dist,
			Incline:  (elevation - bld.prevElevation) * unit / bld.stepLength,
		})
		bld.currHeightAngle = float64(len(bld.geopixels))*totalHeightAngle/float64(bld.geopixelLen) + bottomHeightAngle
	}
	bld.prevElevation = elevation
}

func atBorder(a intStep, b intStep) bool {
	if a > b {
		return a-b > 1 || b < 0
	}

	return b-a > 1 || a < 0
}

type smallSquareIter struct {
	front intStep
	side  intStep
	side2 intStep
}

func (i *smallSquareIter) init(front intStep, side intStep) {
	i.front = front % smallSquareSize
	i.side = side % smallSquareSize
	i.side2 = (side + 1) % smallSquareSize
}

// intStep is used for indices of squares. It is a separate type to make it easy to distinguish it from
// easting/northing. intStep values must be multiplied by 10 to get easting/northing
type intStep int

func (bld *geoPixelBuilder) traceEastWest(elevationMap ElevationMapInterface, eastStepper intStepper, northStepper floatStepper) {
	totalSteps := intStep(maxBlaneDistance / bld.stepLength)
	elevation0 := bld.prevElevation + 9
	prevIter := smallSquareIter{
		front: 10000,
		side:  10000,
		side2: 10000,
	}
	var sIter smallSquareIter
	var sq0 = elevationMap.lookupSquare(eastStepper.start, intStep(math.Floor(northStepper.start)))
	var sq1 = elevationMap.lookupSquare(eastStepper.start, intStep(math.Floor(northStepper.start))+1)
	for i := intStep(1); i < totalSteps; i++ {
		eastStep := eastStepper.step(i)
		northFloat := northStepper.step(i)
		northStep := intStep(math.Floor(northFloat))

		sIter.init(eastStep, northStep)

		if atBorder(prevIter.front, sIter.front) {
			elevationLimit := elevation0 + bld.elevationLimit(i)

			if elevationMap.maxElevation(eastStep, northStep) < elevationLimit &&
				elevationMap.maxElevation(eastStep, northStep+intStep(northStepper.stepLen*smallSquareSize)) < elevationLimit {
				i += (smallSquareSize - 1)
				eastingIndex := eastStepper.step(i)
				northFloat = northStepper.step(i)
				northStep = intStep(math.Floor(northFloat))
				prevIter.init(eastingIndex, northStep)
				continue
			}
			sq0 = elevationMap.lookupSquare(eastStep, northStep)
			if sq0 == nil {
				break
			}
			if sIter.side2 == 0 {
				sq1 = elevationMap.lookupSquare(eastStep, northStep+1)
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
					sq0 = elevationMap.lookupSquare(eastStep, northStep)
					if sq0 == nil {
						break
					}
				}
			}

			if atBorder(prevIter.side2, sIter.side2) {
				if sIter.side2 == 0 {
					sq1 = elevationMap.lookupSquare(eastStep, northStep+1)
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

func (bld *geoPixelBuilder) traceNorthSouth(elevationMap ElevationMapInterface, eastStepper floatStepper, northStepper intStepper) {
	totalSteps := intStep(maxBlaneDistance / bld.stepLength)
	elevation0 := bld.prevElevation + 9
	prevIter := smallSquareIter{
		front: 10000,
		side:  10000,
		side2: 10000,
	}
	var sIter smallSquareIter
	var sq0 = elevationMap.lookupSquare(intStep(math.Floor(eastStepper.start)), northStepper.start)
	var sq1 = elevationMap.lookupSquare(intStep(math.Floor(eastStepper.start))+1, northStepper.start)
	for i := intStep(1); i < totalSteps; i++ {
		northStep := northStepper.step(i)
		eastFloat := eastStepper.step(i)
		eastStep := intStep(math.Floor(eastFloat))

		sIter.init(northStep, eastStep)

		if atBorder(prevIter.front, sIter.front) {
			elevationLimit := elevation0 + bld.elevationLimit(i)

			if elevationMap.maxElevation(eastStep, northStep) < elevationLimit &&
				elevationMap.maxElevation(eastStep+intStep(eastStepper.stepLen*smallSquareSize), northStep) < elevationLimit { //?
				i += (smallSquareSize - 1)
				northStep = northStepper.step(i)
				eastFloat = eastStepper.step(i)
				eastStep = intStep(math.Floor(eastFloat))
				prevIter.init(northStep, eastStep)
				continue
			}
			sq0 = elevationMap.lookupSquare(eastStep, northStep)
			if sq0 == nil {
				break
			}
			if sIter.side2 == 0 {
				sq1 = elevationMap.lookupSquare(eastStep+1, northStep)
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
					sq0 = elevationMap.lookupSquare(eastStep, northStep)
					if sq0 == nil {
						break
					}
				}
			}

			if atBorder(prevIter.side2, sIter.side2) {
				if sIter.side2 == 0 {
					sq1 = elevationMap.lookupSquare(eastStep+1, northStep)
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
	northing0 := math.Round(t.Northing/unit) * unit
	easting0 := math.Round(t.Easting/unit) * unit

	minEasting, maxNorthing := t.ElevMap.offsets()
	var eastingStart = intStep(easting0-minEasting) / unit
	var northingStart = intStep(maxNorthing-northing0) / unit

	bld := geoPixelBuilder{
		geopixels:       make([]Geopixel, 1000),
		currHeightAngle: bottomHeightAngle,
		prevElevation:   t.ElevMap.elevation(eastingStart, northingStart),
		geopixelLen:     t.GeopixelLen,
	}

	sin := math.Sin(rad) // east
	cos := math.Cos(rad) // north

	if math.Abs(sin) > math.Abs(cos) {
		bld.stepLength = unit / math.Abs(sin)
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
		bld.stepLength = unit / math.Abs(cos)
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
