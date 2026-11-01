// Command ci3-hebrew-4k is a Hebrew launcher for Chicken Invaders 3 Xmas. It asks
// for the game mode, characters and sounds, builds the matching files from the
// player's own install, and starts the original game engine.
package main

import (
	"bytes"
	"embed"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"image"
	"image/color"
	"image/jpeg"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/dor2500/ci3-hebrew-4k/internal/art"
	"github.com/dor2500/ci3-hebrew-4k/internal/mod"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

//go:embed assets/fonts/Rubik.ttf
var fontTTF []byte

//go:embed assets/he.json assets/sites.json
var textFS embed.FS

//go:embed assets/packs assets/voice
var soundFS embed.FS

const arialBold = `C:\Windows\Fonts\arialbd.ttf`

var packs = []struct{ id, name string }{
	{"original", "מקורי CI3"},
	{"blocks", "סגנון מיינקראפט"},
	{"kombat", "סגנון מורטל קומבט"},
	{"mix", "מיקס רטרו ומודרני"},
}

type theme struct {
	name   string
	frames [2]*image.NRGBA // nil for the game's own chickens
}

type Config struct {
	Mode, Theme, Sound int
	VoiceOff           bool
	UpscaleOff         bool
}

type App struct {
	txt            *Text
	dat            *mod.Dat
	install, home  string
	exeDir         string
	cfg            Config
	themes         []theme
	row            int
	tick           int
	bg, logo, ship *ebiten.Image
	previews       []*ebiten.Image
	status         string
	upscaleNote    string
	failure        string
	building       bool
	done           chan error
	launch         bool
	autoBuild      bool
	shot           string
	noRun          bool
}

func toNRGBA(m *image.RGBA) *image.NRGBA {
	// Unpremultiply the painter's output for the game's straight-alpha textures.
	out := image.NewNRGBA(m.Rect)
	for i := 0; i+3 < len(m.Pix); i += 4 {
		a := int(m.Pix[i+3])
		out.Pix[i+3] = uint8(a)
		if a > 0 {
			out.Pix[i] = uint8(min(255, int(m.Pix[i])*255/a))
			out.Pix[i+1] = uint8(min(255, int(m.Pix[i+1])*255/a))
			out.Pix[i+2] = uint8(min(255, int(m.Pix[i+2])*255/a))
		}
	}
	return out
}

func newApp(exeDir string) (*App, error) {
	txt, err := NewText(fontTTF)
	if err != nil {
		return nil, err
	}
	home, err := os.UserCacheDir()
	if err != nil {
		return nil, err
	}
	a := &App{txt: txt, exeDir: exeDir, home: filepath.Join(home, "CI3Hebrew4K")}
	if err := os.MkdirAll(a.home, 0o755); err != nil {
		return nil, err
	}
	if b, err := os.ReadFile(filepath.Join(a.home, "launcher.json")); err == nil {
		_ = json.Unmarshal(b, &a.cfg)
	}
	a.install = mod.FindInstall(exeDir)
	if a.install == "" {
		a.failure = "לא נמצא המשחק. יש להתקין אותו, או לכתוב את נתיב התיקייה שלו בקובץ ci3_path.txt"
		return a, nil
	}
	blob, err := os.ReadFile(filepath.Join(a.install, "CI3Xmas.dat"))
	if err != nil {
		return nil, err
	}
	if a.dat, err = mod.ParseDat(blob); err != nil {
		return nil, err
	}
	a.themes = []theme{{name: "תרנגולות מקוריות"}}
	for _, id := range []string{"plumber", "robot", "ghost"} {
		for _, t := range art.Themes {
			if t.ID == id {
				a.themes = append(a.themes, theme{t.Name, [2]*image.NRGBA{toNRGBA(art.Creature(t, 256, 0)), toNRGBA(art.Creature(t, 256, 1))}})
			}
		}
	}
	// Any PNG in skins/ next to the exe is one more choice (Mario, for example).
	if m, _ := filepath.Glob(filepath.Join(exeDir, "skins", "*.png")); m != nil {
		for _, p := range m {
			f, err := os.Open(p)
			if err != nil {
				continue
			}
			im, _, err := image.Decode(f)
			f.Close()
			if err != nil {
				continue
			}
			n := image.NewNRGBA(im.Bounds())
			for y := im.Bounds().Min.Y; y < im.Bounds().Max.Y; y++ {
				for x := im.Bounds().Min.X; x < im.Bounds().Max.X; x++ {
					n.Set(x, y, im.At(x, y))
				}
			}
			name := strings.TrimSuffix(filepath.Base(p), filepath.Ext(p))
			a.themes = append(a.themes, theme{name, [2]*image.NRGBA{n, n}})
		}
	}
	a.cfg.Theme = max(0, min(a.cfg.Theme, len(a.themes)-1))
	a.cfg.Sound = max(0, min(a.cfg.Sound, len(packs)-1))
	a.cfg.Mode = max(0, min(a.cfg.Mode, 1))

	// The screen uses the game's own art.
	if im, err := jpeg.Decode(bytes.NewReader(a.dat.Get("gfx/galaxy.jpg"))); err == nil {
		a.bg = ebiten.NewImageFromImage(im)
	}
	if b := a.dat.Get("xmas-edition/gfx/logo.tga"); b != nil {
		if im, err := mod.ReadTGA(b); err == nil {
			// Skin 0 is the red glow and skin 1 the letters, as the game layers them.
			skins := mod.ParseIni(a.dat.Get("xmas-edition/gfx/logo.tga.ini"))
			glow, letters := skins[0], skins[1]
			l := ebiten.NewImage(glow.Width, glow.Height)
			l.DrawImage(ebiten.NewImageFromImage(im.SubImage(glow.Rect())), nil)
			op := &ebiten.DrawImageOptions{}
			op.GeoM.Translate(float64(glow.Width-letters.Width)/2, float64(glow.Height-letters.Height)/2)
			l.DrawImage(ebiten.NewImageFromImage(im.SubImage(letters.Rect())), op)
			a.logo = l
		}
	}
	if im, err := mod.ReadTGA(a.dat.Get("gfx/spaceship.tga")); err == nil {
		a.ship = ebiten.NewImageFromImage(im.SubImage(mod.ParseIni(a.dat.Get("gfx/spaceship.tga.ini"))[7].Rect()))
	}
	hero, err := mod.OriginalChicken(a.dat)
	if err != nil {
		return nil, err
	}
	for i, t := range a.themes {
		if i == 0 {
			a.previews = append(a.previews, ebiten.NewImageFromImage(hero))
		} else {
			a.previews = append(a.previews, ebiten.NewImageFromImage(t.frames[0]))
		}
	}
	return a, nil
}

// build creates the game folder for the current selection.
func (a *App) build() error {
	game := filepath.Join(a.home, "game")
	a.status = "מכין את התיקייה..."
	if err := mod.PrepareFolder(a.install, game); err != nil {
		return err
	}
	// Render at the monitor's full native size, whatever its shape.
	sw, sh := ebiten.Monitor().Size()
	s := ebiten.Monitor().DeviceScaleFactor()
	resW, resH := max(int(float64(sw)*s), 800), max(int(float64(sh)*s), 600)
	if err := mod.ConfigureGame(game, resW, resH); err != nil {
		return err
	}
	a.status = "מתקין תמיכה ב-4K..."
	if err := mod.InstallDGV(game, a.home, resW, resH); err != nil {
		return err
	}
	a.status = "מתרגם לעברית..."
	var sites []mod.StringSite
	b, _ := textFS.ReadFile("assets/sites.json")
	if err := json.Unmarshal(b, &sites); err != nil {
		return err
	}
	var he map[string]string
	b, _ = textFS.ReadFile("assets/he.json")
	if err := json.Unmarshal(b, &he); err != nil {
		return err
	}
	orig, err := os.ReadFile(filepath.Join(a.install, "CI3Xmas.exe"))
	if err != nil {
		return err
	}
	exe, err := mod.PatchExe(orig, sites, he)
	if err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(game, "CI3Xmas.exe"), mod.HookUpscaleLoader(exe), 0o755); err != nil {
		return err
	}
	ttf, err := os.ReadFile(arialBold)
	if err != nil {
		return err
	}
	repl, err := mod.BuildFonts(a.dat, ttf)
	if err != nil {
		return err
	}
	a.status = "מרכיב דמויות וצלילים..."
	if err := a.applyChoices(repl); err != nil {
		return err
	}
	a.upscaleNote = ""
	if !a.cfg.UpscaleOff {
		// A machine that cannot run the model still gets a working game.
		if err := a.applyUpscale(repl); err != nil {
			a.upscaleNote = err.Error()
			_ = os.WriteFile(filepath.Join(a.home, "upscale-error.txt"), []byte(err.Error()), 0o644)
		}
	}
	return os.WriteFile(filepath.Join(game, "CI3Xmas.dat"), a.dat.Rebuild(repl), 0o644)
}

