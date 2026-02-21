package game

type GameState int

const (
	StateMenu          GameState = iota
	StatePlaying                 // main gameplay
	StateLevelComplete           // all peds eaten
	StateLevelFailed             // snake died
)

type GameSession struct {
	State        GameState
	CurrentLevel int
	ThemeName    string
	Weather      WeatherType
	WeatherSeed  uint64
	LevelTimer   float64
	Score        int

	LastThemeIdx int
	ThemeRoll    uint64
}

func NewGameSession() *GameSession {
	return &GameSession{
		State:        StateMenu,
		LastThemeIdx: -1,
	}
}

// StartLevel resets entities and begins a new level.
func (s *GameSession) StartLevel(level int, world *World, peds *PedestrianSystem, traffic *TrafficSystem, bonuses *BonusSystem, cops *CopSystem, mil *MilitarySystem, snake **Snake, particles *ParticleSystem, seed uint64) {
	s.CurrentLevel = level
	s.Score = 0
	s.State = StatePlaying

	cfg := GetLevelConfig(level)
	if len(Themes) > 0 {
		theme, themeIdx := PickLevelTheme(seed, level, s.ThemeRoll, s.LastThemeIdx)
		s.ThemeRoll++
		cfg.Theme = theme
		s.LastThemeIdx = themeIdx
	}
	// Per-level mixed seed: keeps level flow varied across theme rolls/retries.
	levelSeed := hash2D(seed^uint64(level)*0xA11CE5ED^s.ThemeRoll*0x9E3779B185EBCA87, level, s.LastThemeIdx+1)
	// No-road themes disable traffic; compensate with extra peds.
	if cfg.Theme.NoRoads {
		cfg.Cars = 0
		cfg.Peds = cfg.Peds * 3 / 2
	}
	s.ThemeName = cfg.Theme.Name
	// Random starting time of day.
	r := NewRand(levelSeed ^ 0xBAD5EED)
	s.LevelTimer = r.RangeF(0, DayCyclePeriod)
	s.WeatherSeed = levelSeed ^ 0x57A7E12D4F3CB71D
	s.Weather = PickLevelWeather(cfg.Theme, NewRand(s.WeatherSeed^uint64(level)*0x9E3779B185EBCA87), level)

	// Regenerate world with level-varied seed and theme.
	world.seed = levelSeed
	world.Theme = cfg.Theme
	world.burningTrees = make(map[int64]*TreeBurn)
	world.burningBuildings = make(map[int64]*BuildingBurn)
	world.temp = world.temp[:0]
	world.scheduled = world.scheduled[:0]
	for i := range world.chunks {
		world.chunks[i] = nil
	}
	world.GenerateAll()
	world.BuildSpatialIndex()
	for cy := 0; cy <= world.maxCy; cy++ {
		for cx := 0; cx <= world.maxCx; cx++ {
			c := world.GetChunk(cx, cy)
			if c != nil {
				c.RecomputeShadows(world)
			}
		}
	}

	// Reset pedestrians.
	peds.seed = levelSeed ^ 0xFED5EED
	peds.SetEnvironment(cfg.Theme.FamilyName())
	peds.P = peds.P[:0]
	peds.SpawnRandom(world, cfg.Peds)
	peds.SpawnArmed(world, cfg.ArmedPeds)
	peds.SpawnInfected(world, cfg.InfectedPeds)

	// Reset traffic.
	traffic.seed = levelSeed ^ 0xCAFE5EED
	traffic.SetEnvironment(cfg.Theme.FamilyName())
	traffic.Cars = traffic.Cars[:0]
	traffic.SpawnRandom(world, cfg.Cars)
	traffic.RebuildGrid()

	// Reset bonuses.
	bonuses.seed = levelSeed ^ 0xB0B5EED
	bonuses.spawnSeq = 0
	bonuses.lastKind = -1
	bonuses.SpawnTimer = 3.0 + NewRand(levelSeed^0xB0B5EED^0x51A3E).RangeF(0, 4.0)
	bonuses.Boxes = bonuses.Boxes[:0]
	bonuses.SpawnRandom(cfg.BonusBoxes)

	// Reset cops and military.
	cops.Reset()
	mil.Reset()

	// Reset particles.
	particles.Clear()

	// Spawn snake at world center on a road.
	sx := float64(WorldWidth/2/Pattern*Pattern + RoadWidth/2)
	sy := float64(WorldHeight/2/Pattern*Pattern + RoadWidth/2)
	*snake = NewSnake(sx, sy, LevelSpeed(level))
}

// Update advances the level timer.
func (s *GameSession) Update(dt float64) {
	if s.State == StatePlaying {
		s.LevelTimer += dt
	}
}

// CheckLevelEnd checks win/lose and updates score from snake.
func (s *GameSession) CheckLevelEnd(peds *PedestrianSystem, snake *Snake) {
	if s.State != StatePlaying {
		return
	}
	if snake != nil {
		s.Score = snake.Score
	}

	// Lose: snake is dead.
	if snake == nil || !snake.Alive {
		s.State = StateLevelFailed
		PlaySound(SoundGameOver)
		return
	}

	// Win: all peds eaten.
	if peds.AliveCount() == 0 {
		s.State = StateLevelComplete
		PlaySound(SoundLevelUp)
	}
}
