package dataset5000

import (
	"math"
)

const (
	unit             = 10
	bigSquareSize    = 5000
	smallSquareSize  = 200
	maxBlaneDistance = 200_000.0
	earthRadius      = 6_371_000.0
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
	ElevMap     ElevationMap
	GeopixelLen int
}

const step float64 = 10
const bottomHeightAngle float64 = -0.08
const totalHeightAngle float64 = 0.16

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
	earthCurvatureAngle := atanPrecalc[int(dist/step)]
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
	earthCurvatureAngle := atanPrecalc[int(dist/step)]

	heightAngle := math.Atan2(elevation, dist)

	for bld.currHeightAngle+earthCurvatureAngle <= heightAngle {
		bld.geopixels = append(bld.geopixels, Geopixel{
			Distance: dist,
			Incline:  (elevation - bld.prevElevation) * step / bld.stepLength,
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

func (bld *geoPixelBuilder) traceEastWest(elevationMap ElevationMap, eastStepper intStepper, northStepper floatStepper) {
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
		eastingIndex := eastStepper.step(i)
		northingIndex := northStepper.step(i)
		nrest := intStep(math.Floor(northingIndex))

		sIter.init(eastingIndex, nrest)

		if atBorder(prevIter.front, sIter.front) {
			elevationLimit := elevation0 + bld.elevationLimit(i)

			if elevationMap.maxElevation(eastingIndex, nrest) < elevationLimit &&
				elevationMap.maxElevation(eastingIndex, nrest+intStep(northStepper.stepLen*smallSquareSize)) < elevationLimit {
				i += (smallSquareSize - 1)
				eastingIndex := eastStepper.step(i)
				northingIndex := northStepper.step(i)
				nrest = intStep(math.Floor(northingIndex))
				prevIter.init(eastingIndex, nrest)
				continue
			}
			sq0 = elevationMap.lookupSquare(eastingIndex, nrest)
			if sq0 == nil {
				break
			}
			if sIter.side2 == 0 {
				sq1 = elevationMap.lookupSquare(eastingIndex, nrest+1)
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
					sq0 = elevationMap.lookupSquare(eastingIndex, nrest)
					if sq0 == nil {
						break
					}
				}
			}

			if atBorder(prevIter.side2, sIter.side2) {
				if sIter.side2 == 0 {
					sq1 = elevationMap.lookupSquare(eastingIndex, nrest+1)
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
			northingIndex-float64(nrest))
		bld.updateState(elevation-elevation0, i)

		prevIter = sIter
	}
}

func (bld *geoPixelBuilder) traceNorthSouth(elevationMap ElevationMap, eastStepper floatStepper, northStepper intStepper) {
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
		northingIndex := northStepper.step(i)
		eastingIndex := eastStepper.step(i)
		erest := intStep(math.Floor(eastingIndex))

		sIter.init(northingIndex, erest)

		if atBorder(prevIter.front, sIter.front) {
			elevationLimit := elevation0 + bld.elevationLimit(i)

			if elevationMap.maxElevation(erest, northingIndex) < elevationLimit &&
				elevationMap.maxElevation(erest+intStep(eastStepper.stepLen*smallSquareSize), northingIndex) < elevationLimit { //?
				i += (smallSquareSize - 1)
				northingIndex = northStepper.step(i)
				eastingIndex = eastStepper.step(i)
				erest = intStep(math.Floor(eastingIndex))
				prevIter.init(northingIndex, erest)
				continue
			}
			sq0 = elevationMap.lookupSquare(erest, northingIndex)
			if sq0 == nil {
				break
			}
			if sIter.side2 == 0 {
				sq1 = elevationMap.lookupSquare(erest+1, northingIndex)
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
					sq0 = elevationMap.lookupSquare(erest, northingIndex)
					if sq0 == nil {
						break
					}
				}
			}

			if atBorder(prevIter.side2, sIter.side2) {
				if sIter.side2 == 0 {
					sq1 = elevationMap.lookupSquare(erest+1, northingIndex)
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
			eastingIndex-float64(erest))

		bld.updateState(elevation-elevation0, i)
		prevIter = sIter
	}
}

func (t Transform) TraceDirection(rad float64) []Geopixel {
	northing0 := math.Round(t.Northing/unit) * unit
	easting0 := math.Round(t.Easting/unit) * unit

	var eastingStart = intStep(easting0-t.ElevMap.minEasting) / unit
	var northingStart = intStep(t.ElevMap.maxNorthing-northing0) / unit

	bld := geoPixelBuilder{
		geopixels:       make([]Geopixel, 0),
		currHeightAngle: bottomHeightAngle,
		prevElevation:   t.ElevMap.elevation(intStep(easting0), intStep(northing0)),
		geopixelLen:     t.GeopixelLen,
	}

	sin := math.Sin(rad) // east
	cos := math.Cos(rad) // north

	if math.Abs(sin) > math.Abs(cos) {
		bld.stepLength = step / math.Abs(sin)
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
		bld.stepLength = step / math.Abs(cos)
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