func (a *App) applyUpscale(repl map[string][]byte) error {
	a.status = "מכין שדרוג AI..."
	exe, err := mod.EnsureESRGAN(filepath.Join(a.home, "esrgan"))
	if err != nil {
		return err
	}
	up, err := mod.Upscale(a.dat, mod.UpscaleCandidates(a.dat), repl, exe, filepath.Join(a.home, "upscaled"), func(done, total int) {
		a.status = fmt.Sprintf("משדרג גרפיקה: %d מתוך %d (רק בפעם הראשונה)", done, total)
	})
	if err != nil {
		return err
	}
	for k, v := range up {
		repl[k] = v
	}
	return nil
}

func (a *App) applyChoices(repl map[string][]byte) error {
	t := a.themes[a.cfg.Theme]
	merge := func(m map[string][]byte, err error) error {
		if err != nil {
			return err
		}
		for k, v := range m {
			repl[k] = v
		}
		return nil
	}
	if a.cfg.Mode == 1 {
		hero := t.frames[0]
		if hero == nil {
			h, err := mod.OriginalChicken(a.dat)
			if err != nil {
				return err
			}
			hero = h
		}
		if err := merge(mod.ReverseSprites(a.dat, hero)); err != nil {
			return err
		}
	} else if t.frames[0] != nil {
		if err := merge(mod.ThemeEnemies(a.dat, t.frames)); err != nil {
			return err
		}
	}
	pack := packs[a.cfg.Sound].id
	if pack != "original" {
		for event, names := range mod.SoundMap {
			b, err := soundFS.ReadFile("assets/packs/" + pack + "/" + event + ".ogg")
			if err != nil {
				return err
			}
			for _, n := range names {
				repl["sfx/"+n+".ogg"] = b
			}
		}
	}
	// Files in soundpacks/<id>/ named like the game's own replace them.
	dir := filepath.Join(a.exeDir, "soundpacks", pack)
	if m, _ := filepath.Glob(filepath.Join(dir, "*.ogg")); m != nil {
		for _, p := range m {
			name := "sfx/" + filepath.Base(p)
			if a.dat.Get(name) != nil {
				if b, err := os.ReadFile(p); err == nil {
					repl[name] = b
				}
			}
		}
	}
	if !a.cfg.VoiceOff {
		for name, id := range mod.VoiceMap {
			b, err := soundFS.ReadFile("assets/voice/" + id + ".ogg")
			if err != nil {
				return err
			}
			repl[name] = b
		}
	}
	return nil
}

