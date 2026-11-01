// Command genpacks renders the synthesised sound packs to assets/packs. Run it
// once from the repo root; it needs ffmpeg. The output is embedded in the exe.
package main

import (
	"encoding/binary"
	"fmt"
	"math"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
)

const Rate = 44100

type Sfx int

const (
	Shoot Sfx = iota
	Hit
	EnemyDie
	EggDrop
	Splat
	Pickup
	Power
	PlayerDie
	Missile
	Boom
	Overheat
	Dash
	Rock
	Charge
	Beam
	BossIn
	BossDie
	Extra
	MenuMove
	MenuOK
	numSfx
)

var sfxFile = [numSfx]string{"shoot", "hit", "enemy_die", "egg_drop", "splat", "pickup", "power", "player_die",
	"missile", "boom", "overheat", "dash", "rock", "charge", "beam", "boss_in", "boss_die", "extra", "menu_move", "menu_ok"}

func main() {
	for _, pack := range []string{"blocks", "kombat", "mix"} {
		dir := filepath.Join("assets", "packs", pack)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			panic(err)
		}
		for s := Sfx(0); s < numSfx; s++ {
			pcm := synthVariants(pack, s)[0]
			out := filepath.Join(dir, sfxFile[s]+".ogg")
			c := exec.Command("ffmpeg", "-y", "-v", "error", "-f", "s16le", "-ar", fmt.Sprint(Rate), "-ac", "2", "-i", "-", "-c:a", "libvorbis", "-q:a", "4", out)
			w, _ := c.StdinPipe()
			if err := c.Start(); err != nil {
				panic(err)
			}
			w.Write(pcm)
			w.Close()
			if err := c.Wait(); err != nil {
				panic(err)
			}
		}
	}
}

// ---- synthesis ----

type wave int

const (
	sine wave = iota
	square
	saw
	tri
	noise
)

type layer struct {
	w        wave
	f0, f1   float64 // Hz sweep
	dur, vol float64
	delay    float64
	crush    int // sample-and-hold factor for a low-fi sound
}

func render(ls []layer, jitter float64, r *rand.Rand) []byte {
	total := 0.0
	for _, l := range ls {
		total = math.Max(total, l.delay+l.dur)
	}
	n := int(total*Rate) + 64
	buf := make([]float64, n)
	for _, l := range ls {
		start := int(l.delay * Rate)
		count := int(l.dur * Rate)
		ph := 0.0
		held, hold := 0.0, 0
		for i := 0; i < count && start+i < n; i++ {
			t := float64(i) / float64(count)
			f := (l.f0 + (l.f1-l.f0)*t) * jitter
			ph += f / Rate
			var v float64
			switch l.w {
			case sine:
				v = math.Sin(2 * math.Pi * ph)
			case square:
				v = 1
				if math.Mod(ph, 1) > 0.5 {
					v = -1
				}
			case saw:
				v = 2*math.Mod(ph, 1) - 1
			case tri:
				v = 4*math.Abs(math.Mod(ph, 1)-0.5) - 1
			default:
				v = r.Float64()*2 - 1
			}
			if l.crush > 1 {
				if hold == 0 {
					held = v
				}
				hold = (hold + 1) % l.crush
				v = held
			}
			env := math.Min(1, float64(i)/(0.002*Rate)) * math.Pow(1-t, 1.6)
			buf[start+i] += v * env * l.vol
		}
	}
	return toPCM(buf)
}

func toPCM(buf []float64) []byte {
	out := make([]byte, len(buf)*4)
	for i, v := range buf {
		s := int16(math.Max(-1, math.Min(1, v)) * 32000)
		binary.LittleEndian.PutUint16(out[i*4:], uint16(s))
		binary.LittleEndian.PutUint16(out[i*4+2:], uint16(s))
	}
	return out
}

func synthVariants(pack string, s Sfx) [][]byte {
	r := rand.New(rand.NewSource(int64(s)*31 + int64(len(pack))))
	n := 1
	if s == Shoot || s == Hit || s == EnemyDie || s == Pickup || s == Rock {
		n = 3
	}
	out := make([][]byte, n)
	ls := recipe(pack, s)
	for i := range out {
		out[i] = render(ls, 1+float64(i)*0.07-0.07, r)
	}
	return out
}

func recipe(pack string, s Sfx) []layer {
	switch pack {
	case "blocks":
		return blocksRecipe(s)
	case "kombat":
		return kombatRecipe(s)
	case "mix":
		return mixRecipe(s)
	}
	return classicRecipe(s)
}

