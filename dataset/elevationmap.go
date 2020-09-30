// Package dataset implements access to elevation data stored in files.
//
// Elevation data is stored in files that are mapped into memory using mmap for fast access.
package dataset

import (
	"fmt"
	"math"
)

const (
	// Unit is the number of meters between each elevation point in the grid of elevation points
	Unit = 10

	// ElevationMapletSize is the dimension of an ElevationMaplet
	ElevationMapletSize      = 200
	bigSquareSize            = 5000
	numberOfElevationMaplets = bigSquareSize / ElevationMapletSize
)

// IntStep is used for indices of squares. It is a separate type to make it easy to distinguish it from
// easting/northing. IntStep values must be multiplied by 10 to get easting/northing
type IntStep int

// ElevationMap provides access to all elevation data
type ElevationMap struct {
	minEasting  float64
	maxNorthing float64
	mmapStructs [50][50]*mmap5000
}

// ElevationMaplet is a small piece of the ElevationMap that fits in memory.
// The contents is loaded from a memory mapped file
type ElevationMaplet [ElevationMapletSize][ElevationMapletSize]Elevation16

// Offsets returns minimum easting and maximum northing
func (em *ElevationMap) Offsets() (float64, float64) {
	return em.minEasting, em.maxNorthing
}

func (em *ElevationMap) lookupMmapStruct(e int, n int) *mmap5000 {
	if e < 0 || e >= 50 || n < 0 || n >= 50 {
		return nil
	}

	return em.mmapStructs[e][n]
}

func index2(x IntStep) int {
	return int((x / ElevationMapletSize) % numberOfElevationMaplets)
}

// MaxElevation returns the maximum elevation in the small
func (em *ElevationMap) MaxElevation(e IntStep, n IntStep) float64 {
	if e < 0 || n < 0 {
		return -1
	}

	mmapStruct := em.lookupMmapStruct(int(e/bigSquareSize), int(n/bigSquareSize))
	if mmapStruct == nil {
		return -1
	}

	return float64(mmapStruct.MaxElevations[index2(n)][index2(e)]) / Unit
}

func (em *ElevationMap) LookupElevationMaplet(e IntStep, n IntStep) *ElevationMaplet {
	if e < 0 || n < 0 {
		return nil
	}

	mmapStruct := em.lookupMmapStruct(int(e/bigSquareSize), int(n/bigSquareSize))
	if mmapStruct == nil {
		return nil
	}

	return (*ElevationMaplet)(&mmapStruct.Elevations[index2(n)][index2(e)])
}

func (em *ElevationMap) Elevation(easting IntStep, northing IntStep) float64 {
	mmapStruct := em.lookupMmapStruct(int(easting/bigSquareSize), int(northing/bigSquareSize))
	if mmapStruct == nil {
		return -1
	}
	return float64(mmapStruct.Elevations[index2(northing)][index2(easting)][northing%ElevationMapletSize][easting%ElevationMapletSize]) * Elevation16Unit
}

func LoadFiles(datasetReader DatasetReader, mmapFileDir string, fNames []string) (ElevationMap, error) {
	mmapStructs := []*mmap5000{}
	allElevations := ElevationMap{
		minEasting:  math.MaxFloat64,
		maxNorthing: -math.MaxFloat64,
	}

	for _, fName := range fNames {
		mmapStruct, err := loadAsMmap(datasetReader, mmapFileDir, fName)
		if err != nil {
			fmt.Printf("%s: %v\n", fName, err)
			continue
		}
		allElevations.maxNorthing = math.Max(allElevations.maxNorthing, mmapStruct.NorthingMax)
		allElevations.minEasting = math.Min(allElevations.minEasting, mmapStruct.EastingMin)

		mmapStructs = append(mmapStructs, mmapStruct)
	}
	for _, mmapStruct := range mmapStructs {
		x := (int(mmapStruct.EastingMin) - int(allElevations.minEasting)) / (bigSquareSize * Unit)
		y := (int(allElevations.maxNorthing) - int(mmapStruct.NorthingMax)) / (bigSquareSize * Unit)
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
