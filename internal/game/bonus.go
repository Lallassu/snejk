package game

import (
	"fmt"
	"math"
)

type BonusKind int

const (
	BonusSpeed BonusKind = iota
	BonusFire
	BonusBash              // smash through buildings
	BonusSurge             // suck peds/cops toward head and crush near core
	BonusTeleport          // rapid-fire teleport to 3-8 peds
	BonusSwarm             // body splits into hunting segments
	BonusAI                // snake auto-hunts 3-10 peds at high speed
	BonusNuke              // massive explosion + shockwave kills everything nearby
	BonusSpread            // drops timed bombs along the path
	BonusClone             // mirror clones of the snake at other world positions
	BonusVacuum            // expanding suction bubble draws in entities and pixels
	BonusHealth            // restores 1–5 HP
	BonusBerserk           // giant snake with AI, bash, and rampage
	BonusFlamethrower      // breathe fire in facing direction
	BonusMissile           // launch homing missiles at living targets
	BonusGatling           // mouth gatling: straight rounds that shred targets
	BonusBombSwarm         // ghost segments dive random points, explode, then return
	BonusTargetNuke        // pause world and click a strike point
	BonusTargetWorms       // click to deploy short-lived ped-hunting worms
	BonusTargetGunship     // click to deploy short-lived helicopter support strike
	BonusTargetHeliMissile // click to deploy helicopter support with missiles
	BonusTargetBombBelt    // click 3-5 positions for carpet bombing strikes
	BonusTargetAirSupport  // click 3-5 positions to call plane bomb runs
	BonusTargetPigs        // place roaming exploding pigs (2-5)
	BonusTargetCars        // place roaming exploding cars (2-5)
	BonusTargetSnakes      // place roaming exploding snakes (2-5)

	BonusKindCount // must stay last
)

type BonusBox struct {
	X, Y  float64
	Kind  BonusKind
	Alive bool
	Timer float64 // flash animation
}

type BonusSystem struct {
	Boxes      []BonusBox
	seed       uint64
	spawnSeq   uint64
	lastKind   int
	SpawnTimer float64
	maxBoxes   int
}

func NewBonusSystem(seed uint64, maxBoxes int) *BonusSystem {
	return &BonusSystem{
		seed:       seed,
		lastKind:   -1,
		maxBoxes:   maxBoxes,
		SpawnTimer: 8.0,
	}
}

func clampBonusSpawn(x, y float64) (float64, float64) {
	const margin = 2.0
	maxX := float64(WorldWidth - 3)
	maxY := float64(WorldHeight - 3)
	if maxX < margin {
		maxX = margin
	}
	if maxY < margin {
		maxY = margin
	}
	return clampF(x, margin, maxX), clampF(y, margin, maxY)
}

func (bs *BonusSystem) nextSpawnRand(salt uint64) *Rand {
	bs.spawnSeq++
	// Use splitmix-style avalanche so consecutive spawn seeds do not correlate.
	z := bs.seed ^ salt ^ bs.spawnSeq*0x9E3779B185EBCA87
	z += 0x9E3779B97F4A7C15
	z = (z ^ (z >> 30)) * 0xBF58476D1CE4E5B9
	z = (z ^ (z >> 27)) * 0x94D049BB133111EB
	z ^= z >> 31
	return NewRand(z)
}

func (bs *BonusSystem) randomSpawnPos(r *Rand) (float64, float64) {
	var x, y float64
	locType := r.Intn(4)
	switch locType {
	case 0:
		// Intersection.
		bx := r.Intn(WorldWidth / Pattern)
		by := r.Intn(WorldHeight / Pattern)
		x = float64(bx*Pattern+RoadWidth/2) + r.RangeF(-2, 2)
		y = float64(by*Pattern+RoadWidth/2) + r.RangeF(-2, 2)
	case 1:
		// Horizontal road segment.
		x = r.RangeF(0, float64(WorldWidth-1))
		by := r.Intn(WorldHeight / Pattern)
		y = float64(by*Pattern+RoadWidth/2) + r.RangeF(-1, 1)
	case 2:
		// Vertical road segment.
		bx := r.Intn(WorldWidth / Pattern)
		x = float64(bx*Pattern+RoadWidth/2) + r.RangeF(-1, 1)
		y = r.RangeF(0, float64(WorldHeight-1))
	default:
		// Completely random position (can be on sidewalk/grass).
		x = float64(r.Range(5, WorldWidth-5))
		y = float64(r.Range(5, WorldHeight-5))
	}
	return clampBonusSpawn(x, y)
}

func (bs *BonusSystem) hasNearbyAliveBox(x, y float64, minDist float64) bool {
	minD2 := minDist * minDist
	for i := range bs.Boxes {
		b := &bs.Boxes[i]
		if !b.Alive {
			continue
		}
		dx := b.X - x
		dy := b.Y - y
		if dx*dx+dy*dy < minD2 {
			return true
		}
	}
	return false
}

func (bs *BonusSystem) pickSpawnPos(r *Rand) (float64, float64) {
	x, y := bs.randomSpawnPos(r)
	for tries := 0; tries < 14; tries++ {
		if !bs.hasNearbyAliveBox(x, y, 8.0) {
			return x, y
		}
		x, y = bs.randomSpawnPos(r)
	}
	return x, y
}

func (bs *BonusSystem) pickBonusKind(r *Rand, snakeHP float64) BonusKind {
	kind := BonusKind(r.Intn(int(BonusKindCount)))

	// Bias towards health bonuses when snake is hurt.
	if snakeHP < 0.5 && r.Intn(100) < 40 {
		kind = BonusHealth
	}

	// Avoid obvious repeated same-kind streaks.
	if int(kind) == bs.lastKind && BonusKindCount > 1 {
		off := 1 + r.Intn(int(BonusKindCount)-1)
		kind = BonusKind((int(kind) + off) % int(BonusKindCount))
	}
	bs.lastKind = int(kind)
	return kind
}

