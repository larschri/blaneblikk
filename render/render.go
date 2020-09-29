package render

import (
	"fmt"
	"github.com/larschri/blaner/dataset"
	"github.com/larschri/blaner/transform"
	"image"
	"math"
)

type Renderer struct {
	Start       float64
	Width       float64
	Columns     int
	Easting     float64
	Northing    float64
	Elevations  dataset.ElevationMap
}

// fadeFromDistance is a distance from where we add white to the color to make it fade
const fadeFromDistance = 70000.0

func getRGB(b transform.GeoPixel) rgb {
	incline := math.Max(0, math.Min(1, b.Incline/20))
	color1 := green.add(blue.scale(b.Distance / 10000)).normalize().add(black.scale(incline)).normalize()
	if b.Distance < fadeFromDistance {
		return color1
	}
	return color1.add(white.scale((b.Distance-fadeFromDistance) / fadeFromDistance)).normalize()
}

type Position struct {
	Northing float64
	Easting  float64
}

func (view Renderer) PixelToLatLng(posX int, posY int) (Position, error) {
	subPixels := 3
	geoPixelLen := int(transform.TotalHeightAngle*float64(view.Columns)/view.Width) * subPixels

	trans2 := transform.Transform{
		Easting:     math.Round(view.Easting/10) * 10,
		Northing:    math.Round(view.Northing/10) * 10,
		ElevMap:     view.Elevations,
		GeoPixelLen: geoPixelLen,
	}

	rad := view.Start + (float64(posX) * view.Width / float64(view.Columns))
	geoPixels := trans2.TraceDirection(rad, make([]transform.GeoPixel, 0, 5000))

	idx := geoPixelLen - posY*subPixels
	if idx < len(geoPixels) {
		return Position{
			Northing: trans2.Northing + math.Cos(rad)*geoPixels[idx].Distance,
			Easting:  trans2.Easting + math.Sin(rad)*geoPixels[idx].Distance,
		}, nil
	}

	return Position{}, fmt.Errorf("invalid position")
}

func (view Renderer) CreateImage() *image.RGBA {
	subPixels := 3
	geoPixelLen := int(transform.TotalHeightAngle*float64(view.Columns)/view.Width) * subPixels

	img := image.NewRGBA(image.Rectangle{
		image.Point{X: 0, Y: 0},
		image.Point{X: view.Columns, Y: geoPixelLen / subPixels},
	})

	trans2 := transform.Transform{
		Easting:     math.Round(view.Easting/10) * 10,
		Northing:    math.Round(view.Northing/10) * 10,
		ElevMap:     view.Elevations,
		GeoPixelLen: geoPixelLen,
	}

	var pixels [5000]transform.GeoPixel
	for i := 0; i < view.Columns; i++ {
		rad := view.Start + (float64(view.Columns-i) * view.Width / float64(view.Columns))

		geoPixels := trans2.TraceDirection(rad, pixels[:0])

		len := len(geoPixels)
		if len > geoPixelLen {
			len = geoPixelLen
		}
		for j := 0; j < len; j += subPixels {
			c := getRGB(geoPixels[j])
			alpha := 255 / subPixels
			for k := 1; k < subPixels; k++ {
				if j+k < len {
					c = c.add(getRGB(geoPixels[j+k]))
					alpha += 255 / subPixels
				}
			}
			img.Set(view.Columns-i, (geoPixelLen-j)/subPixels, c.normalize().getColor(uint8(alpha)))
		}

	}
	return img
}
