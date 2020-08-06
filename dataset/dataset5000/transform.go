package dataset5000

import (
	"math"
)

const (
	unit             = 10
	bigSquareSize    = 5000
	smallSquareSize  = 25
	maxBlaneDistance = 200_000.0
)

var atanPrecalc [10 + maxBlaneDistance/unit]float64

func init() {
	for i := 0; i < len(atanPrecalc); i++ {
		atanPrecalc[i] = math.Atan2(float64(unit*i)/2, 6371000.0)
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

type squareIterator struct {
	stepLength float64

	easting  float64
	northing float64

	//
	geopixels       []Geopixel
	currHeightAngle float64
	prevElevation   float64
	elevation0      float64
	geopixelLen     int
}

func sign(i float64) intStep {
	if i < 0 {
		return -1
	} else {
		return 1
	}
}

func (sq *squareIterator) updateState(elevation float64, i intStep) {
	dist := float64(i) * sq.stepLength
	earthCurvatureAngle := atanPrecalc[int(dist/step)]

	heightAngle := math.Atan2(elevation-sq.elevation0, dist)

	for sq.currHeightAngle+earthCurvatureAngle <= heightAngle {
		sq.geopixels = append(sq.geopixels, Geopixel{
			Distance: dist,
			Incline:  (elevation - sq.prevElevation) * step / sq.stepLength,
		})
		sq.currHeightAngle = float64(len(sq.geopixels))*totalHeightAngle/float64(sq.geopixelLen) + bottomHeightAngle
	}
	sq.prevElevation = elevation
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

func (sq *squareIterator) TraceEastWest(elevationMap ElevationMap, eastStepSign intStep, northStepLength float64) {
	totalSteps := intStep(maxBlaneDistance / sq.stepLength)
	prevIter := smallSquareIter{
		front: 10000,
		side:  10000,
		side2: 10000,
	}
	var sIter smallSquareIter
	var eastingStart = intStep(sq.easting-elevationMap.minEasting) / 10
	var northingStart = (elevationMap.maxNorthing - sq.northing) / 10
	var sq0 = elevationMap.lookupSquare(eastingStart, intStep(math.Floor(northingStart)))
	var sq1 = elevationMap.lookupSquare(eastingStart, intStep(math.Floor(northingStart))+1)
	for i := intStep(1); i < totalSteps; i++ {
		eastingIndex := eastingStart + i*eastStepSign
		northingIndex := northingStart - float64(i)*northStepLength
		nrest := intStep(math.Floor(northingIndex))

		sIter.init(eastingIndex, nrest)

		if atBorder(prevIter.front, sIter.front) {
			dist := float64(i) * sq.stepLength
			earthCurvatureAngle := atanPrecalc[int(dist/step)]
			elevationLimit1 := sq.elevation0 + dist*math.Tan(sq.currHeightAngle+earthCurvatureAngle)
			elevationLimit2 := sq.elevation0 + float64(i+smallSquareSize)*sq.stepLength*math.Tan(sq.currHeightAngle+earthCurvatureAngle)
			elevationLimit := math.Min(elevationLimit1, elevationLimit2)

			if elevationMap.maxElevation(eastingIndex, nrest) < elevationLimit &&
				elevationMap.maxElevation(eastingIndex, nrest-intStep(northStepLength*smallSquareSize)) < elevationLimit {
				i += (smallSquareSize - 1)
				eastingIndex = eastingStart + i*eastStepSign
				northingIndex = northingStart - float64(i)*northStepLength
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

		l00 := sq0[sIter.side][sIter.front]
		l01 := sq1[sIter.side2][sIter.front]

		nr := northingIndex - float64(nrest)
		elev2 := (float64(l01)*nr +
			float64(l00)*(1-nr)) / step

		sq.updateState(elev2, i)
		prevIter = sIter
	}
}

func (sq *squareIterator) TraceNorthSouth(elevationMap ElevationMap, eastStepLength float64, northStepSign intStep) {
	totalSteps := intStep(maxBlaneDistance / sq.stepLength)
	prevIter := smallSquareIter{
		front: 10000,
		side:  10000,
		side2: 10000,
	}
	var sIter smallSquareIter
	var eastingStart = (sq.easting - elevationMap.minEasting) / 10
	var northingStart = intStep(elevationMap.maxNorthing-sq.northing) / 10
	var sq0 = elevationMap.lookupSquare(intStep(math.Floor(eastingStart)), northingStart)
	var sq1 = elevationMap.lookupSquare(intStep(math.Floor(eastingStart))+1, northingStart)
	for i := intStep(1); i < totalSteps; i++ {
		northingIndex := northingStart - i*northStepSign
		eastingIndex := eastingStart + float64(i)*eastStepLength
		erest := intStep(math.Floor(eastingIndex))

		sIter.init(northingIndex, erest)

		if atBorder(prevIter.front, sIter.front) {
			dist := float64(i) * sq.stepLength
			earthCurvatureAngle := atanPrecalc[int(dist/step)]
			elevationLimit1 := sq.elevation0 + dist*math.Tan(sq.currHeightAngle+earthCurvatureAngle)
			elevationLimit2 := sq.elevation0 + float64(i+smallSquareSize)*sq.stepLength*math.Tan(sq.currHeightAngle+earthCurvatureAngle)
			elevationLimit := math.Min(elevationLimit1, elevationLimit2)

			if elevationMap.maxElevation(erest, northingIndex) < elevationLimit &&
				elevationMap.maxElevation(erest+intStep(eastStepLength*smallSquareSize), northingIndex) < elevationLimit { //?
				i += (smallSquareSize - 1)
				northingIndex = northingStart - i*northStepSign
				eastingIndex = eastingStart + float64(i)*eastStepLength
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

		l00 := sq0[sIter.front][sIter.side]
		l01 := sq1[sIter.front][sIter.side2]

		nr := eastingIndex - float64(erest)
		elev2 := (float64(l01)*nr +
			float64(l00)*(1-nr)) / step

		sq.updateState(elev2, i)
		prevIter = sIter
	}
}

func (t Transform) TraceDirectionExperimental(rad float64, elevation0 float64) []Geopixel {
	var sq squareIterator
	sq.northing = float64(int(t.Northing)/10) * 10
	sq.easting = float64(int(t.Easting)/10) * 10
	sq.geopixels = make([]Geopixel, 0)
	sq.currHeightAngle = bottomHeightAngle
	sq.prevElevation = elevation0
	sq.elevation0 = elevation0
	sq.geopixelLen = t.GeopixelLen

	sin := math.Sin(rad) // east
	cos := math.Cos(rad) // north

	if math.Abs(sin) > math.Abs(cos) {
		sq.stepLength = step / math.Abs(sin)
		sq.TraceEastWest(t.ElevMap, sign(sin), cos/math.Abs(sin))
	} else {
		sq.stepLength = step / math.Abs(cos)
		sq.TraceNorthSouth(t.ElevMap, sin/math.Abs(cos), sign(cos))
	}

	return sq.geopixels
}

func (t Transform) TraceDirectionPlain(rad float64, elevation0 float64) []Geopixel {
	geopixels := make([]Geopixel, 0)
	sin := math.Sin(rad)
	cos := math.Cos(rad)
	currHeightAngle := bottomHeightAngle
	prevElevation := elevation0
	for dist := step; dist < 200000; dist = dist + step {
		earthCurvatureAngle := math.Atan2(dist/2, 6371000.0)
		elevation := t.ElevMap.GetElevation(t.Easting+sin*dist, t.Northing+cos*dist, 0)

		heightAngle := math.Atan2(elevation-elevation0, dist)

		for currHeightAngle+earthCurvatureAngle <= heightAngle {
			geopixels = append(geopixels, Geopixel{
				Distance: dist,
				Incline:  (elevation - prevElevation),
			})
			currHeightAngle = float64(len(geopixels))*totalHeightAngle/float64(t.GeopixelLen) + bottomHeightAngle
		}
		prevElevation = elevation
	}
	return geopixels
}
