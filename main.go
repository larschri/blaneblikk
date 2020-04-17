package main

import (
	"fmt"
	"github.com/larschri/gdal002/elevationmap"
)

func main() {
	mapstruct, err := elevationmap.LoadAsMmap("dem-files/6603_1_10m_z32.dem")
	fmt.Println(err)
	fmt.Println(mapstruct.NorthingMax)
}