// SpawnRandom places count bonus boxes at random road positions.
func (bs *BonusSystem) SpawnRandom(count int) {
	for i := 0; i < count; i++ {
		r := bs.nextSpawnRand(uint64(i+1) * 0xB0105)
		x, y := bs.pickSpawnPos(r)
		kind := bs.pickBonusKind(r, 1.0)
		bs.Boxes = append(bs.Boxes, BonusBox{
			X: x, Y: y,
			Kind:  kind,
			Alive: true,
			Timer: r.RangeF(0, 1),
		})
	}
}

// Update advances animation timers and respawn logic.
// pedsAlive: current number of alive pedestrians (speeds up spawns when low).
// snakeHP: snake health fraction 0-1 (lower health = faster spawns).
func (bs *BonusSystem) Update(dt float64, pedsAlive int, snakeHP float64) {
	for i := range bs.Boxes {
		bs.Boxes[i].Timer += dt
	}

	alive := 0
	for i := range bs.Boxes {
		if bs.Boxes[i].Alive {
			alive++
		}
	}

	// Raise cap when few peds remain or snake is low on health.
	effectiveMax := bs.maxBoxes
	if pedsAlive < 8 {
		effectiveMax = bs.maxBoxes + 3
	}
	if snakeHP < 0.4 {
		effectiveMax = bs.maxBoxes + 4
	}

	if alive < effectiveMax {
		bs.SpawnTimer -= dt

		// Speed up spawn timer when snake health is low.
		if snakeHP < 0.3 {
			bs.SpawnTimer -= dt * 2.0 // 3x faster spawns at critical health
		} else if snakeHP < 0.5 {
			bs.SpawnTimer -= dt * 1.0 // 2x faster spawns at low health
		} else if snakeHP < 0.7 {
			bs.SpawnTimer -= dt * 0.5 // 1.5x faster spawns
		}

		if bs.SpawnTimer <= 0 {
			salt := uint64(len(bs.Boxes)*0xB4D+alive*0x77F+pedsAlive*0x55D) ^ uint64(clamp(int(snakeHP*1000), 0, 1000))
			r := bs.nextSpawnRand(salt)

			// Base spawn timers - more random variation.
			switch {
			case pedsAlive < 4:
				bs.SpawnTimer = 0.5 + r.RangeF(0, 1.2)
			case pedsAlive < 10:
				bs.SpawnTimer = 1.5 + r.RangeF(0, 3.0)
			default:
				bs.SpawnTimer = 4.0 + r.RangeF(0, 6.0)
			}

			x, y := bs.pickSpawnPos(r)
			kind := bs.pickBonusKind(r, snakeHP)

			spawned := false
			for i := range bs.Boxes {
				if !bs.Boxes[i].Alive {
					bs.Boxes[i] = BonusBox{X: x, Y: y, Kind: kind, Alive: true, Timer: 0}
					spawned = true
					break
				}
			}
			if !spawned {
				bs.Boxes = append(bs.Boxes, BonusBox{X: x, Y: y, Kind: kind, Alive: true})
			}
		}
	}
}

// Collect activates a bonus effect on the snake.
// cx, cy: position of the collector (snake head or swarm ghost) — used for area effects.
// Duration and strength are randomized per box; 28% chance of a secondary combo effect.
func (bs *BonusSystem) Collect(idx int, s *Snake, world *World, peds *PedestrianSystem, traffic *TrafficSystem, particles *ParticleSystem, cam *Camera, cops *CopSystem, mil *MilitarySystem) {
	cx, cy := s.Head()
	bs.collectAt(idx, s, cx, cy, world, peds, traffic, particles, cam, cops, mil)
}

// CollectAt activates a bonus at a specific world position (used by swarm ghosts).
func (bs *BonusSystem) CollectAt(idx int, s *Snake, cx, cy float64, world *World, peds *PedestrianSystem, traffic *TrafficSystem, particles *ParticleSystem, cam *Camera, cops *CopSystem, mil *MilitarySystem) {
	bs.collectAt(idx, s, cx, cy, world, peds, traffic, particles, cam, cops, mil)
}

