package game

import "math"

// Wanted level — scale 0–5.
const (
	WantedMax  = 5.0
	WantedHalf = 1.0 // cops start here

	WantedTier1 = 1.0 // used for star color thresholds in HUD
	WantedTier2 = 3.0
	WantedTier3 = 5.0
)

const (
	carDeployRadius = 15.0 // car stops and deploys cops within this distance
	carRecallRadius = 28.0 // cops recalled when snake moves beyond this
)

type CopCarState int

const (
	CarChasing   CopCarState = iota
	CarDeployed              // stopped, cops are out shooting
	CarRecalling             // waiting for a survivor to return and drive off
)

// CopCarKind controls car movement pattern.
type CopCarKind int

const (
	CarKindChaser      CopCarKind = iota // drives straight at snake
	CarKindInterceptor                   // predicts snake path and cuts it off
)

// CopPedKind controls foot cop movement and shooting pattern.
type CopPedKind int

const (
	PedChaser  CopPedKind = iota // rushes snake head directly
	PedFlanker                   // moves to intercept point ahead of snake
	PedSniper                    // holds distance, shoots faster
)

type CopCar struct {
	X, Y       float64
	Heading    float64
	Speed      float64
	HP         Health
	Alive      bool
	Size       float32
	State      CopCarState
	ID         uint32 // unique ID so deployed peds can find their car
	Kind       CopCarKind
	SirenTimer float64 // timer for continuous siren sound
}

type CopPed struct {
	X, Y         float64
	HP           Health
	Alive        bool
	ShootTimer   float64
	StuckTimer   float64
	PrevX, PrevY float64
	OwnerCarID   uint32 // 0 = standalone; >0 = deployed from this car
	Returning    bool   // heading back to car before it drives off
	Kind         CopPedKind
}

type Helicopter struct {
	X, Y         float64
	Heading      float64 // direction the nose points (tangent of circle)
	RotorAngle   float64 // spins fast for blade animation
	CircleAngle  float64
	CircleRadius float64
	CenterX      float64 // orbit center — lerps toward snake at fixed speed
	CenterY      float64
	HP           Health
	Alive        bool
	BurstTimer   float64 // > 0: actively firing; ≤ 0: on pause between bursts
	PauseTimer   float64 // counts down before next burst
	FireTimer    float64 // time until next individual shot within a burst
	Burning      bool    // true at low HP — crashes after BurnTimer reaches 3s
	BurnTimer    float64
}

// HeliShot is a single gatling round fired by a helicopter.
type HeliShot struct {
	X, Y   float64
	VX, VY float64
	Life   float64
}

type CopSystem struct {
	Cars       []CopCar
	Peds       []CopPed
	Helis      []Helicopter
	Shots      []HeliShot
	seed       uint64
	SpawnTimer float64
	nextCarID  uint32
}

func NewCopSystem(seed uint64) *CopSystem {
	if seed == 0 {
		seed = 1
	}
	return &CopSystem{seed: seed, SpawnTimer: 3.0, nextCarID: 1}
}

func (cs *CopSystem) Reset() {
	cs.Cars = cs.Cars[:0]
	cs.Peds = cs.Peds[:0]
	cs.Helis = cs.Helis[:0]
	cs.Shots = cs.Shots[:0]
	cs.SpawnTimer = 3.0
	cs.nextCarID = 1
}

// carPos returns the position of the car with the given ID, or false if not found.
func (cs *CopSystem) carPos(id uint32) (float64, float64, bool) {
	for _, c := range cs.Cars {
		if c.ID == id && c.Alive {
			return c.X, c.Y, true
		}
	}
	return 0, 0, false
}

// reboardCar marks a patrol car as manned again and sends any remaining patrol on foot.
func (cs *CopSystem) reboardCar(carID uint32) {
	for i := range cs.Cars {
		c := &cs.Cars[i]
		if c.ID == carID && c.Alive {
			c.State = CarChasing
			break
		}
	}
	for i := range cs.Peds {
		p := &cs.Peds[i]
		if p.Alive && p.OwnerCarID == carID {
			p.OwnerCarID = 0
			p.Returning = false
		}
	}
}

func sirenDistanceGain(dist float64) float64 {
	const near = 24.0
	const far = 220.0
	if dist <= near {
		return 1.0
	}
	if dist >= far {
		return 0.0
	}
	t := (dist - near) / (far - near)
	g := 1.0 - t
	return g * g
}

func sirenDopplerFactor(radialSpeed float64) float64 {
	// Pseudo speed-of-sound in world units/s; tuned for subtle but audible shift.
	const c = 520.0
	return clampF(c/(c-radialSpeed), 0.90, 1.12)
}

func sirenStereoPan(srcX, listenerX float64) float64 {
	// -1 = left, +1 = right.
	return clampF((srcX-listenerX)/90.0, -1.0, 1.0)
}

