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

type indices struct {
	i1 intStep
	i2 int
	i3 int
}

func arrayIndices(num intStep) indices {
	return indices{
		i1: num % smallSquareSize,
		i2: int((num / smallSquareSize) % (bigSquareSize / smallSquareSize)),
		i3: int(num / bigSquareSize),
	}
}

func (em ElevationMap) lookupMmapStruct(e int, n int) *Mmap5000 {
	if e < 0 || e >= 50 || n < 0 || n >= 50 {
		return nil
	}

	return em.mmapStructs[e][n]
}

func (em ElevationMap) lookup(e indices, n indices) elevation16 {
	if e.i3 < 0 || e.i3 >= 50 || n.i3 < 0 || n.i3 >= 50 {
		return -1
	}

	ms := em.mmapStructs[e.i3][n.i3]
	if ms == nil {
		return -1
	}

	return ms.Elevations[n.i2][e.i2][n.i1][e.i1]
}

func index2(x intStep) int {
	return int((x / smallSquareSize) % numberOfSmallSquares)
}

func (em ElevationMap) maxElevation(e intStep, n intStep) float64 {
	if e < 0 || n < 0 {
		return -1
	}

	mmapStruct := em.lookupMmapStruct(int(e / bigSquareSize), int(n / bigSquareSize))
	if mmapStruct == nil {
		return -1
	}

	return float64(mmapStruct.MaxElevations[index2(n)][index2(e)]) / unit
}

func (em ElevationMap) lookupSquare(e intStep, n intStep) *[smallSquareSize][smallSquareSize]elevation16 {
	if e < 0 || n < 0 {
		return nil
	}

	mmapStruct := em.lookupMmapStruct(int(e / bigSquareSize), int(n / bigSquareSize))
	if mmapStruct == nil {
		return nil
	}

	return &mmapStruct.Elevations[index2(n)][index2(e)]
}

func (em ElevationMap) elevation(easting intStep, northing intStep) float64 {
	easting2 := (easting - intStep(em.minEasting)) / unit
	northing2 := (intStep(em.maxNorthing) - northing) / unit

	if easting2 < 0 || northing2 < 0 {
		return -1
	}

	n0 := arrayIndices(northing2)
	e0 := arrayIndices(easting2)

	mmapStruct := em.lookupMmapStruct(e0.i3, n0.i3)
	if mmapStruct == nil {
		return -1
	}
	return float64(mmapStruct.Elevations[n0.i2][e0.i2][n0.i1][e0.i1]) * elevation16Unit
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