func (bs *BonusSystem) collectAt(idx int, s *Snake, hx, hy float64, world *World, peds *PedestrianSystem, traffic *TrafficSystem, particles *ParticleSystem, cam *Camera, cops *CopSystem, mil *MilitarySystem) {
	b := &bs.Boxes[idx]
	if !b.Alive {
		return
	}
	b.Alive = false
	PlaySound(SoundBonus)

	// Per-box RNG: varies by position and seed so each box has unique rolls.
	r := NewRand(uint64(b.X*71+b.Y*43) ^ bs.seed ^ 0xC011EC7)

	switch b.Kind {
	case BonusSpeed:
		s.SpeedBoost = 2.0 + r.RangeF(0, 4.0) // 2–6s
		s.SpeedMult = 1.8 + r.RangeF(0, 0.5)  // 1.8–2.3×
		s.PowerupMsg = "SPEED BOOST!"
		s.PowerupTimer = 2.5
		s.PowerupCol = RGB{R: 255, G: 255, B: 25}

	case BonusBash:
		s.BashTimer = 3.0 + r.RangeF(0, 5.0) // 3–8s
		s.PowerupMsg = "BASH WALLS!"
		s.PowerupTimer = 2.5
		s.PowerupCol = RGB{R: 25, G: 230, B: 255}

	case BonusSurge:
		s.SurgeTimer = 2.0 + r.RangeF(0, 5.0) // 2–7s
		s.PowerupMsg = "SURGE!"
		s.PowerupTimer = 2.5
		s.PowerupCol = RGB{R: 180, G: 80, B: 255}
		if particles != nil {
			pr := NewRand(uint64(hx*53+hy*29) ^ 0x5A37)
			for range 24 {
				ang := pr.RangeF(0, 2*math.Pi)
				dist := pr.RangeF(10, 45)
				spd := pr.RangeF(15, 50)
				particles.Add(Particle{
					X: hx + math.Cos(ang)*dist, Y: hy + math.Sin(ang)*dist,
					VX: math.Cos(ang+math.Pi) * spd, VY: math.Sin(ang+math.Pi) * spd,
					Size: 0.5, MaxLife: pr.RangeF(0.4, 0.9),
					Col: RGB{R: 180, G: 80, B: 255}, Kind: ParticleGlow,
				})
			}
		}

	case BonusTeleport:
		count := 3 + r.Range(0, 5) // 3–7 targets
		s.PowerupMsg = "TELEPORT!"
		s.PowerupTimer = 2.5
		s.PowerupCol = RGB{R: 200, G: 200, B: 255}
		s.TeleportQueue = s.TeleportQueue[:0]
		added := 0
		for i := range peds.P {
			if added >= count {
				break
			}
			p := &peds.P[i]
			if !p.Alive || p.Infection == StateSymptomatic {
				continue
			}
			s.TeleportQueue = append(s.TeleportQueue, PathPoint{X: p.X, Y: p.Y})
			added++
		}
		s.TeleportTimer = 0

	case BonusAI:
		s.AITimer = 3.0 + r.RangeF(0, 7.0)  // 3–10s
		s.AITargetsLeft = 3 + r.Range(0, 7) // 3–9 targets
		s.PowerupMsg = "AI MODE!"
		s.PowerupTimer = 2.5
		s.PowerupCol = RGB{R: 255, G: 200, B: 50}

	case BonusSwarm:
		// Can't re-activate swarm while already in swarm mode — that would
		// clear s.Ghosts mid-loop and cause an out-of-bounds panic.
		if len(s.Ghosts) > 0 {
			break
		}
		s.PowerupMsg = "SWARM!"
		s.PowerupTimer = 2.5
		s.PowerupCol = RGB{R: 100, G: 255, B: 100}

		segs := s.Segments()
		numSwarms := clamp(int(s.Length/4.0), 2, 8)
		if numSwarms > len(segs) {
			numSwarms = len(segs)
		}
		segLen := s.Length / float64(numSwarms)
		step := max(1, len(segs)/numSwarms)

		s.GhostTimer = 5.0 + r.RangeF(0, 5.0) // 5–10s hunt time
		s.GhostBombMode = false
		s.Ghosts = s.Ghosts[:0]
		for i := 0; i < numSwarms; i++ {
			si := i * step
			if si >= len(segs) {
				break
			}
			pt := segs[si]
			heading := s.Heading
			if si+1 < len(segs) {
				next := segs[si+1]
				heading = math.Atan2(pt.Y-next.Y, pt.X-next.X)
			}
			s.Ghosts = append(s.Ghosts, GhostSnake{
				X: pt.X, Y: pt.Y,
				Heading: heading,
				TargetX: pt.X, TargetY: pt.Y,
				SegLen: segLen,
			})
		}
		s.Length = 3.0

		if particles != nil {
			for range 30 {
				ang := r.RangeF(0, 2*math.Pi)
				spd := r.RangeF(20, 60)
				particles.Add(Particle{
					X: hx, Y: hy,
					VX: math.Cos(ang) * spd, VY: math.Sin(ang) * spd,
					Size: 0.4, MaxLife: r.RangeF(0.2, 0.5),
					Col: RGB{R: 150, G: 255, B: 150}, Kind: ParticleGlow,
				})
			}
		}

	case BonusBombSwarm:
		// Dive-bomb swarm: segments split, fly to random points, explode, then return.
		if len(s.Ghosts) > 0 {
			break
		}
		s.PowerupMsg = "BOMB SWARM!"
		s.PowerupTimer = 2.5
		s.PowerupCol = RGB{R: 255, G: 155, B: 70}
		s.GhostBombMode = true
		s.GhostTimer = 0

		segs := s.Segments()
		numSwarms := clamp(int(s.Length/4.0), 2, 8)
		if numSwarms > len(segs) {
			numSwarms = len(segs)
		}
		segLen := s.Length / float64(numSwarms)
		step := max(1, len(segs)/numSwarms)

		s.Ghosts = s.Ghosts[:0]
		for i := 0; i < numSwarms; i++ {
			si := i * step
			if si >= len(segs) {
				break
			}
			pt := segs[si]
			heading := s.Heading
			if si+1 < len(segs) {
				next := segs[si+1]
				heading = math.Atan2(pt.Y-next.Y, pt.X-next.X)
			}

			tx, ty := pt.X, pt.Y
			for tries := 0; tries < 60; tries++ {
				rx := r.Range(2, WorldWidth-3)
				ry := r.Range(2, WorldHeight-3)
				if world.HeightAt(rx, ry) == 0 && math.Hypot(float64(rx)-hx, float64(ry)-hy) > 12 {
					tx = float64(rx) + 0.5
					ty = float64(ry) + 0.5
					break
				}
			}

			s.Ghosts = append(s.Ghosts, GhostSnake{
				X: pt.X, Y: pt.Y,
				Heading: heading,
				TargetX: tx, TargetY: ty,
				SegLen: segLen,
			})
		}
		s.Length = 3.0

		if particles != nil {
			for range 36 {
				ang := r.RangeF(0, 2*math.Pi)
				spd := r.RangeF(20, 70)
				particles.Add(Particle{
					X: hx, Y: hy,
					VX: math.Cos(ang) * spd, VY: math.Sin(ang) * spd,
					Size: 0.45, MaxLife: r.RangeF(0.2, 0.6),
					Col: RGB{R: 255, G: 170, B: 70}, Kind: ParticleGlow,
				})
			}
		}

	case BonusFire:
		s.PowerupMsg = "FIRE BOLTS!"
		s.PowerupTimer = 2.5
		s.PowerupCol = RGB{R: 255, G: 80, B: 15}
		s.FireRingTimer = 4.0 + r.RangeF(0, 4.0) // 4–8s; bolts orbit snake head
		s.FireBoltAngle = 0

		// Initial burst flash.
		for range 30 {
			ang := r.RangeF(0, 2*math.Pi)
			spd := r.RangeF(15, 50)
			particles.Add(Particle{
				X: hx, Y: hy,
				VX: math.Cos(ang) * spd, VY: math.Sin(ang) * spd,
				Z: r.RangeF(0, 5), VZ: r.RangeF(20, 60),
				Size: 0.5 + r.RangeF(0, 0.5), MaxLife: r.RangeF(0.2, 0.6),
				Col: Palette.FireHot, Kind: ParticleFire,
			})
		}

	case BonusNuke:
		s.PowerupMsg = "NUKE!"
		s.PowerupTimer = 2.5
		s.PowerupCol = RGB{R: 255, G: 255, B: 200}
		wx := int(math.Round(hx))
		wy := int(math.Round(hy))

		// Central mega-explosion (radius 30 carves a huge crater, big shockwave).
		ExplodeAt(wx, wy, 30, world, particles, peds, traffic, cam, cops, mil)
		SpawnNukeAftermath(wx, wy, world, particles, 1.25)

		// Ring of 6 satellite explosions at evenly spaced angles.
		const ringDist = 18.0
		for i := range 6 {
			ang := float64(i) * math.Pi / 3.0
			sx := wx + int(math.Cos(ang)*ringDist)
			sy := wy + int(math.Sin(ang)*ringDist)
			ExplodeAt(sx, sy, 10, world, particles, peds, traffic, cam, cops, mil)
		}

		// Nuking the city = instant max heat.
		s.WantedLevel = WantedMax

		// Burn trees across a massive radius.
		const nukeRadius = 50.0
		for range 30 {
			ang := r.RangeF(0, 2*math.Pi)
			dist := r.RangeF(0, nukeRadius)
			bx := wx + int(math.Cos(ang)*dist)
			by := wy + int(math.Sin(ang)*dist)
			world.StartTreeBurn(bx, by)
		}

		// Kill all remaining peds in the extended nuke radius with massive blood.
		for i := range peds.P {
			p := &peds.P[i]
			if !p.Alive {
				continue
			}
			if math.Hypot(p.X-hx, p.Y-hy) >= nukeRadius {
				continue
			}
			p.Alive = false
			s.Score += 100
			outAng := math.Atan2(p.Y-hy, p.X-hx)
			particles.SpawnBlood(p.X, p.Y, math.Cos(outAng), math.Sin(outAng), 22, 1.0)
			particles.SpawnBlood(p.X+0.5, p.Y+0.5, math.Cos(outAng+math.Pi/2), math.Sin(outAng+math.Pi/2), 12, 0.7)
		}

		// Kill remaining cars in extended radius.
		for i := range traffic.Cars {
			c := &traffic.Cars[i]
			if !c.Alive {
				continue
			}
			if math.Hypot(c.X-hx, c.Y-hy) < nukeRadius {
				c.Alive = false
				SpawnExplosionWithShockwave(int(c.X), int(c.Y), world.ColorAt(int(c.X), int(c.Y)), 0.7, 0, world, particles)
			}
		}

		// Snake rides the shockwave — speed boost.
		s.SpeedBoost = max(s.SpeedBoost, 2.0+r.RangeF(0, 2.0))
		s.SpeedMult = max(s.SpeedMult, 1.8)

	case BonusTargetNuke:
		s.beginTargetAbility(BonusTargetNuke, 5.0)
		s.PowerupMsg = "CLICK TO NUKE 5.0s"
		s.PowerupTimer = 5.0
		s.PowerupCol = RGB{R: 255, G: 245, B: 170}
		if cam != nil {
			cam.AddShake(0.45, 0.2)
		}

	case BonusTargetWorms:
		s.beginTargetAbility(BonusTargetWorms, 5.0)
		s.PowerupMsg = "CLICK TO DEPLOY WORMS 5.0s"
		s.PowerupTimer = 5.0
		s.PowerupCol = RGB{R: 120, G: 255, B: 140}
		if cam != nil {
			cam.AddShake(0.35, 0.18)
		}

	case BonusTargetGunship:
		s.beginTargetAbility(BonusTargetGunship, 5.0)
		s.PowerupMsg = "CLICK FOR HELICOPTER\nSUPPORT 5.0s"
		s.PowerupTimer = 5.0
		s.PowerupCol = RGB{R: 255, G: 85, B: 70}
		if cam != nil {
			cam.AddShake(0.45, 0.2)
		}

	case BonusTargetHeliMissile:
		s.beginTargetAbility(BonusTargetHeliMissile, 5.0)
		s.PowerupMsg = "CLICK FOR MISSILE HELI\nSUPPORT 5.0s"
		s.PowerupTimer = 5.0
		s.PowerupCol = RGB{R: 255, G: 125, B: 52}
		if cam != nil {
			cam.AddShake(0.45, 0.2)
		}

	case BonusTargetBombBelt:
		s.beginTargetAbility(BonusTargetBombBelt, 8.0)
		s.PowerupMsg = "MARK CARPET BOMB 0/5\n8.0s"
		s.PowerupTimer = 8.0
		s.PowerupCol = RGB{R: 255, G: 185, B: 90}
		if cam != nil {
			cam.AddShake(0.40, 0.18)
		}

	case BonusTargetAirSupport:
		s.beginTargetAbility(BonusTargetAirSupport, 8.0)
		s.PowerupMsg = "MARK AIR SUPPORT 0/5\n8.0s"
		s.PowerupTimer = 8.0
		s.PowerupCol = RGB{R: 145, G: 200, B: 255}
		if cam != nil {
			cam.AddShake(0.42, 0.18)
		}

	case BonusTargetPigs:
		s.beginTargetAbility(BonusTargetPigs, 10.0)
		s.PowerupMsg = fmt.Sprintf("PLACE EXPLODING PIGS 0/%d\n10.0s", max(1, s.TargetMaxClicks))
		s.PowerupTimer = 10.0
		s.PowerupCol = RGB{R: 255, G: 130, B: 175}
		if cam != nil {
			cam.AddShake(0.35, 0.16)
		}

	case BonusTargetCars:
		s.beginTargetAbility(BonusTargetCars, 10.0)
		s.PowerupMsg = fmt.Sprintf("PLACE EXPLODING R/C CARS 0/%d\n10.0s", max(1, s.TargetMaxClicks))
		s.PowerupTimer = 10.0
		s.PowerupCol = RGB{R: 255, G: 150, B: 105}
		if cam != nil {
			cam.AddShake(0.35, 0.16)
		}

	case BonusTargetSnakes:
		s.beginTargetAbility(BonusTargetSnakes, 10.0)
		s.PowerupMsg = fmt.Sprintf("PLACE SNAKE BOMBERS 0/%d\n10.0s", max(1, s.TargetMaxClicks))
		s.PowerupTimer = 10.0
		s.PowerupCol = RGB{R: 140, G: 255, B: 130}
		if cam != nil {
			cam.AddShake(0.35, 0.16)
		}

	case BonusSpread:
		s.SpreadTimer = 5.0 + r.RangeF(0, 4.0) // 5–9s
		s.SpreadBombs = s.SpreadBombs[:0]
		s.SpreadDropTimer = 0
		s.PowerupMsg = "SPREAD BOMBS!"
		s.PowerupTimer = 2.5
		s.PowerupCol = RGB{R: 255, G: 115, B: 12}

	case BonusClone:
		s.CloneTimer = 6.0 + r.RangeF(0, 5.0) // 6–11s
		s.Clones = s.Clones[:0]
		numClones := 2 + r.Range(0, 2) // 2–3 clones
		for range numClones {
			// Spawn at a random offset from the collector position.
			for tries := range 40 {
				_ = tries
				cx := clampF(hx+r.RangeF(-float64(WorldWidth/3), float64(WorldWidth/3)), 0, float64(WorldWidth-1))
				cy := clampF(hy+r.RangeF(-float64(WorldHeight/3), float64(WorldHeight/3)), 0, float64(WorldHeight-1))
				if world.HeightAt(int(math.Round(cx)), int(math.Round(cy))) == 0 {
					path := make([]PathPoint, 1, 256)
					path[0] = PathPoint{X: cx, Y: cy}
					s.Clones = append(s.Clones, CloneSnake{
						X: cx, Y: cy,
						Heading: s.Heading,
						Path:    path,
						Length:  s.Length * 0.6,
					})
					break
				}
			}
		}
		s.PowerupMsg = "CLONE!"
		s.PowerupTimer = 2.5
		s.PowerupCol = RGB{R: 90, G: 215, B: 255}
		if particles != nil {
			for range 20 {
				ang := r.RangeF(0, 2*math.Pi)
				spd := r.RangeF(15, 45)
				particles.Add(Particle{
					X: b.X, Y: b.Y,
					VX: math.Cos(ang) * spd, VY: math.Sin(ang) * spd,
					Size: 0.4, MaxLife: r.RangeF(0.2, 0.5),
					Col: RGB{R: 90, G: 215, B: 255}, Kind: ParticleGlow,
				})
			}
		}

	case BonusVacuum:
		s.VacuumBubbles = append(s.VacuumBubbles, VacuumBubble{
			X: hx, Y: hy,
			MaxTime: 3.5 + r.RangeF(0, 2.0), // 3.5–5.5s
		})
		s.PowerupMsg = "VACUUM!"
		s.PowerupTimer = 2.5
		s.PowerupCol = RGB{R: 155, G: 225, B: 255}

	case BonusHealth:
		s.HP.Heal(1.0 + r.RangeF(0, 4.0)) // +1 to +5 HP up to max
		s.PowerupMsg = "HEALTH BOOST!"
		s.PowerupTimer = 2.5
		s.PowerupCol = RGB{R: 80, G: 255, B: 120}

	case BonusBerserk:
		// Giant snake rampage: 3x size, bash, AI mode, 3-5 seconds.
		s.BerserkTimer = 3.0 + r.RangeF(0, 2.0)
		s.SizeMult = 3.0
		s.BashTimer = s.BerserkTimer + 0.5 // bash lasts slightly longer
		s.AITimer = s.BerserkTimer
		s.AITargetsLeft = 50 // hunt as many as possible
		s.PowerupMsg = "BERSERK!"
		s.PowerupTimer = 2.5
		s.PowerupCol = RGB{R: 255, G: 50, B: 50}
		// Camera shake for dramatic effect.
		if cam != nil {
			cam.AddShake(1.0, 0.5)
		}
		// Spawn rage particles.
		if particles != nil {
			for range 30 {
				ang := r.RangeF(0, 2*math.Pi)
				spd := r.RangeF(20, 60)
				particles.Add(Particle{
					X: hx, Y: hy,
					VX: math.Cos(ang) * spd, VY: math.Sin(ang) * spd,
					Size: 0.8, MaxLife: r.RangeF(0.3, 0.6),
					Col: RGB{R: 255, G: 60, B: 30}, Kind: ParticleFire,
				})
			}
		}

	case BonusFlamethrower:
		// Flamethrower: breathe fire for 5-10 seconds.
		s.FlamethrowerTimer = 5.0 + r.RangeF(0, 5.0)
		s.PowerupMsg = "FLAMETHROWER!"
		s.PowerupTimer = 2.5
		s.PowerupCol = RGB{R: 255, G: 120, B: 20}
		// Camera shake.
		if cam != nil {
			cam.AddShake(0.6, 0.3)
		}
		// Initial fire burst.
		if particles != nil {
			for range 40 {
				ang := r.RangeF(0, 2*math.Pi)
				spd := r.RangeF(30, 80)
				particles.Add(Particle{
					X: hx, Y: hy,
					VX: math.Cos(ang) * spd, VY: math.Sin(ang) * spd,
					Z: r.RangeF(0, 8), VZ: r.RangeF(20, 50),
					Size: 0.7, MaxLife: r.RangeF(0.3, 0.7),
					Col: RGB{R: 255, G: uint8(100 + r.Range(0, 100)), B: 30}, Kind: ParticleFire,
				})
			}
		}

	case BonusMissile:
		s.MissileTimer = 6.0 + r.RangeF(0, 4.0) // 6–10s barrage
		if s.MissileFireTimer <= 0 {
			s.MissileFireTimer = 0.08
		}
		s.PowerupMsg = "MISSILE SWARM!"
		s.PowerupTimer = 2.5
		s.PowerupCol = RGB{R: 255, G: 150, B: 40}
		if particles != nil {
			for range 24 {
				ang := r.RangeF(0, 2*math.Pi)
				spd := r.RangeF(18, 55)
				particles.Add(Particle{
					X: hx, Y: hy,
					VX: math.Cos(ang) * spd, VY: math.Sin(ang) * spd,
					Size: 0.5, MaxLife: r.RangeF(0.2, 0.5),
					Col: RGB{R: 255, G: 170, B: 70}, Kind: ParticleGlow,
				})
			}
		}

	case BonusGatling:
		s.GatlingTimer = 5.0 + r.RangeF(0, 4.0) // 5–9s
		if s.GatlingFireTimer <= 0 {
			s.GatlingFireTimer = 0.01
		}
		s.PowerupMsg = "GATLING MOUTH!"
		s.PowerupTimer = 2.5
		s.PowerupCol = RGB{R: 255, G: 210, B: 90}
		if particles != nil {
			for range 26 {
				ang := s.Heading + r.RangeF(-0.5, 0.5)
				spd := r.RangeF(30, 95)
				particles.Add(Particle{
					X: hx + math.Cos(s.Heading)*2, Y: hy + math.Sin(s.Heading)*2,
					VX: math.Cos(ang) * spd, VY: math.Sin(ang) * spd,
					Size: 0.45, MaxLife: r.RangeF(0.12, 0.35),
					Col: RGB{R: 255, G: 230, B: 110}, Kind: ParticleGlow,
				})
			}
		}
	}

	// 28% chance for a secondary combo effect at half strength.
	if r.RangeF(0, 1) < 0.28 {
		if label := applyBonusCombo(b.Kind, s, r); label != "" {
			s.PowerupMsg += " +" + label
		}
	}
}

