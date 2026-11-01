package art

import (
	"image"
	"math"
)

// Theme is a creature family. In Classic mode it is the enemy; in Reverse mode
// it is the player's own character.
type Theme struct {
	ID, Name string
	Paint    func(c *Canvas, frame int, tint C, hasTint bool)
}

var (
	white  = Hex(0xfaf6ea)
	black  = Hex(0x15121c)
	red    = Hex(0xe0322e)
	yellow = Hex(0xf5b92e)
	green  = Hex(0x2fa84f)
)

var Themes = []Theme{
	{"chicken", "תרנגולות", paintChicken},
	{"xmas", "תרנגולות חג", paintXmas},
	{"plumber", "אינסטלטורים", paintPlumber},
	{"robot", "רובוטים", paintRobot},
	{"ghost", "רוחות", paintGhost},
}

func eye(c *Canvas, x, y, r float64) {
	c.Ell(x, y, r, r*1.15, white, 0)
	c.Ell(x+r*0.1, y+r*0.2, r*0.5, r*0.6, black, 0)
	c.Ell(x-r*0.1, y-r*0.15, r*0.17, r*0.17, white, 0)
}

func paintChicken(c *Canvas, f int, tint C, has bool) {
	body := white
	if has {
		body = tint
	}
	flap := 0.0
	if f == 1 {
		flap = 0.09
	}
	c.Line(0.42, 0.84, 0.40, 0.94, 0.018, yellow)
	c.Line(0.58, 0.84, 0.60, 0.94, 0.018, yellow)
	// wings
	c.Out(0.17, 0.52-flap, 0.13, 0.2, body.Mul(0.88))
	c.Out(0.83, 0.52-flap, 0.13, 0.2, body.Mul(0.88))
	c.Out(0.5, 0.6, 0.27, 0.28, body)
	c.Ell(0.5, 0.68, 0.16, 0.16, body.Mul(1.1).Alpha(0.5), 0)
	// comb, sitting behind the head
	c.Out(0.42, 0.15, 0.06, 0.07, red)
	c.Out(0.5, 0.11, 0.07, 0.08, red)
	c.Out(0.58, 0.15, 0.06, 0.07, red)
	c.Out(0.5, 0.33, 0.2, 0.18, body)
	eye(c, 0.42, 0.3, 0.052)
	eye(c, 0.58, 0.3, 0.052)
	c.Poly([][2]float64{{0.43, 0.38}, {0.57, 0.38}, {0.5, 0.48}}, yellow, 0.4)
	c.Ell(0.5, 0.49, 0.03, 0.045, red, 0.3)
}

func paintXmas(c *Canvas, f int, tint C, has bool) {
	paintChicken(c, f, tint, has)
	// Santa hat over the comb, with a scarf.
	c.Poly([][2]float64{{0.31, 0.25}, {0.69, 0.25}, {0.76, 0.07}, {0.5, 0.03}}, red, 0.5)
	c.Out(0.79, 0.09, 0.05, 0.05, white)
	c.Line(0.32, 0.25, 0.68, 0.25, 0.033, white)
	c.Line(0.33, 0.5, 0.67, 0.5, 0.028, green)
	c.Line(0.6, 0.5, 0.65, 0.66, 0.026, green)
}

