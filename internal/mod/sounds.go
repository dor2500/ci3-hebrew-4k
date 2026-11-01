package mod

// SoundMap names, for each generated sound event, the original files it replaces.
var SoundMap = map[string][]string{
	"shoot":      {"(defaultweapon)", "(gatling)", "(greenWeaponSound)", "(laser)", "(lightningMiss)", "(plasmaMiss)", "(zap2)", "(zap3)", "(zigzagzap)", "(buglezap)", "(sword)"},
	"hit":        {"chicken1b1(pluck)", "chicken1b2(pluck)", "chicken2a1(pluck)", "chicken3b1(pluck)", "chicken3b2(pluck)", "chicken5b(pluck)", "(ricochet1)", "(ricochet2)", "(ricochet3)", "lightningHit", "(plasmaHit)", "(bonk)"},
	"enemy_die":  {"chicken1a(die)", "chicken2b(die)", "chicken3a(die)", "chicken4a(die)", "chicken4b(die)", "chicken5a(die)", "(smallExplosion)", "(mediumExplosion)"},
	"egg_drop":   {"(eggdrop)", "(bubble)"},
	"splat":      {"(splat)", "(eggshellcrack)", "(eggsplat,dopplered)", "(yolkcrack1)", "(yolkcrack2)", "(yolkcrack3)", "(yolkcrack4)", "(balloonPop)", "(tubepop)"},
	"pickup":     {"(eating1)", "(eating2)", "(eating3)", "(eating4)", "(eating5)", "(coin)"},
	"power":      {"(weaponPowerup)"},
	"player_die": {"(playerexplosion)"},
	"missile":    {"(missile)", "(rocket)"},
	"boom":       {"(missileexplosion)"},
	"overheat":   {"(overheat)", "(overheatBeep)"},
	"dash":       {"(cometWhoosh)", "(hyperspaceengine)", "(hyperspaceenginestop)"},
	"rock":       {"(rock1)", "(rock2)", "(rock3)", "(rock4)", "(superMine)"},
	"charge":     {"(yolkbeamcharge)"},
	"beam":       {"(redBeam)", "(greenBeam)", "(yolkbeam)"},
	"boss_in":    {"(chickbossCry)", "(chickbossDie)"},
	"boss_die":   {"(bossexplosion)"},
	"extra":      {"(extraLife)", "(extraMissile)"},
	"menu_move":  {"(beep)", "(messagebeep)", "(speechbeep)", "(bugleBeep)"},
	"menu_ok":    {"clickXP", "pop"},
}

// VoiceMap plays a Hebrew line in place of a sound tied to an achievement.
var VoiceMap = map[string]string{
	"sfx/music/CI3MinorWinTheme.ogg": "kol",
	"sfx/(extraLife).ogg":            "yay",
	"sfx/(extraMissile).ogg":         "alufim",
	"sfx/(weaponPowerup).ogg":        "metzuyan",
	"sfx/(chickbossCry).ogg":         "boss",
}