func (a *App) startBuild() {
	a.building = true
	a.done = make(chan error, 1)
	if b, err := json.Marshal(a.cfg); err == nil {
		_ = os.WriteFile(filepath.Join(a.home, "launcher.json"), b, 0o644)
	}
	go func() { a.done <- a.build() }()
}

const (
	rowMode = iota
	rowTheme
	rowSound
	rowVoice
	rowUpscale
	rowStart
	numRows
)

var rowTitles = []string{"מצב משחק", "דמויות", "חבילת צלילים", "קריינות בעברית", "שדרוג AI לגרפיקה"}

func (a *App) options(r int) []string {
	switch r {
	case rowMode:
		return []string{"קלאסי: חללית נגד תרנגולות", "הפוך: תרנגולת נגד חלליות"}
	case rowTheme:
		o := []string{}
		for _, t := range a.themes {
			o = append(o, t.name)
		}
		return o
	case rowSound:
		o := []string{}
		for _, p := range packs {
			o = append(o, p.name)
		}
		return o
	case rowVoice, rowUpscale:
		return []string{"פעיל", "כבוי"}
	}
	return nil
}

func (a *App) value(r int) int {
	switch r {
	case rowMode:
		return a.cfg.Mode
	case rowTheme:
		return a.cfg.Theme
	case rowSound:
		return a.cfg.Sound
	case rowVoice:
		if a.cfg.VoiceOff {
			return 1
		}
	case rowUpscale:
		if a.cfg.UpscaleOff {
			return 1
		}
	}
	return 0
}

func (a *App) set(r, v int) {
	switch r {
	case rowMode:
		a.cfg.Mode = v
	case rowTheme:
		a.cfg.Theme = v
	case rowSound:
		a.cfg.Sound = v
	case rowVoice:
		a.cfg.VoiceOff = v == 1
	case rowUpscale:
		a.cfg.UpscaleOff = v == 1
	}
}

type chip struct {
	label      string
	x, y, w, h float32
	row, idx   int
}

