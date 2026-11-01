package mod

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"image"
	"image/draw"
	"image/png"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
	"syscall"

	xdraw "golang.org/x/image/draw"
)

const (
	esrganURL    = "https://github.com/xinntao/Real-ESRGAN/releases/download/v0.2.5.0/realesrgan-ncnn-vulkan-20220424-windows.zip"
	esrganSHA256 = "abc02804e17982a3be33675e4d471e91ea374e65b70167abc09e31acb412802d"
	UpscaleScale = 4
	esrganModel  = "realesrgan-x4plus-anime"
)

// The loader hook: textures with odd width and height are marked as upscaled, and
// their logical size becomes the real one divided by the scale. Width mod 4 = 3
// means 4x and 1 means 2x. Everything that sizes a sprite or computes its texture
// coordinates uses the logical size, so the game draws the same sprite in the same
// place, only from more pixels.
const (
	hookSite = 0x4a54d9
	hookBack = 0x4a54e0
	hookCave = 0x51e500 // unused padding at the end of .text
)

// HookUpscaleLoader patches an exe for loading marked textures.
func HookUpscaleLoader(exe []byte) []byte {
	out := append([]byte(nil), exe...)
	cave := []byte{
		0x8B, 0x84, 0x24, 0xA4, 0, 0, 0, // mov eax,[esp+0xa4]  image width
		0x8B, 0x8C, 0x24, 0xA8, 0, 0, 0, // mov ecx,[esp+0xa8]  image height
		0xF6, 0xC1, 0x01, // test cl,1
		0x74, 0x18, // jz done
		0xA8, 0x01, // test al,1
		0x74, 0x14, // jz done
		0xA8, 0x02, // test al,2
		0x74, 0x0A, // jz half
		0xC1, 0x6F, 0x78, 0x02, // shr dword [edi+0x78],2
		0xC1, 0x6F, 0x7C, 0x02, // shr dword [edi+0x7c],2
		0xEB, 0x06, // jmp done
		0xD1, 0x6F, 0x78, // half: shr dword [edi+0x78],1
		0xD1, 0x6F, 0x7C, // shr dword [edi+0x7c],1
		// done: the instructions the hook replaced
		0x51,       // push ecx
		0x8B, 0xD4, // mov edx,esp
		0x89, 0x64, 0x24, 0x2C, // mov [esp+0x2c],esp
	}
	jmpBack := hookBack - (hookCave + len(cave) + 5)
	cave = append(cave, 0xE9)
	cave = binary.LittleEndian.AppendUint32(cave, uint32(int32(jmpBack)))
	copy(out[hookCave-base:], cave)
	site := hookSite - base
	out[site] = 0xE9
	binary.LittleEndian.PutUint32(out[site+1:], uint32(int32(hookCave-(hookSite+5))))
	out[site+5], out[site+6] = 0x90, 0x90
	return out
}

// Effects are soft gradients, where a model adds cost and nothing else. Fonts
// need their exact pixels for the Hebrew glyphs.
var upscaleSkip = []string{"circleblur", "ring", "flare", "spark", "plasma", "debug", "taperbeam", "lightning", "laser",
	"exhaust", "energypull", "glint", "warningstrip", "particle", "crosshair", "progress", "underconstruction",
	"network", "smoke", "splat", "thicklaser", "mapmarker", "dashedline"}

// The engine rejects textures over 2048, so large atlases get 2x and the largest none.
const maxTextureSide = 2048

func upscaleFactor(w, h int) int {
	switch m := max(w, h); {
	case m*4+3 <= maxTextureSide:
		return 4
	case m*2+1 <= maxTextureSide:
		return 2
	}
	return 1
}

// UpscaleCandidates lists the textures worth upscaling.
func UpscaleCandidates(d *Dat) []string {
	var out []string
	for _, n := range d.Names {
		l := strings.ToLower(n)
		if !strings.HasSuffix(l, ".tga") || strings.HasPrefix(l, "system/") {
			continue
		}
		skip := false
		base := path.Base(l)
		for _, s := range upscaleSkip {
			if strings.Contains(base, s) {
				skip = true
			}
		}
		if b := d.Get(n); skip || len(b) < 18 {
			continue
		} else if w, h := binary.LittleEndian.Uint16(b[12:]), binary.LittleEndian.Uint16(b[14:]); upscaleFactor(int(w), int(h)) == 1 {
			continue
		}
		out = append(out, n)
	}
	return out
}

