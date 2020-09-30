package render

import (
	"github.com/larschri/blaneblikk/transform"
	"github.com/lucasb-eyer/go-colorful"
)

type gradient struct {
	gradient [][]rgb
}

var gradient1 = gradient{
	gradient: [][]rgb{
		{green, black},
		{blue, black},
	},
}

func hcl1(h, c, l float64) rgb {
	cl := colorful.Hcl(h, c, l).Clamped()
	return rgb{255 * cl.R, 255 * cl.G, 255 * cl.B, 2}
}

func hcl2(h, c, l float64) []rgb {
	return []rgb{hcl1(h, c, l), hcl1(h, c, l-0.5)}
}

func init() {
	var g [][]rgb
	g = append(g, hcl2(100, 0.45, 0.95))
	g = append(g, hcl2(140, 0.45, 0.95))
	g = append(g, hcl2(170, 0.45, 0.95))
	g = append(g, hcl2(200, 0.45, 0.95))
	g = append(g, hcl2(210, 0.45, 0.95))
	g = append(g, hcl2(220, 0.45, 0.95))
	g = append(g, hcl2(230, 0.45, 1))
	g = append(g, hcl2(240, 0.45, 1.1))
	g = append(g, hcl2(250, 0.45, 1.2))
	g = append(g, hcl2(260, 0.45, 1.3))
	g = append(g, hcl2(270, 0.45, 1.4))
	g = append(g, hcl2(280, 0.45, 1.5))
	gradient1.gradient = g
}

func intAndFraction(value float64, max float64, length int) (int, float64) {

	if value <= 0 {
		return 0, 0
	}

	if value >= max {
		return length - 2, 1
	}

	r := float64(length-1) * value / max
	i := int(r)
	return i, r - float64(i)
}

func (g gradient) getRGB(b transform.GeoPixel) rgb {

	id, rd := intAndFraction(b.Distance, 200_000, len(g.gradient))
	ii, ri := intAndFraction(b.Incline, 20, len(g.gradient[0]))

	c1 := g.gradient[id][ii].scale(1 - rd).add(g.gradient[id+1][ii].scale(rd))
	c2 := g.gradient[id][ii+1].scale(1 - rd).add(g.gradient[id+1][ii+1].scale(rd))

	return c1.scale(1 - ri).add(c2.scale(ri))
}
