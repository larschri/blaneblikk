package dataset5000

import (
	"math"
)

const (
	unit = 10
	bigSquareSize = 5000
	smallSquareSize = 25
)

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
	step             float64

	ElevMap      ElevationMap
	eastStep     float64
	northStep    float64
	easting      float64
	northing     float64

	//
	geopixels       []Geopixel
	currHeightAngle float64
	prevElevation   float64
	elevation0      float64
	geopixelLen     int
}

func sign(i float64) int {
	if i < 0 {
		return -1
	} else {
		return 1
	}
}

func (iter *squareIterator) init(rad float64, northing int, easting int, e ElevationMap) {
	iter.northing = float64(northing / 10) * 10
	iter.easting = float64(easting / 10) * 10
	iter.ElevMap = e
	sin := math.Sin(rad) // east
	cos := math.Cos(rad) // north
	if math.Abs(sin) > math.Abs(cos) {
		iter.eastStep = float64(sign(sin))
		iter.northStep = cos / math.Abs(sin)
		iter.step = float64(1) / math.Abs(sin)
	} else {
		iter.eastStep = sin / math.Abs(cos)
		iter.northStep = float64(sign(cos))
		iter.step = float64(1) / math.Abs(cos)
	}
}

func (sq *squareIterator) updateState(elevation float64, i int) {
	dist := float64(i) * sq.step
	earthCurvatureAngle := math.Atan2(dist/2, 6371000.0)

	heightAngle := math.Atan2(elevation - sq.elevation0, dist)

	for sq.currHeightAngle + earthCurvatureAngle <= heightAngle {
		sq.geopixels = append(sq.geopixels, Geopixel{
			Distance: dist,
			Incline:  (elevation - sq.prevElevation) / sq.step,
		})
		sq.currHeightAngle = float64(len(sq.geopixels)) * totalHeightAngle / float64(sq.geopixelLen) + bottomHeightAngle
	}
	sq.prevElevation = elevation
}

func (sq *squareIterator) TraceEastWest() {
	steps := int(2000000.0 / sq.step)
	emodPrev := uint32(10000)
	nmodPrev := uint32(10000)
	nmod2Prev := uint32(10000)
	var sq0 *[25][25]int16
	var sq1 *[25][25]int16
	for i := int(step); i < steps; i = i + int(step) {
		easting2 := (int(sq.easting)+i*int(sq.eastStep) - int(sq.ElevMap.minEasting)) / 10
		northing2 := (sq.ElevMap.maxNorthing - (sq.northing+float64(i)*sq.northStep)) / 10
		nrest := int(math.Floor(northing2))

		emod := uint32(easting2) % smallSquareSize
		nmod := uint32(nrest) % smallSquareSize

		if math.Abs(float64(emodPrev - emod)) > 1.0 || math.Abs(float64(nmodPrev - nmod)) > 1.0 {
			sq0 = sq.ElevMap.lookupSquare(easting2, nrest)
			if sq0 == nil {
				break
			}
		}

		nmod2 := uint32(uint32(nrest + 1) % smallSquareSize)
		if math.Abs(float64(emodPrev - emod)) > 1.0 || math.Abs(float64(nmod2Prev - nmod2)) > 1.0 {
			sq1 = sq.ElevMap.lookupSquare(easting2, nrest + 1)
			if sq1 == nil {
				break
			}
		}

		l00 := sq0[nmod][emod]
		l01 := sq1[nmod2][emod]

		nr := northing2 - float64(nrest)
		elev2 := (float64(l01) * nr +
			float64(l00) * (1 - nr)) / 10

		sq.updateState(elev2, i)
		emodPrev = emod
		nmodPrev = nmod
		nmod2Prev = nmod2
	}
}

func (t Transform) TraceDirectionExperimental(rad float64, elevation0 float64) []Geopixel {
	var sq squareIterator
	sq.init(rad, int(t.Northing), int(t.Easting), t.ElevMap)
	sq.geopixels = make([]Geopixel, 0)
	sq.currHeightAngle = bottomHeightAngle
	sq.prevElevation = elevation0
	sq.elevation0 = elevation0
	sq.geopixelLen = t.GeopixelLen

	steps := int(2000000.0 / sq.step)
	if math.Abs(sq.eastStep) == 1 {
		sq.TraceEastWest()
	} else {
		for i := int(step); i < steps; i = i + int(step) {
			elevation := sq.ElevMap.GetElevationNorth(sq.easting+float64(i)*sq.eastStep, int(sq.northing)+i*int(sq.northStep))
			sq.updateState(elevation, i)
		}
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

		heightAngle := math.Atan2(elevation - elevation0, dist)

		for currHeightAngle + earthCurvatureAngle <= heightAngle {
			geopixels = append(geopixels, Geopixel{
				Distance: dist,
				Incline:  (elevation - prevElevation),
			})
			currHeightAngle = float64(len(geopixels)) * totalHeightAngle / float64(t.GeopixelLen) + bottomHeightAngle
		}
		prevElevation = elevation
	}
	return geopixels
}

