package game

import (
	"fmt"
	"math"
)

// PathPoint is a 2D position stored in the snake's path ring buffer.
type PathPoint struct {
	X, Y float64
}

// GhostSnake is a body segment split off during the swarm powerup.
// Segments hunt peds independently, then return to reintegrate with the snake.
type GhostSnake struct {
	X, Y             float64
	Heading          float64
	TargetX, TargetY float64
	PrevX, PrevY     float64
	StuckTimer       float64
	SegLen           float64 // body length this segment represents (restored on return)
	Returning        bool    // true = heading back to snake head
	Exploded         bool    // true after the one-shot dive explosion
	FireBoltTimer    float64 // seconds of orbiting fire bolts remaining
	FireBoltAngle    float64 // current rotation angle for the 5-bolt ring
}

// SpreadBomb is a timed explosive dropped along the snake's path.
type SpreadBomb struct {
	X, Y float64
	Fuse float64 // seconds until detonation
}

// CloneSnake mirrors the main snake's heading at a different world position.
type CloneSnake struct {
	X, Y          float64
	Heading       float64
	Path          []PathPoint
	Length        float64
	FireBoltTimer float64 // seconds of orbiting fire bolts remaining
	FireBoltAngle float64 // current rotation angle for the 5-bolt ring
}

// VacuumBubble is an expanding suction field that pulls particles and erases world pixels.
type VacuumBubble struct {
	X, Y    float64
	Timer   float64
	MaxTime float64
}

// HomingMissile is fired by the missile bonus and steers toward nearby targets.
type HomingMissile struct {
	X, Y   float64
	VX, VY float64
	Life   float64
}

// StrikeWorm is a short-lived hunter spawned by targeted worm strike.
type StrikeWorm struct {
	X, Y    float64
	Heading float64
	Life    float64
}

// StrikeHeli is a temporary gunship spawned by targeted airstrike.
type StrikeHeli struct {
	X, Y             float64
	Heading          float64
	RotorAngle       float64
	CenterX, CenterY float64
	AttackX, AttackY float64
	OrbitA, OrbitR   float64
	OrbitSpd         float64
	DeployTimer      float64 // spread to attack position before opening fire
	Life             float64 // attack window once deployed
	FireTimer        float64
	MissileMode      bool // true = missile support, false = gatling support
	Exiting          bool
	ExitVX, ExitVY   float64
	ExitTimer        float64
	Leader           bool // leader helis are rendered brighter red
}

// StrikeHeliShot is a bullet fired by temporary strike helicopters.
type StrikeHeliShot struct {
	X, Y   float64
	VX, VY float64
	Life   float64
}

// StrikeHeliMissile is a short-lived homing missile fired by strike helicopters.
type StrikeHeliMissile struct {
	X, Y       float64
	VX, VY     float64
	Life       float64
	TargetDist float64 // max distance from missile position to acquire a target
}

// SnakeRound is a straight projectile fired by the gatling bonus.
type SnakeRound struct {
	X, Y   float64
	VX, VY float64
	Life   float64
}

// BeltBomb is queued by targeted carpet bomb strikes.
type BeltBomb struct {
	X, Y   float64
	Fuse   float64
	Radius int
}

// StrikePlane is spawned by targeted air support.
type StrikePlane struct {
	X, Y       float64
	VX, VY     float64
	Heading    float64
	Life       float64
	TargetX    float64
	TargetY    float64
	Dropped    bool
	BombsLeft  int
	BombRadius int
}

// StrikePlaneBomb is dropped by strike planes and explodes on fuse expiry.
type StrikePlaneBomb struct {
	X, Y   float64
	VX, VY float64
	Fuse   float64
	Radius int
}

type TimedExploderKind int

const (
	TimedExploderPig TimedExploderKind = iota
	TimedExploderCar
	TimedExploderSnake
)

// TimedExploder roams until its timer expires, then detonates.
type TimedExploder struct {
	Kind    TimedExploderKind
	X, Y    float64
	Heading float64
	Speed   float64
	Timer   float64
	Radius  int
}

// Snake is the player-controlled entity.
type Snake struct {
	Path    []PathPoint // recent head positions, index 0 = newest
	PathCap int

	Length        float64 // current visual length in world pixels
	Heading       float64 // current direction in radians
	TargetHeading float64 // player's desired heading (for post-bounce correction)
	Speed         float64
	BaseSpeed     float64
	SpeedBoost    float64 // seconds remaining
	SpeedMult     float64 // multiplier while boosted

	HP    Health
	Score int

	PukeTimer     float64      // seconds of green puke effect remaining
	BashTimer     float64      // seconds of building-bash powerup remaining
	SurgeTimer    float64      // seconds of ped-suction remaining
	Ghosts        []GhostSnake // active ghost clones (swarm powerup)
	GhostTimer    float64      // seconds of ghost swarm remaining
	GhostBombMode bool         // true: ghosts run dive-bomb behavior instead of hunting

	PowerupMsg   string  // name of last collected powerup
	PowerupTimer float64 // seconds until powerup label fades (2.5s total)
	PowerupCol   RGB     // color matching the powerup box

	// AI mode: snake autonomously hunts peds.
	AITimer       float64 // seconds of AI control remaining
	AITargetsLeft int     // peds left to hunt before AI ends

	// Berserk mode: giant snake rampages with AI and bash.
	BerserkTimer float64 // seconds remaining in berserk mode
	SizeMult     float64 // visual size multiplier (default 1.0)

	// Teleport: queue of positions to jump to in rapid sequence.
	TeleportQueue []PathPoint
	TeleportTimer float64 // delay between jumps

	// Fire bolts: 5 bolts orbit the snake head, burning everything in their path.
	FireRingTimer float64
	FireBoltAngle float64

	// Spread bombs: dropped along path, detonate after fuse.
	SpreadTimer     float64
	SpreadDropTimer float64
	SpreadBombs     []SpreadBomb

	// Clone: mirror snakes at other world positions.
	CloneTimer float64
	Clones     []CloneSnake

	// Vacuum bubbles: expanding suction fields.
	VacuumBubbles []VacuumBubble

	// Missile barrage bonus.
	MissileTimer     float64
	MissileFireTimer float64
	Missiles         []HomingMissile

	// Gatling bonus.
	GatlingTimer     float64
	GatlingFireTimer float64
	GatlingRounds    []SnakeRound

	// Tactical nuke targeting mode.
	TargetNukeTimer   float64   // seconds remaining in point-and-click targeting mode
	TargetAbilityKind BonusKind // active targeting payload while TargetNukeTimer > 0
	TargetMinClicks   int
	TargetMaxClicks   int
	TargetClicksMade  int
	TargetRoll        uint64

	// Point-and-click temporary strike entities.
	StrikeWorms      []StrikeWorm
	StrikeHelis      []StrikeHeli
	StrikeHeliShots  []StrikeHeliShot
	StrikeHeliMiss   []StrikeHeliMissile
	BeltBombs        []BeltBomb
	StrikePlanes     []StrikePlane
	StrikePlaneBombs []StrikePlaneBomb
	TimedExploders   []TimedExploder

	WantedLevel float64 // 0–100: drives cop escalation

	Idle        bool    // true when cursor is stationary over the head
	RattlePhase float64 // oscillation phase for idle figure-8 animation
	IdleBaseX   float64 // world position to center the figure-8 around
	IdleBaseY   float64

	// Kill streak combo tracking.
	KillStreak      int     // consecutive kills within the streak window
	KillStreakTimer float64 // resets to 2.5s on each kill; at 0 the streak ends
	KillMsg         string  // latest streak announcement label
	KillMsgTimer    float64 // display timer, fades over 2s
	KillMsgCol      RGB

	// Gore trail: blood drips after eating.
	GoreTimer   float64 // seconds remaining for dripping blood
	GoreDripAcc float64 // accumulator for drip timing

	// Evolution system.
	EvoPoints int // points earned from eating
	EvoLevel  int // current evolution level (0-5)

	// Flamethrower bonus: breathe fire while active.
	FlamethrowerTimer float64 // seconds remaining for flamethrower
	FireBreathActive  bool    // true while player is holding fire button

	// Wall stuck detection: tracks when snake hasn't moved.
	PrevX, PrevY  float64
	StuckTimer    float64
	StuckAttempts int // consecutive unstuck attempts without meaningful movement

	// Bounce override: after a wall escape, hold the escape heading for a
	// short time so Steer() can't immediately pull the snake back into the wall.
	BounceTimer float64 // seconds remaining; while > 0 overrides mouse steer
	BounceDir   float64 // heading to hold during bounce

	Alive bool
}

func NewSnake(x, y float64, speed float64) *Snake {
	cap := 512
	path := make([]PathPoint, 1, cap)
	path[0] = PathPoint{X: x, Y: y}
	return &Snake{
		Path:      path,
		PathCap:   cap,
		Length:    SnakeStartLength,
		Heading:   0,
		Speed:     speed,
		BaseSpeed: speed,
		SpeedMult: 1.0,
		SizeMult:  1.0,
		HP:        NewHealth(10),
		Alive:     true,
	}
}

// LevelSpeed returns the snake speed for a given level.
// Starts at SnakeBaseSpeed and increases by 3 per level.
func LevelSpeed(level int) float64 {
	return SnakeBaseSpeed + float64(level-1)*3.0
}

func (s *Snake) evoSpeedMult() float64 {
	lvl := clamp(s.EvoLevel, 0, 5)
	return 1.0 + float64(lvl)*0.09
}

func (s *Snake) evoHeadColor() RGB {
	palette := [6]RGB{
		{R: 40, G: 200, B: 26},   // base toxic green
		{R: 70, G: 220, B: 60},   // brighter green
		{R: 80, G: 235, B: 130},  // green-cyan
		{R: 95, G: 215, B: 205},  // cyan
		{R: 145, G: 185, B: 245}, // ice blue
		{R: 225, G: 120, B: 255}, // violet apex
	}
	return palette[clamp(s.EvoLevel, 0, len(palette)-1)]
}

func (s *Snake) evoSegmentColor(t float64) RGB {
	head := s.evoHeadColor()
	// Keep tail darker so body depth remains readable.
	tail := RGB{
		R: uint8(float64(head.R) * 0.42),
		G: uint8(float64(head.G) * 0.42),
		B: uint8(float64(head.B) * 0.42),
	}
	return lerpRGB(head, tail, t)
}

func nearestAlivePed(mx, my float64, peds *PedestrianSystem) (idx int, tx, ty float64, ok bool) {
	if peds == nil {
		return 0, 0, 0, false
	}
	best := math.MaxFloat64
	bestIdx := -1
	for i := range peds.P {
		p := &peds.P[i]
		if !p.Alive {
			continue
		}
		d := math.Hypot(p.X-mx, p.Y-my)
		if d < best {
			best = d
			bestIdx = i
			tx, ty = p.X, p.Y
		}
	}
	if bestIdx < 0 {
		return 0, 0, 0, false
	}
	return bestIdx, tx, ty, true
}

func nearestAlivePedWithin(mx, my float64, peds *PedestrianSystem, maxDist float64) (idx int, tx, ty float64, ok bool) {
	if peds == nil || maxDist <= 0 {
		return 0, 0, 0, false
	}
	best := maxDist * maxDist
	bestIdx := -1
	for i := range peds.P {
		p := &peds.P[i]
		if !p.Alive {
			continue
		}
		dx := p.X - mx
		dy := p.Y - my
		d2 := dx*dx + dy*dy
		if d2 <= best {
			best = d2
			bestIdx = i
			tx, ty = p.X, p.Y
		}
	}
	if bestIdx < 0 {
		return 0, 0, 0, false
	}
	return bestIdx, tx, ty, true
}

func nearestStrikeHeliTargetWithin(mx, my float64, maxDist float64, peds *PedestrianSystem, cops *CopSystem, mil *MilitarySystem) (tx, ty float64, ok bool) {
	if maxDist <= 0 {
		return 0, 0, false
	}
	bestHostile := maxDist * maxDist
	foundHostile := false
	bestPed := maxDist * maxDist
	foundPed := false

	tryHostile := func(x, y float64) {
		dx := x - mx
		dy := y - my
		d2 := dx*dx + dy*dy
		if d2 <= bestHostile {
			bestHostile = d2
			tx = x
			ty = y
			foundHostile = true
		}
	}

	if cops != nil {
		for i := range cops.Peds {
			p := &cops.Peds[i]
			if !p.Alive {
				continue
			}
			tryHostile(p.X, p.Y)
		}
		for i := range cops.Cars {
			c := &cops.Cars[i]
			if !c.Alive {
				continue
			}
			tryHostile(c.X, c.Y)
		}
		for i := range cops.Helis {
			h := &cops.Helis[i]
			if !h.Alive {
				continue
			}
			tryHostile(h.X, h.Y)
		}
	}

	if mil != nil {
		for i := range mil.Troops {
			t := &mil.Troops[i]
			if !t.Alive {
				continue
			}
			tryHostile(t.X, t.Y)
		}
		for i := range mil.Tanks {
			t := &mil.Tanks[i]
			if !t.Alive {
				continue
			}
			tryHostile(t.X, t.Y)
		}
		for i := range mil.Helis {
			h := &mil.Helis[i]
			if !h.Alive {
				continue
			}
			tryHostile(h.X, h.Y)
		}
	}

	if foundHostile {
		return tx, ty, true
	}

	if peds != nil {
		for i := range peds.P {
			p := &peds.P[i]
			if !p.Alive {
				continue
			}
			dx := p.X - mx
			dy := p.Y - my
			d2 := dx*dx + dy*dy
			if d2 <= bestPed {
				bestPed = d2
				tx = p.X
				ty = p.Y
				foundPed = true
			}
		}
	}

	return tx, ty, foundPed
}

func strikeWormWalkable(world *World, x, y float64) bool {
	if world == nil {
		return true
	}
	ix := int(math.Round(x))
	iy := int(math.Round(y))
	if ix < 0 || iy < 0 || ix >= WorldWidth || iy >= WorldHeight {
		return false
	}
	return world.HeightAt(ix, iy) == 0
}

func nearestWalkablePoint(world *World, x, y float64, maxRadius int) (float64, float64, bool) {
	if world == nil {
		return clampF(x, 0, float64(WorldWidth-1)), clampF(y, 0, float64(WorldHeight-1)), true
	}
	ix := clamp(int(math.Round(x)), 0, WorldWidth-1)
	iy := clamp(int(math.Round(y)), 0, WorldHeight-1)
	if world.HeightAt(ix, iy) == 0 {
		return float64(ix) + 0.5, float64(iy) + 0.5, true
	}
	for r := 1; r <= maxRadius; r++ {
		minX := clamp(ix-r, 0, WorldWidth-1)
		maxX := clamp(ix+r, 0, WorldWidth-1)
		minY := clamp(iy-r, 0, WorldHeight-1)
		maxY := clamp(iy+r, 0, WorldHeight-1)
		for px := minX; px <= maxX; px++ {
			if world.HeightAt(px, minY) == 0 {
				return float64(px) + 0.5, float64(minY) + 0.5, true
			}
			if world.HeightAt(px, maxY) == 0 {
				return float64(px) + 0.5, float64(maxY) + 0.5, true
			}
		}
		for py := minY + 1; py <= maxY-1; py++ {
			if world.HeightAt(minX, py) == 0 {
				return float64(minX) + 0.5, float64(py) + 0.5, true
			}
			if world.HeightAt(maxX, py) == 0 {
				return float64(maxX) + 0.5, float64(py) + 0.5, true
			}
		}
	}
	return x, y, false
}

func targetAbilityPickRange(kind BonusKind) (minClicks, maxClicks int) {
	switch kind {
	case BonusTargetBombBelt, BonusTargetAirSupport:
		return 3, 5
	case BonusTargetPigs, BonusTargetCars, BonusTargetSnakes:
		return 3, 5
	default:
		return 1, 1
	}
}

func (s *Snake) beginTargetAbility(kind BonusKind, duration float64) {
	minClicks, maxClicks := targetAbilityPickRange(kind)
	// Roaming exploder abilities use a randomized required count (2-5)
	// so each activation feels different.
	if kind == BonusTargetPigs || kind == BonusTargetCars || kind == BonusTargetSnakes {
		hx, hy := s.Head()
		s.TargetRoll++
		r := NewRand(
			uint64(int(hx))*0x9E3779B185EBCA87 ^
				uint64(int(hy))*0xC2B2AE3D27D4EB4F ^
				uint64(s.Score+1)*0x165667B19E3779F9 ^
				s.TargetRoll*0xA24BAED4963EE407 ^
				uint64(kind+1)*0x94D049BB133111EB,
		)
		count := 2 + r.Intn(4)
		minClicks, maxClicks = count, count
	}
	s.TargetAbilityKind = kind
	s.TargetNukeTimer = duration
	s.TargetMinClicks = minClicks
	s.TargetMaxClicks = maxClicks
	s.TargetClicksMade = 0
}

func (s *Snake) targetProgress() (picked, minClicks, maxClicks int) {
	picked = s.TargetClicksMade
	minClicks = max(1, s.TargetMinClicks)
	maxClicks = max(minClicks, s.TargetMaxClicks)
	if picked < 0 {
		picked = 0
	}
	if picked > maxClicks {
		picked = maxClicks
	}
	return
}

func (s *Snake) TargetingPrompt(rem float64) (string, RGB) {
	if rem < 0 {
		rem = 0
	}
	picked, _, maxClicks := s.targetProgress()
	switch s.TargetAbilityKind {
	case BonusTargetWorms:
		return fmt.Sprintf("CLICK TO DEPLOY WORMS %.1fs", rem), RGB{R: 120, G: 255, B: 145}
	case BonusTargetGunship:
		return fmt.Sprintf("CLICK FOR HELICOPTER\nSUPPORT %.1fs", rem), RGB{R: 255, G: 95, B: 80}
	case BonusTargetHeliMissile:
		return fmt.Sprintf("CLICK FOR MISSILE HELI\nSUPPORT %.1fs", rem), RGB{R: 255, G: 130, B: 55}
	case BonusTargetBombBelt:
		return fmt.Sprintf("MARK CARPET BOMB %d/%d\n%.1fs", picked, maxClicks, rem), RGB{R: 255, G: 185, B: 90}
	case BonusTargetAirSupport:
		return fmt.Sprintf("MARK AIR SUPPORT %d/%d\n%.1fs", picked, maxClicks, rem), RGB{R: 145, G: 200, B: 255}
	case BonusTargetPigs:
		return fmt.Sprintf("PLACE EXPLODING PIGS %d/%d\n%.1fs", picked, maxClicks, rem), RGB{R: 255, G: 130, B: 175}
	case BonusTargetCars:
		return fmt.Sprintf("PLACE EXPLODING R/C CARS %d/%d\n%.1fs", picked, maxClicks, rem), RGB{R: 255, G: 150, B: 105}
	case BonusTargetSnakes:
		return fmt.Sprintf("PLACE SNAKE BOMBERS %d/%d\n%.1fs", picked, maxClicks, rem), RGB{R: 140, G: 255, B: 130}
	default:
		return fmt.Sprintf("CLICK TO NUKE %.1fs", rem), RGB{R: 255, G: 245, B: 170}
	}
}