// applyPositionalBonus fires only the area/spatial effects of a bonus at (hx, hy)
// without touching snake stats. Used to propagate area effects to ghost/clone positions.
func applyPositionalBonus(kind BonusKind, s *Snake, hx, hy float64, r *Rand, world *World, peds *PedestrianSystem, traffic *TrafficSystem, particles *ParticleSystem, cam *Camera, cops *CopSystem, mil *MilitarySystem) {
	// Universal flash so every secondary entity visibly receives the bonus.
	if particles != nil {
		cr, cg, cb := bonusColor(kind)
		bCol := RGB{R: uint8(cr * 255), G: uint8(cg * 255), B: uint8(cb * 255)}
		for range 8 {
			ang := r.RangeF(0, 2*math.Pi)
			spd := r.RangeF(12, 30)
			particles.Add(Particle{
				X: hx, Y: hy,
				VX: math.Cos(ang) * spd, VY: math.Sin(ang) * spd,
				Size: 0.35, MaxLife: r.RangeF(0.15, 0.35),
				Col: bCol, Kind: ParticleGlow,
			})
		}
	}

	switch kind {
	case BonusSurge:
		if particles == nil {
			return
		}
		for range 16 {
			ang := r.RangeF(0, 2*math.Pi)
			dist := r.RangeF(10, 45)
			spd := r.RangeF(15, 50)
			particles.Add(Particle{
				X: hx + math.Cos(ang)*dist, Y: hy + math.Sin(ang)*dist,
				VX: math.Cos(ang+math.Pi) * spd, VY: math.Sin(ang+math.Pi) * spd,
				Size: 0.5, MaxLife: r.RangeF(0.4, 0.9),
				Col: RGB{R: 180, G: 80, B: 255}, Kind: ParticleGlow,
			})
		}

	case BonusFire:
		const fireRadius = 28.0
		bxc := int(math.Round(hx))
		byc := int(math.Round(hy))
		for range 6 {
			bx := bxc + r.Range(-int(fireRadius), int(fireRadius))
			by := byc + r.Range(-int(fireRadius), int(fireRadius))
			world.StartTreeBurn(bx, by)
		}
		if particles != nil {
			for range 80 {
				ang := r.RangeF(0, 2*math.Pi)
				dist := r.RangeF(0, fireRadius)
				particles.Add(Particle{
					X: hx + math.Cos(ang)*dist, Y: hy + math.Sin(ang)*dist,
					VX: r.RangeF(-10, 10), VY: r.RangeF(-10, 10),
					Z: r.RangeF(0, 10), VZ: r.RangeF(20, 70),
					Size: 0.5 + r.RangeF(0, 0.6), MaxLife: r.RangeF(0.3, 0.8),
					Col: Palette.FireHot, Kind: ParticleFire,
				})
			}
		}
		for i := range peds.P {
			p := &peds.P[i]
			if !p.Alive {
				continue
			}
			if math.Hypot(p.X-hx, p.Y-hy) < fireRadius {
				p.Alive = false
				s.Score += 300
				ang := r.RangeF(0, 2*math.Pi)
				if particles != nil {
					particles.SpawnBlood(p.X, p.Y, math.Cos(ang), math.Sin(ang), 18, 1.0)
				}
			}
		}

	case BonusNuke:
		wx := int(math.Round(hx))
		wy := int(math.Round(hy))
		ExplodeAt(wx, wy, 20, world, particles, peds, traffic, cam, cops, mil)
		SpawnNukeAftermath(wx, wy, world, particles, 0.85)
		const nukeRadius = 35.0
		for range 12 {
			ang := r.RangeF(0, 2*math.Pi)
			dist := r.RangeF(0, nukeRadius)
			world.StartTreeBurn(wx+int(math.Cos(ang)*dist), wy+int(math.Sin(ang)*dist))
		}
		for i := range peds.P {
			p := &peds.P[i]
			if !p.Alive {
				continue
			}
			if math.Hypot(p.X-hx, p.Y-hy) < nukeRadius {
				p.Alive = false
				s.Score += 100
				outAng := math.Atan2(p.Y-hy, p.X-hx)
				if particles != nil {
					particles.SpawnBlood(p.X, p.Y, math.Cos(outAng), math.Sin(outAng), 15, 0.9)
				}
			}
		}

	case BonusVacuum:
		s.VacuumBubbles = append(s.VacuumBubbles, VacuumBubble{
			X: hx, Y: hy,
			MaxTime: 3.5,
		})
	}
}