func paintPlumber(c *Canvas, f int, tint C, has bool) {
	shirt, overall := red, Hex(0x2a55c9)
	if has {
		shirt = tint
	}
	sw := 0.0
	if f == 1 {
		sw = 0.05
	}
	c.Out(0.36-sw, 0.91, 0.1, 0.055, Hex(0x6d3f1d))
	c.Out(0.64+sw, 0.91, 0.1, 0.055, Hex(0x6d3f1d))
	c.Line(0.2, 0.55, 0.13, 0.42-sw*2, 0.05, shirt)
	c.Line(0.8, 0.55, 0.87, 0.42+sw*2-0.08, 0.05, shirt)
	c.Out(0.13, 0.4-sw*2, 0.055, 0.055, white)
	c.Out(0.87, 0.34+sw*2, 0.055, 0.055, white)
	c.Out(0.5, 0.67, 0.25, 0.23, overall)
	c.Ell(0.5, 0.52, 0.22, 0.09, shirt, 0.3)
	c.Line(0.38, 0.5, 0.38, 0.62, 0.022, overall)
	c.Line(0.62, 0.5, 0.62, 0.62, 0.022, overall)
	c.Ell(0.38, 0.62, 0.022, 0.022, yellow, 0.3)
	c.Ell(0.62, 0.62, 0.022, 0.022, yellow, 0.3)
	c.Out(0.5, 0.31, 0.2, 0.18, Hex(0xf2bd8d))
	eye(c, 0.43, 0.28, 0.042)
	eye(c, 0.57, 0.28, 0.042)
	c.Out(0.5, 0.35, 0.05, 0.045, Hex(0xeaa577))
	c.Ell(0.42, 0.41, 0.09, 0.03, Hex(0x3b2412), 0)
	c.Ell(0.58, 0.41, 0.09, 0.03, Hex(0x3b2412), 0)
	// A plain cap: no logo, an original character rather than a licensed one.
	c.Ell(0.5, 0.17, 0.22, 0.11, shirt, 0.5)
	c.Ell(0.58, 0.2, 0.2, 0.035, shirt.Mul(0.75), 0)
}

func paintRobot(c *Canvas, f int, tint C, has bool) {
	m := Hex(0x8fa3b8)
	if has {
		m = tint
	}
	blink := Hex(0xff5140)
	if f == 1 {
		blink = Hex(0x48ffa0)
	}
	c.Line(0.5, 0.17, 0.5, 0.07, 0.014, black)
	c.Ell(0.5, 0.07, 0.035, 0.035, blink, 0.2)
	c.Line(0.16, 0.5, 0.09, 0.72, 0.035, m.Mul(0.7))
	c.Line(0.84, 0.5, 0.91, 0.72, 0.035, m.Mul(0.7))
	box := func(x0, y0, x1, y1 float64, col C) {
		c.Poly([][2]float64{{x0 - .012, y0 - .012}, {x1 + .012, y0 - .012}, {x1 + .012, y1 + .012}, {x0 - .012, y1 + .012}}, col.Mul(0.22), 0)
		c.Poly([][2]float64{{x0, y0}, {x1, y0}, {x1, y1}, {x0, y1}}, col, 0.45)
	}
	box(0.36, 0.84, 0.44, 0.95, m.Mul(0.6))
	box(0.56, 0.84, 0.64, 0.95, m.Mul(0.6))
	box(0.22, 0.46, 0.78, 0.86, m)
	box(0.28, 0.16, 0.72, 0.44, m.Mul(1.1))
	c.Ell(0.4, 0.29, 0.055, 0.055, Hex(0x102030), 0)
	c.Ell(0.6, 0.29, 0.055, 0.055, Hex(0x102030), 0)
	c.Ell(0.4, 0.29, 0.032, 0.032, Hex(0x4ee3ff), 0.3)
	c.Ell(0.6, 0.29, 0.032, 0.032, Hex(0x4ee3ff), 0.3)
	c.Line(0.4, 0.38, 0.6, 0.38, 0.012, black)
	c.Ell(0.5, 0.65, 0.09, 0.09, blink, 0.5)
}

func paintGhost(c *Canvas, f int, tint C, has bool) {
	b := Hex(0xdcc8ff)
	if has {
		b = tint
	}
	w := 0.03
	if f == 1 {
		w = -0.03
	}
	pts := [][2]float64{}
	for i := 0; i <= 24; i++ {
		a := math.Pi + float64(i)/24*math.Pi
		pts = append(pts, [2]float64{0.5 + math.Cos(a)*0.34, 0.42 + math.Sin(a)*0.34})
	}
	for i := 0; i <= 8; i++ {
		x := 0.84 - float64(i)/8*0.68
		y := 0.88
		if i%2 == 1 {
			y = 0.78 + w
		} else {
			y -= w
		}
		pts = append(pts, [2]float64{x, y})
	}
	out := make([][2]float64, len(pts))
	for i, p := range pts {
		out[i] = [2]float64{0.5 + (p[0]-0.5)*1.07, 0.5 + (p[1]-0.5)*1.05}
	}
	c.Poly(out, b.Mul(0.22), 0)
	c.Poly(pts, b, 0.45)
	eye(c, 0.39, 0.4, 0.07)
	eye(c, 0.61, 0.4, 0.07)
	c.Ell(0.5, 0.57, 0.06, 0.075, black, 0)
	c.Ell(0.28, 0.5, 0.045, 0.028, Hex(0xff8fb0).Alpha(0.7), 0)
	c.Ell(0.72, 0.5, 0.045, 0.028, Hex(0xff8fb0).Alpha(0.7), 0)
}

