package main

import (
	"fmt"
	"github.com/larschri/blaner/elevationmap"
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

type geopixel struct {
	distance float64
	incline float64
}

func (b geopixel) getRGB() rgb {
	incline := math.Max(0, math.Min(1, b.incline / 20))
	return green.add(blue.scale(b.distance / 10000)).normalize().add(black.scale(incline)).normalize()
}

func createView(view args, elevMap elevationmap.ElevationMap) {
	subPixels := 3
	geopixelLen := int(view.heightAngle * float64(view.columns) / view.width) * subPixels
	elevation0 := elevMap.GetElevation(view.easting, view.northing) + 20

	img := image.NewRGBA(image.Rectangle{
		image.Point{0, 0},
		image.Point{view.columns, geopixelLen / subPixels},
	})

	for i := 0; i < view.columns; i++ {
		geopixels := make([]geopixel, 0)
		rad := view.start + (float64(view.columns - i) * view.width / float64(view.columns))
		sin := math.Sin(rad)
		cos := math.Cos(rad)
		prevElevation := elevation0
		for dist := view.step; dist < 200000; dist = dist + view.step {
			elevation := elevMap.GetElevation(view.easting + sin * dist, view.northing + cos * dist)
			heightAngle := math.Atan2(elevation - elevation0, dist) - math.Atan2(dist / 2, 6371000.0)
			newBoffIdx := int(float64(geopixelLen) * (heightAngle - view.minHeight) / view.heightAngle)

			for len(geopixels) <= newBoffIdx {
				geopixels = append(geopixels, geopixel{
					distance: dist,
					incline:  (elevation - prevElevation),
				})
			}
			prevElevation = elevation
		}

		len := len(geopixels)
		if len > geopixelLen {
			len = geopixelLen
		}
		for j := 0; j < len; j+=subPixels {
			c := geopixels[j].getRGB()
			alpha := 255 / subPixels
			for k := 1; k < subPixels; k++ {
				if j + k < len {
					c = c.add(geopixels[j+k].getRGB())
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