// Used for "original" when the player's own CI3 files are not present.
func classicRecipe(s Sfx) []layer {
	switch s {
	case Shoot:
		return []layer{{saw, 1500, 500, .1, .35, 0, 0}}
	case Hit:
		return []layer{{square, 900, 1300, .05, .25, 0, 0}, {square, 700, 1100, .05, .2, .05, 0}}
	case EnemyDie:
		return []layer{{square, 1200, 500, .09, .35, 0, 0}, {noise, 0, 0, .12, .25, .03, 3}}
	case EggDrop:
		return []layer{{sine, 900, 350, .1, .3, 0, 0}}
	case Splat:
		return []layer{{noise, 0, 0, .1, .22, 0, 6}}
	case Pickup:
		return []layer{{tri, 700, 1100, .05, .3, 0, 0}, {tri, 1000, 1400, .05, .3, .05, 0}}
	case Power:
		return arp(tri, []float64{523, 659, 784, 1047}, .07, .35)
	case PlayerDie:
		return []layer{{noise, 0, 0, .8, .6, 0, 4}, {saw, 300, 40, .7, .4, 0, 0}}
	case Missile:
		return []layer{{saw, 180, 900, .5, .3, 0, 0}, {noise, 0, 0, .5, .2, 0, 2}}
	case Boom:
		return []layer{{noise, 0, 0, 1, .7, 0, 8}, {sine, 120, 30, .9, .7, 0, 0}}
	case Overheat:
		return []layer{{noise, 0, 0, .9, .3, 0, 1}, {square, 300, 120, .5, .2, 0, 0}}
	case Dash:
		return []layer{{noise, 0, 0, .25, .3, 0, 2}, {sine, 400, 1500, .22, .25, 0, 0}}
	case Rock:
		return []layer{{noise, 0, 0, .22, .45, 0, 12}, {sine, 170, 60, .2, .4, 0, 0}}
	case Charge:
		return []layer{{saw, 100, 1400, 1.1, .3, 0, 0}}
	case Beam:
		return []layer{{saw, 110, 90, .6, .45, 0, 0}, {square, 220, 180, .6, .25, 0, 0}, {noise, 0, 0, .6, .2, 0, 3}}
	case BossIn:
		return []layer{{saw, 300, 700, .4, .35, 0, 0}, {saw, 700, 200, .6, .35, .4, 0}}
	case BossDie:
		return []layer{{noise, 0, 0, 1.6, .7, 0, 8}, {sine, 150, 25, 1.5, .8, 0, 0}, {saw, 900, 100, 1, .3, 0, 0}}
	case Extra:
		return arp(square, []float64{659, 784, 1047, 1319, 1568}, .08, .25)
	case MenuMove:
		return []layer{{square, 900, 900, .04, .18, 0, 0}}
	}
	return []layer{{square, 660, 660, .06, .22, 0, 0}, {square, 990, 990, .1, .22, .06, 0}}
}

// Original sounds that evoke a blocky sandbox game: dry digging crunches, thin
// bows, and a chiming experience-orb arpeggio.
func blocksRecipe(s Sfx) []layer {
	switch s {
	case Shoot:
		return []layer{{noise, 0, 0, .09, .3, 0, 2}, {tri, 1200, 300, .1, .3, 0, 2}}
	case Hit:
		return []layer{{noise, 0, 0, .08, .5, 0, 14}, {square, 180, 120, .07, .25, 0, 6}}
	case EnemyDie:
		return []layer{{noise, 0, 0, .12, .5, 0, 20}, {noise, 0, 0, .1, .4, .1, 28}, {square, 240, 90, .22, .2, 0, 8}}
	case EggDrop:
		return []layer{{sine, 500, 180, .12, .35, 0, 4}}
	case Splat:
		return []layer{{noise, 0, 0, .14, .3, 0, 22}}
	case Pickup:
		return []layer{{sine, 1400, 1400, .06, .25, 0, 0}, {sine, 2100, 2100, .1, .22, .05, 0}}
	case Power:
		return arp(sine, []float64{784, 1047, 1319, 1568, 2093}, .08, .3)
	case PlayerDie:
		return []layer{{square, 220, 70, .5, .35, 0, 10}, {noise, 0, 0, .35, .4, 0, 24}}
	case Missile:
		return []layer{{noise, 0, 0, 1.0, .25, 0, 3}, {square, 90, 90, 1.0, .1, 0, 6}}
	case Boom:
		return []layer{{noise, 0, 0, 1.2, .8, 0, 26}, {sine, 90, 30, 1.0, .8, 0, 0}}
	case Overheat:
		return []layer{{noise, 0, 0, .7, .35, 0, 6}, {sine, 180, 180, .5, .15, 0, 0}}
	case Dash:
		return []layer{{noise, 0, 0, .2, .35, 0, 9}}
	case Rock:
		return []layer{{noise, 0, 0, .16, .55, 0, 30}, {noise, 0, 0, .12, .4, .14, 36}, {noise, 0, 0, .1, .3, .26, 40}}
	case Charge:
		return []layer{{tri, 200, 1200, 1.1, .3, 0, 6}}
	case Beam:
		return []layer{{noise, 0, 0, .6, .35, 0, 5}, {square, 130, 110, .6, .25, 0, 8}}
	case BossIn:
		return []layer{{square, 80, 55, 1.1, .4, 0, 14}, {noise, 0, 0, 1.0, .2, 0, 30}}
	case BossDie:
		return []layer{{noise, 0, 0, 1.6, .8, 0, 30}, {sine, 100, 25, 1.5, .8, 0, 0}}
	case Extra:
		return arp(sine, []float64{1047, 1319, 1568, 2093, 2637}, .09, .28)
	case MenuMove:
		return []layer{{noise, 0, 0, .03, .3, 0, 8}}
	}
	return []layer{{square, 500, 500, .03, .25, 0, 4}, {square, 750, 750, .04, .2, .03, 4}}
}

