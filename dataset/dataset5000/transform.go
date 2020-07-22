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
	sideStep float64
	front0 int // modulo smallSquareSize
	side0 int // modulo smallSquareSize
	smallSquareFront int
	smallSquareSide int

	nextSideJump int
}

func (iter squareIterator) updateNextSideJump() {
	deltaY := (smallSquareSize - iter.side0) + smallSquareSize * iter.smallSquareSide
	iter.nextSideJump = int(math.Ceil(float64(deltaY) / iter.sideStep))
}

func (iter squareIterator) next() {
	iter.smallSquareFront++
	if (smallSquareSize - iter.front0) + smallSquareSize * iter.smallSquareFront > iter.nextSideJump {
		iter.smallSquareSide++
		iter.updateNextSideJump()
	}
}

func (iter squareIterator) init2(fronting int, siding int, frontComponent float64, sideComponent float64) {
	iter.sideStep = math.Abs(sideComponent / frontComponent)
	iter.front0 = fronting % smallSquareSize
	iter.side0 = siding % smallSquareSize
	if frontComponent < 0 {
		iter.front0 = smallSquareSize - iter.front0
	}
	if sideComponent < 0 {
		iter.side0 = smallSquareSize - iter.side0
	}
}

func (iter squareIterator) init(rad float64, northing int, easting int) {
	sin := math.Sin(rad) // east
	cos := math.Cos(rad) // north
	if math.Abs(sin) > math.Abs(cos) {
		iter.init2(easting, northing, sin, cos)
	} else {
		iter.init2(northing, easting, cos, sin)
	}
	iter.updateNextSideJump()
}

func ClampToUnit(val float64) int {
	ival := int(val)
	return ival - ival % unit
}

func (t Transform) traverseSmallSquare() {

}

func (t Transform) TraceDirection(rad float64, elevation0 float64) []Geopixel {
	var iter squareIterator
	iter.init(rad, ClampToUnit(t.Northing), ClampToUnit(t.Easting))
	for i := 0; i < (smallSquareSize - iter.front0); i++ {

	}
	iter.next()
	for u := 0; u < 1000; u++ {
		for i := 0; i < smallSquareSize; i++ {
		}
		iter.next()
	}
	return []Geopixel{}
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