func (s *Snake) TargetingLockLostPrompt() (string, RGB) {
	picked, _, maxClicks := s.targetProgress()
	if maxClicks > 1 && picked > 0 {
		switch s.TargetAbilityKind {
		case BonusTargetBombBelt:
			return "CARPET BOMB WINDOW\nCLOSED", RGB{R: 255, G: 185, B: 90}
		case BonusTargetAirSupport:
			return "AIR SUPPORT WINDOW\nCLOSED", RGB{R: 145, G: 200, B: 255}
		case BonusTargetPigs:
			return "EXPLODING PIGS\nWINDOW CLOSED", RGB{R: 255, G: 130, B: 175}
		case BonusTargetCars:
			return "EXPLODING R/C CARS\nWINDOW CLOSED", RGB{R: 255, G: 150, B: 105}
		case BonusTargetSnakes:
			return "SNAKE BOMBER WINDOW\nCLOSED", RGB{R: 140, G: 255, B: 130}
		}
	}
	switch s.TargetAbilityKind {
	case BonusTargetWorms:
		return "WORM LOCK LOST", RGB{R: 120, G: 255, B: 145}
	case BonusTargetGunship:
		return "HELICOPTER SUPPORT\nLOCK LOST", RGB{R: 255, G: 95, B: 80}
	case BonusTargetHeliMissile:
		return "MISSILE HELI SUPPORT\nLOCK LOST", RGB{R: 255, G: 130, B: 55}
	case BonusTargetBombBelt:
		return "CARPET BOMB LOCK LOST", RGB{R: 255, G: 185, B: 90}
	case BonusTargetAirSupport:
		return "AIR SUPPORT LOCK LOST", RGB{R: 145, G: 200, B: 255}
	case BonusTargetPigs:
		return "EXPLODING PIGS\nLOCK LOST", RGB{R: 255, G: 130, B: 175}
	case BonusTargetCars:
		return "EXPLODING R/C CARS\nLOCK LOST", RGB{R: 255, G: 150, B: 105}
	case BonusTargetSnakes:
		return "SNAKE BOMBER LOCK LOST", RGB{R: 140, G: 255, B: 130}
	default:
		return "NUKE LOCK LOST", RGB{R: 255, G: 180, B: 90}
	}
}

func (s *Snake) targetClickPrompt(done bool) (string, RGB) {
	picked, _, maxClicks := s.targetProgress()
	if done {
		switch s.TargetAbilityKind {
		case BonusTargetWorms:
			return "WORM HUNTERS DEPLOYED!", RGB{R: 120, G: 255, B: 145}
		case BonusTargetGunship:
			return "HELICOPTER SUPPORT\nINBOUND!", RGB{R: 255, G: 95, B: 80}
		case BonusTargetHeliMissile:
			return "MISSILE HELI SUPPORT\nINBOUND!", RGB{R: 255, G: 130, B: 55}
		case BonusTargetBombBelt:
			return "CARPET BOMB RUN\nACTIVE!", RGB{R: 255, G: 185, B: 90}
		case BonusTargetAirSupport:
			return "AIR SUPPORT WAVE\nINBOUND!", RGB{R: 145, G: 200, B: 255}
		case BonusTargetPigs:
			return "EXPLODING PIGS\nRELEASED!", RGB{R: 255, G: 130, B: 175}
		case BonusTargetCars:
			return "EXPLODING R/C CARS\nRELEASED!", RGB{R: 255, G: 150, B: 105}
		case BonusTargetSnakes:
			return "SNAKE BOMBERS\nRELEASED!", RGB{R: 140, G: 255, B: 130}
		default:
			return "NUKE LAUNCHED!", RGB{R: 255, G: 230, B: 120}
		}
	}
	switch s.TargetAbilityKind {
	case BonusTargetBombBelt:
		return fmt.Sprintf("CARPET BOMB MARKED %d/%d", picked, maxClicks), RGB{R: 255, G: 185, B: 90}
	case BonusTargetAirSupport:
		return fmt.Sprintf("AIR SUPPORT MARKED %d/%d", picked, maxClicks), RGB{R: 145, G: 200, B: 255}
	case BonusTargetPigs:
		return fmt.Sprintf("EXPLODING PIGS PLACED %d/%d", picked, maxClicks), RGB{R: 255, G: 130, B: 175}
	case BonusTargetCars:
		return fmt.Sprintf("EXPLODING R/C CARS PLACED %d/%d", picked, maxClicks), RGB{R: 255, G: 150, B: 105}
	case BonusTargetSnakes:
		return fmt.Sprintf("SNAKE BOMBERS PLACED %d/%d", picked, maxClicks), RGB{R: 140, G: 255, B: 130}
	default:
		return "", RGB{}
	}
}

func (s *Snake) ActivateTargetAbilityAt(wx, wy int, world *World, peds *PedestrianSystem, traffic *TrafficSystem, particles *ParticleSystem, cam *Camera, cops *CopSystem, mil *MilitarySystem) bool {
	if s == nil || s.TargetNukeTimer <= 0 {
		return true
	}

	switch s.TargetAbilityKind {
	case BonusTargetWorms:
		s.activateTargetWormStrike(wx, wy, world, particles)
	case BonusTargetGunship:
		s.activateTargetGunshipStrike(wx, wy, particles)
	case BonusTargetHeliMissile:
		s.activateTargetHeliMissileStrike(wx, wy, particles)
	case BonusTargetBombBelt:
		s.activateTargetBombBelt(wx, wy, world, particles)
	case BonusTargetAirSupport:
		s.activateTargetAirSupport(wx, wy, particles)
	case BonusTargetPigs:
		s.activateTargetTimedExploder(wx, wy, world, particles, TimedExploderPig)
	case BonusTargetCars:
		s.activateTargetTimedExploder(wx, wy, world, particles, TimedExploderCar)
	case BonusTargetSnakes:
		s.activateTargetTimedExploder(wx, wy, world, particles, TimedExploderSnake)
	default:
		ExplodeAt(wx, wy, 30, world, particles, peds, traffic, cam, cops, mil)
		SpawnNukeAftermath(wx, wy, world, particles, 1.25)
	}

	s.TargetClicksMade++
	_, _, maxClicks := s.targetProgress()
	done := s.TargetClicksMade >= maxClicks
	msg, col := s.targetClickPrompt(done)
	if msg != "" {
		s.PowerupMsg = msg
		s.PowerupCol = col
	}
	if done {
		s.TargetNukeTimer = 0
		s.PowerupTimer = 2.2
	} else {
		s.PowerupTimer = 1.5
	}
	return done
}

func (s *Snake) activateTargetWormStrike(wx, wy int, world *World, particles *ParticleSystem) {
	r := NewRand(uint64(wx*97+wy*53) ^ 0xA11CE5EED)
	count := 5 + r.Range(0, 15) // 5-20 worms
	for i := 0; i < count; i++ {
		ang := r.RangeF(0, 2*math.Pi)
		dist := r.RangeF(0, 6.0)
		sx := clampF(float64(wx)+math.Cos(ang)*dist, 0, float64(WorldWidth-1))
		sy := clampF(float64(wy)+math.Sin(ang)*dist, 0, float64(WorldHeight-1))
		if nx, ny, ok := nearestWalkablePoint(world, sx, sy, 18); ok {
			sx, sy = nx, ny
		}
		s.StrikeWorms = append(s.StrikeWorms, StrikeWorm{
			X:       sx,
			Y:       sy,
			Heading: r.RangeF(0, 2*math.Pi),
			Life:    2.0 + r.RangeF(0, 1.0), // 2-3s
		})
	}
	if particles != nil {
		for i := 0; i < 36; i++ {
			ang := r.RangeF(0, 2*math.Pi)
			spd := r.RangeF(18, 52)
			particles.Add(Particle{
				X: float64(wx), Y: float64(wy),
				VX: math.Cos(ang) * spd, VY: math.Sin(ang) * spd,
				Size: 0.45, MaxLife: r.RangeF(0.25, 0.65),
				Col: RGB{R: 100, G: 255, B: 120}, Kind: ParticleGlow,
			})
		}
	}
}

func (s *Snake) activateTargetHeliStrike(wx, wy int, particles *ParticleSystem, missileMode bool) {
	r := NewRand(uint64(wx*131+wy*61) ^ 0xBADC0DE5EED)
	leaders := 3
	escorts := 2 + r.Range(0, 2) // 2-4 support helicopters
	total := leaders + escorts

	spawnHeli := func(i int, leader bool) {
		baseA := float64(i) * 2 * math.Pi / float64(max(1, total))
		orbitR := 8.0 + r.RangeF(0, 10.0)
		attackSpread := 10.0 + r.RangeF(0, 18.0)
		attackA := r.RangeF(0, 2*math.Pi)
		ax := clampF(float64(wx)+math.Cos(attackA)*attackSpread, 2, float64(WorldWidth-2))
		ay := clampF(float64(wy)+math.Sin(attackA)*attackSpread, 2, float64(WorldHeight-2))
		s.StrikeHelis = append(s.StrikeHelis, StrikeHeli{
			CenterX: clampF(ax+r.RangeF(-5.0, 5.0), 2, float64(WorldWidth-2)),
			CenterY: clampF(ay+r.RangeF(-5.0, 5.0), 2, float64(WorldHeight-2)),
			AttackX: ax, AttackY: ay,
			OrbitA:      baseA + r.RangeF(-0.4, 0.4),
			OrbitR:      orbitR,
			OrbitSpd:    0.8 + r.RangeF(0, 1.0),
			DeployTimer: 0.25 + r.RangeF(0, 0.35),
			Life:        3.5 + r.RangeF(0, 1.5), // 3.5-5.0s attack window
			FireTimer:   r.RangeF(0.03, 0.20),
			Leader:      leader,
			MissileMode: missileMode,
			X:           float64(wx) + math.Cos(baseA)*2.5,
			Y:           float64(wy) + math.Sin(baseA)*2.5,
			Heading:     baseA,
			RotorAngle:  r.RangeF(0, 2*math.Pi),
		})
	}

	for i := 0; i < leaders; i++ {
		spawnHeli(i, true)
	}
	for i := 0; i < escorts; i++ {
		spawnHeli(leaders+i, false)
	}

	PlaySound(SoundHelicopter)
	if particles != nil {
		for i := 0; i < 44; i++ {
			ang := r.RangeF(0, 2*math.Pi)
			spd := r.RangeF(24, 70)
			col := RGB{R: 255, G: 70, B: 60}
			if missileMode {
				col = RGB{R: 255, G: 135, B: 48}
			}
			particles.Add(Particle{
				X: float64(wx), Y: float64(wy),
				VX: math.Cos(ang) * spd, VY: math.Sin(ang) * spd,
				Size: 0.52, MaxLife: r.RangeF(0.20, 0.55),
				Col: col, Kind: ParticleGlow,
			})
		}
	}
}

func (s *Snake) activateTargetGunshipStrike(wx, wy int, particles *ParticleSystem) {
	s.activateTargetHeliStrike(wx, wy, particles, false)
}

func (s *Snake) activateTargetHeliMissileStrike(wx, wy int, particles *ParticleSystem) {
	s.activateTargetHeliStrike(wx, wy, particles, true)
}

func (s *Snake) activateTargetBombBelt(wx, wy int, world *World, particles *ParticleSystem) {
	r := NewRand(uint64(wx*187+wy*97) ^ 0xB0B0B37)
	x := clampF(float64(wx), 0, float64(WorldWidth-1))
	y := clampF(float64(wy), 0, float64(WorldHeight-1))
	if nx, ny, ok := nearestWalkablePoint(world, x, y, 20); ok {
		x, y = nx, ny
	}
	s.BeltBombs = append(s.BeltBombs, BeltBomb{
		X:      x,
		Y:      y,
		Fuse:   0.30 + r.RangeF(0, 0.85),
		Radius: 15,
	})
	if particles != nil {
		for i := 0; i < 18; i++ {
			ang := r.RangeF(0, 2*math.Pi)
			spd := r.RangeF(20, 52)
			particles.Add(Particle{
				X: x, Y: y,
				VX: math.Cos(ang) * spd, VY: math.Sin(ang) * spd,
				Size: 0.42, MaxLife: r.RangeF(0.14, 0.34),
				Col: RGB{R: 255, G: 185, B: 90}, Kind: ParticleGlow,
			})
		}
	}
}

func (s *Snake) activateTargetAirSupport(wx, wy int, particles *ParticleSystem) {
	r := NewRand(uint64(wx*223+wy*149) ^ 0xA1A5A1F0)
	tx := clampF(float64(wx), 0, float64(WorldWidth-1))
	ty := clampF(float64(wy), 0, float64(WorldHeight-1))
	a := r.RangeF(0, 2*math.Pi)
	dx := math.Cos(a)
	dy := math.Sin(a)
	span := math.Hypot(float64(WorldWidth), float64(WorldHeight)) + 60.0
	startX := tx - dx*span*0.5
	startY := ty - dy*span*0.5
	speed := 115.0 + r.RangeF(0, 40.0)
	pathLen := span
	s.StrikePlanes = append(s.StrikePlanes, StrikePlane{
		X: startX, Y: startY,
		VX: dx * speed, VY: dy * speed,
		Heading:    math.Atan2(dy, dx),
		Life:       pathLen/speed + 0.9,
		TargetX:    tx,
		TargetY:    ty,
		BombsLeft:  2 + r.Range(0, 2),
		BombRadius: 10,
	})
	if particles != nil {
		for i := 0; i < 24; i++ {
			ang := r.RangeF(0, 2*math.Pi)
			spd := r.RangeF(18, 62)
			particles.Add(Particle{
				X: tx, Y: ty,
				VX: math.Cos(ang) * spd, VY: math.Sin(ang) * spd,
				Size: 0.46, MaxLife: r.RangeF(0.16, 0.40),
				Col: RGB{R: 145, G: 200, B: 255}, Kind: ParticleGlow,
			})
		}
	}
}

func (s *Snake) activateTargetTimedExploder(wx, wy int, world *World, particles *ParticleSystem, kind TimedExploderKind) {
	seed := uint64(wx*257+wy*83) ^
		uint64(kind+1)*0x600D5EED ^
		uint64(s.TargetClicksMade+1)*0x9E3779B185EBCA87 ^
		uint64(len(s.TimedExploders)+1)*0xC2B2AE3D27D4EB4F ^
		s.TargetRoll*0x94D049BB133111EB
	r := NewRand(seed)
	x := clampF(float64(wx), 0, float64(WorldWidth-1))
	y := clampF(float64(wy), 0, float64(WorldHeight-1))
	if nx, ny, ok := nearestWalkablePoint(world, x, y, 20); ok {
		x, y = nx, ny
	}

	speed := 22.0 + r.RangeF(0, 14.0)
	radius := 6
	col := RGB{R: 255, G: 130, B: 175}
	switch kind {
	case TimedExploderCar:
		speed = 30.0 + r.RangeF(0, 14.0)
		radius = 7
		col = RGB{R: 255, G: 150, B: 105}
	case TimedExploderSnake:
		speed = 20.0 + r.RangeF(0, 16.0)
		radius = 6
		col = RGB{R: 140, G: 255, B: 130}
	}
	s.TimedExploders = append(s.TimedExploders, TimedExploder{
		Kind:    kind,
		X:       x,
		Y:       y,
		Heading: r.RangeF(0, 2*math.Pi),
		Speed:   speed,
		Timer:   3.0 + r.RangeF(0, 4.0),
		Radius:  radius,
	})
	if particles != nil {
		for i := 0; i < 18; i++ {
			ang := r.RangeF(0, 2*math.Pi)
			spd := r.RangeF(12, 44)
			particles.Add(Particle{
				X: x, Y: y,
				VX: math.Cos(ang) * spd, VY: math.Sin(ang) * spd,
				Size: 0.38, MaxLife: r.RangeF(0.16, 0.44),
				Col: col, Kind: ParticleGlow,
			})
		}
	}
}

// ExplodeAt wraps the global ExplodeAt and credits ped kills as WantedLevel.
func (s *Snake) ExplodeAt(wx, wy, radius int, w *World, ps *ParticleSystem, peds *PedestrianSystem, traffic *TrafficSystem, cam *Camera, cops *CopSystem, mil *MilitarySystem) {
	kills := ExplodeAt(wx, wy, radius, w, ps, peds, traffic, cam, cops, mil)
	s.WantedLevel = min(WantedMax, s.WantedLevel+float64(kills)*0.1)
}

// Head returns the current head position.
func (s *Snake) Head() (float64, float64) {
	if len(s.Path) == 0 {
		return 0, 0
	}
	p := s.Path[0]
	return p.X, p.Y
}

func pointSegDist(px, py, ax, ay, bx, by float64) float64 {
	abx := bx - ax
	aby := by - ay
	den := abx*abx + aby*aby
	if den <= 1e-9 {
		return math.Hypot(px-ax, py-ay)
	}
	t := ((px-ax)*abx + (py-ay)*aby) / den
	t = clampF(t, 0, 1)
	cx := ax + abx*t
	cy := ay + aby*t
	return math.Hypot(px-cx, py-cy)
}

// bonusPickupSkewHit gives bonus collection a slight directional skew and
// movement-sweep forgiveness, so circling near a bonus still collects it.
func bonusPickupSkewHit(x, y, heading, bx, by float64, prevX, prevY float64, hasPrev bool) bool {
	baseR := SnakeBonusRadius + 0.45
	dx := bx - x
	dy := by - y
	if math.Hypot(dx, dy) <= baseR {
		return true
	}

	// Wider side pickup while orbiting around a bonus.
	side := -math.Sin(heading)*dx + math.Cos(heading)*dy
	forward := math.Cos(heading)*dx + math.Sin(heading)*dy
	if math.Abs(side) <= baseR+0.9 && math.Abs(forward) <= baseR+0.45 {
		return true
	}

	// Sweep test between previous and current head positions to avoid pass-through misses.
	if hasPrev && pointSegDist(bx, by, prevX, prevY, x, y) <= baseR+0.35 {
		return true
	}
	return false
}

// Steer smoothly turns heading toward targetAngle at SnakeTurnRate rad/s.
func (s *Snake) Steer(targetAngle, dt float64) {
	s.TargetHeading = targetAngle // remember player's desired direction

	diff := angDiff(s.Heading, targetAngle)
	maxTurn := SnakeTurnRate * dt
	if math.Abs(diff) <= maxTurn {
		s.Heading = targetAngle
	} else if diff > 0 {
		s.Heading += maxTurn
	} else {
		s.Heading -= maxTurn
	}
	// Normalize to [-π, π].
	for s.Heading > math.Pi {
		s.Heading -= 2 * math.Pi
	}
	for s.Heading < -math.Pi {
		s.Heading += 2 * math.Pi
	}
}

