package elevationmap

import (
	"fmt"
	"math"
)

type ElevationMap struct {
	minEasting float64
	maxNorthing float64
	mmapStructs [50][50]*Mmapstruct
}

func LoadFiles(fNames []string) (ElevationMap, error) {
	mmapStructs := []*Mmapstruct{}
	allElevations := ElevationMap{
		minEasting:  math.MaxFloat64,
		maxNorthing: - math.MaxFloat64,
	}

	for _, fName := range fNames {
		mmapStruct, err := LoadAsMmap(fName)
		if err != nil {
			fmt.Printf("%s: %v\n", fName, err)
			continue
		}
		allElevations.maxNorthing = math.Max(allElevations.maxNorthing, mmapStruct.NorthingMax)
		allElevations.minEasting = math.Min(allElevations.minEasting, mmapStruct.EastingMin)

		mmapStructs = append(mmapStructs, mmapStruct)
	}
	for _, mmapStruct := range mmapStructs {
		x := (int(mmapStruct.EastingMin) - int(allElevations.minEasting)) / 50000
		y := (int(allElevations.maxNorthing) - int(mmapStruct.NorthingMax)) / 50000
		allElevations.mmapStructs[x][y] = mmapStruct
	}
	return allElevations, nil
}