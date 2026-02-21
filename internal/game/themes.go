package game

type ThemeConfig struct {
	Name          string
	Family        string // canonical family used for gameplay tuning.
	ParkChance    [2]int // min/max percent chance for park blocks.
	BuildingCount [2]int // min/max buildings per block.
	BuildingSize  [2]int // min/max building footprint (width/height).
	TreeCount     [2]int // min/max trees in parks.
	TreeChance    int    // percent chance of trees on sidewalk/lot areas.
	NoRoads       bool   // pure wilderness — no roads, sidewalks, or cars.
}

var (
	ThemeCity = ThemeConfig{
		Name:          "City",
		ParkChance:    [2]int{5, 15},
		BuildingCount: [2]int{5, 10},
		BuildingSize:  [2]int{20, 68},
		TreeCount:     [2]int{5, 20},
		TreeChance:    10,
	}
	ThemeSuburban = ThemeConfig{
		Name:          "Suburban",
		ParkChance:    [2]int{15, 30},
		BuildingCount: [2]int{3, 6},
		BuildingSize:  [2]int{15, 44},
		TreeCount:     [2]int{15, 40},
		TreeChance:    25,
	}
	ThemeForest = ThemeConfig{
		Name:          "Forest",
		ParkChance:    [2]int{50, 80},
		BuildingCount: [2]int{1, 3},
		BuildingSize:  [2]int{12, 30},
		TreeCount:     [2]int{30, 64},
		TreeChance:    40,
		NoRoads:       true,
	}
	ThemeRural = ThemeConfig{
		Name:          "Rural",
		ParkChance:    [2]int{30, 50},
		BuildingCount: [2]int{2, 4},
		BuildingSize:  [2]int{15, 36},
		TreeCount:     [2]int{10, 35},
		TreeChance:    30,
	}
	// ThemeArctic: sparse icy settlement — fewer trees, larger open lanes.
	ThemeArctic = ThemeConfig{
		Name:          "Arctic",
		ParkChance:    [2]int{25, 42},
		BuildingCount: [2]int{2, 4},
		BuildingSize:  [2]int{16, 40},
		TreeCount:     [2]int{2, 10},
		TreeChance:    5,
	}
	// ThemeDesert: dry low-rise blocks — wide open space, almost no trees.
	ThemeDesert = ThemeConfig{
		Name:          "Desert",
		ParkChance:    [2]int{18, 35},
		BuildingCount: [2]int{2, 5},
		BuildingSize:  [2]int{14, 42},
		TreeCount:     [2]int{0, 4},
		TreeChance:    2,
	}
	// ThemeSand: wind-swept sandy district with sparse vegetation.
	ThemeSand = ThemeConfig{
		Name:          "Sand",
		ParkChance:    [2]int{16, 32},
		BuildingCount: [2]int{2, 5},
		BuildingSize:  [2]int{14, 44},
		TreeCount:     [2]int{0, 5},
		TreeChance:    2,
	}
	// ThemeWinter: snowy city blocks with icy open lanes.
	ThemeWinter = ThemeConfig{
		Name:          "Winter",
		ParkChance:    [2]int{22, 44},
		BuildingCount: [2]int{2, 5},
		BuildingSize:  [2]int{14, 42},
		TreeCount:     [2]int{4, 16},
		TreeChance:    8,
	}
	// ThemeBeach: bright coastal district with open beach zones.
	ThemeBeach = ThemeConfig{
		Name:          "Beach",
		ParkChance:    [2]int{40, 68},
		BuildingCount: [2]int{1, 4},
		BuildingSize:  [2]int{12, 34},
		TreeCount:     [2]int{4, 16},
		TreeChance:    14,
	}
	// Forest seasonal palettes.
	ThemeForestSpring = ThemeConfig{
		Name:          "Forest Spring",
		ParkChance:    [2]int{52, 80},
		BuildingCount: [2]int{1, 3},
		BuildingSize:  [2]int{12, 30},
		TreeCount:     [2]int{34, 72},
		TreeChance:    58,
		NoRoads:       true,
	}
	ThemeForestSummer = ThemeConfig{
		Name:          "Forest Summer",
		ParkChance:    [2]int{55, 82},
		BuildingCount: [2]int{1, 3},
		BuildingSize:  [2]int{12, 30},
		TreeCount:     [2]int{38, 78},
		TreeChance:    68,
		NoRoads:       true,
	}
	ThemeForestAutumn = ThemeConfig{
		Name:          "Forest Autumn",
		ParkChance:    [2]int{48, 76},
		BuildingCount: [2]int{1, 3},
		BuildingSize:  [2]int{12, 30},
		TreeCount:     [2]int{26, 58},
		TreeChance:    36,
		NoRoads:       true,
	}
	ThemeForestWinter = ThemeConfig{
		Name:          "Forest Winter",
		ParkChance:    [2]int{45, 72},
		BuildingCount: [2]int{1, 3},
		BuildingSize:  [2]int{12, 30},
		TreeCount:     [2]int{18, 44},
		TreeChance:    18,
		NoRoads:       true,
	}
	// ThemeSpace: outpost biome — wilderness style map with sparse structures.
	ThemeSpace = ThemeConfig{
		Name:          "Space",
		ParkChance:    [2]int{55, 80},
		BuildingCount: [2]int{1, 3},
		BuildingSize:  [2]int{10, 28},
		TreeCount:     [2]int{0, 1},
		TreeChance:    0,
		NoRoads:       true,
	}
	// ThemeUnderwater: reef labyrinth — no roads and scattered obstacle clusters.
	ThemeUnderwater = ThemeConfig{
		Name:          "Underwater",
		ParkChance:    [2]int{45, 70},
		BuildingCount: [2]int{1, 3},
		BuildingSize:  [2]int{12, 34},
		TreeCount:     [2]int{0, 2},
		TreeChance:    0,
		NoRoads:       true,
	}
	// ThemeMegacity: towering urban canyon — massive building footprints, almost no parks.
	ThemeMegacity = ThemeConfig{
		Name:          "Megacity",
		ParkChance:    [2]int{2, 8},
		BuildingCount: [2]int{3, 6},
		BuildingSize:  [2]int{45, 95},
		TreeCount:     [2]int{2, 8},
		TreeChance:    4,
	}
	// ThemeParkCity: green city — large open parks dominate, small scattered buildings.
	ThemeParkCity = ThemeConfig{
		Name:          "Park City",
		ParkChance:    [2]int{45, 75},
		BuildingCount: [2]int{2, 4},
		BuildingSize:  [2]int{12, 32},
		TreeCount:     [2]int{25, 60},
		TreeChance:    38,
	}
	// ThemeVillage: small town — tiny buildings, lots of greenery, open feel.
	ThemeVillage = ThemeConfig{
		Name:          "Village",
		ParkChance:    [2]int{20, 42},
		BuildingCount: [2]int{2, 5},
		BuildingSize:  [2]int{8, 26},
		TreeCount:     [2]int{12, 30},
		TreeChance:    32,
	}
	// ThemeIndustrial: grimy district — medium-large blocky buildings, sparse trees.
	ThemeIndustrial = ThemeConfig{
		Name:          "Industrial",
		ParkChance:    [2]int{5, 12},
		BuildingCount: [2]int{4, 8},
		BuildingSize:  [2]int{28, 72},
		TreeCount:     [2]int{3, 12},
		TreeChance:    6,
	}
	// ThemeWasteland: scorched district with sparse growth and battered structures.
	ThemeWasteland = ThemeConfig{
		Name:          "Wasteland",
		ParkChance:    [2]int{10, 24},
		BuildingCount: [2]int{2, 5},
		BuildingSize:  [2]int{14, 40},
		TreeCount:     [2]int{0, 8},
		TreeChance:    4,
	}
	// ThemeJungle: overgrown ruins with dense foliage and almost no roads.
	ThemeJungle = ThemeConfig{
		Name:          "Jungle",
		ParkChance:    [2]int{55, 82},
		BuildingCount: [2]int{1, 3},
		BuildingSize:  [2]int{10, 30},
		TreeCount:     [2]int{38, 74},
		TreeChance:    62,
		NoRoads:       true,
	}
	// ThemeCanyon: rocky mesa district with sparse trees and broader corridors.
	ThemeCanyon = ThemeConfig{
		Name:          "Canyon",
		ParkChance:    [2]int{18, 36},
		BuildingCount: [2]int{2, 4},
		BuildingSize:  [2]int{16, 44},
		TreeCount:     [2]int{1, 8},
		TreeChance:    5,
	}
	// ThemeVolcanic: ashlands with low vegetation and compact structures.
	ThemeVolcanic = ThemeConfig{
		Name:          "Volcanic",
		ParkChance:    [2]int{16, 30},
		BuildingCount: [2]int{2, 5},
		BuildingSize:  [2]int{14, 38},
		TreeCount:     [2]int{0, 3},
		TreeChance:    1,
	}
	// ThemeSwamp: murky lowlands with uneven structure density.
	ThemeSwamp = ThemeConfig{
		Name:          "Swamp",
		ParkChance:    [2]int{36, 62},
		BuildingCount: [2]int{1, 4},
		BuildingSize:  [2]int{10, 30},
		TreeCount:     [2]int{8, 28},
		TreeChance:    24,
	}
	// ThemeNeon: high-energy district with tight streets and low greenery.
	ThemeNeon = ThemeConfig{
		Name:          "Neon",
		ParkChance:    [2]int{4, 14},
		BuildingCount: [2]int{5, 10},
		BuildingSize:  [2]int{18, 58},
		TreeCount:     [2]int{1, 8},
		TreeChance:    3,
	}
	// ThemeRuins: broken city blocks mixed with reclaimed vegetation.
	ThemeRuins = ThemeConfig{
		Name:          "Ruins",
		ParkChance:    [2]int{28, 54},
		BuildingCount: [2]int{2, 5},
		BuildingSize:  [2]int{12, 34},
		TreeCount:     [2]int{10, 26},
		TreeChance:    20,
	}
	// ThemeFarmland: open agricultural town with sparse low-rise buildings.
	ThemeFarmland = ThemeConfig{
		Name:          "Farmland",
		ParkChance:    [2]int{34, 58},
		BuildingCount: [2]int{1, 3},
		BuildingSize:  [2]int{12, 28},
		TreeCount:     [2]int{8, 24},
		TreeChance:    18,
	}
	// ThemeHighlands: windy plateau settlements with medium structures.
	ThemeHighlands = ThemeConfig{
		Name:          "Highlands",
		ParkChance:    [2]int{26, 44},
		BuildingCount: [2]int{2, 5},
		BuildingSize:  [2]int{14, 36},
		TreeCount:     [2]int{6, 20},
		TreeChance:    14,
	}

	Themes = []ThemeConfig{
		ThemeCity, ThemeSuburban, ThemeForest, ThemeRural,
		ThemeArctic, ThemeDesert, ThemeSand, ThemeWinter, ThemeBeach,
		ThemeForestSpring, ThemeForestSummer, ThemeForestAutumn, ThemeForestWinter,
		ThemeSpace, ThemeUnderwater,
		ThemeMegacity, ThemeParkCity, ThemeVillage, ThemeIndustrial,
		ThemeWasteland, ThemeJungle, ThemeCanyon, ThemeVolcanic,
		ThemeSwamp, ThemeNeon, ThemeRuins, ThemeFarmland, ThemeHighlands,
	}
)