// Update advances the snake by dt seconds.
func (s *Snake) Update(dt float64, world *World, peds *PedestrianSystem, traffic *TrafficSystem, bonuses *BonusSystem, particles *ParticleSystem, cam *Camera, cops *CopSystem, mil *MilitarySystem) {
	if !s.Alive {
		return
	}

	// Tick bounce override timer.
	if s.BounceTimer > 0 {
		s.BounceTimer -= dt
	}

	// Tick powerup label fade.
	if s.PowerupTimer > 0 {
		s.PowerupTimer -= dt
	}

	// Kill streak window.
	if s.KillStreakTimer > 0 {
		s.KillStreakTimer -= dt
		if s.KillStreakTimer <= 0 {
			s.KillStreak = 0
		}
	}
	if s.KillMsgTimer > 0 {
		s.KillMsgTimer -= dt
	}

	// Gore trail: drip blood occasionally after eating.
	if s.GoreTimer > 0 {
		s.GoreTimer -= dt
		s.GoreDripAcc += dt
		// Drip every 0.15-0.25 seconds.
		if s.GoreDripAcc >= 0.18 {
			s.GoreDripAcc = 0
			hx, hy := s.Head()
			if particles != nil {
				// Small blood drip behind snake.
				r := NewRand(uint64(hx*31+hy*17) ^ 0xB100D)
				ang := s.Heading + math.Pi + r.RangeF(-0.5, 0.5) // behind snake
				particles.Add(Particle{
					X: hx - math.Cos(s.Heading)*2, Y: hy - math.Sin(s.Heading)*2,
					VX: math.Cos(ang) * r.RangeF(2, 8), VY: math.Sin(ang) * r.RangeF(2, 8),
					Size: r.RangeF(0.3, 0.6), MaxLife: r.RangeF(0.5, 1.2),
					Col: RGB{R: uint8(110 + r.Range(0, 30)), G: 15, B: 15}, Kind: ParticleBlood,
				})
				// Leave small stain on ground.
				if r.Float64() < 0.4 {
					bx := int(math.Round(hx - math.Cos(s.Heading)*3))
					by := int(math.Round(hy - math.Sin(s.Heading)*3))
					if bx >= 0 && by >= 0 && bx < WorldWidth && by < WorldHeight {
						world.AddTempPaint(bx, by, RGB{R: 100, G: 15, B: 15}, 4.0)
					}
				}
			}
		}
	}

	// Update evolution level.
	newLevel := 0
	switch {
	case s.EvoPoints >= 500:
		newLevel = 5
	case s.EvoPoints >= 300:
		newLevel = 4
	case s.EvoPoints >= 150:
		newLevel = 3
	case s.EvoPoints >= 50:
		newLevel = 2
	case s.EvoPoints >= 20:
		newLevel = 1
	}
	if newLevel > s.EvoLevel {
		s.EvoLevel = newLevel
		// Flash effect on level up.
		if particles != nil {
			hx, hy := s.Head()
			r := NewRand(uint64(s.EvoLevel) ^ 0xEEE)
			col := s.evoHeadColor()
			for i := 0; i < 20; i++ {
				ang := r.RangeF(0, 2*math.Pi)
				spd := r.RangeF(20, 60)
				particles.Add(Particle{
					X: hx, Y: hy,
					VX: math.Cos(ang) * spd, VY: math.Sin(ang) * spd,
					Size: 0.5, MaxLife: r.RangeF(0.4, 0.8),
					Col: col, Kind: ParticleGlow,
				})
			}
		}
	}

	// Flamethrower timer decay.
	if s.FlamethrowerTimer > 0 {
		s.FlamethrowerTimer -= dt
	}

	// Flamethrower: spray fire particles, damage peds/cops/military/cars, ignite buildings.
	if s.FlamethrowerTimer > 0 {
		hx, hy := s.Head()
		r := NewRand(uint64(hx*17+hy*31) ^ 0xF12E)
		// Spawn lots of fire particles in a wide cone.
		for i := 0; i < 8; i++ {
			ang := s.Heading + r.RangeF(-0.6, 0.6)
			spd := r.RangeF(60, 120)
			particles.Add(Particle{
				X: hx + math.Cos(s.Heading)*3, Y: hy + math.Sin(s.Heading)*3,
				VX: math.Cos(ang) * spd, VY: math.Sin(ang) * spd,
				Z: r.RangeF(0, 3), VZ: r.RangeF(5, 20),
				Size: r.RangeF(0.6, 1.2), MaxLife: r.RangeF(0.4, 0.8),
				Col:  RGB{R: 255, G: uint8(80 + r.Range(0, 150)), B: uint8(r.Range(0, 40))},
				Kind: ParticleFire, Burning: true,
			})
		}
		// Large cone: range 30, angle ±45°.
		fireRange := 30.0
		fireAngle := 0.8 // ~45 degrees
		// Burn pedestrians.
		for i := range peds.P {
			p := &peds.P[i]
			if !p.Alive {
				continue
			}
			dx := p.X - hx
			dy := p.Y - hy
			dist := math.Hypot(dx, dy)
			if dist > fireRange || dist < 0.5 {
				continue
			}
			ang := math.Atan2(dy, dx)
			if math.Abs(angDiff(ang, s.Heading)) < fireAngle {
				p.Alive = false
				s.Score += 50
				s.EvoPoints += 5
				s.GoreTimer = 1.0
				particles.SpawnBlood(p.X, p.Y, dx/dist, dy/dist, 8, 0.7)
			}
		}
		// Burn cop peds.
		if cops != nil {
			for i := range cops.Peds {
				cp := &cops.Peds[i]
				if !cp.Alive {
					continue
				}
				dx := cp.X - hx
				dy := cp.Y - hy
				dist := math.Hypot(dx, dy)
				if dist > fireRange || dist < 0.5 {
					continue
				}
				ang := math.Atan2(dy, dx)
				if math.Abs(angDiff(ang, s.Heading)) < fireAngle {
					cp.Alive = false
					s.Score += 100
					s.EvoPoints += 10
					s.WantedLevel = min(WantedMax, s.WantedLevel+0.2)
					particles.SpawnBlood(cp.X, cp.Y, dx/dist, dy/dist, 12, 0.85)
				}
			}
		}
		// Burn military troops.
		if mil != nil {
			for i := range mil.Troops {
				t := &mil.Troops[i]
				if !t.Alive {
					continue
				}
				dx := t.X - hx
				dy := t.Y - hy
				dist := math.Hypot(dx, dy)
				if dist > fireRange || dist < 0.5 {
					continue
				}
				ang := math.Atan2(dy, dx)
				if math.Abs(angDiff(ang, s.Heading)) < fireAngle {
					t.Alive = false
					s.Score += 150
					s.EvoPoints += 15
					s.WantedLevel = min(WantedMax, s.WantedLevel+0.3)
					particles.SpawnBlood(t.X, t.Y, dx/dist, dy/dist, 15, 0.9)
				}
			}
		}
		// Burn cars.
		if traffic != nil {
			for i := range traffic.Cars {
				c := &traffic.Cars[i]
				if !c.Alive {
					continue
				}
				dx := c.X - hx
				dy := c.Y - hy
				dist := math.Hypot(dx, dy)
				if dist > fireRange || dist < 0.5 {
					continue
				}
				ang := math.Atan2(dy, dx)
				if math.Abs(angDiff(ang, s.Heading)) < fireAngle {
					// Flamethrower should fully destroy vehicles in-cone.
					c.HP.Damage(c.HP.Current + c.HP.Max + 1)
					c.Alive = false
					s.Score += 200
					s.EvoPoints += 20
					s.ExplodeAt(int(c.X), int(c.Y), 5, world, particles, peds, traffic, cam, cops, mil)
				}
			}
		}
		// Burn cop cars as well.
		if cops != nil {
			for i := range cops.Cars {
				cc := &cops.Cars[i]
				if !cc.Alive {
					continue
				}
				dx := cc.X - hx
				dy := cc.Y - hy
				dist := math.Hypot(dx, dy)
				if dist > fireRange || dist < 0.5 {
					continue
				}
				ang := math.Atan2(dy, dx)
				if math.Abs(angDiff(ang, s.Heading)) < fireAngle {
					cc.HP.Damage(cc.HP.Current + cc.HP.Max + 1)
					cc.Alive = false
					s.Score += 250
					s.EvoPoints += 20
					s.WantedLevel = min(WantedMax, s.WantedLevel+0.3)
					s.ExplodeAt(int(cc.X), int(cc.Y), 5, world, particles, peds, traffic, cam, cops, mil)
				}
			}
			for i := range cops.Helis {
				ch := &cops.Helis[i]
				if !ch.Alive {
					continue
				}
				dx := ch.X - hx
				dy := ch.Y - hy
				dist := math.Hypot(dx, dy)
				if dist > fireRange || dist < 0.5 {
					continue
				}
				ang := math.Atan2(dy, dx)
				if math.Abs(angDiff(ang, s.Heading)) < fireAngle {
					ch.Alive = false
					s.Score += 320
					s.EvoPoints += 26
					s.WantedLevel = min(WantedMax, s.WantedLevel+0.35)
					s.ExplodeAt(int(ch.X), int(ch.Y), 6, world, particles, peds, traffic, cam, cops, mil)
				}
			}
		}
		// Ignite buildings in cone.
		for dist := 3.0; dist <= fireRange; dist += 2.0 {
			px := hx + math.Cos(s.Heading)*dist
			py := hy + math.Sin(s.Heading)*dist
			wx, wy := int(math.Round(px)), int(math.Round(py))
			if wx >= 0 && wy >= 0 && wx < WorldWidth && wy < WorldHeight {
				if world.HeightAt(wx, wy) > 0 {
					world.StartBuildingBurn(wx, wy)
				}
			}
		}
	}

	// Swarm mode: body segments hunt peds (GhostTimer>0), then return to reassemble (GhostTimer<=0).
	// Active as long as any segments remain (len > 0); main snake is frozen during this time.
	if len(s.Ghosts) > 0 {
		if s.GhostBombMode {
			s.updateGhostBombRun(dt, world, peds, traffic, particles, cam, cops, mil)
			if s.Length < 3 || s.HP.IsDead() {
				s.Alive = false
			}
			return
		}

		if s.GhostTimer > 0 {
			s.GhostTimer -= dt
			if s.GhostTimer <= 0 {
				// Hunting phase over — all segments begin returning.
				for i := range s.Ghosts {
					s.Ghosts[i].Returning = true
				}
			}
		}

		// Keep global powerup timers active while swarm mode is running.
		if s.BashTimer > 0 {
			s.BashTimer -= dt
			if s.BashTimer < 0 {
				s.BashTimer = 0
			}
		}
		surgeActive := false
		if s.SurgeTimer > 0 {
			s.SurgeTimer -= dt
			if s.SurgeTimer < 0 {
				s.SurgeTimer = 0
			} else {
				surgeActive = true
			}
		}

		hx, hy := s.Head()
		swarmMult := 1.8
		if s.SpeedBoost > 0 {
			s.SpeedBoost -= dt
			swarmMult *= s.SpeedMult
			if s.SpeedBoost <= 0 {
				s.SpeedBoost = 0
				s.SpeedMult = 1.0
			}
		}
		if s.BerserkTimer > 0 {
			s.BerserkTimer -= dt
			swarmMult *= 1.8
			if s.BerserkTimer <= 0 {
				s.BerserkTimer = 0
				s.SizeMult = 1.0
			}
		}
		evoSpeed := s.Speed * s.evoSpeedMult()
		ghostSpeed := evoSpeed * swarmMult
		returnSpeed := evoSpeed * (swarmMult + 1.2)

		for i := len(s.Ghosts) - 1; i >= 0; i-- {
			if i >= len(s.Ghosts) {
				break // slice was modified mid-loop (e.g. by bonus collection)
			}
			g := &s.Ghosts[i]
			r := NewRand(uint64(i+1)*0xC0FFEE ^ uint64(math.Abs(s.GhostTimer)*1000+float64(i)*997))

			if g.Returning {
				// Fly back to snake head; when close enough, merge back.
				dx := hx - g.X
				dy := hy - g.Y
				dist := math.Hypot(dx, dy)
				if dist < 2.5 {
					// Reintegrate: restore this segment's body length.
					s.Length += g.SegLen
					if particles != nil {
						// Small flash on arrival.
						for range 6 {
							ang := r.RangeF(0, 2*math.Pi)
							particles.Add(Particle{
								X: hx, Y: hy,
								VX: math.Cos(ang) * r.RangeF(10, 30), VY: math.Sin(ang) * r.RangeF(10, 30),
								Size: 0.3, MaxLife: r.RangeF(0.1, 0.25),
								Col: RGB{R: 150, G: 255, B: 150}, Kind: ParticleGlow,
							})
						}
					}
					// Swap-remove.
					s.Ghosts[i] = s.Ghosts[len(s.Ghosts)-1]
					s.Ghosts = s.Ghosts[:len(s.Ghosts)-1]
					continue
				}
				// Phase through walls during return (move direct, clamp to bounds).
				g.Heading = math.Atan2(dy, dx)
				g.X = clampF(g.X+math.Cos(g.Heading)*returnSpeed*dt, 0, float64(WorldWidth-1))
				g.Y = clampF(g.Y+math.Sin(g.Heading)*returnSpeed*dt, 0, float64(WorldHeight-1))
				continue
			}

			// Hunting: find nearest alive ped or bonus box.
			bestDist := math.MaxFloat64
			var nearX, nearY float64
			found := false
			for j := range peds.P {
				p := &peds.P[j]
				if !p.Alive {
					continue
				}
				d := math.Hypot(p.X-g.X, p.Y-g.Y)
				if d < bestDist {
					bestDist = d
					nearX, nearY = p.X, p.Y
					found = true
				}
			}
			// Also target bonus boxes — prefer them when closer than any ped.
			if bonuses != nil {
				for j := range bonuses.Boxes {
					b := &bonuses.Boxes[j]
					if !b.Alive {
						continue
					}
					d := math.Hypot(b.X-g.X, b.Y-g.Y)
					if d < bestDist {
						bestDist = d
						nearX, nearY = b.X, b.Y
						found = true
					}
				}
			}

			tdist := math.Hypot(g.TargetX-g.X, g.TargetY-g.Y)
			if found && bestDist < 20.0 {
				g.TargetX, g.TargetY = nearX, nearY
			} else if tdist < 1.5 {
				if found {
					g.TargetX, g.TargetY = nearX, nearY
				} else {
					for tries := 0; tries < 40; tries++ {
						tx := r.Range(0, WorldWidth-1)
						ty := r.Range(0, WorldHeight-1)
						if world.HeightAt(tx, ty) == 0 {
							g.TargetX = float64(tx) + 0.5
							g.TargetY = float64(ty) + 0.5
							break
						}
					}
				}
			}

			dx := g.TargetX - g.X
			dy := g.TargetY - g.Y
			dist := math.Hypot(dx, dy)
			if dist > 0.1 {
				// Steer heading toward target gradually — heading persists between frames
				// so wall bounces hold rather than being overwritten each tick.
				targetH := math.Atan2(dy, dx)
				hdiff := angDiff(g.Heading, targetH)
				maxTurn := SnakeTurnRate * 1.5 * dt
				if math.Abs(hdiff) <= maxTurn {
					g.Heading = targetH
				} else if hdiff > 0 {
					g.Heading += maxTurn
				} else {
					g.Heading -= maxTurn
				}
			}

			// Move in current heading direction.
			newX := g.X + math.Cos(g.Heading)*ghostSpeed*dt
			newY := g.Y + math.Sin(g.Heading)*ghostSpeed*dt
			nxi := int(math.Round(newX))
			nyi := int(math.Round(newY))

			if nxi >= 0 && nyi >= 0 && nxi < WorldWidth && nyi < WorldHeight && world.HeightAt(nxi, nyi) == 0 {
				g.X = newX
				g.Y = newY
			} else {
				if s.BashTimer > 0 {
					bx := clamp(nxi, 0, WorldWidth-1)
					by := clamp(nyi, 0, WorldHeight-1)
					s.ExplodeAt(bx, by, 3, world, particles, peds, traffic, cam, cops, mil)
					g.X = clampF(newX, 0, float64(WorldWidth-1))
					g.Y = clampF(newY, 0, float64(WorldHeight-1))
				} else {
					// Bounce: same axis-reflection as the main snake.
					xBlocked := nxi < 0 || nxi >= WorldWidth || world.HeightAt(clamp(nxi, 0, WorldWidth-1), int(math.Round(g.Y))) > 0
					yBlocked := nyi < 0 || nyi >= WorldHeight || world.HeightAt(int(math.Round(g.X)), clamp(nyi, 0, WorldHeight-1)) > 0
					if xBlocked && !yBlocked {
						g.Heading = math.Pi - g.Heading // vertical wall: flip X
						g.X = clampF(g.X+math.Cos(g.Heading)*ghostSpeed*dt, 0, float64(WorldWidth-1))
					} else if yBlocked && !xBlocked {
						g.Heading = -g.Heading // horizontal wall: flip Y
						g.Y = clampF(g.Y+math.Sin(g.Heading)*ghostSpeed*dt, 0, float64(WorldHeight-1))
					} else {
						g.Heading += math.Pi // corner: reverse
						bx := g.X + math.Cos(g.Heading)*ghostSpeed*dt
						by := g.Y + math.Sin(g.Heading)*ghostSpeed*dt
						bxi := int(math.Round(bx))
						byi := int(math.Round(by))
						if bxi >= 0 && byi >= 0 && bxi < WorldWidth && byi < WorldHeight && world.HeightAt(bxi, byi) == 0 {
							g.X = bx
							g.Y = by
						}
					}
					for g.Heading > math.Pi {
						g.Heading -= 2 * math.Pi
					}
					for g.Heading < -math.Pi {
						g.Heading += 2 * math.Pi
					}
				}
			}

			// Stuck detection: much faster timeouts for fluid hunting.
			moved := math.Hypot(g.X-g.PrevX, g.Y-g.PrevY)
			g.PrevX, g.PrevY = g.X, g.Y
			if moved < 0.05*dt {
				g.StuckTimer += dt
				if g.StuckTimer > 1.2 {
					// Teleport to a random walkable spot.
					for tries := 0; tries < 50; tries++ {
						tx := r.Range(0, WorldWidth-1)
						ty := r.Range(0, WorldHeight-1)
						if world.HeightAt(tx, ty) == 0 {
							g.X = float64(tx) + 0.5
							g.Y = float64(ty) + 0.5
							g.StuckTimer = 0
							break
						}
					}
				} else if g.StuckTimer > 0.4 {
					// Pick a new waypoint farther away.
					for tries := 0; tries < 20; tries++ {
						tx := int(math.Round(g.X)) + r.Range(-20, 20)
						ty := int(math.Round(g.Y)) + r.Range(-20, 20)
						if tx >= 0 && ty >= 0 && tx < WorldWidth && ty < WorldHeight && world.HeightAt(tx, ty) == 0 {
							g.TargetX = float64(tx) + 0.5
							g.TargetY = float64(ty) + 0.5
							break
						}
					}
				}
			} else {
				g.StuckTimer = 0
			}

			// Surge applies to every active swarm segment, not only the hidden main head.
			if surgeActive {
				s.applySurgeAt(g.X, g.Y, dt, world, peds, traffic, particles, cam, cops, mil)
			}

			// Eat nearby peds.
			for j := range peds.P {
				p := &peds.P[j]
				if !p.Alive {
					continue
				}
				if math.Hypot(p.X-g.X, p.Y-g.Y) < SnakeEatRadius {
					p.Alive = false
					s.Score += 100
					s.WantedLevel = min(WantedMax, s.WantedLevel+0.1)
					if particles != nil {
						particles.SpawnBlood(g.X, g.Y, math.Cos(g.Heading), math.Sin(g.Heading), 16, 0.8)
						particles.SpawnBlood(g.X+0.4, g.Y+0.4, math.Cos(g.Heading+math.Pi/2), math.Sin(g.Heading+math.Pi/2), 8, 0.5)
					}
					break
				}
			}

			// Collect bonus boxes: area effects trigger at ghost's position.
			if bonuses != nil {
				for j := range bonuses.Boxes {
					b := &bonuses.Boxes[j]
					if !b.Alive {
						continue
					}
					if bonusPickupSkewHit(g.X, g.Y, g.Heading, b.X, b.Y, g.PrevX, g.PrevY, true) {
						kind := b.Kind
						bonuses.CollectAt(j, s, g.X, g.Y, world, peds, traffic, particles, cam, cops, mil)
						if kind == BonusFire {
							g.FireBoltTimer = s.FireRingTimer
							g.FireBoltAngle = 0
							s.setFireBoltsAll()
						}
						// Propagate area effects to all other ghosts and clones.
						propR := NewRand(uint64(g.X*53+g.Y*37) ^ 0x9A1D)
						for k := range s.Ghosts {
							if k == i {
								continue
							}
							og := &s.Ghosts[k]
							applyPositionalBonus(kind, s, og.X, og.Y, propR, world, peds, traffic, particles, cam, cops, mil)
						}
						for k := range s.Clones {
							applyPositionalBonus(kind, s, s.Clones[k].X, s.Clones[k].Y, propR, world, peds, traffic, particles, cam, cops, mil)
						}
						break
					}
				}
			}
			// Orbiting fire bolts for this ghost (independent from main snake).
			runFireBolts(g.X, g.Y, &g.FireBoltTimer, &g.FireBoltAngle, dt, s, world, peds, particles)
		}

		// Vacuum bubbles must continue updating during swarm mode.
		s.updateVacuumBubbles(dt, world, peds, traffic, particles, cam, cops, mil)
		s.updateStrikeWorms(dt, world, peds, particles)
		s.updateStrikeHelis(dt, world, peds, particles, cam, cops, mil)
		s.updateBeltBombs(dt, world, peds, traffic, particles, cam, cops, mil)
		s.updateStrikePlanes(dt, world, peds, traffic, particles, cam, cops, mil)
		s.updateTimedExploders(dt, world, peds, traffic, particles, cam, cops, mil)

		// Death check still applies during swarm.
		if s.Length < 3 || s.HP.IsDead() {
			s.Alive = false
		}
		return
	}

	// AI mode: steer toward nearest alive ped at higher speed.
	if s.AITimer > 0 {
		s.AITimer -= dt
		hx0, hy0 := s.Head()
		bestDist := math.MaxFloat64
		var nearX, nearY float64
		for i := range peds.P {
			p := &peds.P[i]
			if !p.Alive {
				continue
			}
			d := math.Hypot(p.X-hx0, p.Y-hy0)
			if d < bestDist {
				bestDist = d
				nearX, nearY = p.X, p.Y
			}
		}
		if bestDist < math.MaxFloat64 {
			s.Steer(math.Atan2(nearY-hy0, nearX-hx0), dt)
		}
		if s.AITimer <= 0 || s.AITargetsLeft <= 0 {
			s.AITimer = 0
			s.AITargetsLeft = 0
		}
	}

	// Speed boost decay.
	effectiveSpeed := s.Speed * s.evoSpeedMult()
	if s.AITimer > 0 {
		effectiveSpeed *= 1.5
	}
	if s.SpeedBoost > 0 {
		s.SpeedBoost -= dt
		effectiveSpeed *= s.SpeedMult
		if s.SpeedBoost <= 0 {
			s.SpeedBoost = 0
			s.SpeedMult = 1.0
		}
	}

	// Berserk mode decay: reset size when timer expires.
	if s.BerserkTimer > 0 {
		s.BerserkTimer -= dt
		effectiveSpeed *= 1.8 // faster during berserk
		if s.BerserkTimer <= 0 {
			s.BerserkTimer = 0
			s.SizeMult = 1.0
		}
	}

	// Move head forward, or trace a figure-8 when idle.
	hx, hy := s.Head()
	if s.Idle {
		s.RattlePhase += dt * math.Pi * 1.6 // ~0.8 Hz — one full figure-8 every ~2s
		// Lemniscate: head traces a small infinity symbol centred on IdleBaseX/Y.
		// Axis aligned to snake heading: perp direction gives the wide loops,
		// forward direction gives the near/far lobes of the figure-8.
		const R = 2.5
		perpX := -math.Sin(s.Heading)
		perpY := math.Cos(s.Heading)
		fwdX := math.Cos(s.Heading)
		fwdY := math.Sin(s.Heading)
		hx = s.IdleBaseX + fwdX*R*0.5*math.Sin(2*s.RattlePhase) + perpX*R*math.Sin(s.RattlePhase)
		hy = s.IdleBaseY + fwdY*R*0.5*math.Sin(2*s.RattlePhase) + perpY*R*math.Sin(s.RattlePhase)
	} else {
		s.RattlePhase = 0
		hx += math.Cos(s.Heading) * effectiveSpeed * dt
		hy += math.Sin(s.Heading) * effectiveSpeed * dt
		hx = clampF(hx, 0, float64(WorldWidth-1))
		hy = clampF(hy, 0, float64(WorldHeight-1))
		s.IdleBaseX, s.IdleBaseY = hx, hy // anchor for next idle phase
	}

	// Bash powerup decay.
	if s.BashTimer > 0 {
		s.BashTimer -= dt
	}

	// Building collision and path update are skipped while idle (snake is still).
	if !s.Idle {

		wx := int(math.Round(hx))
		wy := int(math.Round(hy))
		if world.HeightAt(wx, wy) > 0 {
			if s.BashTimer > 0 {
				// Bash powerup: explode through without damage.
				s.ExplodeAt(wx, wy, 3, world, particles, peds, traffic, cam, cops, mil)
				s.WantedLevel = min(WantedMax, s.WantedLevel+0.06)
				if cam != nil {
					cam.AddShake(0.3, 0.15)
				}
			} else {
				// Wall collision: find a clear escape direction and commit to it.
				ohx, ohy := s.Head()
				hx, hy = ohx, ohy
				step := effectiveSpeed * dt

				if ea, ok := s.findClearDir(ohx, ohy, s.Heading, step, world); ok {
					hx = ohx + math.Cos(ea)*step
					hy = ohy + math.Sin(ea)*step
					s.Heading = ea
					s.BounceDir = ea
					s.BounceTimer = 0.1 // hold escape direction for 0.1s
				} else if ux, uy, ok := s.forceUnstuck(ohx, ohy, world); ok {
					// Hard fallback: relocate to nearest walkable tile if boxed in.
					hx, hy = ux, uy
				}
			}
		}

		// Teleport: fire next queued jump (eat-ped loop below handles the actual eating).
		if len(s.TeleportQueue) > 0 {
			s.TeleportTimer -= dt
			if s.TeleportTimer <= 0 {
				target := s.TeleportQueue[0]
				s.TeleportQueue = s.TeleportQueue[1:]
				rr := NewRand(uint64(target.X*53+target.Y*79) ^ 0x7E1E)
				// Flash at departure.
				if particles != nil {
					for range 12 {
						ang := rr.RangeF(0, 2*math.Pi)
						spd := rr.RangeF(15, 50)
						particles.Add(Particle{
							X: hx, Y: hy,
							VX: math.Cos(ang) * spd, VY: math.Sin(ang) * spd,
							Size: 0.5, MaxLife: rr.RangeF(0.1, 0.3),
							Col: RGB{R: 200, G: 200, B: 255}, Kind: ParticleGlow,
						})
					}
				}
				// Jump to target; rebuild path so the body appears at the new position immediately.
				hx, hy = target.X, target.Y
				needed := int(math.Ceil(s.Length/1.5)) + 2
				if needed > len(s.Path) {
					needed = len(s.Path)
				}
				if needed < 1 {
					needed = 1
				}
				for pi := 0; pi < needed; pi++ {
					back := float64(pi) * 1.5
					s.Path[pi] = PathPoint{
						X: hx - math.Cos(s.Heading)*back,
						Y: hy - math.Sin(s.Heading)*back,
					}
				}
				s.Path = s.Path[:needed]
				// Flash at arrival.
				if particles != nil {
					for range 12 {
						ang := rr.RangeF(0, 2*math.Pi)
						spd := rr.RangeF(15, 50)
						particles.Add(Particle{
							X: hx, Y: hy,
							VX: math.Cos(ang) * spd, VY: math.Sin(ang) * spd,
							Size: 0.5, MaxLife: rr.RangeF(0.1, 0.3),
							Col: RGB{R: 200, G: 200, B: 255}, Kind: ParticleGlow,
						})
					}
				}
				s.TeleportTimer = 0.25 // 0.25s between jumps → up to 4 per second
			}
		}

	} // end !s.Idle block

	// Stuck detection: if snake hasn't moved, force a clear-road escape.
	if !s.Idle {
		moved := math.Hypot(hx-s.PrevX, hy-s.PrevY)
		if moved < 0.05 {
			s.StuckTimer += dt
			if s.StuckTimer > 0.12 {
				s.StuckAttempts++
				step := max(effectiveSpeed*dt*2.2, 2.0)
				escaped := false

				if ea, ok := s.findClearDir(hx, hy, s.Heading, step, world); ok {
					nx := hx + math.Cos(ea)*step
					ny := hy + math.Sin(ea)*step
					if ix, iy := int(math.Round(nx)), int(math.Round(ny)); ix >= 0 && iy >= 0 && ix < WorldWidth && iy < WorldHeight && world.HeightAt(ix, iy) == 0 {
						hx = nx
						hy = ny
						escaped = true
					}
					s.Heading = ea
					s.BounceDir = ea
					s.BounceTimer = 0.16
				}

				if !escaped {
					// Escalate search radius if we're repeatedly failing to move.
					maxRadius := min(120, 24+s.StuckAttempts*18)
					if ux, uy, ok := s.forceUnstuckWithRadius(hx, hy, world, maxRadius); ok {
						hx, hy = ux, uy
						escaped = true
					}
				}

				if escaped {
					s.PrevX, s.PrevY = hx, hy
					s.StuckAttempts = 0
				}
				s.StuckTimer = 0
			}
		} else {
			s.StuckTimer = 0
			s.StuckAttempts = 0
		}
		s.PrevX, s.PrevY = hx, hy
	}

	// Only prepend if the head actually moved, preventing path collapse into one spot.
	if len(s.Path) == 0 || math.Hypot(hx-s.Path[0].X, hy-s.Path[0].Y) > 0.01 {
		s.Path = append(s.Path, PathPoint{})
		copy(s.Path[1:], s.Path[0:])
		s.Path[0] = PathPoint{X: hx, Y: hy}
	}

	// Trim path to a maximum cap (avoid unbounded growth).
	maxPts := int(math.Ceil(SnakeMaxLength/1.5)) + 8
	if len(s.Path) > maxPts {
		s.Path = s.Path[:maxPts]
	}

	// Surge: pull peds/cops inward and shred anything that reaches the core.
	if s.SurgeTimer > 0 {
		s.SurgeTimer -= dt
		s.applySurgeAt(hx, hy, dt, world, peds, traffic, particles, cam, cops, mil)
	}

	// Eat pedestrians.
	r := NewRand(uint64(hx*100+hy*37) ^ 0xEA7)
	sizeMult := s.SizeMult
	if sizeMult < 1.0 {
		sizeMult = 1.0
	}
	eatRadius := SnakeEatRadius * sizeMult // larger eat radius during berserk
	for i := range peds.P {
		p := &peds.P[i]
		if !p.Alive {
			continue
		}
		d := math.Hypot(p.X-hx, p.Y-hy)
		if d >= eatRadius {
			continue
		}
		p.Alive = false
		if p.Infection == StateSymptomatic {
			// Infected: puke, shrink.
			s.Length -= 3
			s.PukeTimer = 1.5
			green := RGB{R: 60, G: 180, B: 40}
			for k := 0; k < 12; k++ {
				ang := r.RangeF(0, 2*math.Pi)
				spd := r.RangeF(5, 20)
				particles.Add(Particle{
					X: hx + r.RangeF(-0.5, 0.5), Y: hy + r.RangeF(-0.5, 0.5),
					VX: math.Cos(ang) * spd, VY: math.Sin(ang) * spd,
					Size: 0.5, MaxLife: r.RangeF(0.3, 0.7),
					Col: green, Kind: ParticleGlow,
				})
			}
			// Green ground paint.
			bx := int(math.Round(hx))
			by := int(math.Round(hy))
			world.AddTempPaint(bx, by, green, 3.0)
			world.AddTempPaint(bx+1, by, green, 3.0)
			world.AddTempPaint(bx, by+1, green, 3.0)
		} else {
			// Healthy (armed or not): eat, grow.
			scoreAdd := 100
			if p.Armed {
				scoreAdd = 200
			}
			s.Score += scoreAdd
			s.Length += 2
			s.WantedLevel = min(WantedMax, s.WantedLevel+0.1)
			// Kill streak combo.
			s.KillStreak++
			s.KillStreakTimer = 2.5
			if msg, bonus, col := killStreakResult(s.KillStreak); msg != "" {
				s.Score += bonus
				s.KillMsg = msg
				s.KillMsgTimer = 2.2
				s.KillMsgCol = col
			}
			// AI mode: count this kill toward the target quota.
			if s.AITimer > 0 {
				s.AITargetsLeft--
				if s.AITargetsLeft <= 0 {
					s.AITimer = 0
				}
			}
			// Blood splatter: main burst + perpendicular scatter + back spray for messier splat.
			particles.SpawnBlood(hx, hy, math.Cos(s.Heading), math.Sin(s.Heading), 30, 1.0)
			perpAng := s.Heading + math.Pi/2
			particles.SpawnBlood(hx+0.5, hy+0.5, math.Cos(perpAng), math.Sin(perpAng), 15, 0.7)
			particles.SpawnBlood(hx-0.5, hy-0.5, math.Cos(perpAng+math.Pi), math.Sin(perpAng+math.Pi), 12, 0.6)
			// Extra random splatter.
			for k := 0; k < 8; k++ {
				ang := r.RangeF(0, 2*math.Pi)
				spd := r.RangeF(15, 45)
				particles.Add(Particle{
					X: hx + r.RangeF(-1, 1), Y: hy + r.RangeF(-1, 1),
					VX: math.Cos(ang) * spd, VY: math.Sin(ang) * spd,
					Z: r.RangeF(0, 3), VZ: r.RangeF(10, 30),
					Size: r.RangeF(0.3, 0.7), MaxLife: r.RangeF(0.5, 1.2),
					Col:  RGB{R: uint8(100 + r.Range(0, 50)), G: uint8(10 + r.Range(0, 20)), B: uint8(10 + r.Range(0, 20))},
					Kind: ParticleBlood,
				})
			}
			// Start gore trail.
			s.GoreTimer = 2.0 + r.RangeF(0, 1.5)
			s.EvoPoints += 10
			// Wider blood pool: larger diamond pattern on the ground.
			bx := int(math.Round(hx))
			by := int(math.Round(hy))
			blood := RGB{R: 130, G: 20, B: 20}
			for ddx := -3; ddx <= 3; ddx++ {
				for ddy := -3; ddy <= 3; ddy++ {
					if abs(ddx)+abs(ddy) <= 3 {
						world.PaintRGB(clamp(bx+ddx, 0, WorldWidth-1), clamp(by+ddy, 0, WorldHeight-1), blood)
					}
				}
			}
			paintFallenPed(world, bx, by, p.Skin, p.Col)
		}
	}

	// Eat cars.
	for i := range traffic.Cars {
		c := &traffic.Cars[i]
		if !c.Alive {
			continue
		}
		d := math.Hypot(c.X-hx, c.Y-hy)
		if d >= SnakeCarEatRadius {
			continue
		}
		c.Alive = false
		cx := int(math.Round(c.X))
		cy := int(math.Round(c.Y))
		s.ExplodeAt(cx, cy, 5, world, particles, peds, traffic, cam, cops, mil)
		s.HP.Damage(1.5)
		s.SpeedBoost = 2.0
		s.SpeedMult = 1.8
		s.Score += 500
		s.Length += 4
		s.WantedLevel = min(WantedMax, s.WantedLevel+0.4)
	}

	// Collect bonuses.
	if bonuses != nil {
		prevHX, prevHY := hx, hy
		hasPrevHead := len(s.Path) > 1
		if hasPrevHead {
			prevHX = s.Path[1].X
			prevHY = s.Path[1].Y
		}
		for i := range bonuses.Boxes {
			b := &bonuses.Boxes[i]
			if !b.Alive {
				continue
			}
			if bonusPickupSkewHit(hx, hy, s.Heading, b.X, b.Y, prevHX, prevHY, hasPrevHead) {
				kind := b.Kind
				bonuses.Collect(i, s, world, peds, traffic, particles, cam, cops, mil)
				if kind == BonusFire {
					s.setFireBoltsAll()
				}
				// Propagate area effects to all ghosts and clones.
				propR := NewRand(uint64(hx*53+hy*37) ^ 0x9A1D)
				for _, g := range s.Ghosts {
					applyPositionalBonus(kind, s, g.X, g.Y, propR, world, peds, traffic, particles, cam, cops, mil)
				}
				for _, c := range s.Clones {
					applyPositionalBonus(kind, s, c.X, c.Y, propR, world, peds, traffic, particles, cam, cops, mil)
				}
			}
		}
	}

	// Puke particle trail.
	if s.PukeTimer > 0 {
		s.PukeTimer -= dt
		if particles != nil {
			rr := NewRand(uint64(hx*100) ^ uint64(s.PukeTimer*1000))
			// Spawn behind head.
			backAng := s.Heading + math.Pi + rr.RangeF(-0.4, 0.4)
			spd := rr.RangeF(3, 10)
			particles.Add(Particle{
				X: hx + rr.RangeF(-0.5, 0.5), Y: hy + rr.RangeF(-0.5, 0.5),
				VX: math.Cos(backAng) * spd, VY: math.Sin(backAng) * spd,
				Size: 0.5, MaxLife: rr.RangeF(0.2, 0.5),
				Col: RGB{R: 60, G: 180, B: 40}, Kind: ParticleGlow,
			})
		}
	}

	// Orbiting fire bolts: 5 bolts circle the snake head, destroying pixels and killing peds.
	runFireBolts(hx, hy, &s.FireRingTimer, &s.FireBoltAngle, dt, s, world, peds, particles)

	// Weapon bonuses.
	s.updateHomingMissiles(dt, hx, hy, world, peds, traffic, particles, cam, cops, mil)
	s.updateGatling(dt, hx, hy, world, peds, traffic, particles, cam, cops, mil)
	s.updateStrikeWorms(dt, world, peds, particles)
	s.updateStrikeHelis(dt, world, peds, particles, cam, cops, mil)
	s.updateBeltBombs(dt, world, peds, traffic, particles, cam, cops, mil)
	s.updateStrikePlanes(dt, world, peds, traffic, particles, cam, cops, mil)
	s.updateTimedExploders(dt, world, peds, traffic, particles, cam, cops, mil)

	// Spread bombs: drop one every ~0.3s, detonate on fuse expiry.
	if s.SpreadTimer > 0 {
		s.SpreadTimer -= dt
		s.SpreadDropTimer -= dt
		rBomb := NewRand(uint64(hx*53+hy*37) ^ uint64(s.SpreadTimer*1000))
		if s.SpreadDropTimer <= 0 {
			s.SpreadDropTimer = 0.25 + rBomb.RangeF(0, 0.15)
			s.SpreadBombs = append(s.SpreadBombs, SpreadBomb{
				X: hx, Y: hy,
				Fuse: 1.0 + rBomb.RangeF(0, 0.8),
			})
		}
	}
	for i := len(s.SpreadBombs) - 1; i >= 0; i-- {
		b := &s.SpreadBombs[i]
		b.Fuse -= dt
		if b.Fuse <= 0 {
			s.ExplodeAt(int(math.Round(b.X)), int(math.Round(b.Y)), 5, world, particles, peds, traffic, cam, cops, mil)
			s.WantedLevel = min(WantedMax, s.WantedLevel+0.12)
			s.SpreadBombs[i] = s.SpreadBombs[len(s.SpreadBombs)-1]
			s.SpreadBombs = s.SpreadBombs[:len(s.SpreadBombs)-1]
		}
	}

	// Clone: mirror the main snake's heading at other world positions, phase through walls.
	if s.CloneTimer > 0 {
		s.CloneTimer -= dt
		if s.CloneTimer <= 0 {
			s.Clones = s.Clones[:0]
		}
	}
	for ci := range s.Clones {
		c := &s.Clones[ci]
		c.Heading = s.Heading // mirrors player direction exactly
		nx := clampF(c.X+math.Cos(c.Heading)*effectiveSpeed*dt, 0, float64(WorldWidth-1))
		ny := clampF(c.Y+math.Sin(c.Heading)*effectiveSpeed*dt, 0, float64(WorldHeight-1))
		c.X, c.Y = nx, ny
		// Prepend to path.
		c.Path = append(c.Path, PathPoint{})
		copy(c.Path[1:], c.Path[0:])
		c.Path[0] = PathPoint{X: c.X, Y: c.Y}
		maxPts := int(math.Ceil(c.Length/1.5)) + 8
		if len(c.Path) > maxPts {
			c.Path = c.Path[:maxPts]
		}
		// Eat peds.
		for j := range peds.P {
			p := &peds.P[j]
			if !p.Alive {
				continue
			}
			if math.Hypot(p.X-c.X, p.Y-c.Y) < SnakeEatRadius {
				p.Alive = false
				s.Score += 100
				if particles != nil {
					particles.SpawnBlood(c.X, c.Y, math.Cos(c.Heading), math.Sin(c.Heading), 14, 0.8)
				}
			}
		}
		// Collect bonus boxes: area effects trigger at clone's position.
		if bonuses != nil {
			prevCX, prevCY := c.X, c.Y
			hasPrevClone := len(c.Path) > 1
			if hasPrevClone {
				prevCX = c.Path[1].X
				prevCY = c.Path[1].Y
			}
			for j := range bonuses.Boxes {
				b := &bonuses.Boxes[j]
				if !b.Alive {
					continue
				}
				if bonusPickupSkewHit(c.X, c.Y, c.Heading, b.X, b.Y, prevCX, prevCY, hasPrevClone) {
					kind := b.Kind
					bonuses.CollectAt(j, s, c.X, c.Y, world, peds, traffic, particles, cam, cops, mil)
					if kind == BonusFire {
						c.FireBoltTimer = s.FireRingTimer
						c.FireBoltAngle = 0
						s.setFireBoltsAll()
					}
					// Propagate area effects to all ghosts and other clones.
					propR := NewRand(uint64(c.X*53+c.Y*37) ^ 0x9A1D)
					for _, g := range s.Ghosts {
						applyPositionalBonus(kind, s, g.X, g.Y, propR, world, peds, traffic, particles, cam, cops, mil)
					}
					for k2 := range s.Clones {
						if k2 == ci {
							continue
						}
						oc := &s.Clones[k2]
						applyPositionalBonus(kind, s, oc.X, oc.Y, propR, world, peds, traffic, particles, cam, cops, mil)
					}
					break
				}
			}
		}
		// Orbiting fire bolts for this clone (independent from main snake).
		runFireBolts(c.X, c.Y, &c.FireBoltTimer, &c.FireBoltAngle, dt, s, world, peds, particles)
	}

	// Vacuum bubbles: erase world pixels and animate them inward.
	s.updateVacuumBubbles(dt, world, peds, traffic, particles, cam, cops, mil)

	// Clamp length bounds.
	if s.Length > SnakeMaxLength {
		s.Length = SnakeMaxLength
	}

	// Death check.
	if s.Length < 3 || s.HP.IsDead() {
		s.Alive = false
	}
}