// deployedAliveCount returns how many alive peds are currently deployed from the given car.
func (cs *CopSystem) deployedAliveCount(carID uint32) int {
	n := 0
	for _, p := range cs.Peds {
		if p.Alive && p.OwnerCarID == carID {
			n++
		}
	}
	return n
}

// Update advances all cop AI, handles collisions, and spawns reinforcements.
func (cs *CopSystem) Update(dt float64, snake *Snake, world *World, ps *ParticleSystem, cam *Camera, now float64) {
	if snake == nil || !snake.Alive || snake.WantedLevel < WantedHalf {
		return
	}

	hx, hy := snake.Head()

	// --- Cop cars ---
	for ci := range cs.Cars {
		c := &cs.Cars[ci]
		if !c.Alive {
			continue
		}
		dist := math.Hypot(hx-c.X, hy-c.Y)

		// Continuous siren sound.
		c.SirenTimer -= dt
		if c.SirenTimer <= 0 {
			dx := hx - c.X
			dy := hy - c.Y
			ux, uy := 0.0, 0.0
			if dist > 0.001 {
				ux = dx / dist
				uy = dy / dist
			}
			carVX := math.Cos(c.Heading) * c.Speed
			carVY := math.Sin(c.Heading) * c.Speed
			radial := carVX*ux + carVY*uy
			PlayPoliceSirenSpatial(
				sirenDistanceGain(dist),
				sirenDopplerFactor(radial),
				sirenStereoPan(c.X, hx),
			)
			// Slight per-car cadence offset keeps fleets from sounding phase-locked.
			c.SirenTimer = 1.05 + float64(c.ID%5)*0.11
		}

		switch c.State {
		case CarChasing:
			// Road-following AI: navigate on the road grid toward the snake.
			var targetX, targetY float64
			if c.Kind == CarKindInterceptor {
				lead := 16.0
				targetX = hx + math.Cos(snake.Heading)*lead
				targetY = hy + math.Sin(snake.Heading)*lead
			} else {
				targetX, targetY = hx, hy
			}

			ix := int(c.X)
			iy := int(c.Y)
			onVertRoad := ix%Pattern < RoadWidth
			onHorizRoad := iy%Pattern < RoadWidth

			if onVertRoad || onHorizRoad {
				// On a road: follow the grid.
				if onVertRoad && onHorizRoad {
					// At intersection: pick best cardinal direction toward target.
					c.Heading = bestCardinalToward(c.X, c.Y, targetX, targetY)
				} else if onVertRoad {
					// On a vertical road: heading must be N or S.
					if math.Abs(math.Cos(c.Heading)) > math.Abs(math.Sin(c.Heading)) {
						if targetY > c.Y {
							c.Heading = math.Pi / 2
						} else {
							c.Heading = -math.Pi / 2
						}
					}
				} else {
					// On a horizontal road: heading must be E or W.
					if math.Abs(math.Sin(c.Heading)) > math.Abs(math.Cos(c.Heading)) {
						if targetX > c.X {
							c.Heading = 0
						} else {
							c.Heading = math.Pi
						}
					}
				}

				nx := c.X + math.Cos(c.Heading)*c.Speed*dt
				ny := c.Y + math.Sin(c.Heading)*c.Speed*dt

				// Gently center on road.
				if onVertRoad {
					centerX := float64((ix/Pattern)*Pattern) + float64(RoadWidth)/2.0
					nx += (centerX - c.X) * 5.0 * dt
				}
				if onHorizRoad {
					centerY := float64((iy/Pattern)*Pattern) + float64(RoadWidth)/2.0
					ny += (centerY - c.Y) * 5.0 * dt
				}
				c.X = clampF(nx, 0, float64(WorldWidth-1))
				c.Y = clampF(ny, 0, float64(WorldHeight-1))
			} else {
				// Off-road: steer directly toward target to reach a road.
				dx := targetX - c.X
				dy := targetY - c.Y
				if d := math.Hypot(dx, dy); d > 0.1 {
					targetH := math.Atan2(dy, dx)
					diff := angDiff(c.Heading, targetH)
					maxTurn := 3.5 * dt
					if math.Abs(diff) <= maxTurn {
						c.Heading = targetH
					} else if diff > 0 {
						c.Heading += maxTurn
					} else {
						c.Heading -= maxTurn
					}
				}
				nx := c.X + math.Cos(c.Heading)*c.Speed*dt
				ny := c.Y + math.Sin(c.Heading)*c.Speed*dt
				if world.HeightAt(int(math.Round(nx)), int(math.Round(ny))) > 0 {
					nx, ny = c.X, c.Y
				}
				c.X = clampF(nx, 0, float64(WorldWidth-1))
				c.Y = clampF(ny, 0, float64(WorldHeight-1))
			}

			// Ram snake on contact.
			if dist < float64(c.Size)*0.7 {
				if len(snake.Ghosts) == 0 {
					snake.HP.Damage(0.15)
				}
				snake.WantedLevel = min(WantedMax, snake.WantedLevel+0.5)
				c.Alive = false
				if ps != nil {
					SpawnExplosionWithShockwave(int(c.X), int(c.Y), RGB{50, 100, 200}, 0.5, 0, world, ps)
				}
				if cam != nil {
					cam.AddShake(0.4, 0.2)
				}
				continue
			}

			// Deploy when close enough.
			if dist < carDeployRadius {
				c.State = CarDeployed
				cs.spawnDeployedPeds(c, world)
			}

		case CarDeployed:
			// Car is parked. Recall only if at least one patrol cop is still alive.
			alivePatrol := cs.deployedAliveCount(c.ID)
			if dist > carRecallRadius && alivePatrol > 0 {
				cs.recallPeds(c.ID)
				c.State = CarRecalling
			}

		case CarRecalling:
			// Patrol wiped out before anyone made it back: leave the parked car.
			if cs.deployedAliveCount(c.ID) == 0 {
				c.State = CarDeployed
			}
		}
	}

	// --- Cop peds ---
	for i := range cs.Peds {
		p := &cs.Peds[i]
		if !p.Alive {
			continue
		}

		dx := hx - p.X
		dy := hy - p.Y
		dist := math.Hypot(dx, dy)

		if p.Returning {
			// Move back to parked car; disappear on arrival.
			carX, carY, found := cs.carPos(p.OwnerCarID)
			if !found {
				// Car is gone — become standalone.
				p.OwnerCarID = 0
				p.Returning = false
				continue
			}
			rdx := carX - p.X
			rdy := carY - p.Y
			rd := math.Hypot(rdx, rdy)
			if rd < 1.8 {
				carID := p.OwnerCarID
				p.Alive = false // absorbed back into car
				cs.reboardCar(carID)
				continue
			}
			p.X += rdx / rd * 20.0 * dt
			p.Y += rdy / rd * 20.0 * dt
			continue
		}

		// Movement: kind-specific target and speed.
		var moveX, moveY, spd float64
		switch p.Kind {
		case PedFlanker:
			// Move to intercept point ahead of snake; once there, rush the head.
			fx := hx + math.Cos(snake.Heading)*20
			fy := hy + math.Sin(snake.Heading)*20
			fdx := fx - p.X
			fdy := fy - p.Y
			if fd := math.Hypot(fdx, fdy); fd > 3.0 {
				moveX, moveY = fdx/fd, fdy/fd
			} else if dist > 1.5 {
				moveX, moveY = dx/dist, dy/dist
			}
			spd = 22.0
		case PedSniper:
			// Hold at ~24px; back off when too close, close in when too far.
			const preferDist = 24.0
			if dist < preferDist-4 && dist > 0.1 {
				moveX, moveY = -dx/dist, -dy/dist
			} else if dist > preferDist+4 {
				moveX, moveY = dx/dist, dy/dist
			}
			spd = 14.0
		default: // PedChaser
			if dist > 1.5 {
				moveX, moveY = dx/dist, dy/dist
			}
			spd = 18.0
		}

		if moveX != 0 || moveY != 0 {
			nx := p.X + moveX*spd*dt
			ny := p.Y + moveY*spd*dt
			// Building collision: only move if destination is clear.
			if world.HeightAt(int(math.Round(nx)), int(math.Round(ny))) > 0 {
				// Blocked: pick a new position.
				r := NewRand(cs.seed ^ uint64(i)*0xC0C0 ^ uint64(p.StuckTimer*100))
				for range 12 {
					tx := int(math.Round(p.X)) + r.Range(-8, 8)
					ty := int(math.Round(p.Y)) + r.Range(-8, 8)
					if tx >= 0 && ty >= 0 && tx < WorldWidth && ty < WorldHeight && world.HeightAt(tx, ty) == 0 {
						p.X = float64(tx) + 0.5
						p.Y = float64(ty) + 0.5
						break
					}
				}
			} else {
				p.X = nx
				p.Y = ny
			}
			if moved := math.Hypot(p.X-p.PrevX, p.Y-p.PrevY); moved < 0.05*dt {
				p.StuckTimer += dt
				if p.StuckTimer > 2.0 {
					r := NewRand(cs.seed ^ uint64(i)*0xFED ^ uint64(p.StuckTimer*100))
					for range 20 {
						tx := int(math.Round(hx)) + r.Range(-20, 20)
						ty := int(math.Round(hy)) + r.Range(-20, 20)
						if tx >= 0 && ty >= 0 && tx < WorldWidth && ty < WorldHeight && world.HeightAt(tx, ty) == 0 {
							p.X = float64(tx)
							p.Y = float64(ty)
							p.StuckTimer = 0
							break
						}
					}
				}
			} else {
				p.StuckTimer = 0
			}
			p.PrevX, p.PrevY = p.X, p.Y
		}

		shootInterval := 1.4
		if p.Kind == PedSniper {
			shootInterval = 0.7
		}
		p.ShootTimer -= dt
		if p.ShootTimer <= 0 && dist < 35.0 && HasLineOfSight(p.X, p.Y, hx, hy, world) {
			p.ShootTimer = shootInterval
			PlaySound(SoundGunshot)
			if ps != nil {
				ang := math.Atan2(dy, dx)
				ps.Add(Particle{
					X: p.X, Y: p.Y,
					VX: math.Cos(ang) * 85, VY: math.Sin(ang) * 85,
					Size: 0.35, MaxLife: 0.35,
					Col: RGB{R: 255, G: 255, B: 120}, Kind: ParticleGlow,
				})
			}
			if dist < 28.0 && len(snake.Ghosts) == 0 {
				snake.HP.Damage(0.03)
			}
		}

		if dist < SnakeEatRadius {
			if len(snake.Ghosts) == 0 {
				snake.HP.Damage(0.03)
			}
			// Blood splatter when eaten.
			if ps != nil {
				ang := math.Atan2(p.Y-hy, p.X-hx)
				ps.SpawnBlood(p.X, p.Y, math.Cos(ang), math.Sin(ang), 16, 0.85)
				ps.SpawnBlood(p.X+0.3, p.Y+0.3, math.Cos(ang+math.Pi/2), math.Sin(ang+math.Pi/2), 8, 0.5)
			}
			p.Alive = false
		}
	}

	// --- Helicopters ---
	for i := range cs.Helis {
		h := &cs.Helis[i]
		if !h.Alive {
			continue
		}

		// Lerp orbit center toward snake head at a fixed speed (independent of snake speed).
		const heliCenterSpeed = 35.0
		cdx := hx - h.CenterX
		cdy := hy - h.CenterY
		cd := math.Hypot(cdx, cdy)
		if cd > heliCenterSpeed*dt {
			h.CenterX += cdx / cd * heliCenterSpeed * dt
			h.CenterY += cdy / cd * heliCenterSpeed * dt
		} else {
			h.CenterX, h.CenterY = hx, hy
		}

		h.CircleAngle += 0.75 * dt
		h.X = clampF(h.CenterX+math.Cos(h.CircleAngle)*h.CircleRadius, 2, float64(WorldWidth-2))
		h.Y = clampF(h.CenterY+math.Sin(h.CircleAngle)*h.CircleRadius, 2, float64(WorldHeight-2))
		h.Heading = h.CircleAngle + math.Pi/2
		h.RotorAngle += 14.0 * dt

		// Start burning at low HP; crash after 3 seconds on fire.
		if !h.Burning && h.HP.Fraction() < 0.35 {
			h.Burning = true
		}
		if h.Burning {
			h.BurnTimer += dt
			h.RotorAngle += 10.0 * dt // erratic extra spin
			if ps != nil {
				rr := NewRand(cs.seed ^ uint64(i+1)*0xF1AE ^ uint64(h.BurnTimer*100))
				ps.Add(Particle{
					X: h.X + rr.RangeF(-1, 1), Y: h.Y + rr.RangeF(-1, 1),
					VX: rr.RangeF(-5, 5), VY: rr.RangeF(-5, 5),
					Z: rr.RangeF(0, 4), VZ: rr.RangeF(15, 35),
					Size: 0.6, MaxLife: rr.RangeF(0.25, 0.55),
					Col: Palette.FireHot, Kind: ParticleFire,
				})
				ps.Add(Particle{
					X: h.X + rr.RangeF(-1.5, 1.5), Y: h.Y + rr.RangeF(-1.5, 1.5),
					VX: rr.RangeF(-3, 3), VY: rr.RangeF(-3, 3),
					Size: 1.0, MaxLife: rr.RangeF(0.3, 0.7),
					Col: RGB{R: 90, G: 85, B: 80}, Kind: ParticleSmoke,
				})
			}
			if h.BurnTimer >= 3.0 {
				h.Alive = false
				ExplodeAt(int(math.Round(h.X)), int(math.Round(h.Y)), 8, world, ps, nil, nil, cam, nil, nil)
				continue
			}
		}

		if h.BurstTimer > 0 {
			h.BurstTimer -= dt
			h.FireTimer -= dt
			if h.FireTimer <= 0 {
				h.FireTimer = 0.08
				PlaySound(SoundGunshot)
				r := NewRand(cs.seed ^ uint64(i)*0xBAD ^ uint64(now*1000))
				lead := 1.5
				tx := hx + math.Cos(snake.Heading)*lead
				ty := hy + math.Sin(snake.Heading)*lead
				ang := math.Atan2(ty-h.Y, tx-h.X) + r.RangeF(-0.18, 0.18)
				spd := 140.0
				cs.Shots = append(cs.Shots, HeliShot{
					X: h.X, Y: h.Y,
					VX: math.Cos(ang) * spd, VY: math.Sin(ang) * spd,
					Life: 0.7,
				})
				if ps != nil {
					ps.Add(Particle{
						X: h.X + math.Cos(ang)*1.5, Y: h.Y + math.Sin(ang)*1.5,
						VX: math.Cos(ang) * 20, VY: math.Sin(ang) * 20,
						Size: 0.6, MaxLife: 0.06,
						Col: RGB{R: 255, G: 220, B: 100}, Kind: ParticleGlow,
					})
				}
			}
			if h.BurstTimer <= 0 {
				h.PauseTimer = 1.8 + NewRand(cs.seed^uint64(i)*0xC0DE^uint64(now*10)).RangeF(0, 1.0)
			}
		} else {
			h.PauseTimer -= dt
			if h.PauseTimer <= 0 {
				h.BurstTimer = 1.5
				h.FireTimer = 0
			}
		}
	}

	// --- Gatling shots ---
	for si := len(cs.Shots) - 1; si >= 0; si-- {
		shot := &cs.Shots[si]
		shot.Life -= dt
		shot.X += shot.VX * dt
		shot.Y += shot.VY * dt

		ix := int(math.Round(shot.X))
		iy := int(math.Round(shot.Y))
		hit := false

		if ix < 0 || iy < 0 || ix >= WorldWidth || iy >= WorldHeight {
			hit = true
		} else if world.HeightAt(ix, iy) > 0 {
			if ps != nil {
				SpawnExplosionWithShockwave(ix, iy, RGB{180, 140, 80}, 0.15, 0, world, ps)
			}
			hit = true
		} else if math.Hypot(shot.X-hx, shot.Y-hy) < 2.0 {
			if ps != nil {
				SpawnExplosionWithShockwave(ix, iy, RGB{255, 120, 40}, 0.15, 0, world, ps)
				cam.AddShake(0.15, 0.08)
			}
			if len(snake.Ghosts) == 0 {
				snake.HP.Damage(0.03)
			}
			hit = true
		} else if shot.Life <= 0 {
			if ps != nil {
				SpawnExplosionWithShockwave(ix, iy, RGB{120, 100, 70}, 0.1, 0, world, ps)
			}
			hit = true
		}

		if hit {
			cs.Shots[si] = cs.Shots[len(cs.Shots)-1]
			cs.Shots = cs.Shots[:len(cs.Shots)-1]
		}
	}

	// Spawn reinforcements.
	cs.SpawnTimer -= dt
	if cs.SpawnTimer <= 0 {
		r := NewRand(cs.seed ^ uint64(snake.WantedLevel*100) ^ uint64(len(cs.Cars)*0xABC))
		cs.SpawnTimer = 5.0 + r.RangeF(0, 3.0)
		cs.spawnForWanted(snake, world, r)
	}
}

