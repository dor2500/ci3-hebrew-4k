package mod

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

const (
	dgvURL    = "https://github.com/dege-diosg/dgVoodoo2/releases/download/2.87.6/dgVoodoo2_87_6.zip"
	dgvSHA256 = "9afa493396feec7108f42a2d97661eef96bbaac5addf0dfb5cfbc3001cd76a0a"
	FPS       = 65
)

// FindInstall locates the player's copy of the game.
func FindInstall(exeDir string) string {
	cands := []string{}
	if b, err := os.ReadFile(filepath.Join(exeDir, "ci3_path.txt")); err == nil {
		cands = append(cands, strings.TrimSpace(string(b)))
	}
	cands = append(cands, exeDir, `C:\Program Files (x86)\ChickenInvadersROTYXmas`, `C:\Program Files\ChickenInvadersROTYXmas`)
	for _, c := range cands {
		if _, err := os.Stat(filepath.Join(c, "CI3Xmas.dat")); err == nil {
			return c
		}
	}
	return ""
}

func copyIfMissing(src, dst string) error {
	if _, err := os.Stat(dst); err == nil {
		return nil
	}
	b, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return os.WriteFile(dst, b, 0o644)
}

// PrepareFolder makes the private game folder. The original install is only read.
func PrepareFolder(install, game string) error {
	if err := os.MkdirAll(game, 0o755); err != nil {
		return err
	}
	if err := copyIfMissing(filepath.Join(install, "bass.dll"), filepath.Join(game, "bass.dll")); err != nil {
		return err
	}
	if err := copyIfMissing(filepath.Join(install, "CI3Xmas.cfg"), filepath.Join(game, "CI3Xmas.cfg")); err != nil {
		return err
	}
	// Progress and high scores carry over once, if they exist.
	for _, f := range []string{"CI3Xmas.hst", "CI3Xmas.pro"} {
		_ = copyIfMissing(filepath.Join(install, f), filepath.Join(game, f))
	}
	return nil
}

func fetchDGV(cache string) ([]byte, error) {
	if b, err := os.ReadFile(cache); err == nil {
		return b, nil
	}
	resp, err := http.Get(dgvURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("dgVoodoo2 download: %s", resp.Status)
	}
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	sum := sha256.Sum256(b)
	if hex.EncodeToString(sum[:]) != dgvSHA256 {
		return nil, fmt.Errorf("dgVoodoo2 checksum mismatch")
	}
	return b, os.WriteFile(cache, b, 0o644)
}

func setConfKey(text, section, key, value string) (string, error) {
	lines := strings.Split(text, "\n")
	cur := ""
	sec := regexp.MustCompile(`^\s*\[(.+)\]`)
	k := regexp.MustCompile(`^\s*` + regexp.QuoteMeta(key) + `\s*=`)
	for i, l := range lines {
		if m := sec.FindStringSubmatch(l); m != nil {
			cur = m[1]
		} else if cur == section && k.MatchString(l) {
			lines[i] = key + " = " + value
			return strings.Join(lines, "\n"), nil
		}
	}
	return "", fmt.Errorf("dgVoodoo.conf has no %s.%s", section, key)
}

// InstallDGV places the DirectX wrapper that renders the game at the given size
// with the 65 FPS limit.
func InstallDGV(game, cacheDir string, resW, resH int) error {
	b, err := fetchDGV(filepath.Join(cacheDir, "dgVoodoo2_87_6.zip"))
	if err != nil {
		return err
	}
	z, err := zip.NewReader(bytes.NewReader(b), int64(len(b)))
	if err != nil {
		return err
	}
	read := func(name string) ([]byte, error) {
		f, err := z.Open(name)
		if err != nil {
			return nil, err
		}
		defer f.Close()
		return io.ReadAll(f)
	}
	for _, n := range []string{"DDraw.dll", "D3DImm.dll"} {
		data, err := read("MS/x86/" + n)
		if err != nil {
			return err
		}
		if err := os.WriteFile(filepath.Join(game, n), data, 0o644); err != nil {
			return err
		}
	}
	confB, err := read("dgVoodoo.conf")
	if err != nil {
		return err
	}
	conf := strings.ReplaceAll(strings.TrimPrefix(string(confB), "\xef\xbb\xbf"), "\r\n", "\n")
	for _, s := range [][3]string{
		{"General", "OutputAPI", "bestavailable"},
		{"General", "FullScreenMode", "true"},
		{"General", "ScalingMode", "stretched"},
		{"GeneralExt", "FPSLimit", fmt.Sprint(FPS)},
		{"DirectX", "VRAM", "1024"},
		{"DirectX", "Resolution", fmt.Sprintf("%dx%d", resW, resH)},
		{"DirectX", "ForceVerticalSync", "false"},
		{"DirectX", "dgVoodooWatermark", "false"},
	} {
		if conf, err = setConfKey(conf, s[0], s[1], s[2]); err != nil {
			return err
		}
	}
	return os.WriteFile(filepath.Join(game, "dgVoodoo.conf"), []byte(conf), 0o644)
}

// ConfigureGame writes the engine settings: a field of view matching the
// monitor's shape so the picture fills it undistorted, the 65 Hz simulation step
// and the larger font atlases.
func ConfigureGame(game string, resW, resH int) error {
	p := filepath.Join(game, "CI3Xmas.cfg")
	b, err := os.ReadFile(p)
	if err != nil {
		return err
	}
	text := string(b)
	for _, kv := range [][2]string{
		{"fovWidth", fmt.Sprint(600 * resW / resH)}, {"fovHeight", "600"},
		{"screenWidth", "800"}, {"screenHeight", "600"}, {"bFullScreen", "Yes"},
		{"frameRate", fmt.Sprintf("%.8f", 1.0/FPS)}, {"bSimulate256x256TextureHW", "No"},
	} {
		re := regexp.MustCompile(`(?m)^` + kv[0] + `=[^\r\n]*`)
		if !re.MatchString(text) {
			return fmt.Errorf("CI3Xmas.cfg has no %s", kv[0])
		}
		text = re.ReplaceAllString(text, kv[0]+"="+kv[1])
	}
	return os.WriteFile(p, []byte(text), 0o644)
}