// runFireBolts advances the orbiting fire-bolt effect centered at (cx, cy).
// Modifies timer and angle in place. Safe to call when timer <= 0 (no-op).
func runFireBolts(cx, cy float64, timer, angle *float64, dt float64, s *Snake, world *World, peds *PedestrianSystem, particles *ParticleSystem) {
	if *timer <= 0 {
		return
	}
	*timer -= dt
	*angle += 3.0 * dt

	const numBolts = 5
	const orbitRadius = 12.0
	const boltKillRadius = 2.5

	rr := NewRand(uint64(cx*53+cy*37) ^ uint64(*timer*1000))

	for bi := range numBolts {
		ang := *angle + float64(bi)*(2*math.Pi/numBolts)
		boltX := cx + math.Cos(ang)*orbitRadius
		boltY := cy + math.Sin(ang)*orbitRadius

		if particles != nil {
			for range 3 {
				particles.Add(Particle{
					X: boltX + rr.RangeF(-0.5, 0.5), Y: boltY + rr.RangeF(-0.5, 0.5),
					VX: rr.RangeF(-8, 8), VY: rr.RangeF(-8, 8),
					Z: rr.RangeF(0, 3), VZ: rr.RangeF(10, 40),
					Size: 0.4 + rr.RangeF(0, 0.4), MaxLife: rr.RangeF(0.1, 0.3),
					Col: Palette.FireHot, Kind: ParticleFire,
				})
			}
		}

		ibx, iby := int(math.Round(boltX)), int(math.Round(boltY))
		for dx := -2; dx <= 2; dx++ {
			for dy := -2; dy <= 2; dy++ {
				if dx*dx+dy*dy > 4 {
					continue
				}
				px, py := ibx+dx, iby+dy
				if px < 0 || py < 0 || px >= WorldWidth || py >= WorldHeight {
					continue
				}
				if world.HeightAt(px, py) > 0 {
					world.PaintRGB(px, py, Palette.Lot)
				}
			}
		}

		if rr.RangeF(0, 1) < dt*5.0 {
			world.StartTreeBurn(ibx, iby)
		}

		for i := range peds.P {
			p := &peds.P[i]
			if !p.Alive {
				continue
			}
			if math.Hypot(p.X-boltX, p.Y-boltY) < boltKillRadius {
				p.Alive = false
				s.Score += 300
				if particles != nil {
					particles.SpawnBlood(p.X, p.Y, rr.RangeF(-1, 1), rr.RangeF(-1, 1), 10, 0.7)
				}
			}
		}
	}
}

