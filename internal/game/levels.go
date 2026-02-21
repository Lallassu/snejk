package game

type LevelConfig struct {
	Peds         int
	Cars         int
	ArmedPeds    int
	InfectedPeds int
	BonusBoxes   int
	Theme        ThemeConfig
}

// GetLevelConfig returns settings for a given level.
// The base population/bonus targets are defined per level.
// Theme selection is randomized at StartLevel.
// Levels 1–14 are hand-crafted; beyond that scales up procedurally.
func GetLevelConfig(level int) LevelConfig {
	var cfg LevelConfig

	switch level {
	case 1:
		// Easy intro — familiar suburban streets, light traffic.
		cfg = LevelConfig{Peds: 25, Cars: 6, ArmedPeds: 1, InfectedPeds: 1, BonusBoxes: 3, Theme: ThemeSuburban}
	case 2:
		// Open countryside — few buildings, room to maneuver.
		cfg = LevelConfig{Peds: 35, Cars: 8, ArmedPeds: 3, InfectedPeds: 2, BonusBoxes: 3, Theme: ThemeRural}
	case 3:
		// City streets — denser buildings, more traffic.
		cfg = LevelConfig{Peds: 50, Cars: 14, ArmedPeds: 5, InfectedPeds: 4, BonusBoxes: 4, Theme: ThemeCity}
	case 4:
		// Dense woodland — no roads, peds boosted, no cars.
		cfg = LevelConfig{Peds: 60, Cars: 0, ArmedPeds: 8, InfectedPeds: 5, BonusBoxes: 4, Theme: ThemeForest}
	case 5:
		// Quiet village — tiny buildings, tight alleys between cottages.
		cfg = LevelConfig{Peds: 65, Cars: 14, ArmedPeds: 10, InfectedPeds: 7, BonusBoxes: 3, Theme: ThemeVillage}
	case 6:
		// Green city — wide parks to chase through, light traffic.
		cfg = LevelConfig{Peds: 75, Cars: 16, ArmedPeds: 13, InfectedPeds: 9, BonusBoxes: 3, Theme: ThemeParkCity}
	case 7:
		// Industrial zone — blocky warehouses, lots of cars, grim.
		cfg = LevelConfig{Peds: 85, Cars: 22, ArmedPeds: 16, InfectedPeds: 11, BonusBoxes: 3, Theme: ThemeIndustrial}
	case 8:
		// Deep forest — brutal wilderness, dense trees, armed hunters.
		cfg = LevelConfig{Peds: 90, Cars: 0, ArmedPeds: 22, InfectedPeds: 15, BonusBoxes: 3, Theme: ThemeForest}
	case 9:
		// Megacity — towering blocks, urban canyons, heavy traffic.
		cfg = LevelConfig{Peds: 100, Cars: 28, ArmedPeds: 22, InfectedPeds: 18, BonusBoxes: 3, Theme: ThemeMegacity}
	case 10:
		// Park City under siege — open parks, maximum peds, chaos.
		cfg = LevelConfig{Peds: 120, Cars: 26, ArmedPeds: 28, InfectedPeds: 22, BonusBoxes: 2, Theme: ThemeParkCity}
	case 11:
		// Arctic raid — cold open lanes with bundled survivors and patrol vehicles.
		cfg = LevelConfig{Peds: 130, Cars: 18, ArmedPeds: 30, InfectedPeds: 20, BonusBoxes: 3, Theme: ThemeArctic}
	case 12:
		// Desert hunt — faster raiders on wide dusty blocks.
		cfg = LevelConfig{Peds: 145, Cars: 22, ArmedPeds: 34, InfectedPeds: 20, BonusBoxes: 3, Theme: ThemeDesert}
	case 13:
		// Space outpost — no roads, tanky astronauts and hostile crews.
		cfg = LevelConfig{Peds: 120, Cars: 0, ArmedPeds: 34, InfectedPeds: 24, BonusBoxes: 4, Theme: ThemeSpace}
	case 14:
		// Underwater zone — slow heavy divers, dense no-road pursuit.
		cfg = LevelConfig{Peds: 115, Cars: 0, ArmedPeds: 28, InfectedPeds: 32, BonusBoxes: 4, Theme: ThemeUnderwater}
	default:
		// Levels 15+ scale enemy/population pressure aggressively.
		// Theme is randomized in StartLevel; this is only a fallback.
		extra := level - 15
		fallbackTheme := ThemeCity
		if len(Themes) > 0 {
			fallbackTheme = Themes[(level-1)%len(Themes)]
		}
		cfg = LevelConfig{
			Peds:         160 + extra*20,
			Cars:         34 + extra*4,
			ArmedPeds:    36 + extra*5,
			InfectedPeds: 30 + extra*4,
			BonusBoxes:   2 + extra/2,
			Theme:        fallbackTheme,
		}
	}

	// Slightly denser population across all levels.
	cfg.Peds += max(3, cfg.Peds/8)

	return cfg
}