func (t ThemeConfig) FamilyName() string {
	if t.Family != "" {
		return t.Family
	}
	return t.Name
}

func jitterRange(base [2]int, r *Rand, lo, hi, dMin, dMax, minSpan int) [2]int {
	minV := clamp(base[0]+r.Range(-dMin, dMin), lo, hi)
	maxV := clamp(base[1]+r.Range(-dMax, dMax), lo, hi)
	if minV > maxV {
		minV, maxV = maxV, minV
	}
	if maxV-minV < minSpan {
		need := minSpan - (maxV - minV)
		minV -= need / 2
		maxV += need - need/2
		if minV < lo {
			maxV += lo - minV
			minV = lo
		}
		if maxV > hi {
			minV -= maxV - hi
			maxV = hi
		}
		minV = clamp(minV, lo, hi)
		maxV = clamp(maxV, lo, hi)
		if minV > maxV {
			minV = maxV
		}
	}
	return [2]int{minV, maxV}
}

func themeVariantTag(family string, r *Rand) string {
	common := []string{
		"Wild", "Dense", "Sparse", "Chaotic", "Stormfront",
		"Rift", "Prime", "Delta", "Echo", "Shift",
	}
	switch family {
	case ThemeArctic.Name:
		return []string{"Whiteout", "Frostbite", "Drift", "Cold Snap", "Aurora"}[r.Intn(5)]
	case ThemeDesert.Name:
		return []string{"Dust Storm", "Dune Run", "Heatwave", "Red Sand", "Mirage"}[r.Intn(5)]
	case ThemeSand.Name:
		return []string{"Drifting Dunes", "Dust Devil", "Sunbaked", "Blown Sand", "Sirocco"}[r.Intn(5)]
	case ThemeWinter.Name:
		return []string{"Snowfall", "Ice Drift", "Black Ice", "Frostline", "Cold Front"}[r.Intn(5)]
	case ThemeBeach.Name:
		return []string{"Low Tide", "High Surf", "Salt Wind", "Coastline", "Boardwalk"}[r.Intn(5)]
	case ThemeForestSpring.Name:
		return []string{"Bloom", "Fresh Growth", "Rainy Trail", "Wildflower", "New Canopy"}[r.Intn(5)]
	case ThemeForestSummer.Name:
		return []string{"Thick Canopy", "Heat Haze", "Cicada", "Sunlit Grove", "Deep Green"}[r.Intn(5)]
	case ThemeForestAutumn.Name:
		return []string{"Leaf Fall", "Amber Woods", "Harvest Wind", "Copper Trail", "Rust Canopy"}[r.Intn(5)]
	case ThemeForestWinter.Name:
		return []string{"Frozen Pines", "Snow Canopy", "White Timber", "Frostwood", "Ice Bark"}[r.Intn(5)]
	case ThemeSpace.Name:
		return []string{"Voidline", "Orbital", "Asteroid", "Nebula", "Zero-G"}[r.Intn(5)]
	case ThemeUnderwater.Name:
		return []string{"Abyss", "Trench", "Kelp Maze", "Pressure", "Current"}[r.Intn(5)]
	case ThemeForest.Name:
		return []string{"Deepwood", "Canopy", "Thicket", "Old Growth", "Night Grove"}[r.Intn(5)]
	case ThemeNeon.Name:
		return []string{"Hypergrid", "Synth", "Afterglow", "Night Pulse", "Chromatic"}[r.Intn(5)]
	case ThemeVolcanic.Name:
		return []string{"Ashfall", "Magma Drift", "Cinder", "Basalt", "Eruption"}[r.Intn(5)]
	case ThemeJungle.Name:
		return []string{"Overgrowth", "Monsoon", "Vine Trap", "Rainfall", "Tangle"}[r.Intn(5)]
	default:
		return common[r.Intn(len(common))]
	}
}

