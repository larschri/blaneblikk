package render

import (
	"fmt"
	"github.com/larschri/blaneblikk/dataset"
	"github.com/larschri/blaneblikk/transform"
	"image"
	"math"
)

type Renderer struct {
	Start      float64
	Width      float64
	Columns    int
	Easting    float64
	Northing   float64
	Elevations dataset.ElevationMap
}

// fadeFromDistance is a distance from where we add white to the color to make it fade
const fadeFromDistance = 70000.0
const subPixels = 3

func getRGB(b transform.GeoPixel) rgb {
	incline := math.Max(0, math.Min(1, b.Incline/20))
	color1 := green.add(blue.scale(b.Distance / 10000)).normalize().add(black.scale(incline)).normalize()
	if b.Distance < fadeFromDistance {
		return color1
	}
	return color1.add(white.scale((b.Distance - fadeFromDistance) / fadeFromDistance)).normalize()
}

func (r Renderer) transform() transform.Transform {
	return transform.Transform{
		Easting:     math.Round(r.Easting/10) * 10,
		Northing:    math.Round(r.Northing/10) * 10,
		ElevMap:     r.Elevations,
		GeoPixelLen: int(transform.TotalHeightAngle*float64(r.Columns)/r.Width) * subPixels,
	}
}

// PixelToLatLng convert pixel position to UTM easting+northing
func (r Renderer) PixelToUTM(posX int, posY int) (easting float64, northing float64, err error) {
	trans := r.transform()

	rad := r.Start + (float64(posX) * r.Width / float64(r.Columns))
	geoPixels := trans.TraceDirection(rad, make([]transform.GeoPixel, 0, 5000))

	idx := trans.GeoPixelLen - posY*subPixels
	if idx >= len(geoPixels) {
		return math.NaN(), math.NaN(), fmt.Errorf("invalid position")
	}

	easting = trans.Easting + math.Sin(rad)*geoPixels[idx].Distance
	northing = trans.Northing + math.Cos(rad)*geoPixels[idx].Distance
	return
}

// CreateImage builds the image from the elevation data
func (r Renderer) CreateImage() *image.RGBA {
	trans := r.transform()

	img := image.NewRGBA(image.Rectangle{
		Min: image.Point{X: 0, Y: 0},
		Max: image.Point{X: r.Columns, Y: trans.GeoPixelLen / subPixels},
	})

	var pixels [5000]transform.GeoPixel
	for i := 0; i < r.Columns; i++ {
		rad := r.Start + (float64(r.Columns-i) * r.Width / float64(r.Columns))

		geoPixels := trans.TraceDirection(rad, pixels[:0])

		l := len(geoPixels)
		if l > trans.GeoPixelLen {
			l = trans.GeoPixelLen
		}
		for j := 0; j < l; j += subPixels {
			c := getRGB(geoPixels[j])
			alpha := 255 / subPixels
			for k := 1; k < subPixels; k++ {
				if j+k < l {
					c = c.add(getRGB(geoPixels[j+k]))
					alpha += 255 / subPixels
				}
			}
			img.Set(r.Columns-i, (trans.GeoPixelLen-j)/subPixels, c.normalize().getColor(uint8(alpha)))
		}

	}
	return img
}