func Creature(t Theme, n, frame int) *image.RGBA {
	c := NewCanvas(n)
	t.Paint(c, frame, C{}, false)
	return c.Image()
}

// Ship is the player's fighter in Classic and the enemy in Reverse, nose up.
func Ship(n, variant int) *image.RGBA {
	c := NewCanvas(n)
	hulls := []C{Hex(0xc5d0de), Hex(0xd9574d), Hex(0x57a6d9), Hex(0x9d6fe0)}
	hull := hulls[variant%len(hulls)]
	wing := hull.Mul(0.62)
	c.Poly([][2]float64{{0.5, 0.3}, {0.04, 0.78}, {0.1, 0.87}, {0.5, 0.72}, {0.9, 0.87}, {0.96, 0.78}}, black, 0)
	c.Poly([][2]float64{{0.5, 0.34}, {0.09, 0.78}, {0.12, 0.82}, {0.5, 0.69}, {0.88, 0.82}, {0.91, 0.78}}, wing, 0.5)
	c.Poly([][2]float64{{0.5, 0.03}, {0.36, 0.5}, {0.38, 0.85}, {0.62, 0.85}, {0.64, 0.5}}, black, 0)
	c.Poly([][2]float64{{0.5, 0.07}, {0.39, 0.5}, {0.41, 0.82}, {0.59, 0.82}, {0.61, 0.5}}, hull, 0.6)
	c.Ell(0.5, 0.42, 0.055, 0.1, Hex(0x37d7ff), 0.8)
	c.Line(0.17, 0.74, 0.17, 0.62, 0.016, red)
	c.Line(0.83, 0.74, 0.83, 0.62, 0.016, red)
	c.Ell(0.5, 0.9, 0.07, 0.06, Hex(0xffa52e), 0.6)
	c.Ell(0.5, 0.89, 0.035, 0.035, Hex(0xfff0b0), 0)
	return c.Image()
}

