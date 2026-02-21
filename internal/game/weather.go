package game

type WeatherType uint8

const (
	WeatherNone WeatherType = iota
	WeatherRain
	WeatherSnow
)

// PickLevelWeather chooses precipitation for a level based on theme family.
func PickLevelWeather(theme ThemeConfig, r *Rand, level int) WeatherType {
	if r == nil {
		r = NewRand(uint64(level+1) * 0x9E3779B185EBCA87)
	}
	family := theme.FamilyName()
	rainChance := 0
	snowChance := 0

	switch family {
	case ThemeArctic.Name, ThemeWinter.Name, ThemeForestWinter.Name:
		snowChance = 90
	case ThemeHighlands.Name:
		rainChance = 18
		snowChance = 26
	case ThemeSwamp.Name, ThemeJungle.Name:
		rainChance = 74
	case ThemeForestSpring.Name, ThemeForestSummer.Name, ThemeForestAutumn.Name, ThemeForest.Name:
		rainChance = 46
	case ThemeBeach.Name, ThemeSuburban.Name, ThemeCity.Name, ThemeRural.Name, ThemeParkCity.Name, ThemeVillage.Name:
		rainChance = 32
	case ThemeDesert.Name, ThemeSand.Name, ThemeVolcanic.Name, ThemeCanyon.Name, ThemeSpace.Name, ThemeUnderwater.Name:
		rainChance = 0
		snowChance = 0
	default:
		rainChance = 20
	}

	// Slightly more weather pressure on higher levels.
	bonus := clamp(level/6, 0, 10)
	rainChance = clamp(rainChance+bonus, 0, 95)
	snowChance = clamp(snowChance+bonus/2, 0, 95)

	roll := r.Intn(100)
	if roll < snowChance {
		return WeatherSnow
	}
	if roll < snowChance+rainChance {
		return WeatherRain
	}
	return WeatherNone
}

type WeatherSystem struct {
	seed      uint64
	mode      WeatherType
	intensity float64
	windX     float64
	spawnAcc  float64
	gustAcc   float64
	spawnSeq  uint64
}

func NewWeatherSystem(seed uint64) *WeatherSystem {
	if seed == 0 {
		seed = 1
	}
	return &WeatherSystem{
		seed:      seed,
		mode:      WeatherNone,
		intensity: 1.0,
	}
}

func (ws *WeatherSystem) Configure(mode WeatherType, seed uint64) {
	if ws == nil {
		return
	}
	if seed == 0 {
		seed = 1
	}
	ws.seed = seed ^ 0x57A7E12D
	ws.mode = mode
	ws.spawnAcc = 0
	ws.gustAcc = 0
	ws.spawnSeq = 0

	r := NewRand(ws.seed ^ 0xA24BAED4)
	ws.intensity = 0.78 + r.RangeF(0, 0.62)
	ws.windX = r.RangeF(-14.0, 14.0)
}

func (ws *WeatherSystem) UpdateAndSpawn(ps *ParticleSystem, dt float64) {
	if ws == nil || ps == nil || dt <= 0 || ws.mode == WeatherNone {
		return
	}

	// Slow gust drift so rain/snow direction changes over time.
	ws.gustAcc += dt
	if ws.gustAcc >= 0.6 {
		g := NewRand(ws.seed ^ uint64(int(ws.gustAcc*1000)+1)*0xC2B2AE3D27D4EB4F ^ ws.spawnSeq)
		ws.windX = clampF(ws.windX+g.RangeF(-2.8, 2.8), -18.0, 18.0)
		ws.gustAcc = 0
	}

	rate := 0.0
	switch ws.mode {
	case WeatherRain:
		rate = 150.0 * ws.intensity
	case WeatherSnow:
		rate = 82.0 * ws.intensity
	default:
		return
	}

	ws.spawnAcc += rate * dt
	count := int(ws.spawnAcc)
	if count <= 0 {
		return
	}
	ws.spawnAcc -= float64(count)

	for i := 0; i < count; i++ {
		ws.spawnSeq++
		r := NewRand(ws.seed ^ ws.spawnSeq*0x9E3779B185EBCA87)
		x := r.RangeF(-10.0, float64(WorldWidth)+10.0)
		y := r.RangeF(-10.0, float64(WorldHeight)+10.0)

		switch ws.mode {
		case WeatherRain:
			ps.Add(Particle{
				X: x, Y: y,
				VX:      ws.windX*0.35 + r.RangeF(-8.0, 8.0),
				VY:      94.0 + r.RangeF(0.0, 52.0),
				Size:    0.50 + r.RangeF(0.0, 0.20),
				Life:    0,
				MaxLife: 0.70 + r.RangeF(0.0, 0.70),
				Col:     RGB{R: 175, G: 195, B: 220},
				Kind:    ParticleRain,
			})
		case WeatherSnow:
			ps.Add(Particle{
				X: x, Y: y,
				VX:      ws.windX + r.RangeF(-9.0, 9.0),
				VY:      18.0 + r.RangeF(0.0, 18.0),
				Size:    0.78 + r.RangeF(0.0, 0.95),
				Life:    0,
				MaxLife: 2.20 + r.RangeF(0.0, 2.00),
				Col:     RGB{R: 235, G: 242, B: 250},
				Kind:    ParticleSnow,
			})
		}
	}
}