// applyBonusCombo applies a reduced-strength secondary effect and returns its label.
// Skips effects of the same kind as the primary.
func applyBonusCombo(primary BonusKind, s *Snake, r *Rand) string {
	type combo struct {
		kind  BonusKind
		label string
	}
	candidates := [3]combo{
		{BonusSpeed, "SPEED"},
		{BonusBash, "BASH"},
		{BonusSurge, "SURGE"},
	}

	// Build available list excluding primary.
	var available [3]combo
	n := 0
	for _, c := range candidates {
		if c.kind != primary {
			available[n] = c
			n++
		}
	}
	if n == 0 {
		return ""
	}

	chosen := available[r.Intn(n)]
	switch chosen.kind {
	case BonusSpeed:
		extra := 1.0 + r.RangeF(0, 2.0) // half-strength: 1–3s
		if extra > s.SpeedBoost {
			s.SpeedBoost = extra
		}
		if s.SpeedMult < 1.5 {
			s.SpeedMult = 1.5
		}
	case BonusBash:
		extra := 1.5 + r.RangeF(0, 2.5) // half-strength: 1.5–4s
		if extra > s.BashTimer {
			s.BashTimer = extra
		}
	case BonusSurge:
		extra := 1.0 + r.RangeF(0, 2.5) // half-strength: 1–3.5s
		if extra > s.SurgeTimer {
			s.SurgeTimer = extra
		}
	}
	return chosen.label
}

