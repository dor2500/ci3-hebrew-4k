package mod

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"regexp"
	"strings"
)

// The patch addresses are valid only for this build, v3.20 Christmas Edition.
const OriginalExeSHA256 = "b38e576fb22ee668d7b0d7157a31790aface6c077ae6f7896e8ce9e5ac2f6be7"

const (
	base          = 0x400000
	compositeFunc = 0x450940
)

// StringSite says where the code references one translatable string.
type StringSite struct {
	Addr  int   `json:"addr"`
	Sites []int `json:"sites"`
}

type tableEntry struct {
	site           int
	prefix, hebrew string
	suffix         string
}

// UI slots whose English text also serves as config keys or carries an icon.
var tableOnly = []tableEntry{
	{0x471075 + 3, "", "כן", ""},
	{0x47107C + 3, "", "לא", ""},
	{0x47108A + 3, "", "סיום", ""},
	{0x47109F + 3, "\x06  ", "חזרה", ""},
	{0x4710A6 + 3, "", "הבא", "  \x07"},
	{0x4710AD + 3, "\x02  ", "ביטול", ""},
	{0x4710BB + 3, "\t  ", "עזרה", ""},
	{0x4710C2 + 3, "\x02  ", "יציאה", ""},
	{0x471486 + 6, "", "הוסף", ""},
}

var (
	ltrRe  = regexp.MustCompile(`[A-Za-z0-9%][A-Za-z0-9%.,:/ +\-]*[A-Za-z0-9%]|[A-Za-z0-9%]`)
	mirror = strings.NewReplacer("(", ")", ")", "(", "[", "]", "]", "[", "{", "}", "}", "{", "<", ">", ">", "<")
)

func visualLine(text string) string {
	var parts []string
	pos := 0
	addChars := func(s string) {
		for _, r := range s {
			parts = append(parts, mirror.Replace(string(r)))
		}
	}
	for _, m := range ltrRe.FindAllStringIndex(text, -1) {
		addChars(text[pos:m[0]])
		parts = append(parts, text[m[0]:m[1]])
		pos = m[1]
	}
	addChars(text[pos:])
	var b strings.Builder
	for i := len(parts) - 1; i >= 0; i-- {
		b.WriteString(parts[i])
	}
	return b.String()
}

// Visual orders a logical Hebrew string for an engine that only draws left to
// right. Lines are separated by | in the game's text format.
func Visual(text string) string {
	lines := strings.Split(text, "|")
	for i, l := range lines {
		lines[i] = visualLine(l)
	}
	return strings.Join(lines, "|")
}

// CP1255 encodes the Hebrew letters into the code points the patched fonts hold.
func CP1255(s string) ([]byte, error) {
	out := make([]byte, 0, len(s))
	for _, r := range s {
		switch {
		case r >= 0x5D0 && r <= 0x5EA:
			out = append(out, byte(0xE0+r-0x5D0))
		case r < 0x80:
			out = append(out, byte(r))
		default:
			return nil, fmt.Errorf("cannot encode %q in %q", r, s)
		}
	}
	return out, nil
}

// PatchExe redirects string references to a new section holding the Hebrew text.
func PatchExe(orig []byte, sites []StringSite, he map[string]string) ([]byte, error) {
	sum := sha256.Sum256(orig)
	if hex.EncodeToString(sum[:]) != OriginalExeSHA256 {
		return nil, fmt.Errorf("unsupported CI3Xmas.exe build")
	}
	exe := append([]byte(nil), orig...)
	u32 := func(o int) int { return int(binary.LittleEndian.Uint32(exe[o:])) }
	u16 := func(o int) int { return int(binary.LittleEndian.Uint16(exe[o:])) }
	pe := u32(0x3C)
	count := u16(pe + 6)
	table := pe + 24 + u16(pe+20)
	last := table + 40*(count-1)
	vsize, va, rsize, raw := u32(last+8), u32(last+12), u32(last+16), u32(last+20)
	fileAlign, sectAlign := u32(pe+24+36), u32(pe+24+32)
	alignUp := func(v, a int) int { return (v + a - 1) &^ (a - 1) }
	newVA := alignUp(va+max(vsize, rsize), sectAlign)
	newRaw := alignUp(raw+rsize, fileAlign)
	if newRaw < len(exe) {
		return nil, fmt.Errorf("exe has an overlay")
	}
	if table+40*(count+1) > u32(table+20) {
		return nil, fmt.Errorf("no room for a section header")
	}

	var blob []byte
	add := func(prefix, hebrew, suffix string) (int, error) {
		enc, err := CP1255(Visual(hebrew))
		if err != nil {
			return 0, err
		}
		target := base + newVA + len(blob)
		blob = append(blob, prefix...)
		blob = append(blob, enc...)
		blob = append(blob, suffix...)
		blob = append(blob, 0)
		return target, nil
	}
	for _, row := range sites {
		text, ok := he[fmt.Sprintf("0x%x", row.Addr)]
		if !ok {
			continue
		}
		target, err := add("", text, "")
		if err != nil {
			return nil, err
		}
		for _, site := range row.Sites {
			off := site - base
			before := exe[off-1]
			isPush := before == 0x68 || (before >= 0xB8 && before <= 0xBF)
			isMovMem := false
			for _, d := range []int{2, 3, 4, 6, 7} {
				if exe[off-d] == 0xC7 {
					isMovMem = true
				}
			}
			if isPush || isMovMem {
				binary.LittleEndian.PutUint32(exe[off:], uint32(target))
			}
		}
	}
	for _, t := range tableOnly {
		target, err := add(t.prefix, t.hebrew, t.suffix)
		if err != nil {
			return nil, err
		}
		binary.LittleEndian.PutUint32(exe[t.site-base:], uint32(target))
	}
	// Disables the composite text path that the Hebrew glyph range would break.
	copy(exe[compositeFunc-base:], []byte{0xC2, 0x14, 0x00})

	size := len(blob)
	rawSize := alignUp(size, fileAlign)
	exe = append(exe, make([]byte, newRaw-len(exe))...)
	exe = append(exe, blob...)
	exe = append(exe, make([]byte, rawSize-size)...)
	h := exe[table+40*count : table+40*(count+1)]
	copy(h, ".heb\x00\x00\x00\x00")
	binary.LittleEndian.PutUint32(h[8:], uint32(size))
	binary.LittleEndian.PutUint32(h[12:], uint32(newVA))
	binary.LittleEndian.PutUint32(h[16:], uint32(rawSize))
	binary.LittleEndian.PutUint32(h[20:], uint32(newRaw))
	binary.LittleEndian.PutUint32(h[36:], 0x40000040)
	binary.LittleEndian.PutUint16(exe[pe+6:], uint16(count+1))
	binary.LittleEndian.PutUint32(exe[pe+24+56:], uint32(alignUp(newVA+size, sectAlign)))
	binary.LittleEndian.PutUint32(exe[pe+24+64:], 0)
	return exe, nil
}
