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

type loopValues struct {
	earthCurvatureAngle float64
	elevation float64
}

type squareIterator struct {
	sideStep         float64
	step             float64
	front0           int // modulo smallSquareSize
	side0            int // modulo smallSquareSize
	smallSquareFront int
	smallSquareSide  int

	nextSideJump int
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

func (iter *squareIterator) updateNextSideJump() {
	deltaY := (smallSquareSize - iter.side0) + smallSquareSize * iter.smallSquareSide
	iter.nextSideJump = int(math.Ceil(float64(deltaY) / iter.sideStep))
}

func (iter *squareIterator) next() {
	iter.smallSquareFront++
	if (smallSquareSize - iter.front0) + smallSquareSize * iter.smallSquareFront > iter.nextSideJump {
		iter.smallSquareSide++
		iter.updateNextSideJump()
	}
}

func (iter *squareIterator) init2(fronting int, siding int, frontComponent float64, sideComponent float64) {
	iter.sideStep = math.Abs(sideComponent / frontComponent)
	iter.step = float64(1) / math.Abs(frontComponent)
	iter.front0 = fronting % smallSquareSize
	iter.side0 = siding % smallSquareSize
	if frontComponent < 0 {
		iter.front0 = smallSquareSize - iter.front0
	}
	if sideComponent < 0 {
		iter.side0 = smallSquareSize - iter.side0
	}
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
		iter.init2(easting, northing, sin, cos)
	} else {
		iter.eastStep = sin / math.Abs(cos)
		iter.northStep = float64(sign(cos))
		iter.init2(northing, easting, cos, sin)
	}
	iter.updateNextSideJump()
}

func (iter *squareIterator) elevation(step int) float64 {
	if math.Abs(iter.eastStep) == 1 {
		return iter.ElevMap.GetElevationEast(int(iter.easting) + step * int(iter.eastStep), iter.northing + float64(step) * iter.northStep)
	} else {
		return iter.ElevMap.GetElevationNorth(iter.easting + float64(step) * iter.eastStep, int(iter.northing) + step * int(iter.northStep))
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
		for i := int(step); i < steps; i = i + int(step) {
			elevation := sq.ElevMap.GetElevationEast(int(sq.easting)+i*int(sq.eastStep), sq.northing+float64(i)*sq.northStep)
			sq.updateState(elevation, i)
		}
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