func (s *Snake) applySurgeAt(hx, hy, dt float64, world *World, peds *PedestrianSystem, traffic *TrafficSystem, particles *ParticleSystem, cam *Camera, cops *CopSystem, mil *MilitarySystem) {
	const surgeRadius = 45.0
	const surgeSpeed = 90.0
	const surgeKillRadius = 2.2

	for i := range peds.P {
		p := &peds.P[i]
		if !p.Alive {
			continue
		}
		dx := hx - p.X
		dy := hy - p.Y
		d := math.Hypot(dx, dy)
		if d > 0 && d < surgeRadius {
			p.X += dx / d * surgeSpeed * dt
			p.Y += dy / d * surgeSpeed * dt
			if d < surgeKillRadius {
				p.Alive = false
				s.Score += 120
				s.EvoPoints += 10
				if particles != nil {
					particles.SpawnBlood(p.X, p.Y, dx, dy, 14, 0.9)
				}
			}
		}
	}
	if cops != nil {
		for i := range cops.Peds {
			cp := &cops.Peds[i]
			if !cp.Alive {
				continue
			}
			dx := hx - cp.X
			dy := hy - cp.Y
			d := math.Hypot(dx, dy)
			if d > 0 && d < surgeRadius {
				cp.X += dx / d * surgeSpeed * dt
				cp.Y += dy / d * surgeSpeed * dt
				if d < surgeKillRadius {
					cp.Alive = false
					s.Score += 180
					s.EvoPoints += 14
					s.WantedLevel = min(WantedMax, s.WantedLevel+0.25)
					if particles != nil {
						particles.SpawnBlood(cp.X, cp.Y, dx, dy, 18, 1.0)
					}
				}
			}
		}
		for i := range cops.Cars {
			cc := &cops.Cars[i]
			if !cc.Alive {
				continue
			}
			dx := hx - cc.X
			dy := hy - cc.Y
			d := math.Hypot(dx, dy)
			if d > 0 && d < surgeRadius {
				pull := 40.0
				cc.X += dx / d * pull * dt
				cc.Y += dy / d * pull * dt
				if d < surgeKillRadius+0.9 {
					cc.Alive = false
					s.Score += 320
					s.EvoPoints += 24
					s.WantedLevel = min(WantedMax, s.WantedLevel+0.4)
					s.ExplodeAt(int(math.Round(cc.X)), int(math.Round(cc.Y)), 5, world, particles, peds, traffic, cam, cops, mil)
				}
			}
		}
		for i := range cops.Helis {
			ch := &cops.Helis[i]
			if !ch.Alive {
				continue
			}
			dx := hx - ch.X
			dy := hy - ch.Y
			d := math.Hypot(dx, dy)
			if d > 0 && d < surgeRadius {
				pull := 24.0
				ch.X += dx / d * pull * dt
				ch.Y += dy / d * pull * dt
				if d < surgeKillRadius+1.6 {
					ch.Alive = false
					s.Score += 420
					s.EvoPoints += 30
					s.WantedLevel = min(WantedMax, s.WantedLevel+0.45)
					s.ExplodeAt(int(math.Round(ch.X)), int(math.Round(ch.Y)), 6, world, particles, peds, traffic, cam, cops, mil)
				}
			}
		}
	}
}

func (s *Snake) updateVacuumBubbles(dt float64, world *World, peds *PedestrianSystem, traffic *TrafficSystem, particles *ParticleSystem, cam *Camera, cops *CopSystem, mil *MilitarySystem) {
	for vi := len(s.VacuumBubbles) - 1; vi >= 0; vi-- {
		vb := &s.VacuumBubbles[vi]
		vb.Timer += dt
		progress := clampF(vb.Timer/vb.MaxTime, 0, 1)
		radius := 25.0 * math.Sin(progress*math.Pi) // bell curve: 0 -> 25 -> 0

		if particles != nil && radius > 1.0 {
			rVac := NewRand(uint64(vb.X*53+vb.Y*37) ^ uint64(vb.Timer*1000))
			for range 50 {
				ang := rVac.RangeF(0, 2*math.Pi)
				dist := rVac.RangeF(0, radius)
				fpx := vb.X + math.Cos(ang)*dist
				fpy := vb.Y + math.Sin(ang)*dist
				ipx := int(math.Round(fpx))
				ipy := int(math.Round(fpy))
				if ipx < 0 || ipy < 0 || ipx >= WorldWidth || ipy >= WorldHeight {
					continue
				}
				col := world.ColorAt(ipx, ipy)
				bg := Palette.Lot
				if col.R == bg.R && col.G == bg.G && col.B == bg.B {
					continue
				}
				world.PaintRGB(ipx, ipy, bg)
				dx := vb.X - fpx
				dy := vb.Y - fpy
				d := math.Hypot(dx, dy)
				if d < 0.1 {
					continue
				}
				nx, ny := dx/d, dy/d
				speed := 60.0 + dist*1.5
				life := d / speed
				particles.Add(Particle{X: fpx, Y: fpy, VX: nx * speed, VY: ny * speed, Size: 0.5, MaxLife: life, Col: col, Kind: ParticleGlow})
			}
		}

		// Pull entities toward the bubble and kill them at the core.
		if radius > 0.8 {
			coreRadius := 1.6 + radius*0.08
			pullPed := 115.0
			pullCar := 55.0

			for i := range peds.P {
				p := &peds.P[i]
				if !p.Alive {
					continue
				}
				dx := vb.X - p.X
				dy := vb.Y - p.Y
				d := math.Hypot(dx, dy)
				if d > 0 && d < radius {
					p.X += dx / d * pullPed * dt
					p.Y += dy / d * pullPed * dt
					if d < coreRadius {
						p.Alive = false
						s.Score += 120
						s.EvoPoints += 10
						if particles != nil {
							particles.SpawnBlood(p.X, p.Y, dx, dy, 12, 0.9)
						}
					}
				}
			}

			if cops != nil {
				for i := range cops.Peds {
					cp := &cops.Peds[i]
					if !cp.Alive {
						continue
					}
					dx := vb.X - cp.X
					dy := vb.Y - cp.Y
					d := math.Hypot(dx, dy)
					if d > 0 && d < radius {
						cp.X += dx / d * pullPed * dt
						cp.Y += dy / d * pullPed * dt
						if d < coreRadius {
							cp.Alive = false
							s.Score += 180
							s.EvoPoints += 14
							s.WantedLevel = min(WantedMax, s.WantedLevel+0.2)
							if particles != nil {
								particles.SpawnBlood(cp.X, cp.Y, dx, dy, 16, 1.0)
							}
						}
					}
				}
				for i := range cops.Cars {
					cc := &cops.Cars[i]
					if !cc.Alive {
						continue
					}
					dx := vb.X - cc.X
					dy := vb.Y - cc.Y
					d := math.Hypot(dx, dy)
					if d > 0 && d < radius {
						cc.X += dx / d * pullCar * dt
						cc.Y += dy / d * pullCar * dt
						if d < coreRadius+0.7 {
							cc.Alive = false
							s.Score += 320
							s.EvoPoints += 22
							s.ExplodeAt(int(math.Round(cc.X)), int(math.Round(cc.Y)), 5, world, particles, peds, traffic, cam, cops, mil)
						}
					}
				}
				for i := range cops.Helis {
					ch := &cops.Helis[i]
					if !ch.Alive {
						continue
					}
					dx := vb.X - ch.X
					dy := vb.Y - ch.Y
					d := math.Hypot(dx, dy)
					if d > 0 && d < radius {
						ch.X += dx / d * (pullCar * 0.65) * dt
						ch.Y += dy / d * (pullCar * 0.65) * dt
						if d < coreRadius+1.2 {
							ch.Alive = false
							s.Score += 420
							s.EvoPoints += 30
							s.ExplodeAt(int(math.Round(ch.X)), int(math.Round(ch.Y)), 6, world, particles, peds, traffic, cam, cops, mil)
						}
					}
				}
			}
		}

		if vb.Timer >= vb.MaxTime {
			s.VacuumBubbles[vi] = s.VacuumBubbles[len(s.VacuumBubbles)-1]
			s.VacuumBubbles = s.VacuumBubbles[:len(s.VacuumBubbles)-1]
		}
	}
}

func (s *Snake) updateGhostBombRun(dt float64, world *World, peds *PedestrianSystem, traffic *TrafficSystem, particles *ParticleSystem, cam *Camera, cops *CopSystem, mil *MilitarySystem) {
	hx, hy := s.Head()
	evoSpeed := s.Speed * s.evoSpeedMult()
	outSpeed := evoSpeed * 2.4
	returnSpeed := evoSpeed * 3.3
	r := NewRand(uint64(hx*67+hy*41) ^ uint64(len(s.Ghosts))*0xD15EA5E)

	for i := len(s.Ghosts) - 1; i >= 0; i-- {
		g := &s.Ghosts[i]

		if g.Returning {
			dx := hx - g.X
			dy := hy - g.Y
			dist := math.Hypot(dx, dy)
			if dist < 2.6 {
				s.Length += g.SegLen
				if particles != nil {
					for range 6 {
						ang := r.RangeF(0, 2*math.Pi)
						particles.Add(Particle{
							X: hx, Y: hy,
							VX: math.Cos(ang) * r.RangeF(12, 30), VY: math.Sin(ang) * r.RangeF(12, 30),
							Size: 0.35, MaxLife: r.RangeF(0.12, 0.28),
							Col: RGB{R: 255, G: 190, B: 90}, Kind: ParticleGlow,
						})
					}
				}
				s.Ghosts[i] = s.Ghosts[len(s.Ghosts)-1]
				s.Ghosts = s.Ghosts[:len(s.Ghosts)-1]
				continue
			}
			g.Heading = math.Atan2(dy, dx)
			g.X = clampF(g.X+math.Cos(g.Heading)*returnSpeed*dt, 0, float64(WorldWidth-1))
			g.Y = clampF(g.Y+math.Sin(g.Heading)*returnSpeed*dt, 0, float64(WorldHeight-1))
			continue
		}

		dx := g.TargetX - g.X
		dy := g.TargetY - g.Y
		dist := math.Hypot(dx, dy)
		g.Heading = math.Atan2(dy, dx)
		if dist < 1.8 {
			if !g.Exploded {
				rad := 5
				if g.SegLen > 6 {
					rad = 6
				}
				s.ExplodeAt(int(math.Round(g.X)), int(math.Round(g.Y)), rad, world, particles, peds, traffic, cam, cops, mil)
				g.Exploded = true
			}
			g.Returning = true
			continue
		}
		step := outSpeed * dt
		if step > dist {
			step = dist
		}
		g.X = clampF(g.X+math.Cos(g.Heading)*step, 0, float64(WorldWidth-1))
		g.Y = clampF(g.Y+math.Sin(g.Heading)*step, 0, float64(WorldHeight-1))
	}

	if len(s.Ghosts) == 0 {
		s.GhostBombMode = false
		s.GhostTimer = 0
	}
}