func (a *App) chips() []chip {
	var out []chip
	const right, left = 1830.0, 620.0
	y := 410.0
	for r := 0; r < rowStart; r++ {
		x := right - 290.0
		for i, l := range a.options(r) {
			w := a.txt.Width(l, 34) + 44
			if x-w < left {
				x = right - 290
				y += 66
			}
			out = append(out, chip{l, float32(x - w), float32(y), float32(w), 56, r, i})
			x -= w + 14
		}
		y += 82
	}
	return out
}

func startRect() (x, y, w, h float32) { return 1000, 880, 420, 100 }

func in(mx, my int, x, y, w, h float32) bool {
	return float32(mx) >= x && float32(mx) < x+w && float32(my) >= y && float32(my) < y+h
}

func (a *App) Update() error {
	a.tick++
	if a.launch {
		return ebiten.Termination
	}
	if a.building {
		select {
		case err := <-a.done:
			a.building = false
			if err != nil {
				a.failure = "הבנייה נכשלה: " + err.Error()
				return nil
			}
			a.launch = true
		default:
		}
		return nil
	}
	if a.failure != "" || a.dat == nil {
		if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
			return ebiten.Termination
		}
		return nil
	}
	if a.autoBuild && a.tick == 5 {
		a.startBuild()
		return nil
	}
	j := inpututil.IsKeyJustPressed
	if j(ebiten.KeyEscape) {
		return ebiten.Termination
	}
	if j(ebiten.KeyArrowUp) {
		a.row = (a.row + numRows - 1) % numRows
	}
	if j(ebiten.KeyArrowDown) {
		a.row = (a.row + 1) % numRows
	}
	if n := len(a.options(a.row)); n > 0 {
		// Options run right to left, so the left key moves to the next one.
		if j(ebiten.KeyArrowLeft) {
			a.set(a.row, (a.value(a.row)+1)%n)
		}
		if j(ebiten.KeyArrowRight) {
			a.set(a.row, (a.value(a.row)+n-1)%n)
		}
	}
	mx, my := ebiten.CursorPosition()
	click := inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft)
	if click {
		for _, c := range a.chips() {
			if in(mx, my, c.x, c.y, c.w, c.h) {
				a.row = c.row
				a.set(c.row, c.idx)
			}
		}
	}
	x, y, w, h := startRect()
	if click && in(mx, my, x, y, w, h) {
		a.startBuild()
	}
	if j(ebiten.KeyEnter) || j(ebiten.KeyNumpadEnter) {
		if a.row == rowStart {
			a.startBuild()
		} else {
			a.row++
		}
	}
	return nil
}

func (a *App) spr(dst, im *ebiten.Image, cx, cy, size float64) {
	b := im.Bounds()
	s := size / float64(max(b.Dx(), b.Dy()))
	op := &ebiten.DrawImageOptions{Filter: ebiten.FilterLinear}
	op.GeoM.Translate(-float64(b.Dx())/2, -float64(b.Dy())/2)
	op.GeoM.Scale(s, s)
	op.GeoM.Translate(cx, cy)
	dst.DrawImage(im, op)
}

var (
	gold  = color.RGBA{255, 208, 74, 255}
	white = color.RGBA{245, 245, 250, 255}
	dim   = color.RGBA{170, 180, 205, 255}
	panel = color.RGBA{10, 14, 36, 225}
	chipC = color.RGBA{36, 44, 88, 240}
	sel   = color.RGBA{214, 84, 30, 255}
)

