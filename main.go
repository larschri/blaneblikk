package main

import (
	"fmt"
	"github.com/larschri/blaner/elevationmap"
	"github.com/larschri/blaner/transform"
	"image"
	"image/png"
	"math"
	"os"
	"path/filepath"
)

type args struct {
	start float64
	width float64
	columns int
	step float64
	easting float64
	northing float64
	heightAngle float64
	minHeight float64
}

var args1 = args{
	start:        3.1,
	width:       .1,
	columns:     400,
	step:        10,
	easting : 463561,
	northing : 6833871,
	//easting:     591307,
	//northing:    6782052,
	heightAngle: .16,
	minHeight:   -.08,
}

func getRGB(b transform.Geopixel) rgb {
	incline := math.Max(0, math.Min(1, b.Incline / 20))
	return green.add(blue.scale(b.Distance / 10000)).normalize().add(black.scale(incline)).normalize()
}

func createView(view args, elevMap elevationmap.ElevationMap) {
	subPixels := 3
	geopixelLen := int(view.heightAngle * float64(view.columns) / view.width) * subPixels
	elevation0 := elevMap.GetElevation(view.easting, view.northing) + 20

	img := image.NewRGBA(image.Rectangle{
		image.Point{0, 0},
		image.Point{view.columns, geopixelLen / subPixels},
	})

	trans := transform.Transform{
		Easting:  view.easting,
		Northing: view.northing,
		ElevMap: elevMap,
		GeopixelLen: geopixelLen,
	}

	for i := 0; i < view.columns; i++ {
		rad := view.start + (float64(view.columns - i) * view.width / float64(view.columns))
		geopixels := trans.TraceDirection(rad, elevation0)

		len := len(geopixels)
		if len > geopixelLen {
			len = geopixelLen
		}
		for j := 0; j < len; j+=subPixels {
			c := getRGB(geopixels[j])
			alpha := 255 / subPixels
			for k := 1; k < subPixels; k++ {
				if j + k < len {
					c = c.add(getRGB(geopixels[j+k]))
					alpha += 255 / subPixels
				}
			}
			img.Set(i, (geopixelLen - j) / subPixels, c.normalize().getColor(uint8(alpha)))
		}

		fmt.Println("col", i)
	}
	f, _ := os.Create("foo.png")
	png.Encode(f, img)
}


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

	createView(args1, elevmap)
}