func (s *Snake) updateStrikeWorms(dt float64, world *World, peds *PedestrianSystem, particles *ParticleSystem) {
	for i := len(s.StrikeWorms) - 1; i >= 0; i-- {
		w := &s.StrikeWorms[i]
		w.Life -= dt
		if w.Life <= 0 {
			s.StrikeWorms[i] = s.StrikeWorms[len(s.StrikeWorms)-1]
			s.StrikeWorms = s.StrikeWorms[:len(s.StrikeWorms)-1]
			continue
		}
		if !strikeWormWalkable(world, w.X, w.Y) {
			if nx, ny, ok := nearestWalkablePoint(world, w.X, w.Y, 18); ok {
				w.X, w.Y = nx, ny
			}
		}

		speed := 60.0
		targetIdx := -1
		if idx, tx, ty, ok := nearestAlivePed(w.X, w.Y, peds); ok {
			targetIdx = idx
			desired := math.Atan2(ty-w.Y, tx-w.X)
			turn := 7.8 * dt
			diff := angDiff(w.Heading, desired)
			if math.Abs(diff) <= turn {
				w.Heading = desired
			} else if diff > 0 {
				w.Heading += turn
			} else {
				w.Heading -= turn
			}
			speed = 130.0
		}

		step := speed * dt
		nx := w.X + math.Cos(w.Heading)*step
		ny := w.Y + math.Sin(w.Heading)*step
		moved := false
		if strikeWormWalkable(world, nx, ny) {
			w.X = nx
			w.Y = ny
			moved = true
		} else if world != nil {
			clearStep := step
			if clearStep < 1.0 {
				clearStep = 1.0
			}
			if escape, ok := s.findClearDir(w.X, w.Y, w.Heading, clearStep, world); ok {
				w.Heading = escape
				nx = w.X + math.Cos(w.Heading)*step
				ny = w.Y + math.Sin(w.Heading)*step
				if strikeWormWalkable(world, nx, ny) {
					w.X = nx
					w.Y = ny
					moved = true
				}
			}

			if !moved {
				xTry := w.X + math.Cos(w.Heading)*step
				yTry := w.Y + math.Sin(w.Heading)*step
				canX := strikeWormWalkable(world, xTry, w.Y)
				canY := strikeWormWalkable(world, w.X, yTry)
				switch {
				case canX && !canY:
					w.X = xTry
					if math.Cos(w.Heading) >= 0 {
						w.Heading = 0
					} else {
						w.Heading = math.Pi
					}
					moved = true
				case canY && !canX:
					w.Y = yTry
					if math.Sin(w.Heading) >= 0 {
						w.Heading = math.Pi / 2
					} else {
						w.Heading = -math.Pi / 2
					}
					moved = true
				case canX && canY:
					// Diagonal destination is blocked; slide along one clear axis.
					if math.Abs(math.Cos(w.Heading)) >= math.Abs(math.Sin(w.Heading)) {
						w.X = xTry
						if math.Cos(w.Heading) >= 0 {
							w.Heading = 0
						} else {
							w.Heading = math.Pi
						}
					} else {
						w.Y = yTry
						if math.Sin(w.Heading) >= 0 {
							w.Heading = math.Pi / 2
						} else {
							w.Heading = -math.Pi / 2
						}
					}
					moved = true
				}
			}
		}
		if !moved {
			if world != nil {
				w.Heading += math.Pi * 0.75
			} else {
				w.X = clampF(nx, 0, float64(WorldWidth-1))
				w.Y = clampF(ny, 0, float64(WorldHeight-1))
			}
		}

		if targetIdx >= 0 && targetIdx < len(peds.P) {
			p := &peds.P[targetIdx]
			if p.Alive && math.Hypot(p.X-w.X, p.Y-w.Y) < 1.8 {
				p.Alive = false
				s.Score += 80
				s.EvoPoints += 8
				if particles != nil {
					dx := p.X - w.X
					dy := p.Y - w.Y
					if d := math.Hypot(dx, dy); d > 0.01 {
						particles.SpawnBlood(p.X, p.Y, dx/d, dy/d, 14, 0.9)
					} else {
						particles.SpawnBlood(p.X, p.Y, 1, 0, 14, 0.9)
					}
				}
			}
		}
	}
}

func (s *Snake) updateStrikeHelis(dt float64, world *World, peds *PedestrianSystem, particles *ParticleSystem, cam *Camera, cops *CopSystem, mil *MilitarySystem) {
	const (
		strikeHeliEngageRange = 34.0
		missileHitRadius      = 2.1
		missileBlastRadius    = 2.8
	)

	r := NewRand(
		uint64(len(s.StrikeHelis))*0xD01A57 ^
			uint64(len(s.StrikeHeliShots))*0x51A07 ^
			uint64(len(s.StrikeHeliMiss))*0x91AC3,
	)
	for i := len(s.StrikeHelis) - 1; i >= 0; i-- {
		h := &s.StrikeHelis[i]
		h.RotorAngle += (12.0 + h.OrbitSpd*4.0) * dt
		if !h.Exiting {
			if h.DeployTimer > 0 {
				h.DeployTimer -= dt
				dx := h.AttackX - h.X
				dy := h.AttackY - h.Y
				d := math.Hypot(dx, dy)
				if d > 0.01 {
					h.Heading = math.Atan2(dy, dx)
					spd := 95.0 + h.OrbitSpd*35.0
					step := spd * dt
					if step > d {
						step = d
					}
					h.X = clampF(h.X+dx/d*step, 0, float64(WorldWidth-1))
					h.Y = clampF(h.Y+dy/d*step, 0, float64(WorldHeight-1))
				}
			} else {
				h.Life -= dt
				h.OrbitA += h.OrbitSpd * dt
				h.X = clampF(h.CenterX+math.Cos(h.OrbitA)*h.OrbitR, 0, float64(WorldWidth-1))
				h.Y = clampF(h.CenterY+math.Sin(h.OrbitA)*h.OrbitR, 0, float64(WorldHeight-1))
				h.Heading = h.OrbitA + math.Pi/2

				h.FireTimer -= dt
				for h.FireTimer <= 0 {
					if h.MissileMode {
						h.FireTimer += 0.42 + r.RangeF(0, 0.30)
						if tx, ty, ok := nearestStrikeHeliTargetWithin(h.X, h.Y, strikeHeliEngageRange, peds, cops, mil); ok {
							ang := math.Atan2(ty-h.Y, tx-h.X) + r.RangeF(-0.08, 0.08)
							spd := 82.0 + r.RangeF(0, 20.0)
							s.StrikeHeliMiss = append(s.StrikeHeliMiss, StrikeHeliMissile{
								X:          h.X + math.Cos(h.Heading)*2.8,
								Y:          h.Y + math.Sin(h.Heading)*2.8,
								VX:         math.Cos(ang) * spd,
								VY:         math.Sin(ang) * spd,
								Life:       1.05 + r.RangeF(0, 0.35),
								TargetDist: 14.0 + r.RangeF(0, 8.0),
							})
						}
					} else {
						h.FireTimer += 0.09 + r.RangeF(0, 0.13)
						if tx, ty, ok := nearestStrikeHeliTargetWithin(h.X, h.Y, strikeHeliEngageRange, peds, cops, mil); ok {
							ang := math.Atan2(ty-h.Y, tx-h.X) + r.RangeF(-0.07, 0.07)
							spd := 190.0
							s.StrikeHeliShots = append(s.StrikeHeliShots, StrikeHeliShot{
								X:    h.X + math.Cos(h.Heading)*2.6,
								Y:    h.Y + math.Sin(h.Heading)*2.6,
								VX:   math.Cos(ang) * spd,
								VY:   math.Sin(ang) * spd,
								Life: 0.40 + r.RangeF(0, 0.10),
							})
						}
					}
				}

				if h.Life <= 0 {
					h.Exiting = true
					h.ExitTimer = 1.0 + r.RangeF(0, 0.6)
					out := math.Atan2(h.Y-h.AttackY, h.X-h.AttackX) + r.RangeF(-0.35, 0.35)
					spd := 125.0 + r.RangeF(0, 55.0)
					h.ExitVX = math.Cos(out) * spd
					h.ExitVY = math.Sin(out) * spd
					h.Heading = out
				}
			}
		} else {
			h.ExitTimer -= dt
			h.X += h.ExitVX * dt
			h.Y += h.ExitVY * dt
			h.Heading = math.Atan2(h.ExitVY, h.ExitVX)
			const exitMargin = 20.0
			outOfBounds := h.X < -exitMargin || h.X > float64(WorldWidth)+exitMargin ||
				h.Y < -exitMargin || h.Y > float64(WorldHeight)+exitMargin
			if outOfBounds || h.ExitTimer <= 0 {
				s.StrikeHelis[i] = s.StrikeHelis[len(s.StrikeHelis)-1]
				s.StrikeHelis = s.StrikeHelis[:len(s.StrikeHelis)-1]
				continue
			}
		}
	}

	for i := len(s.StrikeHeliShots) - 1; i >= 0; i-- {
		b := &s.StrikeHeliShots[i]
		b.Life -= dt
		if b.Life <= 0 {
			s.StrikeHeliShots[i] = s.StrikeHeliShots[len(s.StrikeHeliShots)-1]
			s.StrikeHeliShots = s.StrikeHeliShots[:len(s.StrikeHeliShots)-1]
			continue
		}

		b.X += b.VX * dt
		b.Y += b.VY * dt
		if b.X < 0 || b.Y < 0 || b.X >= float64(WorldWidth) || b.Y >= float64(WorldHeight) {
			s.StrikeHeliShots[i] = s.StrikeHeliShots[len(s.StrikeHeliShots)-1]
			s.StrikeHeliShots = s.StrikeHeliShots[:len(s.StrikeHeliShots)-1]
			continue
		}

		if s.gatlingHitEntity(b.X, b.Y, world, peds, nil, particles, cam, cops, mil) {
			s.StrikeHeliShots[i] = s.StrikeHeliShots[len(s.StrikeHeliShots)-1]
			s.StrikeHeliShots = s.StrikeHeliShots[:len(s.StrikeHeliShots)-1]
			continue
		}
	}

	for i := len(s.StrikeHeliMiss) - 1; i >= 0; i-- {
		m := &s.StrikeHeliMiss[i]
		m.Life -= dt
		if m.Life <= 0 {
			s.StrikeHeliMiss[i] = s.StrikeHeliMiss[len(s.StrikeHeliMiss)-1]
			s.StrikeHeliMiss = s.StrikeHeliMiss[:len(s.StrikeHeliMiss)-1]
			continue
		}

		if tx, ty, ok := nearestStrikeHeliTargetWithin(m.X, m.Y, m.TargetDist, peds, cops, mil); ok {
			desired := math.Atan2(ty-m.Y, tx-m.X)
			cur := math.Atan2(m.VY, m.VX)
			turn := 5.8 * dt
			diff := angDiff(cur, desired)
			if math.Abs(diff) > turn {
				if diff > 0 {
					cur += turn
				} else {
					cur -= turn
				}
			} else {
				cur = desired
			}
			speed := clampF(math.Hypot(m.VX, m.VY)+dt*24.0, 80, 132)
			m.VX = math.Cos(cur) * speed
			m.VY = math.Sin(cur) * speed
		}

		m.X += m.VX * dt
		m.Y += m.VY * dt

		if particles != nil {
			particles.Add(Particle{
				X: m.X, Y: m.Y,
				VX: -m.VX * 0.11, VY: -m.VY * 0.11,
				Size: 0.30, MaxLife: 0.14,
				Col: RGB{R: 255, G: 145, B: 52}, Kind: ParticleGlow,
			})
		}

		if m.X < 0 || m.Y < 0 || m.X >= float64(WorldWidth) || m.Y >= float64(WorldHeight) {
			s.StrikeHeliMiss[i] = s.StrikeHeliMiss[len(s.StrikeHeliMiss)-1]
			s.StrikeHeliMiss = s.StrikeHeliMiss[:len(s.StrikeHeliMiss)-1]
			continue
		}

		if _, _, ok := nearestStrikeHeliTargetWithin(m.X, m.Y, missileHitRadius, peds, cops, mil); !ok {
			continue
		}

		if world != nil {
			s.ExplodeAt(int(math.Round(m.X)), int(math.Round(m.Y)), int(math.Ceil(missileBlastRadius)), world, particles, peds, nil, cam, cops, mil)
		}
		if particles != nil {
			for burst := 0; burst < 16; burst++ {
				ang := r.RangeF(0, 2*math.Pi)
				spd := r.RangeF(24, 90)
				particles.Add(Particle{
					X: m.X, Y: m.Y,
					VX: math.Cos(ang) * spd, VY: math.Sin(ang) * spd,
					Size: 0.52, MaxLife: r.RangeF(0.14, 0.35),
					Col: RGB{R: 255, G: 130, B: 45}, Kind: ParticleFire,
				})
			}
		}
		s.StrikeHeliMiss[i] = s.StrikeHeliMiss[len(s.StrikeHeliMiss)-1]
		s.StrikeHeliMiss = s.StrikeHeliMiss[:len(s.StrikeHeliMiss)-1]
	}
}

func (s *Snake) updateBeltBombs(dt float64, world *World, peds *PedestrianSystem, traffic *TrafficSystem, particles *ParticleSystem, cam *Camera, cops *CopSystem, mil *MilitarySystem) {
	for i := len(s.BeltBombs) - 1; i >= 0; i-- {
		b := &s.BeltBombs[i]
		b.Fuse -= dt
		if b.Fuse > 0 {
			continue
		}
		s.ExplodeAt(int(math.Round(b.X)), int(math.Round(b.Y)), b.Radius, world, particles, peds, traffic, cam, cops, mil)
		s.BeltBombs[i] = s.BeltBombs[len(s.BeltBombs)-1]
		s.BeltBombs = s.BeltBombs[:len(s.BeltBombs)-1]
	}
}

func (s *Snake) updateStrikePlanes(dt float64, world *World, peds *PedestrianSystem, traffic *TrafficSystem, particles *ParticleSystem, cam *Camera, cops *CopSystem, mil *MilitarySystem) {
	r := NewRand(
		uint64(len(s.StrikePlanes))*0x3A5E ^
			uint64(len(s.StrikePlaneBombs))*0xBED17,
	)

	for i := len(s.StrikePlanes) - 1; i >= 0; i-- {
		p := &s.StrikePlanes[i]
		p.Life -= dt
		p.X += p.VX * dt
		p.Y += p.VY * dt
		p.Heading = math.Atan2(p.VY, p.VX)

		if !p.Dropped && p.BombsLeft > 0 {
			nearTarget := math.Hypot(p.X-p.TargetX, p.Y-p.TargetY)
			threshold := 2.2 + math.Hypot(p.VX, p.VY)*dt*0.85
			if nearTarget <= threshold {
				for bi := 0; bi < p.BombsLeft; bi++ {
					s.StrikePlaneBombs = append(s.StrikePlaneBombs, StrikePlaneBomb{
						X:      p.TargetX,
						Y:      p.TargetY,
						VX:     0,
						VY:     0,
						Fuse:   0.18 + float64(bi)*0.10 + r.RangeF(0, 0.08),
						Radius: p.BombRadius,
					})
				}
				p.Dropped = true
				p.BombsLeft = 0
			}
		}

		if p.Life <= 0 {
			s.StrikePlanes[i] = s.StrikePlanes[len(s.StrikePlanes)-1]
			s.StrikePlanes = s.StrikePlanes[:len(s.StrikePlanes)-1]
		}
	}

	for i := len(s.StrikePlaneBombs) - 1; i >= 0; i-- {
		b := &s.StrikePlaneBombs[i]
		b.Fuse -= dt
		b.X += b.VX * dt
		b.Y += b.VY * dt
		if particles != nil {
			particles.Add(Particle{
				X: b.X, Y: b.Y,
				VX: -b.VX * 0.25, VY: -b.VY * 0.25,
				Size: 0.26, MaxLife: 0.15,
				Col: RGB{R: 255, G: 160, B: 90}, Kind: ParticleGlow,
			})
		}
		if b.Fuse > 0 {
			continue
		}
		ix := int(math.Round(b.X))
		iy := int(math.Round(b.Y))
		if ix >= 0 && iy >= 0 && ix < WorldWidth && iy < WorldHeight {
			s.ExplodeAt(ix, iy, b.Radius, world, particles, peds, traffic, cam, cops, mil)
		}
		s.StrikePlaneBombs[i] = s.StrikePlaneBombs[len(s.StrikePlaneBombs)-1]
		s.StrikePlaneBombs = s.StrikePlaneBombs[:len(s.StrikePlaneBombs)-1]
	}
}

func timedExploderWalkable(world *World, x, y float64) bool {
	if world == nil {
		return true
	}
	ix := int(math.Round(x))
	iy := int(math.Round(y))
	if ix < 0 || iy < 0 || ix >= WorldWidth || iy >= WorldHeight {
		return false
	}
	return world.HeightAt(ix, iy) == 0
}

func (s *Snake) updateTimedExploders(dt float64, world *World, peds *PedestrianSystem, traffic *TrafficSystem, particles *ParticleSystem, cam *Camera, cops *CopSystem, mil *MilitarySystem) {
	r := NewRand(uint64(len(s.TimedExploders))*0x77A41 ^ uint64(s.Score)*0xB4D)
	for i := len(s.TimedExploders) - 1; i >= 0; i-- {
		e := &s.TimedExploders[i]
		e.Timer -= dt
		if e.Timer <= 0 {
			s.ExplodeAt(int(math.Round(e.X)), int(math.Round(e.Y)), e.Radius, world, particles, peds, traffic, cam, cops, mil)
			s.TimedExploders[i] = s.TimedExploders[len(s.TimedExploders)-1]
			s.TimedExploders = s.TimedExploders[:len(s.TimedExploders)-1]
			continue
		}

		turnScale := 2.2
		switch e.Kind {
		case TimedExploderCar:
			turnScale = 1.4
		case TimedExploderSnake:
			turnScale = 3.0
		}
		e.Heading += r.RangeF(-turnScale, turnScale) * dt
		step := e.Speed * dt
		nx := e.X + math.Cos(e.Heading)*step
		ny := e.Y + math.Sin(e.Heading)*step
		if timedExploderWalkable(world, nx, ny) {
			e.X = nx
			e.Y = ny
			continue
		}
		clearStep := max(step, 1.2)
		if escape, ok := s.findClearDir(e.X, e.Y, e.Heading, clearStep, world); ok {
			e.Heading = escape
			nx = e.X + math.Cos(e.Heading)*step
			ny = e.Y + math.Sin(e.Heading)*step
			if timedExploderWalkable(world, nx, ny) {
				e.X = nx
				e.Y = ny
				continue
			}
		}
		e.Heading += math.Pi * (0.45 + r.RangeF(0, 0.4))
	}
}

func (s *Snake) updateHomingMissiles(dt, hx, hy float64, world *World, peds *PedestrianSystem, traffic *TrafficSystem, particles *ParticleSystem, cam *Camera, cops *CopSystem, mil *MilitarySystem) {
	if s.MissileTimer > 0 {
		s.MissileTimer -= dt
		s.MissileFireTimer -= dt
		for s.MissileFireTimer <= 0 {
			s.MissileFireTimer += 0.34
			muzzleX := hx + math.Cos(s.Heading)*3.0
			muzzleY := hy + math.Sin(s.Heading)*3.0
			speed := 95.0
			s.Missiles = append(s.Missiles, HomingMissile{
				X: muzzleX, Y: muzzleY,
				VX:   math.Cos(s.Heading) * speed,
				VY:   math.Sin(s.Heading) * speed,
				Life: 3.2,
			})
		}
	}

	for i := len(s.Missiles) - 1; i >= 0; i-- {
		m := &s.Missiles[i]
		m.Life -= dt
		if m.Life <= 0 {
			s.Missiles[i] = s.Missiles[len(s.Missiles)-1]
			s.Missiles = s.Missiles[:len(s.Missiles)-1]
			continue
		}

		if tx, ty, ok := nearestSnakeMissileTarget(m.X, m.Y, peds); ok {
			desired := math.Atan2(ty-m.Y, tx-m.X)
			cur := math.Atan2(m.VY, m.VX)
			turn := 4.8 * dt
			diff := angDiff(cur, desired)
			if math.Abs(diff) > turn {
				if diff > 0 {
					cur += turn
				} else {
					cur -= turn
				}
			} else {
				cur = desired
			}
			speed := math.Hypot(m.VX, m.VY)
			speed = clampF(speed+dt*35, 90, 150)
			m.VX = math.Cos(cur) * speed
			m.VY = math.Sin(cur) * speed
		}

		m.X += m.VX * dt
		m.Y += m.VY * dt

		if particles != nil {
			particles.Add(Particle{
				X: m.X, Y: m.Y,
				VX: -m.VX * 0.12, VY: -m.VY * 0.12,
				Size: 0.35, MaxLife: 0.16,
				Col: RGB{R: 255, G: 170, B: 60}, Kind: ParticleGlow,
			})
		}

		mx := int(math.Round(m.X))
		my := int(math.Round(m.Y))
		if mx < 0 || my < 0 || mx >= WorldWidth || my >= WorldHeight {
			s.Missiles[i] = s.Missiles[len(s.Missiles)-1]
			s.Missiles = s.Missiles[:len(s.Missiles)-1]
			continue
		}
		// Missile bonus rounds fly over buildings and only detonate on pedestrian contact.
		if snakeMissileHitTarget(m.X, m.Y, 2.2, peds) {
			s.ExplodeAt(mx, my, 5, world, particles, peds, traffic, cam, cops, mil)
			s.Missiles[i] = s.Missiles[len(s.Missiles)-1]
			s.Missiles = s.Missiles[:len(s.Missiles)-1]
			continue
		}
	}
}

