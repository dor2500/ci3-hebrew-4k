// Package mod rebuilds a player's own Chicken Invaders 3 Xmas install: it reads
// the data file, swaps resources, and patches the Hebrew text into the exe. No
// game files are stored in this repository.
package mod

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"image"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

const (
	key   = 0xCC
	entry = 88
)

type Dat struct {
	blob  []byte
	Names []string
	index map[string][2]int
}

func ParseDat(blob []byte) (*Dat, error) {
	if len(blob) < 4 {
		return nil, fmt.Errorf("dat too short")
	}
	n := int(binary.LittleEndian.Uint32(blob))
	if n <= 0 || 4+n*entry > len(blob) {
		return nil, fmt.Errorf("bad dat count %d", n)
	}
	d := &Dat{blob: blob, index: map[string][2]int{}}
	for i := 0; i < n; i++ {
		o := 4 + i*entry
		name := make([]byte, 0, 80)
		for _, c := range blob[o : o+80] {
			if c^key == 0 {
				break
			}
			name = append(name, c^key)
		}
		off := int(binary.LittleEndian.Uint32(blob[o+80:]))
		size := int(binary.LittleEndian.Uint32(blob[o+84:]))
		if off < 0 || size < 0 || off+size > len(blob) {
			return nil, fmt.Errorf("bad entry %s", name)
		}
		d.Names = append(d.Names, string(name))
		d.index[string(name)] = [2]int{off, size}
	}
	return d, nil
}

func (d *Dat) Get(name string) []byte {
	e, ok := d.index[name]
	if !ok {
		return nil
	}
	return d.blob[e[0] : e[0]+e[1]]
}

// Rebuild writes a new dat with the same table order and replaced contents.
func (d *Dat) Rebuild(repl map[string][]byte) []byte {
	n := len(d.Names)
	var body bytes.Buffer
	table := append([]byte(nil), d.blob[:4+n*entry]...)
	cursor := len(table)
	for i, name := range d.Names {
		data, ok := repl[name]
		if !ok {
			data = d.Get(name)
		}
		o := 4 + i*entry
		binary.LittleEndian.PutUint32(table[o+80:], uint32(cursor))
		binary.LittleEndian.PutUint32(table[o+84:], uint32(len(data)))
		body.Write(data)
		cursor += len(data)
	}
	return append(table, body.Bytes()...)
}

// ReadTGA decodes the game's uncompressed 32-bit textures to straight-alpha RGBA.
func ReadTGA(b []byte) (*image.NRGBA, error) {
	if len(b) < 18 || b[2] != 2 || b[16] != 32 {
		return nil, fmt.Errorf("unsupported tga")
	}
	w, h := int(binary.LittleEndian.Uint16(b[12:])), int(binary.LittleEndian.Uint16(b[14:]))
	start := 18 + int(b[0])
	if len(b) < start+w*h*4 {
		return nil, fmt.Errorf("short tga")
	}
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	topDown := b[17]&0x20 != 0
	for y := 0; y < h; y++ {
		sy := h - 1 - y
		if topDown {
			sy = y
		}
		src := b[start+sy*w*4:]
		dst := img.Pix[y*img.Stride:]
		for x := 0; x < w; x++ {
			dst[x*4], dst[x*4+1], dst[x*4+2], dst[x*4+3] = src[x*4+2], src[x*4+1], src[x*4], src[x*4+3]
		}
	}
	return img, nil
}

func WriteTGA(img *image.NRGBA) []byte {
	w, h := img.Rect.Dx(), img.Rect.Dy()
	out := make([]byte, 18+w*h*4)
	out[2] = 2
	binary.LittleEndian.PutUint16(out[12:], uint16(w))
	binary.LittleEndian.PutUint16(out[14:], uint16(h))
	out[16], out[17] = 32, 8
	for y := 0; y < h; y++ {
		src := img.Pix[y*img.Stride:]
		dst := out[18+(h-1-y)*w*4:]
		for x := 0; x < w; x++ {
			dst[x*4], dst[x*4+1], dst[x*4+2], dst[x*4+3] = src[x*4+2], src[x*4+1], src[x*4], src[x*4+3]
		}
	}
	return out
}

type Skin struct {
	Left, Top, Width, Height int
	Extra                    [][2]string // other keys, kept in file order
}

func (s Skin) Rect() image.Rectangle {
	return image.Rect(s.Left, s.Top, s.Left+s.Width, s.Top+s.Height)
}

var (
	skinRe = regexp.MustCompile(`\[Skin (\d+)\]([^\[]*)`)
	kvRe   = regexp.MustCompile(`(\w+)=(-?\d+)`)
)

func ParseIni(text []byte) map[int]Skin {
	out := map[int]Skin{}
	for _, m := range skinRe.FindAllSubmatch(text, -1) {
		n, _ := strconv.Atoi(string(m[1]))
		var s Skin
		for _, kv := range kvRe.FindAllSubmatch(m[2], -1) {
			v, _ := strconv.Atoi(string(kv[2]))
			switch string(kv[1]) {
			case "left":
				s.Left = v
			case "top":
				s.Top = v
			case "width":
				s.Width = v
			case "height":
				s.Height = v
			default:
				s.Extra = append(s.Extra, [2]string{string(kv[1]), string(kv[2])})
			}
		}
		out[n] = s
	}
	return out
}

func WriteIni(skins map[int]Skin) []byte {
	keys := make([]int, 0, len(skins))
	for k := range skins {
		keys = append(keys, k)
	}
	sort.Ints(keys)
	var b strings.Builder
	for _, k := range keys {
		s := skins[k]
		fmt.Fprintf(&b, "[Skin %d]\nleft=%d\ntop=%d\nwidth=%d\nheight=%d\n", k, s.Left, s.Top, s.Width, s.Height)
		for _, e := range s.Extra {
			fmt.Fprintf(&b, "%s=%s\n", e[0], e[1])
		}
		b.WriteByte('\n')
	}
	return []byte(b.String())
}