// bonusColor returns the fill color for a bonus kind.
func bonusColor(k BonusKind) (r, g, b float32) {
	switch k {
	case BonusSpeed:
		return 1.0, 0.96, 0.05
	case BonusFire:
		return 1.0, 0.28, 0.02
	case BonusBash:
		return 0.0, 0.86, 1.0
	case BonusSurge:
		return 0.66, 0.04, 1.0
	case BonusTeleport:
		return 0.52, 0.70, 1.0
	case BonusSwarm:
		return 0.30, 1.0, 0.16
	case BonusAI:
		return 1.0, 0.74, 0.0
	case BonusNuke:
		return 1.0, 0.96, 0.58
	case BonusSpread:
		return 1.0, 0.50, 0.0
	case BonusClone:
		return 0.18, 0.76, 1.0
	case BonusVacuum:
		return 0.0, 1.0, 0.82
	case BonusHealth:
		return 0.0, 1.0, 0.34
	case BonusBerserk:
		return 0.95, 0.0, 0.10
	case BonusFlamethrower:
		return 1.0, 0.62, 0.06
	case BonusMissile:
		return 0.42, 0.61, 1.0
	case BonusGatling:
		return 0.84, 0.80, 0.30
	case BonusBombSwarm:
		return 0.80, 0.44, 0.14
	case BonusTargetNuke:
		return 1.0, 1.0, 0.84
	case BonusTargetWorms:
		return 0.90, 0.14, 0.82
	case BonusTargetGunship:
		return 0.86, 0.08, 0.08
	case BonusTargetHeliMissile:
		return 0.96, 0.24, 0.44
	case BonusTargetBombBelt:
		return 0.95, 0.62, 0.24
	case BonusTargetAirSupport:
		return 0.45, 0.70, 1.0
	case BonusTargetPigs:
		return 1.0, 0.42, 0.66
	case BonusTargetCars:
		return 1.0, 0.48, 0.26
	case BonusTargetSnakes:
		return 0.40, 0.95, 0.36
	}
	return 1, 1, 1
}

