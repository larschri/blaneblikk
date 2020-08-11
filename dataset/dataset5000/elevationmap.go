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

func (em ElevationMap) maxElevation(e intStep, n intStep) float64 {
	if e < 0 || n < 0 {
		return -1
	}
	n0 := arrayIndices(n)
	e0 := arrayIndices(e)

	mmapStruct := em.lookupMmapStruct(e0.i3, n0.i3)
	if mmapStruct == nil {
		return -1
	}
	return float64(mmapStruct.MaxElevations[n0.i2][e0.i2]) / unit
}

func (em ElevationMap) lookupSquare(e intStep, n intStep) *[smallSquareSize][smallSquareSize]elevation16 {
	if e < 0 || n < 0 {
		return nil
	}

	mmapStruct := em.lookupMmapStruct(int(e / bigSquareSize), int(n / bigSquareSize))
	if mmapStruct == nil {
		return nil
	}
	return &mmapStruct.Elevations[int((n/smallSquareSize) % numberOfSmallSquares)][int((e/smallSquareSize) % numberOfSmallSquares)]
}

func (em ElevationMap) GetElevation(easting float64, northing float64, limit float64) float64 {
	easting2 := (easting - em.minEasting) / unit
	northing2 := (em.maxNorthing - northing) / unit
	erest := intStep(math.Floor(easting2))
	nrest := intStep(math.Floor(northing2))

	if erest < 0 || nrest < 0 {
		return -1
	}

	n0 := arrayIndices(nrest)
	e0 := arrayIndices(erest)

	mmapStruct := em.lookupMmapStruct(e0.i3, n0.i3)
	if mmapStruct == nil || float64(mmapStruct.MaxElevations[n0.i2][e0.i2])/unit < limit {
		return -1
	}

	n1 := arrayIndices(nrest + 1)
	e1 := arrayIndices(erest + 1)

	// Optimisation, assume the same mmapStruct for all corners
	l00 := mmapStruct.Elevations[n0.i2][e0.i2][n0.i1][e0.i1]
	l01 := mmapStruct.Elevations[n1.i2][e0.i2][n1.i1][e0.i1]
	l10 := mmapStruct.Elevations[n0.i2][e1.i2][n0.i1][e1.i1]
	l11 := mmapStruct.Elevations[n1.i2][e1.i2][n1.i1][e1.i1]

	if nrest/bigSquareSize != (nrest+1)/bigSquareSize || erest/bigSquareSize != (erest+1)/bigSquareSize {
		l00 = em.lookup(e0, n0)
		l01 = em.lookup(e0, n1)
		l10 = em.lookup(e1, n0)
		l11 = em.lookup(e1, n1)
	}

	if l00 == -1 || l01 == -1 || l10 == -1 || l11 == -1 {
		return -1
	}

	er := easting2 - float64(erest)
	nr := northing2 - float64(nrest)

	return (float64(l11)*er*nr +
		float64(l10)*er*(1-nr) +
		float64(l01)*(1-er)*nr +
		float64(l00)*(1-er)*(1-nr)) / unit
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