// spawnDeployedPeds exits 1–4 cop peds from the corners of a parked car.
// The count is randomised per car so squads vary in size.
func (cs *CopSystem) spawnDeployedPeds(c *CopCar, world *World) {
	r := NewRand(uint64(c.X*97+c.Y*53) ^ uint64(c.ID)*0xC0FFEE)
	numCops := 1 + r.Intn(4) // 1, 2, 3, or 4

	fwdX := math.Cos(c.Heading)
	fwdY := math.Sin(c.Heading)
	perpX := -math.Sin(c.Heading)
	perpY := math.Cos(c.Heading)

	// Up to 4 exit positions: front-left, front-right, rear-left, rear-right.
	offsets := [4][2]float64{
		{fwdX*1.4 + perpX*1.8, fwdY*1.4 + perpY*1.8},
		{fwdX*1.4 - perpX*1.8, fwdY*1.4 - perpY*1.8},
		{-fwdX*1.4 + perpX*1.8, -fwdY*1.4 + perpY*1.8},
		{-fwdX*1.4 - perpX*1.8, -fwdY*1.4 - perpY*1.8},
	}
	for i := 0; i < numCops; i++ {
		off := offsets[i]
		px := clampF(c.X+off[0], 0, float64(WorldWidth-1))
		py := clampF(c.Y+off[1], 0, float64(WorldHeight-1))
		if world.HeightAt(int(math.Round(px)), int(math.Round(py))) > 0 {
			px, py = c.X, c.Y // fallback to car center
		}
		cs.Peds = append(cs.Peds, CopPed{
			X: px, Y: py,
			HP:         NewHealth(4.0),
			Alive:      true,
			ShootTimer: float64(i) * 0.15, // stagger initial shots
			OwnerCarID: c.ID,
		})
	}
}