// BossSprite decorates a large creature (or mothership) so each boss reads as
// its own character.
func BossSprite(t Theme, n int, kind int, ships bool, frame int) *image.RGBA {
	c := NewCanvas(n)
	tints := []C{Hex(0xfff1c9), Hex(0xcdeeff), Hex(0xffb98a), Hex(0x6b5a8d), Hex(0xffdf5a), Hex(0xffb000)}
	if ships {
		mothership(c, kind, tints[kind], frame)
		return c.Image()
	}
	t.Paint(c, frame, tints[kind], true)
	gold := Hex(0xf7c52d)
	switch kind {
	case 0: // crown of the Mother Hen
		c.Poly([][2]float64{{0.34, 0.2}, {0.3, 0.02}, {0.42, 0.11}, {0.5, 0}, {0.58, 0.11}, {0.7, 0.02}, {0.66, 0.2}}, gold, 0.5)
	case 1: // top hat and carrot of the Snow King
		c.Poly([][2]float64{{0.36, 0.02}, {0.64, 0.02}, {0.64, 0.17}, {0.36, 0.17}}, black, 0.6)
		c.Line(0.27, 0.18, 0.73, 0.18, 0.018, black)
		c.Line(0.37, 0.15, 0.63, 0.15, 0.012, red)
		c.Poly([][2]float64{{0.47, 0.36}, {0.47, 0.42}, {0.68, 0.42}}, Hex(0xf58a1f), 0.3)
	case 2: // battle goggles for the Twins
		c.Line(0.32, 0.3, 0.68, 0.3, 0.018, black)
		c.Ell(0.42, 0.3, 0.075, 0.075, Hex(0x37d7ff).Alpha(0.75), 0.3)
		c.Ell(0.58, 0.3, 0.075, 0.075, Hex(0x37d7ff).Alpha(0.75), 0.3)
	case 3: // horns and burning eyes of the Dark Rooster
		c.Poly([][2]float64{{0.33, 0.2}, {0.2, 0.02}, {0.41, 0.15}}, Hex(0x2a2238), 0.3)
		c.Poly([][2]float64{{0.67, 0.2}, {0.8, 0.02}, {0.59, 0.15}}, Hex(0x2a2238), 0.3)
		c.Ell(0.42, 0.3, 0.06, 0.06, Hex(0xff2a2a), 0.2)
		c.Ell(0.58, 0.3, 0.06, 0.06, Hex(0xff2a2a), 0.2)
	case 4: // the Yolk Eye: one giant lens
		c.Ell(0.5, 0.31, 0.15, 0.15, white, 0.2)
		c.Ell(0.5, 0.33, 0.1, 0.1, Hex(0xff5a00), 0.6)
		c.Ell(0.5, 0.34, 0.05, 0.05, black, 0)
	default: // the Yolk Star: spikes
		for i := 0; i < 8; i++ {
			a := float64(i)/8*2*math.Pi + 0.2
			ca, sa := math.Cos(a), math.Sin(a)
			c.Line(0.5+ca*0.3, 0.5+sa*0.3, 0.5+ca*0.48, 0.5+sa*0.48, 0.016, gold)
		}
		c.Poly([][2]float64{{0.34, 0.2}, {0.3, 0.02}, {0.42, 0.11}, {0.5, 0}, {0.58, 0.11}, {0.7, 0.02}, {0.66, 0.2}}, gold, 0.5)
	}
	return c.Image()
}

func mothership(c *Canvas, kind int, tint C, frame int) {
	c.Ell(0.5, 0.5, 0.48, 0.22, tint.Mul(0.22), 0)
	c.Ell(0.5, 0.5, 0.465, 0.205, tint.Mul(0.72), 0.7)
	c.Ell(0.5, 0.37, 0.2, 0.17, Hex(0x37d7ff), 0.8)
	c.Ell(0.45, 0.31, 0.06, 0.04, white.Alpha(0.6), 0)
	lights := 5 + kind
	for i := 0; i < lights; i++ {
		x := 0.13 + 0.74*float64(i)/float64(lights-1)
		col := Hex(0xff5140)
		if (i+frame)%2 == 0 {
			col = Hex(0xffe063)
		}
		c.Ell(x, 0.56+0.03*math.Sin(float64(i)), 0.024, 0.024, col, 0.3)
	}
	for i := 0; i < 1+kind/2; i++ {
		x := 0.5 + (float64(i)-float64(kind/2)/2)*0.22
		c.Line(x, 0.68, x, 0.86, 0.03, Hex(0x20232e))
		c.Ell(x, 0.87, 0.04, 0.03, Hex(0xff5140), 0.4)
	}
}

func Egg(n int) *image.RGBA {
	c := NewCanvas(n)
	c.Ell(0.5, 0.52, 0.34, 0.45, Hex(0x4a3c22), 0)
	c.Ell(0.5, 0.52, 0.29, 0.4, Hex(0xfff3d3), 0.6)
	return c.Image()
}

func Snowball(n int) *image.RGBA {
	c := NewCanvas(n)
	c.Ell(0.5, 0.5, 0.47, 0.47, Hex(0x2a4a6a), 0)
	c.Ell(0.5, 0.5, 0.43, 0.43, Hex(0xe6f6ff), 0.5)
	for _, p := range [][2]float64{{0.35, 0.35}, {0.6, 0.42}, {0.45, 0.65}, {0.67, 0.65}} {
		c.Ell(p[0], p[1], 0.045, 0.045, Hex(0xa9d5ef), 0)
	}
	return c.Image()
}