func (s *Snake) updateGatling(dt, hx, hy float64, world *World, peds *PedestrianSystem, traffic *TrafficSystem, particles *ParticleSystem, cam *Camera, cops *CopSystem, mil *MilitarySystem) {
	if s.GatlingTimer > 0 {
		s.GatlingTimer -= dt
		s.GatlingFireTimer -= dt
		for s.GatlingFireTimer <= 0 {
			s.GatlingFireTimer += 0.045
			muzzleX := hx + math.Cos(s.Heading)*3.2
			muzzleY := hy + math.Sin(s.Heading)*3.2
			spread := 0.08 * math.Sin(float64(len(s.GatlingRounds))*0.7)
			a := s.Heading + spread
			speed := 235.0
			s.GatlingRounds = append(s.GatlingRounds, SnakeRound{
				X: muzzleX, Y: muzzleY,
				VX:   math.Cos(a) * speed,
				VY:   math.Sin(a) * speed,
				Life: 1.0,
			})
		}
	}

	for i := len(s.GatlingRounds) - 1; i >= 0; i-- {
		r := &s.GatlingRounds[i]
		r.Life -= dt
		if r.Life <= 0 {
			s.GatlingRounds[i] = s.GatlingRounds[len(s.GatlingRounds)-1]
			s.GatlingRounds = s.GatlingRounds[:len(s.GatlingRounds)-1]
			continue
		}

		px, py := r.X, r.Y
		nx := r.X + r.VX*dt
		ny := r.Y + r.VY*dt
		steps := max(1, int(math.Ceil(math.Hypot(nx-px, ny-py))))
		hit := false
		for si := 1; si <= steps; si++ {
			t := float64(si) / float64(steps)
			sx := px + (nx-px)*t
			sy := py + (ny-py)*t
			ix := int(math.Round(sx))
			iy := int(math.Round(sy))
			if ix < 0 || iy < 0 || ix >= WorldWidth || iy >= WorldHeight {
				hit = true
				break
			}
			if world.HeightAt(ix, iy) > 0 {
				s.ExplodeAt(ix, iy, 1, world, particles, peds, traffic, cam, cops, mil)
				hit = true
				break
			}
			if s.gatlingHitEntity(sx, sy, world, peds, traffic, particles, cam, cops, mil) {
				hit = true
				break
			}
		}

		if hit {
			s.GatlingRounds[i] = s.GatlingRounds[len(s.GatlingRounds)-1]
			s.GatlingRounds = s.GatlingRounds[:len(s.GatlingRounds)-1]
			continue
		}

		r.X = nx
		r.Y = ny
		if particles != nil {
			particles.Add(Particle{
				X: r.X, Y: r.Y,
				VX: -r.VX * 0.10, VY: -r.VY * 0.10,
				Size: 0.22, MaxLife: 0.08,
				Col: RGB{R: 255, G: 210, B: 90}, Kind: ParticleGlow,
			})
		}
	}
}

func nearestSnakeMissileTarget(mx, my float64, peds *PedestrianSystem) (float64, float64, bool) {
	best := math.MaxFloat64
	tx, ty := 0.0, 0.0

	for i := range peds.P {
		p := &peds.P[i]
		if !p.Alive {
			continue
		}
		d := math.Hypot(p.X-mx, p.Y-my)
		if d < best {
			best = d
			tx, ty = p.X, p.Y
		}
	}

	return tx, ty, best < math.MaxFloat64
}

func snakeMissileHitTarget(mx, my, radius float64, peds *PedestrianSystem) bool {
	for i := range peds.P {
		p := &peds.P[i]
		if p.Alive && math.Hypot(p.X-mx, p.Y-my) <= radius {
			return true
		}
	}
	return false
}

func (s *Snake) gatlingHitEntity(x, y float64, world *World, peds *PedestrianSystem, traffic *TrafficSystem, particles *ParticleSystem, cam *Camera, cops *CopSystem, mil *MilitarySystem) bool {
	const softHit = 1.1
	const hardHit = 1.5

	for i := range peds.P {
		p := &peds.P[i]
		if !p.Alive || math.Hypot(p.X-x, p.Y-y) > softHit {
			continue
		}
		p.Alive = false
		s.Score += 110
		s.EvoPoints += 8
		if particles != nil {
			particles.SpawnBlood(p.X, p.Y, p.X-x, p.Y-y, 10, 0.8)
		}
		return true
	}

	if cops != nil {
		for i := range cops.Peds {
			p := &cops.Peds[i]
			if !p.Alive || math.Hypot(p.X-x, p.Y-y) > softHit {
				continue
			}
			p.Alive = false
			s.Score += 170
			s.EvoPoints += 12
			s.WantedLevel = min(WantedMax, s.WantedLevel+0.15)
			if particles != nil {
				particles.SpawnBlood(p.X, p.Y, p.X-x, p.Y-y, 12, 0.9)
			}
			return true
		}
		for i := range cops.Cars {
			c := &cops.Cars[i]
			if !c.Alive || math.Hypot(c.X-x, c.Y-y) > hardHit {
				continue
			}
			c.Alive = false
			s.Score += 300
			s.EvoPoints += 22
			s.ExplodeAt(int(math.Round(c.X)), int(math.Round(c.Y)), 3, world, particles, peds, traffic, cam, cops, mil)
			return true
		}
		for i := range cops.Helis {
			h := &cops.Helis[i]
			if !h.Alive || math.Hypot(h.X-x, h.Y-y) > hardHit+0.6 {
				continue
			}
			h.Alive = false
			s.Score += 520
			s.EvoPoints += 28
			s.ExplodeAt(int(math.Round(h.X)), int(math.Round(h.Y)), 4, world, particles, peds, traffic, cam, cops, mil)
			return true
		}
	}

	if traffic != nil {
		for i := range traffic.Cars {
			c := &traffic.Cars[i]
			if !c.Alive || math.Hypot(c.X-x, c.Y-y) > hardHit {
				continue
			}
			c.Alive = false
			s.Score += 220
			s.EvoPoints += 18
			s.ExplodeAt(int(math.Round(c.X)), int(math.Round(c.Y)), 3, world, particles, peds, traffic, cam, cops, mil)
			return true
		}
	}

	if mil != nil {
		for i := range mil.Troops {
			t := &mil.Troops[i]
			if !t.Alive || math.Hypot(t.X-x, t.Y-y) > softHit {
				continue
			}
			t.Alive = false
			s.Score += 220
			s.EvoPoints += 16
			if particles != nil {
				particles.SpawnBlood(t.X, t.Y, t.X-x, t.Y-y, 12, 0.9)
			}
			return true
		}
		for i := range mil.Tanks {
			t := &mil.Tanks[i]
			if !t.Alive || math.Hypot(t.X-x, t.Y-y) > hardHit+0.5 {
				continue
			}
			t.Alive = false
			s.Score += 520
			s.EvoPoints += 28
			s.ExplodeAt(int(math.Round(t.X)), int(math.Round(t.Y)), 4, world, particles, peds, traffic, cam, cops, mil)
			return true
		}
		for i := range mil.Helis {
			h := &mil.Helis[i]
			if !h.Alive || math.Hypot(h.X-x, h.Y-y) > hardHit+0.5 {
				continue
			}
			h.Alive = false
			s.Score += 520
			s.EvoPoints += 28
			s.ExplodeAt(int(math.Round(h.X)), int(math.Round(h.Y)), 4, world, particles, peds, traffic, cam, cops, mil)
			return true
		}
	}

	return false
}

// setFireBoltsAll copies the main snake's fire bolt duration to every ghost and clone,
// so they each orbit their own position independently.
func (s *Snake) setFireBoltsAll() {
	for i := range s.Ghosts {
		s.Ghosts[i].FireBoltTimer = s.FireRingTimer
		s.Ghosts[i].FireBoltAngle = 0
	}
	for i := range s.Clones {
		s.Clones[i].FireBoltTimer = s.FireRingTimer
		s.Clones[i].FireBoltAngle = 0
	}
}

// Segments returns evenly-spaced positions along the snake path for rendering.
// spacing is the distance between samples (e.g. 1.5 px).
func (s *Snake) Segments() []PathPoint {
	spacing := 1.5
	if len(s.Path) < 2 {
		return s.Path
	}

	numSegs := int(math.Ceil(s.Length/spacing)) + 1
	out := make([]PathPoint, 0, numSegs)

	dist := 0.0
	out = append(out, s.Path[0])

	for i := 1; i < len(s.Path) && dist < s.Length; i++ {
		prev := s.Path[i-1]
		cur := s.Path[i]
		d := math.Hypot(cur.X-prev.X, cur.Y-prev.Y)
		if d < 0.001 {
			continue
		}
		dist += d
		if dist <= s.Length {
			out = append(out, cur)
		}
	}

	return out
}

// SnakeRenderData builds point sprite data for snake body rendering.
// Each sprite: x, y, size, r, g, b, a, rotation (8 floats).
func (s *Snake) SnakeRenderData() []float32 {
	if !s.Alive {
		return nil
	}
	segs := s.Segments()
	if len(segs) == 0 {
		return nil
	}

	swarmActive := len(s.Ghosts) > 0

	// During swarm: only the head is drawn — body is split into segments.
	renderSegs := segs
	if swarmActive && len(segs) > 1 {
		renderSegs = segs[:1]
	}

	buf := make([]float32, 0, (len(renderSegs)+len(s.Ghosts)*4)*2*8)
	total := float32(len(renderSegs))
	sizeMult := float32(s.SizeMult)
	if sizeMult < 1.0 {
		sizeMult = 1.0
	}

	for i, seg := range renderSegs {
		t := float32(i) / total // 0=head, 1=tail

		// Evolution changes body hue over time; berserk overrides with red.
		var red, green, blue uint8
		evoCol := s.evoSegmentColor(float64(t))
		red, green, blue = evoCol.R, evoCol.G, evoCol.B
		if s.BerserkTimer > 0 {
			red = lerpU8(255, 150, float64(t))
			green = lerpU8(60, 30, float64(t))
			blue = lerpU8(30, 8, float64(t))
		}
		size := float32(SnakeHeadSize) * sizeMult * (1.0 - t*0.55)
		if size < 1.2*sizeMult {
			size = 1.2 * sizeMult
		}

		// When idle, a slow sine wave travels from head to tail.
		rx, ry := seg.X, seg.Y

		// Shadow: offset south-east, larger than sprite, semi-transparent.
		buf = append(buf,
			float32(rx)+0.4, float32(ry)+0.9,
			size*1.6,
			0, 0, 0, 0.32, 0,
		)
		// Body point.
		buf = append(buf,
			float32(rx), float32(ry),
			size,
			float32(red)/255.0, float32(green)/255.0, float32(blue)/255.0, 1.0, 0,
		)
	}

	// Clone segments: blue-cyan tinted worm mirroring player movement.
	for _, c := range s.Clones {
		segs := cloneSegments(c)
		total := float32(len(segs))
		for i, seg := range segs {
			t := float32(i) / total
			cyan := lerpU8(220, 80, float64(t))
			sz := float32(SnakeHeadSize-0.5) * (1.0 - t*0.55)
			if sz < 1.0 {
				sz = 1.0
			}
			buf = append(buf, float32(seg.X), float32(seg.Y)+0.4, sz*1.1, 0.05, 0.05, 0.05, 0.4, 0)
			buf = append(buf, float32(seg.X), float32(seg.Y), sz,
				0.1, float32(cyan)/255.0, float32(lerpU8(200, 70, float64(t)))/255.0, 0.88, 0)
		}
	}

	// Swarm segments: each rendered as a 4-dot mini worm with heading trail.
	for _, g := range s.Ghosts {
		const numDots = 4
		for j := 0; j < numDots; j++ {
			t := float32(j) / float32(numDots)
			// Trail extends behind the segment's heading.
			tx := float32(g.X - math.Cos(g.Heading)*float64(j)*1.5)
			ty := float32(g.Y - math.Sin(g.Heading)*float64(j)*1.5)
			sz := float32(2.4 - float32(j)*0.35)
			if sz < 0.8 {
				sz = 0.8
			}

			var cr, cg, cb float32
			if g.Returning {
				// Warm yellow: heading home.
				cr = float32(lerpU8(230, 90, float64(t))) / 255.0
				cg = float32(lerpU8(220, 90, float64(t))) / 255.0
				cb = 0.08
			} else {
				// Bright green: hunting.
				cr = float32(lerpU8(50, 15, float64(t))) / 255.0
				cg = float32(lerpU8(255, 100, float64(t))) / 255.0
				cb = 0.1
			}
			// Shadow then body dot.
			buf = append(buf, tx, ty+0.4, sz*1.1, 0.05, 0.05, 0.05, 0.4, 0)
			buf = append(buf, tx, ty, sz, cr, cg, cb, 1.0, 0)
		}
	}

	// Targeted strike worms: small fast hunter worms.
	for i := range s.StrikeWorms {
		w := &s.StrikeWorms[i]
		// Lifetime fade toward end.
		t := clampF(w.Life/3.0, 0, 1)
		cr := float32(0.30 + 0.40*t)
		cg := float32(0.95)
		cb := float32(0.45 + 0.25*t)
		buf = append(buf, float32(w.X)+0.18, float32(w.Y)+0.35, 1.5, 0.04, 0.04, 0.04, 0.38, 0)
		buf = append(buf, float32(w.X), float32(w.Y), 1.2, cr, cg, cb, 0.95, 0)
		tx := float32(w.X - math.Cos(w.Heading)*1.1)
		ty := float32(w.Y - math.Sin(w.Heading)*1.1)
		buf = append(buf, tx, ty, 0.9, cr*0.72, cg*0.72, cb*0.72, 0.88, 0)
	}

	// Targeted strike helicopters: cop-heli silhouette, recolored red with black rotors.
	for i := range s.StrikeHelis {
		h := &s.StrikeHelis[i]
		fwdX := math.Cos(h.Heading)
		fwdY := math.Sin(h.Heading)
		perpX := -math.Sin(h.Heading)
		perpY := math.Cos(h.Heading)
		hx32 := float32(h.X)
		hy32 := float32(h.Y)

		var bodyR, bodyG, bodyB float32
		if h.Leader {
			bodyR, bodyG, bodyB = 0.92, 0.16, 0.14
		} else {
			bodyR, bodyG, bodyB = 0.85, 0.24, 0.16
		}
		cockR, cockG, cockB := float32(1.0), float32(0.42), float32(0.24)
		rotR, rotG, rotB := float32(0.08), float32(0.08), float32(0.08)
		if h.MissileMode {
			cockR, cockG, cockB = 1.0, 0.56, 0.18
		}

		buf = append(buf, hx32, hy32+1.8, 7.0, 0.05, 0.05, 0.05, 0.22, 0) // shadow
		buf = append(buf, hx32, hy32, 3.5, bodyR, bodyG, bodyB, 1.0, 0)   // body
		buf = append(buf,
			float32(h.X+fwdX*1.8), float32(h.Y+fwdY*1.8),
			2.2, cockR, cockG, cockB, 1.0, 0) // cockpit

		for step := 1; step <= 3; step++ {
			sz := float32(1.6 - float32(step)*0.25)
			buf = append(buf,
				float32(h.X-fwdX*float64(step)*1.3),
				float32(h.Y-fwdY*float64(step)*1.3),
				sz, bodyR*0.82, bodyG*0.82, bodyB*0.82, 1.0, 0)
		}

		tailX := h.X - fwdX*4.5
		tailY := h.Y - fwdY*4.5
		buf = append(buf, float32(tailX+perpX*1.1), float32(tailY+perpY*1.1), 1.1, bodyR, bodyG, bodyB, 1.0, 0)
		buf = append(buf, float32(tailX-perpX*1.1), float32(tailY-perpY*1.1), 1.1, bodyR, bodyG, bodyB, 1.0, 0)
		buf = append(buf, float32(tailX), float32(tailY), 1.0, bodyR*0.82, bodyG*0.82, bodyB*0.82, 1.0, 0)

		const bladeR = 3.2
		rotX := math.Cos(h.RotorAngle)
		rotY := math.Sin(h.RotorAngle)
		buf = append(buf, float32(h.X+rotX*bladeR), float32(h.Y+rotY*bladeR), 1.0, rotR, rotG, rotB, 0.95, 0)
		buf = append(buf, float32(h.X+rotX*bladeR*0.5), float32(h.Y+rotY*bladeR*0.5), 1.0, rotR, rotG, rotB, 0.95, 0)
		buf = append(buf, float32(h.X-rotX*bladeR*0.5), float32(h.Y-rotY*bladeR*0.5), 1.0, rotR, rotG, rotB, 0.95, 0)
		buf = append(buf, float32(h.X-rotX*bladeR), float32(h.Y-rotY*bladeR), 1.0, rotR, rotG, rotB, 0.95, 0)

		barR, barG, barB := float32(0.16), float32(0.16), float32(0.16)
		if h.MissileMode {
			barR, barG, barB = 0.30, 0.16, 0.08
		}
		buf = append(buf, float32(h.X+fwdX*3.2), float32(h.Y+fwdY*3.2), 0.8, barR, barG, barB, 1.0, 0)
	}
	for i := range s.StrikeHeliShots {
		b := &s.StrikeHeliShots[i]
		buf = append(buf, float32(b.X), float32(b.Y), 0.72, 1.0, 0.33, 0.18, 1.0, 0)
	}
	for i := range s.StrikeHeliMiss {
		m := &s.StrikeHeliMiss[i]
		buf = append(buf, float32(m.X)+0.2, float32(m.Y)+0.28, 1.35, 0.04, 0.04, 0.04, 0.42, 0)
		buf = append(buf, float32(m.X), float32(m.Y), 1.15, 1.0, 0.52, 0.18, 1.0, 0)
	}

	// Bomb belt markers (pending detonations).
	for i := range s.BeltBombs {
		b := &s.BeltBombs[i]
		buf = append(buf,
			float32(b.X)+0.18, float32(b.Y)+0.28, 1.6,
			0.05, 0.05, 0.05, 0.44, 0,
		)
		buf = append(buf,
			float32(b.X), float32(b.Y), 1.3,
			1.0, 0.72, 0.36, 1.0, 0,
		)
	}

	// Air support planes and dropped bombs.
	for i := range s.StrikePlanes {
		p := &s.StrikePlanes[i]
		fx := math.Cos(p.Heading)
		fy := math.Sin(p.Heading)
		px := -fy
		py := fx
		buf = append(buf, float32(p.X), float32(p.Y)+1.7, 7.2, 0.04, 0.04, 0.04, 0.20, 0) // shadow
		buf = append(buf, float32(p.X), float32(p.Y), 3.1, 0.55, 0.58, 0.62, 1.0, 0)      // body
		buf = append(buf, float32(p.X+fx*3.2), float32(p.Y+fy*3.2), 1.35, 0.72, 0.75, 0.80, 1.0, 0)
		buf = append(buf, float32(p.X+px*2.4), float32(p.Y+py*2.4), 1.0, 0.50, 0.55, 0.60, 1.0, 0)
		buf = append(buf, float32(p.X-px*2.4), float32(p.Y-py*2.4), 1.0, 0.50, 0.55, 0.60, 1.0, 0)
		buf = append(buf, float32(p.X-fx*2.4), float32(p.Y-fy*2.4), 1.0, 0.45, 0.48, 0.52, 1.0, 0)
	}
	for i := range s.StrikePlaneBombs {
		b := &s.StrikePlaneBombs[i]
		buf = append(buf, float32(b.X)+0.12, float32(b.Y)+0.22, 1.25, 0.04, 0.04, 0.04, 0.40, 0)
		buf = append(buf, float32(b.X), float32(b.Y), 1.0, 1.0, 0.62, 0.30, 1.0, 0)
	}

	for i := range s.TimedExploders {
		e := &s.TimedExploders[i]
		tPulse := float32(0.86 + 0.14*math.Sin(e.Timer*8.0))
		switch e.Kind {
		case TimedExploderPig:
			buf = append(buf, float32(e.X), float32(e.Y)+0.4, 3.0, 0.06, 0.06, 0.06, 0.34, 0)
			buf = append(buf, float32(e.X), float32(e.Y), 1.9*tPulse, 1.0, 0.58, 0.72, 1.0, 0)
			buf = append(buf, float32(e.X+0.9), float32(e.Y+0.9), 0.72, 1.0, 0.66, 0.80, 1.0, 0)
			buf = append(buf, float32(e.X-0.9), float32(e.Y+0.9), 0.72, 1.0, 0.66, 0.80, 1.0, 0)
		case TimedExploderCar:
			fx := math.Cos(e.Heading)
			fy := math.Sin(e.Heading)
			buf = append(buf, float32(e.X), float32(e.Y)+0.5, 3.4, 0.06, 0.06, 0.06, 0.38, 0)
			buf = append(buf, float32(e.X), float32(e.Y), 2.2*tPulse, 0.98, 0.46, 0.28, 1.0, 0)
			buf = append(buf, float32(e.X+fx*1.4), float32(e.Y+fy*1.4), 1.0, 1.0, 0.70, 0.34, 1.0, 0)
			buf = append(buf, float32(e.X-fx*1.4), float32(e.Y-fy*1.4), 0.9, 0.85, 0.30, 0.18, 1.0, 0)
		case TimedExploderSnake:
			fx := math.Cos(e.Heading)
			fy := math.Sin(e.Heading)
			buf = append(buf, float32(e.X), float32(e.Y)+0.4, 2.8, 0.05, 0.05, 0.05, 0.32, 0)
			buf = append(buf, float32(e.X), float32(e.Y), 1.6*tPulse, 0.56, 1.0, 0.50, 1.0, 0)
			buf = append(buf, float32(e.X-fx*1.0), float32(e.Y-fy*1.0), 1.2*tPulse, 0.42, 0.88, 0.40, 1.0, 0)
			buf = append(buf, float32(e.X-fx*2.0), float32(e.Y-fy*2.0), 0.9*tPulse, 0.30, 0.68, 0.30, 1.0, 0)
		}
	}

	// Homing missiles.
	for _, m := range s.Missiles {
		buf = append(buf,
			float32(m.X)+0.25, float32(m.Y)+0.35, 1.5,
			0.04, 0.04, 0.04, 0.45, 0,
		)
		buf = append(buf,
			float32(m.X), float32(m.Y), 1.25,
			1.0, 0.58, 0.10, 1.0, 0,
		)
	}

	// Gatling rounds.
	for _, b := range s.GatlingRounds {
		buf = append(buf,
			float32(b.X), float32(b.Y), 0.65,
			1.0, 0.90, 0.35, 1.0, 0,
		)
	}
	return buf
}