// bonusRotSpeed returns radians per second for the box spin.
func bonusRotSpeed(k BonusKind) float64 {
	switch k {
	case BonusSpeed:
		return math.Pi
	case BonusFire:
		return 1.5 * math.Pi
	case BonusBash:
		return 0.8 * math.Pi
	case BonusSurge:
		return 1.2 * math.Pi
	case BonusTeleport:
		return 2.0 * math.Pi
	case BonusSwarm:
		return 0.7 * math.Pi
	case BonusAI:
		return 0.9 * math.Pi
	case BonusNuke:
		return 1.8 * math.Pi
	case BonusSpread:
		return 2.2 * math.Pi
	case BonusClone:
		return 1.1 * math.Pi
	case BonusVacuum:
		return 0.6 * math.Pi
	case BonusHealth:
		return 1.3 * math.Pi
	case BonusBerserk:
		return 2.5 * math.Pi
	case BonusFlamethrower:
		return 1.8 * math.Pi
	case BonusMissile:
		return 2.4 * math.Pi
	case BonusGatling:
		return 3.0 * math.Pi
	case BonusBombSwarm:
		return 2.1 * math.Pi
	case BonusTargetNuke:
		return 1.6 * math.Pi
	case BonusTargetWorms:
		return 2.0 * math.Pi
	case BonusTargetGunship:
		return 2.3 * math.Pi
	case BonusTargetHeliMissile:
		return 2.45 * math.Pi
	case BonusTargetBombBelt:
		return 2.3 * math.Pi
	case BonusTargetAirSupport:
		return 2.1 * math.Pi
	case BonusTargetPigs:
		return 1.9 * math.Pi
	case BonusTargetCars:
		return 2.0 * math.Pi
	case BonusTargetSnakes:
		return 2.2 * math.Pi
	}
	return math.Pi
}