// recallPeds marks all peds deployed from this car as returning.
func (cs *CopSystem) recallPeds(carID uint32) {
	for i := range cs.Peds {
		p := &cs.Peds[i]
		if p.Alive && p.OwnerCarID == carID && !p.Returning {
			p.Returning = true
		}
	}
}

func (cs *CopSystem) spawnForWanted(snake *Snake, world *World, r *Rand) {
	hx, hy := snake.Head()
	targetCars, targetPeds, targetHelis := wantedTargets(snake.WantedLevel)

	aliveCars, alivePeds, aliveHelis := 0, 0, 0
	for _, c := range cs.Cars {
		if c.Alive {
			aliveCars++
		}
	}
	for _, p := range cs.Peds {
		if p.Alive && p.OwnerCarID == 0 { // only count standalone peds
			alivePeds++
		}
	}
	for _, h := range cs.Helis {
		if h.Alive {
			aliveHelis++
		}
	}

	if aliveCars < targetCars {
		cs.nextCarID++
		sx, sy := edgeSpawnPos(hx, hy, r)
		kind := CarKindChaser
		if snake.WantedLevel >= 3.5 && r.Intn(2) == 0 {
			kind = CarKindInterceptor
		}
		cs.Cars = append(cs.Cars, CopCar{
			X: sx, Y: sy,
			Heading: math.Atan2(hy-sy, hx-sx),
			Speed:   35.0 + r.RangeF(0, 10.0),
			HP:      NewHealth(10.0),
			Alive:   true,
			Size:    CarSize,
			State:   CarChasing,
			ID:      cs.nextCarID,
			Kind:    kind,
			// Randomize first siren to prevent synchronized bursts.
			SirenTimer: 0.20 + r.RangeF(0, 0.75),
		})
	}

	if alivePeds < targetPeds {
		sx, sy := edgeSpawnPos(hx, hy, r)
		needed := targetPeds - alivePeds
		kind := randomPedKind(snake.WantedLevel, r)
		groupSize := 1
		if snake.WantedLevel >= 4.0 && needed >= 3 {
			groupSize = 3
		} else if snake.WantedLevel >= 3.0 && needed >= 2 {
			groupSize = 2
		}
		cs.spawnPedGroup(groupSize, sx, sy, kind, world, r)
	}

	if aliveHelis < targetHelis {
		radius := 30.0
		ang := r.RangeF(0, 2*math.Pi)
		cs.Helis = append(cs.Helis, Helicopter{
			X: hx + math.Cos(ang)*radius, Y: hy + math.Sin(ang)*radius,
			CenterX:      hx,
			CenterY:      hy,
			CircleAngle:  ang,
			CircleRadius: radius,
			HP:           NewHealth(20.0),
			Alive:        true,
			PauseTimer:   2.0 + r.RangeF(0, 1.0),
		})
		PlaySound(SoundHelicopter)
	}
}

