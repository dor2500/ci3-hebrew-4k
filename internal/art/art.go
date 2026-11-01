// Package art paints every sprite and background procedurally on the CPU, at
// the size it will be shown in 4K. Nothing is upscaled and no third-party art
// is included.
package art

import (
	"image"
	"image/color"
	"math"
	"math/rand"
)

type C struct{ R, G, B, A float64 }

func Hex(v uint32) C {
	return C{float64(v>>16&255) / 255, float64(v>>8&255) / 255, float64(v&255) / 255, 1}
}

func (c C) Mul(f float64) C { return C{c.R * f, c.G * f, c.B * f, c.A} }
func (c C) Alpha(a float64) C {
	c.A = a
	return c
}

// Canvas uses premultiplied float pixels and unit coordinates from 0 to 1.
type Canvas struct {
	N   int
	Pix []float64
}

func NewCanvas(n int) *Canvas { return &Canvas{N: n, Pix: make([]float64, n*n*4)} }

func (c *Canvas) blend(i int, col C, cov float64) {
	a := col.A * cov
	c.Pix[i] = c.Pix[i]*(1-a) + col.R*a
	c.Pix[i+1] = c.Pix[i+1]*(1-a) + col.G*a
	c.Pix[i+2] = c.Pix[i+2]*(1-a) + col.B*a
	c.Pix[i+3] = c.Pix[i+3]*(1-a) + a
}

// shade fills by signed distance. fill gets normalised y inside the bounds.
func (c *Canvas) shade(x0, y0, x1, y1 float64, dist func(x, y float64) float64, fill func(x, y float64) C) {
	n := float64(c.N)
	ix0, iy0 := max(0, int(x0*n)-2), max(0, int(y0*n)-2)
	ix1, iy1 := min(c.N, int(x1*n)+3), min(c.N, int(y1*n)+3)
	for y := iy0; y < iy1; y++ {
		for x := ix0; x < ix1; x++ {
			ux, uy := (float64(x)+0.5)/n, (float64(y)+0.5)/n
			cov := math.Max(0, math.Min(1, 0.5-dist(ux, uy)*n))
			if cov > 0 {
				c.blend((y*c.N+x)*4, fill(ux, uy), cov)
			}
		}
	}
}

// Ell draws a shaded ellipse lit from the top left.
func (c *Canvas) Ell(cx, cy, rx, ry float64, col C, lit float64) {
	mr := math.Min(rx, ry)
	c.shade(cx-rx, cy-ry, cx+rx, cy+ry, func(x, y float64) float64 {
		return (math.Hypot((x-cx)/rx, (y-cy)/ry) - 1) * mr
	}, func(x, y float64) C {
		l := 1 + lit*(-(x-cx)/rx*0.35-(y-cy)/ry*0.65)
		return C{math.Min(1, col.R*l), math.Min(1, col.G*l), math.Min(1, col.B*l), col.A}
	})
}

// Out draws an ellipse with a dark outline, the base look of the sprites.
func (c *Canvas) Out(cx, cy, rx, ry float64, col C) {
	const o = 0.012
	c.Ell(cx, cy, rx+o, ry+o, col.Mul(0.22), 0)
	c.Ell(cx, cy, rx, ry, col, 0.5)
}

func (c *Canvas) Poly(pts [][2]float64, col C, lit float64) {
	x0, y0, x1, y1 := 2.0, 2.0, -1.0, -1.0
	for _, p := range pts {
		x0, y0 = math.Min(x0, p[0]), math.Min(y0, p[1])
		x1, y1 = math.Max(x1, p[0]), math.Max(y1, p[1])
	}
	h := math.Max(y1-y0, 1e-6)
	c.shade(x0, y0, x1, y1, func(x, y float64) float64 { return polyDist(pts, x, y) }, func(x, y float64) C {
		l := 1 + lit*(0.5-(y-y0)/h)
		return C{math.Min(1, col.R*l), math.Min(1, col.G*l), math.Min(1, col.B*l), col.A}
	})
}

func polyDist(p [][2]float64, x, y float64) float64 {
	d := math.MaxFloat64
	inside := false
	for i, j := 0, len(p)-1; i < len(p); j, i = i, i+1 {
		ax, ay, bx, by := p[j][0], p[j][1], p[i][0], p[i][1]
		ex, ey := bx-ax, by-ay
		t := math.Max(0, math.Min(1, ((x-ax)*ex+(y-ay)*ey)/(ex*ex+ey*ey)))
		d = math.Min(d, math.Hypot(x-ax-ex*t, y-ay-ey*t))
		if (ay > y) != (by > y) && x < ax+(y-ay)/(by-ay)*(bx-ax) {
			inside = !inside
		}
	}
	if inside {
		return -d
	}
	return d
}