// RenderData returns sprite data for all alive bonus boxes (rotated box shader format).
func (bs *BonusSystem) RenderData() []float32 {
	buf := make([]float32, 0, len(bs.Boxes)*8)
	for i := range bs.Boxes {
		b := &bs.Boxes[i]
		if !b.Alive {
			continue
		}

		cr, cg, cb := bonusColor(b.Kind)
		rotation := float32(math.Mod(b.Timer*bonusRotSpeed(b.Kind), 2*math.Pi))

		// Subtle size pulse so the box breathes slightly.
		base := float32(4.0)
		if b.Kind == BonusNuke {
			base = 5.5
		}
		size := base + float32(0.4*math.Sin(b.Timer*4.0))

		buf = append(buf, float32(b.X), float32(b.Y), size, cr, cg, cb, 0.95, rotation)
	}
	return buf
}

// SpawnSparks emits gentle ember-like sparks close to each alive bonus box.
// Call once per frame from the game loop.
func (bs *BonusSystem) SpawnSparks(ps *ParticleSystem, dt float64) {
	if ps == nil {
		return
	}
	for i := range bs.Boxes {
		b := &bs.Boxes[i]
		if !b.Alive {
			continue
		}

		cr, cg, cb := bonusColor(b.Kind)
		col := RGB{R: uint8(cr * 255), G: uint8(cg * 255), B: uint8(cb * 255)}

		timeBucket := uint64(b.Timer * 40)
		rr := NewRand(uint64(b.X*53+b.Y*37) ^ bs.seed ^ timeBucket)

		// ~20 tiny glitter sparks per second, spawned in a ring around the box.
		for range 5 {
			if rr.RangeF(0, 1) > 4.0*dt {
				continue
			}
			ang := rr.RangeF(0, 2*math.Pi)
			ring := rr.RangeF(1.2, 2.8)
			ps.Add(Particle{
				X:  b.X + math.Cos(ang)*ring,
				Y:  b.Y + math.Sin(ang)*ring,
				VX: rr.RangeF(-1.5, 1.5),
				VY: rr.RangeF(-1.5, 1.5),
				Z:  0, VZ: rr.RangeF(4, 12),
				Size:    rr.RangeF(0.08, 0.18),
				MaxLife: rr.RangeF(0.25, 0.5),
				Col:     col,
				Kind:    ParticleGlow,
			})
		}
	}
}

// GlowData returns additive glow sprites for bonus boxes (soft color halo around each box).
func (bs *BonusSystem) GlowData() []float32 {
	buf := make([]float32, 0, len(bs.Boxes)*8)
	for i := range bs.Boxes {
		b := &bs.Boxes[i]
		if !b.Alive {
			continue
		}

		cr, cg, cb := bonusColor(b.Kind)

		// Breathing glow: pulses in and out.
		intensity := float32(0.12 + 0.06*math.Sin(b.Timer*3.0))
		glowSize := float32(10.0)
		if b.Kind == BonusNuke {
			glowSize = 16.0
			intensity *= 1.4
		}

		buf = append(buf, float32(b.X), float32(b.Y), glowSize,
			cr*intensity, cg*intensity, cb*intensity, 1, 0)
	}
	return buf
}