// randomPedKind chooses a ped kind weighted by wanted level.
func randomPedKind(wanted float64, r *Rand) CopPedKind {
	if wanted >= 4.5 {
		switch r.Intn(3) {
		case 0:
			return PedFlanker
		case 1:
			return PedSniper
		default:
			return PedChaser
		}
	}
	if wanted >= 3.5 && r.Intn(2) == 0 {
		return PedFlanker
	}
	return PedChaser
}

// spawnPedGroup spawns n cop peds clustered around cx,cy with the given kind.
func (cs *CopSystem) spawnPedGroup(n int, cx, cy float64, kind CopPedKind, world *World, r *Rand) {
	for i := range n {
		for range 20 {
			tx := int(math.Round(cx)) + r.Range(-6, 6)
			ty := int(math.Round(cy)) + r.Range(-6, 6)
			if tx >= 0 && ty >= 0 && tx < WorldWidth && ty < WorldHeight && world.HeightAt(tx, ty) == 0 {
				cs.Peds = append(cs.Peds, CopPed{
					X: float64(tx), Y: float64(ty),
					HP:         NewHealth(6.0),
					Alive:      true,
					ShootTimer: float64(i)*0.25 + r.RangeF(0, 0.4),
					Kind:       kind,
				})
				break
			}
		}
	}
}