func (a *App) Draw(s *ebiten.Image) {
	s.Fill(color.RGBA{6, 8, 24, 255})
	if a.bg != nil {
		op := &ebiten.DrawImageOptions{Filter: ebiten.FilterLinear}
		b := a.bg.Bounds()
		op.GeoM.Scale(1920/float64(b.Dx()), 1080/float64(b.Dy()))
		s.DrawImage(a.bg, op)
	}
	if a.logo != nil {
		a.spr(s, a.logo, 960, 170, 620)
	}
	if a.failure != "" {
		a.txt.Draw(s, a.failure, 960, 620, 40, AlignCenter, color.RGBA{255, 120, 100, 255})
		a.txt.Draw(s, "Esc: יציאה", 960, 700, 34, AlignCenter, dim)
		a.finish(s)
		return
	}
	vector.FillRect(s, 590, 360, 1270, 650, panel, true)
	vector.FillRect(s, 60, 360, 500, 650, panel, true)
	last := -1
	for _, c := range a.chips() {
		if c.row != last {
			col := dim
			if a.row == c.row {
				col = gold
			}
			a.txt.Draw(s, rowTitles[c.row], 1830, float64(c.y)+28, 36, AlignRight, col)
			last = c.row
		}
		f := chipC
		if a.value(c.row) == c.idx {
			f = sel
		}
		vector.FillRect(s, c.x, c.y, c.w, c.h, f, true)
		if a.row == c.row && a.value(c.row) == c.idx {
			vector.StrokeRect(s, c.x, c.y, c.w, c.h, 4, gold, true)
		}
		a.txt.Draw(s, c.label, float64(c.x+c.w/2), float64(c.y+c.h/2), 34, AlignCenter, white)
	}
	x, y, w, h := startRect()
	vector.FillRect(s, x, y, w, h, color.RGBA{44, 150, 76, 255}, true)
	if a.row == rowStart {
		vector.StrokeRect(s, x, y, w, h, 5, gold, true)
	}
	label := "התחל!"
	if a.building {
		label = "רגע..."
	}
	a.txt.Draw(s, label, float64(x+w/2), float64(y+h/2), 58, AlignCenter, white)
	if a.building {
		a.txt.Draw(s, a.status, 980, 930, 30, AlignRight, gold)
	}

	if len(a.previews) > a.cfg.Theme {
		pv := a.previews[a.cfg.Theme]
		a.txt.Draw(s, "תצוגה מקדימה", 310, 410, 36, AlignCenter, gold)
		if a.cfg.Mode == 0 {
			for i := 0; i < 3; i++ {
				a.spr(s, pv, 150+float64(i)*160, 540+float64(i%2)*50, 140)
			}
			if a.ship != nil {
				a.spr(s, a.ship, 310, 830, 150)
			}
			a.txt.Draw(s, "במשחק: החללית נגד הדמויות", 310, 960, 28, AlignCenter, white)
		} else {
			a.spr(s, pv, 310, 830, 170)
			if a.ship != nil {
				for i := 0; i < 3; i++ {
					a.spr(s, a.ship, 150+float64(i)*160, 540+float64(i%2)*50, 130)
				}
			}
			a.txt.Draw(s, "במשחק: הדמות נגד החלליות", 310, 960, 28, AlignCenter, white)
		}
	}
	a.txt.Draw(s, "חצים או עכבר: בחירה   •   Enter: אישור   •   Esc: יציאה", 960, 1045, 28, AlignCenter, dim)
	a.finish(s)
}

func (a *App) finish(s *ebiten.Image) {
	if a.shot != "" && a.tick == 30 {
		saveShot(s, a.shot)
	}
}

func (a *App) Layout(_, _ int) (int, int) { return 1920, 1080 }

func main() {
	var f struct {
		mode, themeI, sound int
		build, noRun        bool
		shot                string
		duration            int
	}
	flag.IntVar(&f.mode, "mode", -1, "0 classic, 1 reverse")
	flag.IntVar(&f.themeI, "theme", -1, "character index")
	flag.IntVar(&f.sound, "sound", -1, "sound pack index")
	flag.BoolVar(&f.build, "build", false, "build without showing choices")
	flag.BoolVar(&f.noRun, "no-run", false, "build only, do not start the game")
	flag.StringVar(&f.shot, "shot", "", "save the launcher screen to a PNG")
	flag.Parse()

	exeDir := "."
	if e, err := os.Executable(); err == nil {
		exeDir = filepath.Dir(e)
	}
	a, err := newApp(exeDir)
	if err != nil {
		log.Fatal(err)
	}
	if f.mode >= 0 {
		a.cfg.Mode = f.mode
	}
	if f.themeI >= 0 && f.themeI < len(a.themes) {
		a.cfg.Theme = f.themeI
	}
	if f.sound >= 0 {
		a.cfg.Sound = f.sound
	}
	a.autoBuild, a.noRun, a.shot = f.build, f.noRun, f.shot

	ebiten.SetWindowTitle("פולשי התרנגולות 3: נקמת החלמון")
	ebiten.SetWindowSize(1280, 720)
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)
	ebiten.SetRunnableOnUnfocused(true)
	if err := ebiten.RunGame(a); err != nil && !errors.Is(err, ebiten.Termination) {
		log.Fatal(err)
	}
	if a.launch && !a.noRun {
		game := filepath.Join(a.home, "game")
		c := exec.Command(filepath.Join(game, "CI3Xmas.exe"))
		c.Dir = game
		if err := c.Start(); err != nil {
			log.Fatal(fmt.Errorf("start game: %w", err))
		}
	}
}