func Rock(n, variant int) *image.RGBA {
	c := NewCanvas(n)
	pts := [][2]float64{}
	in := [][2]float64{}
	for i := 0; i < 11; i++ {
		a := float64(i) / 11 * 2 * math.Pi
		r := 0.4 + 0.07*math.Sin(float64(i*(3+variant))+float64(variant))
		pts = append(pts, [2]float64{0.5 + math.Cos(a)*(r+0.03), 0.5 + math.Sin(a)*(r+0.03)})
		in = append(in, [2]float64{0.5 + math.Cos(a)*r, 0.5 + math.Sin(a)*r})
	}
	c.Poly(pts, Hex(0x1b1712), 0)
	c.Poly(in, Hex(0x8c7763), 0.7)
	c.Ell(0.38, 0.4, 0.08, 0.07, Hex(0x5c4d40), 0)
	c.Ell(0.62, 0.58, 0.11, 0.09, Hex(0x5c4d40), 0)
	c.Ell(0.45, 0.68, 0.05, 0.05, Hex(0x5c4d40), 0)
	return c.Image()
}

func Food(n int) *image.RGBA {
	c := NewCanvas(n)
	c.Line(0.66, 0.66, 0.84, 0.84, 0.05, Hex(0x2c1c10))
	c.Line(0.66, 0.66, 0.84, 0.84, 0.03, Hex(0xf6ead0))
	c.Ell(0.86, 0.8, 0.05, 0.05, Hex(0xf6ead0), 0.3)
	c.Ell(0.8, 0.87, 0.05, 0.05, Hex(0xf6ead0), 0.3)
	c.Out(0.42, 0.42, 0.3, 0.28, Hex(0xc77a2a))
	c.Ell(0.34, 0.33, 0.1, 0.07, Hex(0xf0b163).Alpha(0.8), 0)
	return c.Image()
}

func Corn(n int) *image.RGBA {
	c := NewCanvas(n)
	c.Out(0.5, 0.5, 0.26, 0.42, Hex(0xf5c93a))
	for y := 0; y < 6; y++ {
		for x := 0; x < 3; x++ {
			c.Ell(0.38+float64(x)*0.12, 0.2+float64(y)*0.12, 0.045, 0.04, Hex(0xffe27a), 0.5)
		}
	}
	c.Poly([][2]float64{{0.2, 0.97}, {0.5, 0.6}, {0.45, 0.97}}, green, 0.4)
	c.Poly([][2]float64{{0.8, 0.97}, {0.5, 0.6}, {0.55, 0.97}}, green, 0.4)
	return c.Image()
}

func Gift(n, weapon int) *image.RGBA {
	c := NewCanvas(n)
	cols := []C{Hex(0xe0322e), Hex(0x2fa84f), Hex(0x2e8fe0)}
	col := cols[weapon%len(cols)]
	box := func(x0, y0, x1, y1 float64, cl C) {
		c.Poly([][2]float64{{x0, y0}, {x1, y0}, {x1, y1}, {x0, y1}}, cl, 0.4)
	}
	box(0.09, 0.3, 0.91, 0.94, black)
	box(0.13, 0.34, 0.87, 0.9, col)
	box(0.43, 0.34, 0.57, 0.9, Hex(0xffe063))
	box(0.13, 0.54, 0.87, 0.66, Hex(0xffe063))
	c.Out(0.34, 0.2, 0.15, 0.1, Hex(0xffe063))
	c.Out(0.66, 0.2, 0.15, 0.1, Hex(0xffe063))
	c.Out(0.5, 0.25, 0.07, 0.07, Hex(0xffc22e))
	return c.Image()
}

func MissileIcon(n int) *image.RGBA {
	c := NewCanvas(n)
	c.Poly([][2]float64{{0.5, 0.02}, {0.34, 0.3}, {0.34, 0.75}, {0.15, 0.95}, {0.85, 0.95}, {0.66, 0.75}, {0.66, 0.3}}, black, 0)
	c.Poly([][2]float64{{0.5, 0.08}, {0.38, 0.31}, {0.38, 0.75}, {0.62, 0.75}, {0.62, 0.31}}, Hex(0xe8e8ee), 0.6)
	c.Poly([][2]float64{{0.38, 0.72}, {0.22, 0.9}, {0.78, 0.9}, {0.62, 0.72}}, red, 0.4)
	c.Ell(0.5, 0.42, 0.06, 0.06, Hex(0x37d7ff), 0.5)
	return c.Image()
}