func wantedTargets(level float64) (cars, peds, helis int) {
	switch {
	case level >= WantedMax: // 5.0 — full force + 2 helicopters
		return 5, 6, 2
	case level >= 4.5:
		return 4, 4, 0
	case level >= 4.0:
		return 4, 3, 0
	case level >= 3.5:
		return 3, 2, 0
	case level >= 3.0:
		return 2, 1, 0
	default: // 2.5–3.0 — just cars
		return 1, 0, 0
	}
}

// bestCardinalToward returns the cardinal direction (E/S/W/N) that best
// reduces distance from (cx,cy) toward (tx,ty).
func bestCardinalToward(cx, cy, tx, ty float64) float64 {
	dx := tx - cx
	dy := ty - cy
	cardinals := [4]float64{0, math.Pi / 2, math.Pi, -math.Pi / 2}
	best := cardinals[0]
	bestDot := -math.MaxFloat64
	for _, h := range cardinals {
		dot := dx*math.Cos(h) + dy*math.Sin(h)
		if dot > bestDot {
			bestDot = dot
			best = h
		}
	}
	return best
}

func edgeSpawnPos(hx, hy float64, r *Rand) (float64, float64) {
	switch r.Intn(4) {
	case 0:
		return r.RangeF(1, float64(WorldWidth-2)), 1
	case 1:
		return r.RangeF(1, float64(WorldWidth-2)), float64(WorldHeight - 2)
	case 2:
		return 1, r.RangeF(1, float64(WorldHeight-2))
	default:
		return float64(WorldWidth - 2), r.RangeF(1, float64(WorldHeight-2))
	}
}

