package render

import (
	"fmt"
	"github.com/larschri/blaner/dataset/dataset5000"
	"github.com/larschri/blaner/dataset/dtm10utm32"
	"image"
	"math"
)

type Args struct {
	Start       float64
	Width       float64
	Columns     int
	Step        float64
	Easting     float64
	Northing    float64
	HeightAngle float64
	MinHeight   float64
}

func getRGB(b dataset5000.Geopixel) rgb {
	incline := math.Max(0, math.Min(1, b.Incline/20))
	return green.add(blue.scale(b.Distance / 10000)).normalize().add(black.scale(incline)).normalize()
}

type LatLng struct {
	Lat float64 `json:"lat"`
	Lng float64 `json:"lng"`
}

func PixelToLatLng(view Args, elevMap dataset5000.ElevationMap, posX int, posY int) LatLng {
	subPixels := 3
	geopixelLen := int(view.HeightAngle*float64(view.Columns)/view.Width) * subPixels

	trans2 := dataset5000.Transform{
		Easting:     math.Round(view.Easting/10) * 10,
		Northing:    math.Round(view.Northing/10) * 10,
		ElevMap:     elevMap,
		GeopixelLen: geopixelLen,
	}

	rad := view.Start + (float64(/*view.Columns-*/posX) * view.Width / float64(view.Columns))
	geopixels := trans2.TraceDirection(rad)

	idx := geopixelLen - posY * subPixels
	fmt.Println(geopixelLen, posY)
	if idx < len(geopixels) {
		sin := math.Sin(rad) // east
		cos := math.Cos(rad) // north
		fmt.Println("dist: ", geopixels[idx])
		lat, lng := dtm10utm32.ITranslate(trans2.Northing + cos * geopixels[idx].Distance, trans2.Easting + sin * geopixels[idx].Distance)
		return LatLng{
			Lat: lat,
			Lng: lng,
		}
	} else {
		fmt.Println("noes")
	}

	return LatLng{
		Lat: 0,
		Lng: 0,
	}
}

func CreateImage(view Args, elevMap dataset5000.ElevationMap) *image.RGBA {
	subPixels := 3
	geopixelLen := int(view.HeightAngle*float64(view.Columns)/view.Width) * subPixels

	img := image.NewRGBA(image.Rectangle{
		image.Point{0, 0},
		image.Point{view.Columns, geopixelLen / subPixels},
	})

	trans2 := dataset5000.Transform{
		Easting:     math.Round(view.Easting/10) * 10,
		Northing:    math.Round(view.Northing/10) * 10,
		ElevMap:     elevMap,
		GeopixelLen: geopixelLen,
	}

	for i := 0; i < view.Columns; i++ {
		rad := view.Start + (float64(view.Columns-i) * view.Width / float64(view.Columns))
		geopixels := trans2.TraceDirection(rad)

		len := len(geopixels)
		if len > geopixelLen {
			len = geopixelLen
		}
		for j := 0; j < len; j += subPixels {
			c := getRGB(geopixels[j])
			alpha := 255 / subPixels
			for k := 1; k < subPixels; k++ {
				if j+k < len {
					c = c.add(getRGB(geopixels[j+k]))
					alpha += 255 / subPixels
				}
			}
			img.Set(view.Columns-i, (geopixelLen-j)/subPixels, c.normalize().getColor(uint8(alpha)))
		}

	}
	return img
}
