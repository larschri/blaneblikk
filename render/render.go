package render

import (
	"github.com/larschri/blaner/elevationmap"
	"github.com/larschri/blaner/transform"
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

func getRGB(b transform.Geopixel) rgb {
	incline := math.Max(0, math.Min(1, b.Incline/20))
	return green.add(blue.scale(b.Distance / 10000)).normalize().add(black.scale(incline)).normalize()
}

func CreateImage(view Args, elevMap elevationmap.ElevationMap) *image.RGBA {
	subPixels := 3
	geopixelLen := int(view.HeightAngle*float64(view.Columns)/view.Width) * subPixels
	elevation0 := elevMap.GetElevation(view.Easting, view.Northing, 0) + 20

	img := image.NewRGBA(image.Rectangle{
		image.Point{0, 0},
		image.Point{view.Columns, geopixelLen / subPixels},
	})

	trans := transform.Transform{
		Easting:     view.Easting,
		Northing:    view.Northing,
		ElevMap:     elevMap,
		GeopixelLen: geopixelLen,
	}

	for i := 0; i < view.Columns; i++ {
		rad := view.Start + (float64(view.Columns-i) * view.Width / float64(view.Columns))
		geopixels := trans.TraceDirection(rad, elevation0)

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

		//fmt.Println("col", i)
	}
	return img
}