// cloneSegments extracts evenly-spaced positions from a CloneSnake path for rendering.
func cloneSegments(c CloneSnake) []PathPoint {
	spacing := 1.5
	if len(c.Path) < 2 {
		return c.Path
	}
	numSegs := int(math.Ceil(c.Length/spacing)) + 1
	out := make([]PathPoint, 0, numSegs)
	dist := 0.0
	out = append(out, c.Path[0])
	for i := 1; i < len(c.Path) && dist < c.Length; i++ {
		prev := c.Path[i-1]
		cur := c.Path[i]
		d := math.Hypot(cur.X-prev.X, cur.Y-prev.Y)
		if d < 0.001 {
			continue
		}
		dist += d
		if dist <= c.Length {
			out = append(out, cur)
		}
	}
	return out
}

// killStreakResult returns the combo label, bonus score, and display color for a
// given kill streak count. Returns empty string when no milestone is reached.
func killStreakResult(streak int) (msg string, bonus int, col RGB) {
	switch streak {
	case 2:
		return "DOUBLE KILL", 100, RGB{R: 255, G: 220, B: 50}
	case 3:
		return "TRIPLE KILL", 300, RGB{R: 255, G: 220, B: 50}
	case 4:
		return "MULTI KILL", 600, RGB{R: 255, G: 120, B: 20}
	case 5:
		return "MEGA KILL", 1000, RGB{R: 255, G: 120, B: 20}
	case 6:
		return "ULTRA KILL", 1500, RGB{R: 255, G: 120, B: 20}
	case 7:
		return "KILLING SPREE", 2000, RGB{R: 255, G: 40, B: 200}
	case 8:
		return "MONSTER KILL", 3000, RGB{R: 255, G: 40, B: 200}
	case 9:
		return "LUDICROUS KILL", 5000, RGB{R: 255, G: 40, B: 200}
	}
	if streak >= 10 {
		return "HOLY SHIT!", 8000, RGB{R: 255, G: 255, B: 255}
	}
	return "", 0, RGB{}
}

// GlowData returns additive glow sprites for spread bombs and vacuum bubbles.
func (s *Snake) GlowData() []float32 {
	buf := make([]float32, 0, 64)

	// Spread bombs: pulsing orange-red dot, size swells as fuse runs out.
	for i := range s.SpreadBombs {
		b := &s.SpreadBombs[i]
		// fuse ranges 0..~1.8; near 0 = about to blow (bigger, brighter).
		t := clampF(1.0-b.Fuse/1.8, 0, 1)
		size := float32(4.0 + 5.0*t)
		intensity := float32(0.15 + 0.25*t)
		buf = append(buf, float32(b.X), float32(b.Y), size,
			intensity, intensity*0.35, 0.0, 1, 0)
	}

	// Vacuum bubbles: no ring indicator — the pixel-suction animation is the only visual.

	// Targeted strike entities.
	for i := range s.StrikeWorms {
		w := &s.StrikeWorms[i]
		buf = append(buf, float32(w.X), float32(w.Y), 2.2,
			0.10, 0.22, 0.09, 1, 0)
	}
	for i := range s.StrikeHelis {
		h := &s.StrikeHelis[i]
		// Red beacon blink.
		if int((h.RotorAngle*0.65))%2 == 0 {
			blinkR, blinkG, blinkB := float32(0.95), float32(0.10), float32(0.08)
			haloR, haloG, haloB := float32(0.32), float32(0.05), float32(0.04)
			if h.MissileMode {
				blinkR, blinkG, blinkB = 1.0, 0.52, 0.12
				haloR, haloG, haloB = 0.36, 0.18, 0.06
			}
			buf = append(buf, float32(h.X), float32(h.Y)-1.2, 1.6, blinkR, blinkG, blinkB, 1, 0)
			buf = append(buf, float32(h.X), float32(h.Y)-1.2, 6.0, haloR, haloG, haloB, 1, 0)
		}
		// Nose muzzle glow while actively attacking.
		if !h.Exiting && h.DeployTimer <= 0 && h.FireTimer < 0.05 {
			muzzleR, muzzleG, muzzleB := float32(0.55), float32(0.28), float32(0.10)
			if h.MissileMode {
				muzzleR, muzzleG, muzzleB = 0.82, 0.43, 0.10
			}
			buf = append(buf,
				float32(h.X+math.Cos(h.Heading)*3.2),
				float32(h.Y+math.Sin(h.Heading)*3.2),
				3.0, muzzleR, muzzleG, muzzleB, 1, 0)
		}
	}
	for i := range s.StrikeHeliShots {
		b := &s.StrikeHeliShots[i]
		buf = append(buf, float32(b.X), float32(b.Y), 1.5,
			0.28, 0.07, 0.03, 1, 0)
	}
	for i := range s.StrikeHeliMiss {
		m := &s.StrikeHeliMiss[i]
		buf = append(buf, float32(m.X), float32(m.Y), 3.0,
			0.30, 0.14, 0.03, 1, 0)
	}

	for i := range s.BeltBombs {
		b := &s.BeltBombs[i]
		t := clampF(1.0-b.Fuse/1.1, 0, 1)
		size := float32(3.4 + 2.6*t)
		buf = append(buf, float32(b.X), float32(b.Y), size,
			0.30, 0.18, 0.06, 1, 0)
	}

	for i := range s.StrikePlanes {
		p := &s.StrikePlanes[i]
		buf = append(buf, float32(p.X), float32(p.Y), 4.2,
			0.12, 0.18, 0.26, 1, 0)
	}
	for i := range s.StrikePlaneBombs {
		b := &s.StrikePlaneBombs[i]
		t := clampF(1.0-b.Fuse/0.7, 0, 1)
		buf = append(buf, float32(b.X), float32(b.Y), float32(2.6+2.0*t),
			0.34, 0.20, 0.08, 1, 0)
	}

	for i := range s.TimedExploders {
		e := &s.TimedExploders[i]
		t := clampF(1.0-e.Timer/7.0, 0, 1)
		size := float32(2.4 + 2.2*t)
		switch e.Kind {
		case TimedExploderPig:
			buf = append(buf, float32(e.X), float32(e.Y), size, 0.34, 0.12, 0.22, 1, 0)
		case TimedExploderCar:
			buf = append(buf, float32(e.X), float32(e.Y), size, 0.34, 0.16, 0.08, 1, 0)
		case TimedExploderSnake:
			buf = append(buf, float32(e.X), float32(e.Y), size, 0.14, 0.30, 0.10, 1, 0)
		}
	}

	// Homing missiles.
	for i := range s.Missiles {
		m := &s.Missiles[i]
		buf = append(buf, float32(m.X), float32(m.Y), 3.2,
			0.22, 0.11, 0.02, 1, 0)
	}

	// Gatling rounds.
	for i := range s.GatlingRounds {
		b := &s.GatlingRounds[i]
		buf = append(buf, float32(b.X), float32(b.Y), 1.5,
			0.20, 0.16, 0.04, 1, 0)
	}

	return buf
}

// WallAvoidAngle returns a steering angle that routes around obstacles.
// It looks ahead along targetAngle and, if a wall is found within lookDist,
// scans perpendicular to find which side has the nearer open gap and steers
// toward that building edge. This is called every frame before Steer() so
// the snake starts turning early enough to clear the wall.
//
// lookDist should be ~speed/turnRate pixels so the snake has room to turn.
func WallAvoidAngle(hx, hy, targetAngle float64, world *World) float64 {
	const lookDist = 12.0 // pixels ahead to scan
	const scanStep = 1.0  // resolution of ray march

	// Ray-march along target heading. Find first wall pixel.
	hitDist := -1.0
	var hitX, hitY float64
	for d := scanStep; d <= lookDist; d += scanStep {
		px := hx + math.Cos(targetAngle)*d
		py := hy + math.Sin(targetAngle)*d
		if int(math.Round(px)) < 0 || int(math.Round(py)) < 0 ||
			int(math.Round(px)) >= WorldWidth || int(math.Round(py)) >= WorldHeight {
			break
		}
		if world.HeightAt(int(math.Round(px)), int(math.Round(py))) > 0 {
			hitDist = d
			hitX, hitY = px, py
			break
		}
	}

	if hitDist < 0 {
		return targetAngle // clear path, no adjustment needed
	}

	// Wall found. Scan left and right perpendicularly from the hit point
	// to find the nearer open gap (building edge).
	const edgeScan = 25.0
	leftAngle := targetAngle - math.Pi/2
	rightAngle := targetAngle + math.Pi/2

	leftGap := -1.0
	for e := scanStep; e <= edgeScan; e += scanStep {
		lx := hitX + math.Cos(leftAngle)*e
		ly := hitY + math.Sin(leftAngle)*e
		lxi, lyi := int(math.Round(lx)), int(math.Round(ly))
		if lxi < 0 || lyi < 0 || lxi >= WorldWidth || lyi >= WorldHeight {
			break
		}
		if world.HeightAt(lxi, lyi) == 0 {
			leftGap = e
			break
		}
	}

	rightGap := -1.0
	for e := scanStep; e <= edgeScan; e += scanStep {
		rx := hitX + math.Cos(rightAngle)*e
		ry := hitY + math.Sin(rightAngle)*e
		rxi, ryi := int(math.Round(rx)), int(math.Round(ry))
		if rxi < 0 || ryi < 0 || rxi >= WorldWidth || ryi >= WorldHeight {
			break
		}
		if world.HeightAt(rxi, ryi) == 0 {
			rightGap = e
			break
		}
	}

	// Steer toward the nearer gap (open edge of the building).
	// If one side hits the world boundary or has no gap, prefer the other.
	useLeft := false
	switch {
	case leftGap >= 0 && rightGap >= 0:
		useLeft = leftGap <= rightGap
	case leftGap >= 0:
		useLeft = true
	case rightGap >= 0:
		useLeft = false
	default:
		return targetAngle // no gap found on either side, keep heading
	}

	// Aim at the open edge point so the snake arcs smoothly around the corner.
	var edgeX, edgeY float64
	if useLeft {
		edgeX = hitX + math.Cos(leftAngle)*leftGap
		edgeY = hitY + math.Sin(leftAngle)*leftGap
	} else {
		edgeX = hitX + math.Cos(rightAngle)*rightGap
		edgeY = hitY + math.Sin(rightAngle)*rightGap
	}
	return math.Atan2(edgeY-hy, edgeX-hx)
}

// findClearDir finds the closest direction to fromAngle (fanning left/right
// alternately) where the next clearNeed pixels are all walkable. Returns the
// angle and true if found; false if completely boxed in.
//
// Starting at perpendicular (±90°) from fromAngle ensures the snake slides
// along the wall surface rather than oscillating into it.
func (s *Snake) findClearDir(x, y, fromAngle, step float64, world *World) (float64, bool) {
	const clearNeed = 8.0 // pixels of clear road required ahead

	for i := 0; i < 64; i++ {
		sign := 1.0
		if i%2 == 1 {
			sign = -1.0
		}
		// Fan starting at 90° perpendicular, widening by 360°/64 per step.
		offset := math.Pi/2 + float64(i/2)*(math.Pi/32.0)
		a := fromAngle + sign*offset

		// First step must land on a walkable pixel.
		ex := x + math.Cos(a)*step
		ey := y + math.Sin(a)*step
		exi, eyi := int(math.Round(ex)), int(math.Round(ey))
		if exi < 0 || eyi < 0 || exi >= WorldWidth || eyi >= WorldHeight || world.HeightAt(exi, eyi) > 0 {
			continue
		}

		// Then check that the next clearNeed pixels ahead are also clear.
		allClear := true
		for d := step; d <= clearNeed; d += 1.0 {
			px := x + math.Cos(a)*d
			py := y + math.Sin(a)*d
			pxi, pyi := int(math.Round(px)), int(math.Round(py))
			if pxi < 0 || pyi < 0 || pxi >= WorldWidth || pyi >= WorldHeight || world.HeightAt(pxi, pyi) > 0 {
				allClear = false
				break
			}
		}
		if allClear {
			return a, true
		}
	}
	return fromAngle, false
}

// forceUnstuck relocates the snake head to the nearest walkable tile and
// rebuilds the path so the body does not remain embedded in obstacles.
func (s *Snake) forceUnstuck(hx, hy float64, world *World) (float64, float64, bool) {
	return s.forceUnstuckWithRadius(hx, hy, world, 64)
}

func (s *Snake) forceUnstuckWithRadius(hx, hy float64, world *World, maxRadius int) (float64, float64, bool) {
	if world == nil {
		return hx, hy, false
	}
	ix := clamp(int(math.Round(hx)), 0, WorldWidth-1)
	iy := clamp(int(math.Round(hy)), 0, WorldHeight-1)

	// If already on walkable ground, keep position and only refresh path.
	if world.HeightAt(ix, iy) == 0 {
		s.rebuildPathFromHead(hx, hy)
		return hx, hy, true
	}

	// Search outward in a square ring to find the closest walkable tile.
	for r := 1; r <= maxRadius; r++ {
		minX := clamp(ix-r, 0, WorldWidth-1)
		maxX := clamp(ix+r, 0, WorldWidth-1)
		minY := clamp(iy-r, 0, WorldHeight-1)
		maxY := clamp(iy+r, 0, WorldHeight-1)

		for x := minX; x <= maxX; x++ {
			if world.HeightAt(x, minY) == 0 {
				nx := float64(x) + 0.5
				ny := float64(minY) + 0.5
				s.rebuildPathFromHead(nx, ny)
				return nx, ny, true
			}
			if world.HeightAt(x, maxY) == 0 {
				nx := float64(x) + 0.5
				ny := float64(maxY) + 0.5
				s.rebuildPathFromHead(nx, ny)
				return nx, ny, true
			}
		}
		for y := minY + 1; y <= maxY-1; y++ {
			if world.HeightAt(minX, y) == 0 {
				nx := float64(minX) + 0.5
				ny := float64(y) + 0.5
				s.rebuildPathFromHead(nx, ny)
				return nx, ny, true
			}
			if world.HeightAt(maxX, y) == 0 {
				nx := float64(maxX) + 0.5
				ny := float64(y) + 0.5
				s.rebuildPathFromHead(nx, ny)
				return nx, ny, true
			}
		}
	}
	return hx, hy, false
}

func (s *Snake) rebuildPathFromHead(hx, hy float64) {
	needed := int(math.Ceil(s.Length/1.5)) + 2
	if needed < 1 {
		needed = 1
	}
	maxPts := int(math.Ceil(SnakeMaxLength/1.5)) + 8
	if needed > maxPts {
		needed = maxPts
	}
	if len(s.Path) < needed {
		s.Path = append(s.Path, make([]PathPoint, needed-len(s.Path))...)
	}
	s.Path = s.Path[:needed]
	for i := 0; i < needed; i++ {
		back := float64(i) * 1.5
		s.Path[i] = PathPoint{
			X: hx - math.Cos(s.Heading)*back,
			Y: hy - math.Sin(s.Heading)*back,
		}
	}
	s.PrevX, s.PrevY = hx, hy
}
