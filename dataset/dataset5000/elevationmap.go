package dataset5000

import (
	"fmt"
	"math"
)

const (
	Unit              = 10
	BigSquareSize     = 5000
	SmallSquareSize   = 200
)

// IntStep is used for indices of squares. It is a separate type to make it easy to distinguish it from
// easting/northing. IntStep values must be multiplied by 10 to get easting/northing
type IntStep int

type ElevationMap struct {
	minEasting  float64
	maxNorthing float64
	mmapStructs [50][50]*Mmap5000
}

func (em ElevationMap) Offsets() (float64, float64) {
	return em.minEasting, em.maxNorthing
}

func (em ElevationMap) lookupMmapStruct(e int, n int) *Mmap5000 {
	if e < 0 || e >= 50 || n < 0 || n >= 50 {
		return nil
	}

	return em.mmapStructs[e][n]
}

func index2(x IntStep) int {
	return int((x / SmallSquareSize) % numberOfSmallSquares)
}

func (em ElevationMap) MaxElevation(e IntStep, n IntStep) float64 {
	if e < 0 || n < 0 {
		return -1
	}

	mmapStruct := em.lookupMmapStruct(int(e/BigSquareSize), int(n/BigSquareSize))
	if mmapStruct == nil {
		return -1
	}

	return float64(mmapStruct.MaxElevations[index2(n)][index2(e)]) / Unit
}

func (em ElevationMap) LookupSquare(e IntStep, n IntStep) *[SmallSquareSize][SmallSquareSize]Elevation16 {
	if e < 0 || n < 0 {
		return nil
	}

	mmapStruct := em.lookupMmapStruct(int(e/BigSquareSize), int(n/BigSquareSize))
	if mmapStruct == nil {
		return nil
	}

	return &mmapStruct.Elevations[index2(n)][index2(e)]
}

type square [SmallSquareSize][SmallSquareSize]Elevation16

func (sq *square) elevation(easting IntStep, northing IntStep) Elevation16 {
	return sq[northing][easting]
}

func (em ElevationMap) Elevation(easting IntStep, northing IntStep) float64 {
	mmapStruct := em.lookupMmapStruct(int(easting /BigSquareSize), int(northing /BigSquareSize))
	if mmapStruct == nil {
		return -1
	}
	return float64(mmapStruct.Elevations[index2(northing)][index2(easting)][northing %SmallSquareSize][easting %SmallSquareSize]) * Elevation16Unit
}

func LoadFiles(datasetReader DatasetReader, fNames []string) (ElevationMap, error) {
	mmapStructs := []*Mmap5000{}
	allElevations := ElevationMap{
		minEasting:  math.MaxFloat64,
		maxNorthing: -math.MaxFloat64,
	}

	for _, fName := range fNames {
		mmapStruct, err := LoadAsMmap(datasetReader, fName)
		if err != nil {
			fmt.Printf("%s: %v\n", fName, err)
			continue
		}
		allElevations.maxNorthing = math.Max(allElevations.maxNorthing, mmapStruct.NorthingMax)
		allElevations.minEasting = math.Min(allElevations.minEasting, mmapStruct.EastingMin)

		mmapStructs = append(mmapStructs, mmapStruct)
	}
	for _, mmapStruct := range mmapStructs {
		x := (int(mmapStruct.EastingMin) - int(allElevations.minEasting)) / (BigSquareSize * Unit)
		y := (int(allElevations.maxNorthing) - int(mmapStruct.NorthingMax)) / (BigSquareSize * Unit)
		allElevations.mmapStructs[x][y] = mmapStruct
	}
	for _, xx := range allElevations.mmapStructs {
		for _, yy := range xx {
			if yy != nil {
				fmt.Print("X")
			} else {
				fmt.Print("-")
			}
		}
		fmt.Println()
	}
	return allElevations, nil
}
