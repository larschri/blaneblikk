package main

import (
	"fmt"
	"github.com/larschri/gdal002/elevationmap"
	"path/filepath"
)

func main() {
	mapstruct, err := elevationmap.LoadAsMmap("dem-files/6603_1_10m_z32.dem")
	fmt.Println(err)
	fmt.Println(mapstruct.NorthingMax)
	files, err := filepath.Glob("dem-files/[^.]*.dem")
	if err != nil {
		panic(err)
	}
	elevmap, err := elevationmap.LoadFiles(files)
	if err != nil {
		panic(err)
	}
	fmt.Print(elevmap)
}
