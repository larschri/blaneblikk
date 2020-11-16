package render

import (
	"github.com/larschri/blaneblikk/transform"
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

func xx(value float64, max float64, length int) (int, float64) {
	if value <= 0 {
		return 0, 0
	}

	if value >= max {
		return length - 2, 1
	}

	r := value / max
	i := int(r)
	return i, r - float64(i)
}

func (g gradient) getRGB(b transform.GeoPixel) rgb {

	id, rd := xx(b.Distance, 200_000, len(g.gradient))
	ii, ri := xx(b.Incline, 20, len(g.gradient[0]))

	c1 := g.gradient[id][ii].scale(1 - rd).add(g.gradient[id+1][ii].scale(rd))
	c2 := g.gradient[id][ii+1].scale(1 - rd).add(g.gradient[id+1][ii+1].scale(rd))

	return c1.scale(1 - ri).add(c2.scale(ri))
}
