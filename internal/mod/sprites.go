package mod

import (
	"image"
	"image/draw"
	"math"

	xdraw "golang.org/x/image/draw"
)

type atlasFile struct {
	name  string
	img   *image.NRGBA
	skins map[int]Skin
}

func loadAtlas(d *Dat, name string) (*atlasFile, error) {
	img, err := ReadTGA(d.Get(name))
	if err != nil {
		return nil, err
	}
	return &atlasFile{name, img, ParseIni(d.Get(name + ".ini"))}, nil
}

func crop(img *image.NRGBA, r image.Rectangle) *image.NRGBA {
	out := image.NewNRGBA(image.Rect(0, 0, r.Dx(), r.Dy()))
	draw.Draw(out, out.Rect, img, r.Min, draw.Src)
	return out
}

func bbox(img *image.NRGBA) image.Rectangle {
	r := image.Rectangle{}
	for y := img.Rect.Min.Y; y < img.Rect.Max.Y; y++ {
		for x := img.Rect.Min.X; x < img.Rect.Max.X; x++ {
			if img.NRGBAAt(x, y).A > 0 {
				r = r.Union(image.Rect(x, y, x+1, y+1))
			}
		}
	}
	if r.Empty() {
		return img.Rect
	}
	return r
}

func scaled(img *image.NRGBA, w, h int) *image.NRGBA {
	out := image.NewNRGBA(image.Rect(0, 0, max(1, w), max(1, h)))
	xdraw.CatmullRom.Scale(out, out.Rect, img, img.Rect, draw.Src, nil)
	return out
}

// rotated turns counter-clockwise by deg around the centre, growing the canvas.
func rotated(img *image.NRGBA, deg float64) *image.NRGBA {
	a := deg * math.Pi / 180
	s, c := math.Sin(a), math.Cos(a)
	w, h := float64(img.Rect.Dx()), float64(img.Rect.Dy())
	nw := int(math.Ceil(math.Abs(w*c) + math.Abs(h*s)))
	nh := int(math.Ceil(math.Abs(w*s) + math.Abs(h*c)))
	out := image.NewNRGBA(image.Rect(0, 0, nw, nh))
	for y := 0; y < nh; y++ {
		for x := 0; x < nw; x++ {
			dx, dy := float64(x)+0.5-float64(nw)/2, float64(y)+0.5-float64(nh)/2
			sx, sy := dx*c-dy*s+w/2, dx*s+dy*c+h/2
			if sx >= 0 && sy >= 0 && sx < w && sy < h {
				out.SetNRGBA(x, y, img.NRGBAAt(int(sx), int(sy)))
			}
		}
	}
	return out
}

// pasteFit clears a skin's cell and centres the sprite in it, scaled to fit.
func pasteFit(atlas *image.NRGBA, skin Skin, sprite *image.NRGBA) {
	sprite = crop(sprite, bbox(sprite))
	sw, sh := sprite.Rect.Dx(), sprite.Rect.Dy()
	f := math.Min(float64(skin.Width)/float64(sw), float64(skin.Height)/float64(sh))
	small := scaled(sprite, int(float64(sw)*f), int(float64(sh)*f))
	r := skin.Rect()
	draw.Draw(atlas, r, image.Transparent, image.Point{}, draw.Src)
	at := r.Min.Add(image.Pt((skin.Width-small.Rect.Dx())/2, (skin.Height-small.Rect.Dy())/2))
	draw.Draw(atlas, image.Rectangle{at, at.Add(small.Rect.Size())}, small, image.Point{}, draw.Src)
}

func tinted(img *image.NRGBA, r, g, b float64) *image.NRGBA {
	out := image.NewNRGBA(img.Rect)
	copy(out.Pix, img.Pix)
	for i := 0; i+3 < len(out.Pix); i += 4 {
		out.Pix[i] = uint8(float64(out.Pix[i]) * (0.45 + 0.55*r/255))
		out.Pix[i+1] = uint8(float64(out.Pix[i+1]) * (0.45 + 0.55*g/255))
		out.Pix[i+2] = uint8(float64(out.Pix[i+2]) * (0.45 + 0.55*b/255))
	}
	return out
}

// OriginalChicken composes the game's own chicken from its body and wing frames.
func OriginalChicken(d *Dat) (*image.NRGBA, error) {
	body, err := loadAtlas(d, "xmas-edition/gfx/chickenBody.tga")
	if err != nil {
		return nil, err
	}
	wings, err := loadAtlas(d, "gfx/chickenWings.tga")
	if err != nil {
		return nil, err
	}
	w, t := crop(wings.img, wings.skins[0].Rect()), crop(body.img, body.skins[0].Rect())
	hero := image.NewNRGBA(image.Rect(0, 0, max(w.Rect.Dx(), t.Rect.Dx()), t.Rect.Dy()+10))
	over := func(s *image.NRGBA, x, y int) {
		draw.Draw(hero, image.Rect(x, y, x+s.Rect.Dx(), y+s.Rect.Dy()), s, image.Point{}, draw.Over)
	}
	over(w, (hero.Rect.Dx()-w.Rect.Dx())/2, hero.Rect.Dy()-w.Rect.Dy())
	over(t, (hero.Rect.Dx()-t.Rect.Dx())/2, 0)
	return hero, nil
}