// Line draws a thick capsule.
func (c *Canvas) Line(ax, ay, bx, by, r float64, col C) {
	ex, ey := bx-ax, by-ay
	l := ex*ex + ey*ey
	c.shade(math.Min(ax, bx)-r, math.Min(ay, by)-r, math.Max(ax, bx)+r, math.Max(ay, by)+r, func(x, y float64) float64 {
		t := math.Max(0, math.Min(1, ((x-ax)*ex+(y-ay)*ey)/l))
		return math.Hypot(x-ax-ex*t, y-ay-ey*t) - r
	}, func(x, y float64) C { return col })
}

func (c *Canvas) Image() *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, c.N, c.N))
	for i := 0; i < len(c.Pix); i++ {
		img.Pix[i] = uint8(math.Min(1, c.Pix[i])*255 + 0.5)
	}
	return img
}

// Glow is a soft radial sprite for additive drawing.
func Glow(n int, col C, power float64) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, n, n))
	for y := 0; y < n; y++ {
		for x := 0; x < n; x++ {
			d := math.Hypot(float64(x)+0.5-float64(n)/2, float64(y)+0.5-float64(n)/2) / (float64(n) / 2)
			a := math.Pow(math.Max(0, 1-d), power)
			img.SetRGBA(x, y, color.RGBA{uint8(col.R * a * 255), uint8(col.G * a * 255), uint8(col.B * a * 255), uint8(a * 255)})
		}
	}
	return img
}

// Spotlight is black except for a soft hole, for the blackout chapter.
func Spotlight(n int) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, n, n))
	for y := 0; y < n; y++ {
		for x := 0; x < n; x++ {
			d := math.Hypot(float64(x)+0.5-float64(n)/2, float64(y)+0.5-float64(n)/2) / (float64(n) / 2)
			t := math.Max(0, math.Min(1, (d-0.4)/0.55))
			img.SetRGBA(x, y, color.RGBA{0, 0, 0, uint8((t * t * (3 - 2*t)) * 247)})
		}
	}
	return img
}

// Nebula renders a soft space backdrop in the given palette.
func Nebula(w, h int, seed int64, a, b, c C) *image.RGBA {
	r := rand.New(rand.NewSource(seed))
	const g = 64
	var grid [g][g]float64
	for i := range grid {
		for j := range grid[i] {
			grid[i][j] = r.Float64()
		}
	}
	noise := func(x, y float64) float64 {
		ix, iy := int(math.Floor(x)), int(math.Floor(y))
		fx, fy := x-float64(ix), y-float64(iy)
		fx, fy = fx*fx*(3-2*fx), fy*fy*(3-2*fy)
		v := func(i, j int) float64 { return grid[(i%g+g)%g][(j%g+g)%g] }
		return (v(ix, iy)*(1-fx)+v(ix+1, iy)*fx)*(1-fy) + (v(ix, iy+1)*(1-fx)+v(ix+1, iy+1)*fx)*fy
	}
	fbm := func(x, y float64) float64 {
		s, m := 0.0, 0.5
		for o := 0; o < 5; o++ {
			s += noise(x, y) * m
			x, y, m = x*2.03+7, y*2.03+3, m*0.5
		}
		return s
	}
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			u, v := float64(x)/float64(w)*5, float64(y)/float64(h)*3
			n1 := fbm(u+fbm(u, v)*1.6, v)
			n2 := fbm(u*1.4+9, v*1.4+fbm(v, u))
			t1 := math.Pow(math.Max(0, n1-0.32)*2.1, 2)
			t2 := math.Pow(math.Max(0, n2-0.38)*2.3, 2)
			pr := 0.035 + a.R*t1 + b.R*t2
			pg := 0.035 + a.G*t1 + b.G*t2
			pb := 0.07 + a.B*t1 + b.B*t2
			// A subtle vertical tint keeps the play area from looking flat.
			f := float64(y) / float64(h)
			pr, pg, pb = pr+c.R*f*0.1, pg+c.G*f*0.1, pb+c.B*f*0.1
			img.SetRGBA(x, y, color.RGBA{uint8(math.Min(1, pr) * 255), uint8(math.Min(1, pg) * 255), uint8(math.Min(1, pb) * 255), 255})
		}
	}
	return img
}
