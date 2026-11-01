package mod

import (
	"fmt"
	"image"
	"image/draw"

	"golang.org/x/image/font"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/math/fixed"
)

const firstHebrewCode = 0xE0

type fontDef struct {
	name      string
	firstCode int
	outlined  bool
	refCode   int
}

// large-3 holds no capitals, so it measures from a lowercase letter.
var fontDefs = []fontDef{
	{"system/font-tiny.tga", 33, false, 'H'},
	{"system/font-small.tga", 0, false, 'H'},
	{"system/font-small-outline.tga", 0, true, 'H'},
	{"system/font-medium.tga", 0, false, 'H'},
	{"system/font-large-3.tga", 121, false, 'z'},
}

const glyphScale = 8

// glyph returns coverage for a letter at target size: core ink and, for
// outlined fonts, the dilated edge. Rows are offset by a 3-pixel top margin.
func glyph(ft *opentype.Font, letter rune, capHeight int, outlined bool) (core, edge [][]uint8, err error) {
	measure := func(size float64) (font.Face, int, error) {
		f, err := opentype.NewFace(ft, &opentype.FaceOptions{Size: size, DPI: 72, Hinting: font.HintingNone})
		if err != nil {
			return nil, 0, err
		}
		b, _ := font.BoundString(f, "ה")
		return f, (b.Max.Y - b.Min.Y).Ceil(), nil
	}
	_, h100, err := measure(200)
	if err != nil {
		return nil, nil, err
	}
	face, _, err := measure(200 * float64(capHeight*glyphScale) / float64(h100))
	if err != nil {
		return nil, nil, err
	}
	box, adv := font.BoundString(face, string(letter))
	width := max((box.Max.X-box.Min.X).Ceil(), adv.Ceil()) + 4*glyphScale
	width += (glyphScale - width%glyphScale) % glyphScale
	height := (capHeight + 8) * glyphScale
	canvas := image.NewAlpha(image.Rect(0, 0, width, height))
	d := font.Drawer{Dst: canvas, Src: image.Opaque, Face: face}
	d.Dot = fixed.Point26_6{X: fixed.I(2*glyphScale) - box.Min.X, Y: fixed.I(3*glyphScale + capHeight*glyphScale)}
	d.DrawString(string(letter))

	sw, sh := width/glyphScale, height/glyphScale
	small := make([][]uint8, sh)
	for y := range small {
		small[y] = make([]uint8, sw)
		for x := range small[y] {
			sum := 0
			for yy := 0; yy < glyphScale; yy++ {
				row := canvas.Pix[(y*glyphScale+yy)*canvas.Stride:]
				for xx := 0; xx < glyphScale; xx++ {
					sum += int(row[x*glyphScale+xx])
				}
			}
			small[y][x] = uint8(sum / (glyphScale * glyphScale))
		}
	}
	dil := small
	if outlined {
		dil = make([][]uint8, sh)
		for y := range dil {
			dil[y] = make([]uint8, sw)
			for x := range dil[y] {
				var m uint8
				for yy := max(0, y-1); yy <= min(sh-1, y+1); yy++ {
					for xx := max(0, x-1); xx <= min(sw-1, x+1); xx++ {
						m = max(m, small[yy][xx])
					}
				}
				dil[y][x] = m
			}
		}
	}
	left, right := sw, 0
	for x := 0; x < sw; x++ {
		for y := 0; y < sh; y++ {
			if dil[y][x] > 12 {
				left, right = min(left, x), max(right, x+1)
			}
		}
	}
	if right <= left {
		return nil, nil, fmt.Errorf("empty glyph %q", letter)
	}
	core, edge = make([][]uint8, sh), make([][]uint8, sh)
	for y := range core {
		core[y], edge[y] = small[y][left:right], dil[y][left:right]
	}
	return core, edge, nil
}

// buildFont appends the 27 Hebrew glyphs to a font atlas, matching the height
// and colour gradient of the font's own letters. The atlas doubles to 512 rows.
func buildFont(tga, ini []byte, def fontDef, ft *opentype.Font) ([]byte, []byte, error) {
	px, err := ReadTGA(tga)
	if err != nil {
		return nil, nil, err
	}
	skins := ParseIni(ini)
	ref, ok := skins[def.refCode-def.firstCode]
	if !ok {
		return nil, nil, fmt.Errorf("%s has no reference glyph", def.name)
	}
	inkTop, inkBottom := -1, 0
	rowCol := make([][3]float64, ref.Height)
	for y := 0; y < ref.Height; y++ {
		var r, g, b, n float64
		maxA := uint8(0)
		for x := 0; x < ref.Width; x++ {
			c := px.NRGBAAt(ref.Left+x, ref.Top+y)
			maxA = max(maxA, c.A)
			if c.A > 128 {
				r, g, b, n = r+float64(c.R), g+float64(c.G), b+float64(c.B), n+1
			}
		}
		if maxA > 96 {
			if inkTop < 0 {
				inkTop = y
			}
			inkBottom = y + 1
		}
		rowCol[y] = [3]float64{255, 255, 255}
		if n > 0 {
			rowCol[y] = [3]float64{r / n, g / n, b / n}
		}
	}
	if inkTop < 0 {
		return nil, nil, fmt.Errorf("%s reference glyph is empty", def.name)
	}
	capHeight := inkBottom - inkTop
	used := 0
	for _, s := range skins {
		used = max(used, s.Top+s.Height)
	}
	w := px.Rect.Dx()
	atlas := image.NewNRGBA(image.Rect(0, 0, w, 512))
	draw.Draw(atlas, px.Rect, px, image.Point{}, draw.Src)
	x, y := 1, max(used+2, 258)
	for i := 0; i < 27; i++ {
		core, edge, err := glyph(ft, rune(0x5D0+i), capHeight, def.outlined)
		if err != nil {
			return nil, nil, err
		}
		gw := len(core[0]) + 1
		if x+gw+1 > w {
			x, y = 1, y+ref.Height+2
		}
		if y+ref.Height > 512 {
			return nil, nil, fmt.Errorf("%s atlas overflow", def.name)
		}
		for row := 0; row < ref.Height; row++ {
			src := row - inkTop + 3
			if src < 0 || src >= len(core) {
				continue
			}
			c := rowCol[row]
			for col := range core[src] {
				ac, ae := int(core[src][col]), int(edge[src][col])
				if ae == 0 {
					continue
				}
				blend := float64(ac) / float64(ae)
				o := atlas.PixOffset(x+col, y+row)
				atlas.Pix[o], atlas.Pix[o+1], atlas.Pix[o+2], atlas.Pix[o+3] = uint8(c[0]*blend), uint8(c[1]*blend), uint8(c[2]*blend), uint8(ae)
			}
		}
		skins[firstHebrewCode+i-def.firstCode] = Skin{Left: x, Top: y, Width: gw, Height: ref.Height}
		x += gw + 2
	}
	blank := skins[0]
	for n := 0; n < 256-def.firstCode; n++ {
		if _, ok := skins[n]; !ok {
			skins[n] = blank
		}
	}
	return WriteTGA(atlas), WriteIni(skins), nil
}

// BuildFonts returns the replaced font resources.
func BuildFonts(d *Dat, ttf []byte) (map[string][]byte, error) {
	ft, err := opentype.Parse(ttf)
	if err != nil {
		return nil, err
	}
	out := map[string][]byte{}
	for _, def := range fontDefs {
		t, i, err := buildFont(d.Get(def.name), d.Get(def.name+".ini"), def, ft)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", def.name, err)
		}
		out[def.name], out[def.name+".ini"] = t, i
	}
	return out, nil
}