// Original sounds that evoke an arcade fighter: whooshes, heavy body blows,
// metal clangs and a low announcer-like rumble.
func kombatRecipe(s Sfx) []layer {
	switch s {
	case Shoot:
		return []layer{{noise, 0, 0, .1, .28, 0, 1}, {sine, 900, 300, .1, .2, 0, 0}}
	case Hit:
		return []layer{{sine, 180, 55, .12, .7, 0, 0}, {noise, 0, 0, .07, .4, 0, 3}}
	case EnemyDie:
		return []layer{{sine, 150, 40, .25, .8, 0, 0}, {noise, 0, 0, .18, .5, 0, 3}, {saw, 420, 90, .3, .25, .03, 0}}
	case EggDrop:
		return []layer{{noise, 0, 0, .1, .2, 0, 1}, {sine, 300, 800, .1, .15, 0, 0}}
	case Splat:
		return []layer{{noise, 0, 0, .18, .35, 0, 7}, {sine, 200, 50, .15, .4, 0, 0}}
	case Pickup:
		return []layer{{square, 2200, 2200, .05, .15, 0, 0}, {square, 3300, 3300, .15, .13, .02, 0}, {sine, 1100, 1100, .2, .2, 0, 0}}
	case Power:
		return []layer{{saw, 110, 110, .8, .35, 0, 0}, {saw, 165, 165, .8, .3, .1, 0}, {saw, 220, 220, .8, .3, .2, 0}, {sine, 55, 55, 1, .5, 0, 0}}
	case PlayerDie:
		return []layer{{saw, 420, 60, .9, .55, 0, 0}, {sine, 90, 25, 1, .9, 0, 0}, {noise, 0, 0, .5, .5, 0, 3}}
	case Missile:
		return []layer{{noise, 0, 0, .6, .4, 0, 1}, {saw, 60, 200, .6, .3, 0, 0}}
	case Boom:
		return []layer{{sine, 90, 20, 1.3, 1, 0, 0}, {noise, 0, 0, 1.1, .7, 0, 5}}
	case Overheat:
		return []layer{{square, 1800, 1700, .35, .15, 0, 0}, {square, 2600, 2500, .35, .12, 0, 0}, {noise, 0, 0, .5, .25, 0, 1}}
	case Dash:
		return []layer{{noise, 0, 0, .28, .5, 0, 1}, {sine, 200, 900, .28, .25, 0, 0}}
	case Rock:
		return []layer{{sine, 110, 35, .3, .9, 0, 0}, {noise, 0, 0, .2, .55, 0, 8}}
	case Charge:
		return []layer{{saw, 55, 440, 1.1, .4, 0, 0}, {saw, 58, 450, 1.1, .3, 0, 0}}
	case Beam:
		return []layer{{saw, 70, 60, .6, .6, 0, 0}, {saw, 73, 62, .6, .5, 0, 0}, {noise, 0, 0, .6, .3, 0, 2}}
	case BossIn:
		return []layer{{saw, 55, 42, 1.5, .55, 0, 0}, {saw, 82, 62, 1.5, .4, 0, 0}, {square, 110, 84, 1.5, .18, 0, 0}, {noise, 0, 0, 1.4, .2, 0, 12}}
	case BossDie:
		return []layer{{sine, 100, 18, 2, 1, 0, 0}, {saw, 300, 30, 1.7, .5, 0, 0}, {noise, 0, 0, 1.7, .7, 0, 6}}
	case Extra:
		return []layer{{sine, 196, 196, .9, .4, 0, 0}, {sine, 294, 294, .9, .35, .12, 0}, {sine, 392, 392, .9, .35, .24, 0}}
	case MenuMove:
		return []layer{{square, 2100, 2000, .05, .13, 0, 0}, {sine, 150, 90, .06, .4, 0, 0}}
	}
	return []layer{{sine, 200, 50, .25, .85, 0, 0}, {square, 2300, 2200, .2, .18, 0, 0}, {noise, 0, 0, .12, .35, 0, 2}}
}