func (cs *CopSystem) RemoveDead() {
	for i := 0; i < len(cs.Cars); {
		if !cs.Cars[i].Alive {
			cs.Cars[i] = cs.Cars[len(cs.Cars)-1]
			cs.Cars = cs.Cars[:len(cs.Cars)-1]
		} else {
			i++
		}
	}
	for i := 0; i < len(cs.Peds); {
		if !cs.Peds[i].Alive {
			cs.Peds[i] = cs.Peds[len(cs.Peds)-1]
			cs.Peds = cs.Peds[:len(cs.Peds)-1]
		} else {
			i++
		}
	}
	for i := 0; i < len(cs.Helis); {
		if !cs.Helis[i].Alive {
			cs.Helis[i] = cs.Helis[len(cs.Helis)-1]
			cs.Helis = cs.Helis[:len(cs.Helis)-1]
		} else {
			i++
		}
	}
}

// CopRenderData returns point sprite data for cop peds, helis, and gatling shots.
// Cop cars are rendered as textured quads via DrawCopCars.
func (cs *CopSystem) CopRenderData(now float64) []float32 {
	buf := make([]float32, 0, (len(cs.Peds)*3+len(cs.Helis)*16+len(cs.Shots))*8)

	// Cop peds: color-coded by kind (blue=chaser, green=flanker, red=sniper).
	for _, p := range cs.Peds {
		if !p.Alive {
			continue
		}
		var ur, ug, ub float32
		switch p.Kind {
		case PedFlanker:
			ur, ug, ub = 0.2, 0.85, 0.3
		case PedSniper:
			ur, ug, ub = 0.9, 0.25, 0.2
		default:
			ur, ug, ub = 0.2, 0.4, 1.0
		}
		buf = append(buf, float32(p.X), float32(p.Y)+0.3, 1.6, 0.04, 0.04, 0.04, 0.4, 0)
		buf = append(buf, float32(p.X), float32(p.Y), 1.5, ur, ug, ub, 1.0, 0)
		buf = append(buf, float32(p.X), float32(p.Y)-0.8, 0.7, 1.0, 1.0, 1.0, 1.0, 0)
	}

	// Helicopters: multi-sprite top-down — fuselage, tail boom, tail rotor, main rotor.
	for _, h := range cs.Helis {
		if !h.Alive {
			continue
		}
		fwdX := math.Cos(h.Heading)
		fwdY := math.Sin(h.Heading)
		perpX := -math.Sin(h.Heading)
		perpY := math.Cos(h.Heading)
		hx32 := float32(h.X)
		hy32 := float32(h.Y)

		buf = append(buf, hx32, hy32+1.8, 7.0, 0.05, 0.05, 0.05, 0.22, 0) // shadow
		buf = append(buf, hx32, hy32, 3.5, 0.88, 0.92, 1.0, 1.0, 0)       // body
		buf = append(buf,
			float32(h.X+fwdX*1.8), float32(h.Y+fwdY*1.8),
			2.2, 0.6, 0.75, 1.0, 1.0, 0) // cockpit

		for step := 1; step <= 3; step++ {
			sz := float32(1.6 - float32(step)*0.25)
			buf = append(buf,
				float32(h.X-fwdX*float64(step)*1.3),
				float32(h.Y-fwdY*float64(step)*1.3),
				sz, 0.55, 0.65, 0.85, 1.0, 0)
		}

		tailX := h.X - fwdX*4.5
		tailY := h.Y - fwdY*4.5
		buf = append(buf, float32(tailX+perpX*1.1), float32(tailY+perpY*1.1), 1.1, 0.85, 0.9, 1.0, 1.0, 0)
		buf = append(buf, float32(tailX-perpX*1.1), float32(tailY-perpY*1.1), 1.1, 0.85, 0.9, 1.0, 1.0, 0)
		buf = append(buf, float32(tailX), float32(tailY), 1.0, 0.7, 0.78, 0.95, 1.0, 0)

		const bladeR = 3.2
		rotX := math.Cos(h.RotorAngle)
		rotY := math.Sin(h.RotorAngle)
		buf = append(buf, float32(h.X+rotX*bladeR), float32(h.Y+rotY*bladeR), 1.0, 0.95, 0.97, 1.0, 0.85, 0)
		buf = append(buf, float32(h.X+rotX*bladeR*0.5), float32(h.Y+rotY*bladeR*0.5), 1.0, 0.9, 0.93, 1.0, 0.9, 0)
		buf = append(buf, float32(h.X-rotX*bladeR*0.5), float32(h.Y-rotY*bladeR*0.5), 1.0, 0.9, 0.93, 1.0, 0.9, 0)
		buf = append(buf, float32(h.X-rotX*bladeR), float32(h.Y-rotY*bladeR), 1.0, 0.95, 0.97, 1.0, 0.85, 0)

		buf = append(buf, float32(h.X+fwdX*3.2), float32(h.Y+fwdY*3.2), 0.8, 0.25, 0.25, 0.3, 1.0, 0) // barrel
	}

	// Gatling shots: tiny yellow-white tracers.
	for _, s := range cs.Shots {
		buf = append(buf, float32(s.X), float32(s.Y), 0.6, 1.0, 0.95, 0.6, 1.0, 0)
	}

	return buf
}

