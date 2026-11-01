package main

import (
	"image"
	"image/png"
	_ "image/png"
	"os"

	"github.com/hajimehoshi/ebiten/v2"
)

func saveShot(s *ebiten.Image, path string) {
	r := s.Bounds()
	buf := make([]byte, 4*r.Dx()*r.Dy())
	s.ReadPixels(buf)
	for i := 3; i < len(buf); i += 4 {
		buf[i] = 255
	}
	f, err := os.Create(path)
	if err != nil {
		return
	}
	defer f.Close()
	_ = png.Encode(f, &image.RGBA{Pix: buf, Stride: 4 * r.Dx(), Rect: r})
}