// Eight-bit blips layered with modern sub-bass drops and risers.
func mixRecipe(s Sfx) []layer {
	switch s {
	case Shoot:
		return []layer{{square, 1800, 400, .08, .26, 0, 0}}
	case Hit:
		return []layer{{square, 500, 300, .04, .28, 0, 0}, {noise, 0, 0, .03, .25, 0, 2}}
	case EnemyDie:
		return []layer{{square, 900, 100, .16, .3, 0, 4}, {sine, 120, 45, .18, .6, 0, 0}}
	case EggDrop:
		return []layer{{tri, 1000, 300, .1, .28, 0, 0}}
	case Splat:
		return []layer{{noise, 0, 0, .09, .25, 0, 9}}
	case Pickup:
		return []layer{{square, 988, 988, .05, .22, 0, 0}, {square, 1319, 1319, .13, .22, .05, 0}}
	case Power:
		return append(arp(square, []float64{392, 523, 659, 784, 1047, 1319}, .05, .22), layer{sine, 60, 60, .5, .6, 0, 0})
	case PlayerDie:
		return []layer{{square, 1000, 60, .6, .3, 0, 3}, {sine, 180, 22, .8, .9, 0, 0}, {noise, 0, 0, .4, .3, 0, 5}}
	case Missile:
		return []layer{{saw, 100, 1800, .55, .28, 0, 0}, {saw, 103, 1830, .55, .22, 0, 0}}
	case Boom:
		return []layer{{sine, 180, 22, 1.2, 1, 0, 0}, {noise, 0, 0, .9, .55, 0, 4}}
	case Overheat:
		return []layer{{square, 880, 880, .1, .2, 0, 0}, {square, 880, 880, .1, .2, .18, 0}, {square, 880, 880, .1, .2, .36, 0}}
	case Dash:
		return []layer{{sine, 200, 2000, .2, .3, 0, 0}, {noise, 0, 0, .2, .2, 0, 1}}
	case Rock:
		return []layer{{noise, 0, 0, .15, .4, 0, 14}, {sine, 100, 40, .2, .6, 0, 0}}
	case Charge:
		return []layer{{saw, 110, 1760, 1.1, .25, 0, 0}, {saw, 112, 1790, 1.1, .2, 0, 0}, {noise, 0, 0, 1.1, .1, 0, 1}}
	case Beam:
		return []layer{{saw, 55, 55, .6, .5, 0, 0}, {saw, 110, 108, .6, .3, 0, 0}, {square, 220, 220, .6, .12, 0, 3}}
	case BossIn:
		return []layer{{square, 110, 110, .25, .28, 0, 0}, {square, 110, 110, .25, .28, .3, 0}, {square, 82, 82, .5, .28, .6, 0}, {sine, 55, 36, 1.2, .8, 0, 0}}
	case BossDie:
		return []layer{{sine, 220, 20, 1.8, 1, 0, 0}, {square, 800, 40, 1.2, .25, 0, 4}, {noise, 0, 0, 1.5, .55, 0, 6}}
	case Extra:
		return arp(square, []float64{1319, 1568, 2637, 2093, 2349, 3136}, .07, .2)
	case MenuMove:
		return []layer{{square, 1200, 1200, .03, .18, 0, 0}}
	}
	return []layer{{square, 784, 784, .05, .2, 0, 0}, {square, 1175, 1175, .09, .2, .05, 0}, {sine, 90, 60, .15, .4, 0, 0}}
}

func arp(w wave, notes []float64, step, vol float64) []layer {
	var out []layer
	for i, n := range notes {
		out = append(out, layer{w, n, n, step * 2.4, vol, float64(i) * step, 0})
	}
	return out
}
