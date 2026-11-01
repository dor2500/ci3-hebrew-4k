package main

import (
	"bytes"
	"image/color"
	"unicode"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"golang.org/x/text/language"
)

// Text does its own run splitting because text/v2 shapes one direction per
// call: the paragraph is RTL, and digits or Latin words inside it stay LTR.
type Align int

const (
	AlignRight Align = iota
	AlignCenter
	AlignLeft
)

type textKey struct {
	size int
	rtl  bool
}

type Text struct {
	src   *text.GoTextFaceSource
	faces map[textKey]*text.GoTextFace
}

func NewText(ttf []byte) (*Text, error) {
	src, err := text.NewGoTextFaceSource(bytes.NewReader(ttf))
	if err != nil {
		return nil, err
	}
	return &Text{src: src, faces: map[textKey]*text.GoTextFace{}}, nil
}

func (t *Text) face(size int, rtl bool) *text.GoTextFace {
	k := textKey{size, rtl}
	if f, ok := t.faces[k]; ok {
		return f
	}
	f := &text.GoTextFace{Source: t.src, Size: float64(size)}
	f.SetVariation(text.MustParseTag("wght"), 600)
	if rtl {
		f.Direction = text.DirectionRightToLeft
		f.Language = language.Hebrew
	}
	t.faces[k] = f
	return f
}

type run struct {
	s   string
	rtl bool
}

func isHebrew(r rune) bool { return r >= 0x0590 && r <= 0x05FF }

// splitRuns applies the parts of the Unicode bidi algorithm this UI needs.
func splitRuns(s string) []run {
	rs := []rune(s)
	strong := func(r rune) int { // 1 RTL, -1 LTR, 0 neutral
		switch {
		case isHebrew(r):
			return 1
		case unicode.IsLetter(r), unicode.IsDigit(r):
			return -1
		}
		return 0
	}
	var out []run
	cur, curRTL := []rune{}, true
	flush := func() {
		if len(cur) > 0 {
			out = append(out, run{string(cur), curRTL})
			cur = cur[:0]
		}
	}
	for i, r := range rs {
		d := strong(r)
		rtl := curRTL
		switch d {
		case 1:
			rtl = true
		case -1:
			rtl = false
		default:
			// A neutral stays in an LTR run only if LTR text continues after it.
			if !curRTL {
				rtl = true
				for _, n := range rs[i+1:] {
					if nd := strong(n); nd != 0 {
						rtl = nd == 1
						break
					}
				}
			}
		}
		if rtl != curRTL {
			flush()
			curRTL = rtl
		}
		cur = append(cur, r)
	}
	flush()
	return out
}

func (t *Text) Width(s string, size int) float64 {
	w := 0.0
	for _, r := range splitRuns(s) {
		w += text.Advance(r.s, t.face(size, r.rtl))
	}
	return w
}

// Draw places a paragraph with the given anchor at x; y is the vertical centre.
func (t *Text) Draw(dst *ebiten.Image, s string, x, y float64, size int, a Align, c color.Color) {
	total := t.Width(s, size)
	right := x
	switch a {
	case AlignCenter:
		right = x + total/2
	case AlignLeft:
		right = x + total
	}
	for _, r := range splitRuns(s) {
		f := t.face(size, r.rtl)
		w := text.Advance(r.s, f)
		op := &text.DrawOptions{}
		op.SecondaryAlign = text.AlignCenter
		op.Filter = ebiten.FilterLinear
		// Soft shadow keeps text readable over the moving backdrop.
		if r.rtl {
			op.GeoM.Translate(right, y)
		} else {
			op.GeoM.Translate(right-w, y)
		}
		shadow := *op
		shadow.GeoM.Translate(float64(size)/22, float64(size)/22)
		shadow.ColorScale.ScaleWithColor(color.RGBA{0, 0, 0, 200})
		text.Draw(dst, r.s, f, &shadow)
		op.ColorScale.ScaleWithColor(c)
		text.Draw(dst, r.s, f, op)
		right -= w
	}
}
