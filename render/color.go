package render

import "image/color"

type rgb struct {
	r float64
	g float64
	b float64
	w float64
}

func (c rgb) scale(s float64) rgb {
	return rgb{
		r: c.r * s,
		g: c.g * s,
		b: c.b * s,
		w: c.w * s,
	}
}

func (c rgb) add(c2 rgb) rgb {
	return rgb{
		r: c.r + c2.r,
		g: c.g + c2.g,
		b: c.b + c2.b,
		w: c.w + c2.w,
	}
}

var (
	green = rgb{
		r: 24,
		g: 161,
		b: 61,
		w: 1,
	}

	blue = rgb{
		r: 76,
		g: 150,
		b: 224,
		w: 1,
	}

	black = rgb{
		r: 0,
		g: 0,
		b: 0,
		w: 1,
	}

	white = rgb{
		r: 255,
		g: 255,
		b: 255,
		w: 1,
	}
)

func (c rgb) normalize() rgb {
	return rgb{
		c.r / c.w,
		c.g / c.w,
		c.b / c.w,
		1,
	}
}

func (c rgb) getColor(alpha uint8) color.RGBA {
	n := c.normalize()
	return color.RGBA{
		R: uint8(n.r),
		G: uint8(n.g),
		B: uint8(n.b),
		A: alpha}
}