// ThemeEnemies draws a replacement character over every chicken frame. The wing
// layer holds the character and the body layer is cleared, so the original
// animation timing and hitboxes stay the same.
func ThemeEnemies(d *Dat, frames [2]*image.NRGBA) (map[string][]byte, error) {
	wings, err := loadAtlas(d, "gfx/chickenWings.tga")
	if err != nil {
		return nil, err
	}
	body, err := loadAtlas(d, "xmas-edition/gfx/chickenBody.tga")
	if err != nil {
		return nil, err
	}
	blink, err := loadAtlas(d, "xmas-edition/gfx/chickenBlink.tga")
	if err != nil {
		return nil, err
	}
	nw := image.NewNRGBA(wings.img.Rect)
	for n, s := range wings.skins {
		pasteFit(nw, s, frames[n/5%2])
	}
	return map[string][]byte{
		wings.name: WriteTGA(nw),
		body.name:  WriteTGA(image.NewNRGBA(body.img.Rect)),
		blink.name: WriteTGA(image.NewNRGBA(blink.img.Rect)),
	}, nil
}

// ReverseSprites makes the player a character and every chicken a spaceship
// flying nose-down, then trades eggs and the player's basic bullets. The rules
// stay the original game's: this changes who looks like what.
func ReverseSprites(d *Dat, hero *image.NRGBA) (map[string][]byte, error) {
	ship, err := loadAtlas(d, "gfx/spaceship.tga")
	if err != nil {
		return nil, err
	}
	ship360, err := loadAtlas(d, "gfx/spaceship360.tga")
	if err != nil {
		return nil, err
	}
	wings, err := loadAtlas(d, "gfx/chickenWings.tga")
	if err != nil {
		return nil, err
	}
	body, err := loadAtlas(d, "xmas-edition/gfx/chickenBody.tga")
	if err != nil {
		return nil, err
	}
	blink, err := loadAtlas(d, "xmas-edition/gfx/chickenBlink.tga")
	if err != nil {
		return nil, err
	}
	egg, err := loadAtlas(d, "gfx/egg.tga")
	if err != nil {
		return nil, err
	}
	bullet, err := loadAtlas(d, "gfx/bulletDefault.tga")
	if err != nil {
		return nil, err
	}
	enemyShip := rotated(crop(ship.img, ship.skins[0].Rect()), 180)

	newShip := image.NewNRGBA(ship.img.Rect)
	copy(newShip.Pix, ship.img.Pix)
	centre := float64(len(ship.skins)-1) / 2
	for n, s := range ship.skins {
		pasteFit(newShip, s, rotated(hero, (centre-float64(n))*2.5))
	}
	new360 := image.NewNRGBA(ship360.img.Rect)
	copy(new360.Pix, ship360.img.Pix)
	last := 0
	for n := range ship360.skins {
		last = max(last, n)
	}
	for n, s := range ship360.skins {
		pasteFit(new360, s, rotated(hero, -180*(1-float64(n)/float64(last))))
	}
	newWings := image.NewNRGBA(wings.img.Rect)
	for _, s := range wings.skins {
		pasteFit(newWings, s, enemyShip)
	}
	// Body frames carry each chicken type's colour; it becomes the ship's cockpit.
	newBody := image.NewNRGBA(body.img.Rect)
	shipCrop := crop(enemyShip, bbox(enemyShip))
	for _, s := range body.skins {
		var r, g, b, n float64
		for y := s.Top + s.Height/2; y < s.Top+s.Height; y++ {
			for x := s.Left; x < s.Left+s.Width; x++ {
				if c := body.img.NRGBAAt(x, y); c.A > 200 {
					r, g, b, n = r+float64(c.R), g+float64(c.G), b+float64(c.B), n+1
				}
			}
		}
		if n == 0 {
			r, g, b, n = 255, 255, 255, 1
		}
		pasteFit(newBody, s, tinted(shipCrop, r/n, g/n, b/n))
	}
	newEgg := image.NewNRGBA(egg.img.Rect)
	shot := crop(bullet.img, bullet.skins[0].Rect())
	for _, s := range egg.skins {
		pasteFit(newEgg, s, rotated(shot, 90))
	}
	newBullet := image.NewNRGBA(bullet.img.Rect)
	eggImg := crop(egg.img, egg.skins[2].Rect())
	for _, s := range bullet.skins {
		pasteFit(newBullet, s, eggImg)
	}
	return map[string][]byte{
		ship.name:    WriteTGA(newShip),
		ship360.name: WriteTGA(new360),
		wings.name:   WriteTGA(newWings),
		body.name:    WriteTGA(newBody),
		blink.name:   WriteTGA(image.NewNRGBA(blink.img.Rect)),
		egg.name:     WriteTGA(newEgg),
		bullet.name:  WriteTGA(newBullet),
	}, nil
}