// CopGlowData returns additive glow sprites for police car lights and helicopter beacon.
func (cs *CopSystem) CopGlowData(now float64) []float32 {
	buf := make([]float32, 0, (len(cs.Cars)*4+len(cs.Helis)*3+len(cs.Shots))*8)

	// Police car roof lights: left=red, right=blue, swapping each blink.
	for _, c := range cs.Cars {
		if !c.Alive {
			continue
		}
		perpX := float32(-math.Sin(c.Heading) * 1.1)
		perpY := float32(math.Cos(c.Heading) * 1.1)
		lx := float32(c.X) - perpX
		ly := float32(c.Y) - perpY
		rx := float32(c.X) + perpX
		ry := float32(c.Y) + perpY

		leftRed := int(now*5)%2 == 0
		if leftRed {
			buf = append(buf, lx, ly, 1.5, 1.0, 0.1, 0.1, 1, 0)
			buf = append(buf, rx, ry, 1.5, 0.15, 0.4, 1.0, 1, 0)
			buf = append(buf, lx, ly, 5.0, 0.35, 0.03, 0.03, 1, 0)
			buf = append(buf, rx, ry, 5.0, 0.04, 0.16, 0.45, 1, 0)
		} else {
			buf = append(buf, lx, ly, 1.5, 0.15, 0.4, 1.0, 1, 0)
			buf = append(buf, rx, ry, 1.5, 1.0, 0.1, 0.1, 1, 0)
			buf = append(buf, lx, ly, 5.0, 0.04, 0.16, 0.45, 1, 0)
			buf = append(buf, rx, ry, 5.0, 0.35, 0.03, 0.03, 1, 0)
		}
	}

	// Helicopter beacon + muzzle glow when firing.
	for _, h := range cs.Helis {
		if !h.Alive {
			continue
		}
		if int(now*8)%2 == 0 {
			buf = append(buf, float32(h.X), float32(h.Y)-1.2, 1.5, 1.0, 0.1, 0.1, 1, 0)
			buf = append(buf, float32(h.X), float32(h.Y)-1.2, 6.0, 0.3, 0.03, 0.03, 1, 0)
		} else {
			buf = append(buf, float32(h.X), float32(h.Y)-1.2, 1.5, 0.15, 0.45, 1.0, 1, 0)
			buf = append(buf, float32(h.X), float32(h.Y)-1.2, 6.0, 0.05, 0.15, 0.4, 1, 0)
		}
		if h.BurstTimer > 0 {
			buf = append(buf,
				float32(h.X+math.Cos(h.Heading)*3.2),
				float32(h.Y+math.Sin(h.Heading)*3.2),
				3.5, 0.5, 0.4, 0.1, 1, 0)
		}
	}

	// Shot glow: warm tracer halos.
	for _, s := range cs.Shots {
		buf = append(buf, float32(s.X), float32(s.Y), 2.5, 0.3, 0.25, 0.05, 1, 0)
	}

	return buf
}

// ExplodeAffectCops damages cop entities within an explosion radius.
func ExplodeAffectCops(wx, wy, radius int, w *World, ps *ParticleSystem, cops *CopSystem) {
	if cops == nil || radius <= 0 {
		return
	}
	fwx := float64(wx)
	fwy := float64(wy)
	fr := float64(radius)

	for i := range cops.Cars {
		c := &cops.Cars[i]
		if !c.Alive {
			continue
		}
		d := math.Hypot(c.X-fwx, c.Y-fwy)
		if d < fr*1.5 {
			nd := d / (fr * 1.5)
			c.HP.Damage((1.0 - nd) * 25.0)
			if c.HP.IsDead() {
				c.Alive = false
				if ps != nil {
					SpawnExplosionWithShockwave(int(c.X), int(c.Y), RGB{50, 100, 200}, 0.6, 0, w, ps)
				}
			}
		}
	}

	for i := range cops.Peds {
		p := &cops.Peds[i]
		if !p.Alive {
			continue
		}
		if math.Hypot(p.X-fwx, p.Y-fwy) < fr {
			p.Alive = false
		}
	}

	for i := range cops.Helis {
		h := &cops.Helis[i]
		if !h.Alive {
			continue
		}
		d := math.Hypot(h.X-fwx, h.Y-fwy)
		if d < fr*2.0 {
			nd := d / (fr * 2.0)
			h.HP.Damage((1.0 - nd) * 20.0)
			if h.HP.IsDead() && !h.Burning {
				h.Burning = true // starts burning; crashes after BurnTimer reaches 3s
			}
		}
	}
}