func ThemeVariant(base ThemeConfig, r *Rand, level int) ThemeConfig {
	v := base
	v.Family = base.FamilyName()

	intensity := clamp(level/5, 0, 10)
	delta := 7 + intensity

	v.ParkChance = jitterRange(base.ParkChance, r, 0, 90, 9+delta, 12+delta, 6)
	v.BuildingCount = jitterRange(base.BuildingCount, r, 1, 16, 3+delta/3, 4+delta/2, 1)
	v.BuildingSize = jitterRange(base.BuildingSize, r, 8, 96, 6+delta, 10+delta, 6)
	v.TreeCount = jitterRange(base.TreeCount, r, 0, 90, 6+delta, 9+delta, 3)
	v.TreeChance = clamp(base.TreeChance+r.Range(-(12+delta), 12+delta), 0, 90)

	if v.NoRoads {
		v.BuildingCount = jitterRange(base.BuildingCount, r, 1, 10, 2, 3+delta/3, 1)
		v.TreeChance = clamp(v.TreeChance+r.Range(10, 24), 0, 95)
	}

	v.Name = v.Family + " [" + themeVariantTag(v.Family, r) + "]"
	return v
}

func themePickWeight(t ThemeConfig) int {
	switch t.FamilyName() {
	case ThemeWinter.Name, ThemeForestWinter.Name:
		return 4
	case ThemeArctic.Name:
		return 2
	default:
		return 1
	}
}

func PickLevelTheme(seed uint64, level int, roll uint64, lastIdx int) (ThemeConfig, int) {
	if len(Themes) == 0 {
		return ThemeCity, -1
	}
	r := NewRand(seed ^ uint64(level)*0xBAD5EED ^ (roll+1)*0x9E3779B185EBCA87 ^ 0xC0FFEE55AA55C0DE)
	if len(Themes) == 1 {
		return ThemeVariant(Themes[0], r, level), 0
	}
	totalWeight := 0
	for i := range Themes {
		if i == lastIdx {
			continue
		}
		w := themePickWeight(Themes[i])
		if w < 1 {
			w = 1
		}
		totalWeight += w
	}
	if totalWeight <= 0 {
		themeIdx := r.Intn(len(Themes))
		return ThemeVariant(Themes[themeIdx], r, level), themeIdx
	}
	rollWeight := r.Intn(totalWeight)
	themeIdx := 0
	for i := range Themes {
		if i == lastIdx {
			continue
		}
		w := themePickWeight(Themes[i])
		if w < 1 {
			w = 1
		}
		if rollWeight < w {
			themeIdx = i
			break
		}
		rollWeight -= w
	}
	return ThemeVariant(Themes[themeIdx], r, level), themeIdx
}
