package mod

import (
	"bytes"
	"encoding/binary"
	"image"
	"testing"
)

func TestDatRoundTrip(t *testing.T) {
	blob := make([]byte, 4+2*entry)
	binary.LittleEndian.PutUint32(blob, 2)
	put := func(i int, name, body string) {
		o := 4 + i*entry
		for j := 0; j < 80; j++ {
			c := byte(0)
			if j < len(name) {
				c = name[j]
			}
			blob[o+j] = c ^ key
		}
		binary.LittleEndian.PutUint32(blob[o+80:], uint32(len(blob)))
		binary.LittleEndian.PutUint32(blob[o+84:], uint32(len(body)))
		blob = append(blob, body...)
	}
	put(0, "a.txt", "hello")
	put(1, "b.txt", "world")
	d, err := ParseDat(blob)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(d.Rebuild(nil), blob) {
		t.Fatal("identity rebuild changed the file")
	}
	d2, err := ParseDat(d.Rebuild(map[string][]byte{"a.txt": []byte("longer text")}))
	if err != nil {
		t.Fatal(err)
	}
	if string(d2.Get("a.txt")) != "longer text" || string(d2.Get("b.txt")) != "world" {
		t.Fatalf("got %q %q", d2.Get("a.txt"), d2.Get("b.txt"))
	}
}

func TestTGARoundTrip(t *testing.T) {
	im := image.NewNRGBA(image.Rect(0, 0, 3, 2))
	for i := range im.Pix {
		im.Pix[i] = byte(i * 9)
	}
	back, err := ReadTGA(WriteTGA(im))
	if err != nil || !bytes.Equal(back.Pix, im.Pix) {
		t.Fatalf("err %v", err)
	}
}

func TestVisual(t *testing.T) {
	cases := map[string]string{
		"שלום":       "םולש",
		"גל 12":      "12 לג",
		"(כן)":       "(ןכ)",
		"אב|גד":      "בא|דג",
		"שיא: 1,200": "1,200 :איש",
	}
	for in, want := range cases {
		if got := Visual(in); got != want {
			t.Errorf("%q: got %q want %q", in, got, want)
		}
	}
}

func TestIniKeepsExtraKeys(t *testing.T) {
	s := ParseIni([]byte("[Skin 2]\nleft=1\ntop=2\nwidth=3\nheight=4\norigin_adjust_x=-1\n"))
	back := ParseIni(WriteIni(s))
	if back[2].Width != 3 || len(back[2].Extra) != 1 || back[2].Extra[0][1] != "-1" {
		t.Fatalf("%+v", back[2])
	}
}

func TestUpscaleFactorRespectsEngineLimit(t *testing.T) {
	for _, c := range []struct{ w, h, want int }{{256, 256, 4}, {510, 64, 4}, {512, 256, 2}, {762, 762, 2}, {1000, 1000, 2}, {1024, 256, 1}} {
		s := upscaleFactor(c.w, c.h)
		if s != c.want {
			t.Errorf("%dx%d: %d, want %d", c.w, c.h, s, c.want)
		}
		if s > 1 && max(c.w, c.h)*s+3 > maxTextureSide {
			t.Errorf("%dx%d exceeds the limit at %d", c.w, c.h, s)
		}
	}
}
