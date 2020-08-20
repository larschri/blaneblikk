package dataset5000

import (
	"fmt"
	"math"
)

type ElevationMap struct {
	minEasting  float64
	maxNorthing float64
	mmapStructs [50][50]*Mmap5000
}

func (em ElevationMap) offsets() (float64, float64) {
	return em.minEasting, em.maxNorthing
}

func (em ElevationMap) lookupMmapStruct(e int, n int) *Mmap5000 {
	if e < 0 || e >= 50 || n < 0 || n >= 50 {
		return nil
	}

	return em.mmapStructs[e][n]
}

func index2(x intStep) int {
	return int((x / smallSquareSize) % numberOfSmallSquares)
}

func (em ElevationMap) maxElevation(e intStep, n intStep) float64 {
	if e < 0 || n < 0 {
		return -1
	}

	mmapStruct := em.lookupMmapStruct(int(e/bigSquareSize), int(n/bigSquareSize))
	if mmapStruct == nil {
		return -1
	}

	return float64(mmapStruct.MaxElevations[index2(n)][index2(e)]) / unit
}

func (em ElevationMap) lookupSquare(e intStep, n intStep) SmallSquare {
	if e < 0 || n < 0 {
		return nil
	}

	mmapStruct := em.lookupMmapStruct(int(e/bigSquareSize), int(n/bigSquareSize))
	if mmapStruct == nil {
		return nil
	}

	return (*square)(&mmapStruct.Elevations[index2(n)][index2(e)])
}

type square [smallSquareSize][smallSquareSize]elevation16

func (sq *square) elevation(easting intStep, northing intStep) elevation16 {
	return sq[northing][easting]
}

func (em ElevationMap) elevation(easting intStep, northing intStep) float64 {
	mmapStruct := em.lookupMmapStruct(int(easting / bigSquareSize), int(northing / bigSquareSize))
	if mmapStruct == nil {
		return -1
	}
	return float64(mmapStruct.Elevations[index2(northing)][index2(easting)][northing % smallSquareSize][easting % smallSquareSize]) * elevation16Unit
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
		x := (int(mmapStruct.EastingMin) - int(allElevations.minEasting)) / (bigSquareSize * unit)
		y := (int(allElevations.maxNorthing) - int(mmapStruct.NorthingMax)) / (bigSquareSize * unit)
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