// EnsureESRGAN unpacks Real-ESRGAN into dir and returns its exe path.
func EnsureESRGAN(dir string) (string, error) {
	exe := filepath.Join(dir, "realesrgan-ncnn-vulkan.exe")
	if _, err := os.Stat(exe); err == nil {
		return exe, nil
	}
	resp, err := http.Get(esrganURL)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return "", fmt.Errorf("Real-ESRGAN download: %s", resp.Status)
	}
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(b)
	if hex.EncodeToString(sum[:]) != esrganSHA256 {
		return "", fmt.Errorf("Real-ESRGAN checksum mismatch")
	}
	z, err := zip.NewReader(bytes.NewReader(b), int64(len(b)))
	if err != nil {
		return "", err
	}
	for _, f := range z.File {
		name := filepath.Base(f.Name)
		isModel := strings.HasPrefix(f.Name, "models/") && strings.HasPrefix(name, esrganModel+".")
		if !isModel && !strings.HasSuffix(name, ".dll") && name != filepath.Base(exe) {
			continue
		}
		target := filepath.Join(dir, name)
		if isModel {
			target = filepath.Join(dir, "models", name)
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return "", err
		}
		r, err := f.Open()
		if err != nil {
			return "", err
		}
		data, err := io.ReadAll(r)
		r.Close()
		if err != nil {
			return "", err
		}
		if err := os.WriteFile(target, data, 0o755); err != nil {
			return "", err
		}
	}
	return exe, nil
}

func tgaKey(b []byte) string {
	s := sha256.Sum256(b)
	return "v2-" + hex.EncodeToString(s[:12])
}

// Upscale returns repl-style entries for the named textures, using the cache and
// running the model only on what is missing. Sources come from repl when the
// player's choices already replaced them. progress receives done and total.
func Upscale(d *Dat, names []string, repl map[string][]byte, esrgan, cache string, progress func(done, total int)) (map[string][]byte, error) {
	if err := os.MkdirAll(cache, 0o755); err != nil {
		return nil, err
	}
	in := filepath.Join(cache, "in")
	work := filepath.Join(cache, "out")
	_ = os.RemoveAll(in)
	_ = os.RemoveAll(work)
	if err := os.MkdirAll(in, 0o755); err != nil {
		return nil, err
	}
	if err := os.MkdirAll(work, 0o755); err != nil {
		return nil, err
	}
	src := map[string][]byte{}
	origSize := map[string]image.Point{}
	missing := 0
	for _, n := range names {
		b, ok := repl[n]
		if !ok {
			b = d.Get(n)
		}
		src[n] = b
		k := tgaKey(b)
		if _, err := os.Stat(filepath.Join(cache, k+".tga")); err == nil {
			continue
		}
		im, err := ReadTGA(b)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", n, err)
		}
		origSize[k] = im.Rect.Size()
		f, err := os.Create(filepath.Join(in, k+".png"))
		if err != nil {
			return nil, err
		}
		err = png.Encode(f, im)
		f.Close()
		if err != nil {
			return nil, err
		}
		missing++
	}
	done := 0
	for missing > 0 {
		entries, err := os.ReadDir(in)
		if err != nil {
			return nil, err
		}
		if len(entries) == 0 {
			break
		}
		e := entries[0]
		k := strings.TrimSuffix(e.Name(), ".png")
		p := filepath.Join(in, e.Name())
		o := filepath.Join(work, e.Name())
		c := exec.Command(esrgan, "-i", p, "-o", o, "-n", esrganModel, "-s", "4", "-f", "png")
		c.Dir = filepath.Dir(esrgan)
		c.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: 0x08000000}
		if msg, err := c.CombinedOutput(); err != nil {
			return nil, fmt.Errorf("Real-ESRGAN: %w: %s", err, tail(msg))
		}
		if err := finishUpscaled(o, filepath.Join(cache, k+".tga"), origSize[k]); err != nil {
			return nil, err
		}
		_ = os.Remove(p)
		_ = os.Remove(o)
		done++
		if progress != nil {
			progress(done, missing)
		}
	}
	out := map[string][]byte{}
	for _, n := range names {
		b, err := os.ReadFile(filepath.Join(cache, tgaKey(src[n])+".tga"))
		if err != nil {
			return nil, err
		}
		out[n] = b
	}
	return out, nil
}

func tail(b []byte) string {
	if len(b) > 300 {
		b = b[len(b)-300:]
	}
	return string(b)
}

// finishUpscaled scales the model's 4x output to the texture's factor and adds the
// transparent rows and columns that mark it for the loader hook.
func finishUpscaled(pngPath, tgaPath string, orig image.Point) error {
	f, err := os.Open(pngPath)
	if err != nil {
		return err
	}
	defer f.Close()
	im, err := png.Decode(f)
	if err != nil {
		return err
	}
	s := upscaleFactor(orig.X, orig.Y)
	pad := 1
	if s == 4 {
		pad = 3
	}
	w, h := orig.X*s, orig.Y*s
	out := image.NewNRGBA(image.Rect(0, 0, w+pad, h+pad))
	xdraw.CatmullRom.Scale(out, image.Rect(0, 0, w, h), im, im.Bounds(), draw.Src, nil)
	return os.WriteFile(tgaPath, WriteTGA(out), 0o644)
}
