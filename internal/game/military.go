package game

import "math"

const (
	MilitaryActivationDelay = 30.0 // seconds at max wanted before military arrives
)

// TroopKind controls military foot soldier behavior.
type TroopKind int

const (
	TroopRegular TroopKind = iota // chases snake in groups
	TroopSniper                   // long range, high damage, slow fire
	TroopMinigun                  // short range, very rapid fire
)

type Tank struct {
	X, Y      float64
	Heading   float64
	TurretAng float64
	Speed     float64
	HP        Health
	Alive     bool
	FireTimer float64
	Size      float32
}

type MilHeli struct {
	X, Y         float64
	Heading      float64
	RotorAngle   float64
	CircleAngle  float64
	CircleRadius float64
	CenterX      float64 // orbit center — lerps toward snake at fixed speed
	CenterY      float64
	HP           Health
	Alive        bool
	BurstTimer   float64
	PauseTimer   float64
	FireTimer    float64
	Burning      bool // true at low HP — crashes after BurnTimer reaches 3s
	BurnTimer    float64
}

type Missile struct {
	X, Y   float64
	VX, VY float64
	Life   float64
	Big    bool // tank shell = true, heli missile = false
	Homing bool
}

type Mine struct {
	X, Y     float64
	Alive    bool
	ArmTimer float64 // armed when >= 1.0
}

type MilTroop struct {
	X, Y          float64
	HP            Health
	Alive         bool
	ShootTimer    float64
	StuckTimer    float64
	PrevX, PrevY  float64
	WaypointX     float64
	WaypointY     float64
	WaypointTimer float64
	Kind          TroopKind
	GroupID       uint32
}

type MilitarySystem struct {
	Tanks    []Tank
	Helis    []MilHeli
	Missiles []Missile
	Mines    []Mine
	Troops   []MilTroop

	Active      bool
	ActiveTimer float64 // counts up while wanted is at max
	SpawnTimer  float64
	MineTimer   float64
	seed        uint64
	nextGroupID uint32
}

func NewMilitarySystem(seed uint64) *MilitarySystem {
	if seed == 0 {
		seed = 1
	}
	return &MilitarySystem{seed: seed, SpawnTimer: 5.0, MineTimer: 8.0, nextGroupID: 1}
}

func (ms *MilitarySystem) Reset() {
	ms.Tanks = ms.Tanks[:0]
	ms.Helis = ms.Helis[:0]
	ms.Missiles = ms.Missiles[:0]
	ms.Mines = ms.Mines[:0]
	ms.Troops = ms.Troops[:0]
	ms.Active = false
	ms.ActiveTimer = 0
	ms.SpawnTimer = 5.0
	ms.MineTimer = 8.0
}

func (ms *MilitarySystem) Update(dt float64, snake *Snake, world *World, ps *ParticleSystem, cam *Camera, now float64) {
	if snake == nil || !snake.Alive {
		return
	}
	hx, hy := snake.Head()

	// Count up while wanted is at max; activate military after delay.
	if snake.WantedLevel >= WantedMax {
		ms.ActiveTimer += dt
		if ms.ActiveTimer >= MilitaryActivationDelay {
			ms.Active = true
		}
	}
	if !ms.Active {
		return
	}

	// --- Tanks ---
	for ti := range ms.Tanks {
		t := &ms.Tanks[ti]
		if !t.Alive {
			continue
		}
		dx := hx - t.X
		dy := hy - t.Y
		dist := math.Hypot(dx, dy)

		// Steer tank body.
		if dist > 2.0 {
			targetH := math.Atan2(dy, dx)
			diff := angDiff(t.Heading, targetH)
			maxTurn := 1.2 * dt
			if math.Abs(diff) <= maxTurn {
				t.Heading = targetH
			} else if diff > 0 {
				t.Heading += maxTurn
			} else {
				t.Heading -= maxTurn
			}
		}

		nx := t.X + math.Cos(t.Heading)*t.Speed*dt
		ny := t.Y + math.Sin(t.Heading)*t.Speed*dt
		// Tanks plow through thin obstacles.
		if world.HeightAt(int(math.Round(nx)), int(math.Round(ny))) > 0 {
			if ps != nil {
				SpawnExplosionWithShockwave(int(math.Round(nx)), int(math.Round(ny)), RGB{100, 90, 70}, 0.15, 0, world, ps)
			}
		}
		t.X = clampF(nx, 0, float64(WorldWidth-1))
		t.Y = clampF(ny, 0, float64(WorldHeight-1))

		// Turret tracks snake head directly.
		t.TurretAng = math.Atan2(hy-t.Y, hx-t.X)

		// Fire shell.
		t.FireTimer -= dt
		if t.FireTimer <= 0 && dist < 80.0 {
			t.FireTimer = 3.5
			ang := math.Atan2(hy-t.Y, hx-t.X)
			ms.Missiles = append(ms.Missiles, Missile{
				X: t.X + math.Cos(ang)*3, Y: t.Y + math.Sin(ang)*3,
				VX: math.Cos(ang) * 90, VY: math.Sin(ang) * 90,
				Life: 1.2, Big: true,
			})
			if ps != nil {
				ps.Add(Particle{
					X: t.X + math.Cos(ang)*3.5, Y: t.Y + math.Sin(ang)*3.5,
					VX: math.Cos(ang) * 15, VY: math.Sin(ang) * 15,
					Size: 1.0, MaxLife: 0.08,
					Col: RGB{255, 200, 80}, Kind: ParticleGlow,
				})
			}
		}

		// Ram snake.
		if dist < float64(t.Size)*0.8 {
			if len(snake.Ghosts) == 0 {
				snake.HP.Damage(3.0)
			}
			t.Alive = false
			if ps != nil {
				SpawnExplosionWithShockwave(int(t.X), int(t.Y), RGB{80, 100, 50}, 1.0, 0, world, ps)
			}
			if cam != nil {
				cam.AddShake(0.8, 0.5)
			}
		}
	}

	// --- Military helicopters ---
	for i := range ms.Helis {
		h := &ms.Helis[i]
		if !h.Alive {
			continue
		}
		// Lerp orbit center toward snake head at a fixed speed (independent of snake speed).
		const milHeliCenterSpeed = 30.0
		cdx := hx - h.CenterX
		cdy := hy - h.CenterY
		cd := math.Hypot(cdx, cdy)
		if cd > milHeliCenterSpeed*dt {
			h.CenterX += cdx / cd * milHeliCenterSpeed * dt
			h.CenterY += cdy / cd * milHeliCenterSpeed * dt
		} else {
			h.CenterX, h.CenterY = hx, hy
		}

		h.CircleAngle += 0.55 * dt
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
				rr := NewRand(ms.seed ^ uint64(i+1)*0xF1AE ^ uint64(h.BurnTimer*100))
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
				h.FireTimer = 0.5
				ang := math.Atan2(hy-h.Y, hx-h.X)
				ms.Missiles = append(ms.Missiles, Missile{
					X: h.X, Y: h.Y,
					VX: math.Cos(ang) * 70, VY: math.Sin(ang) * 70,
					Life: 1.5, Homing: true,
				})
			}
			if h.BurstTimer <= 0 {
				h.PauseTimer = 3.0 + NewRand(ms.seed^uint64(i)*0xF1A1^uint64(now*10)).RangeF(0, 2.0)
			}
		} else {
			h.PauseTimer -= dt
			if h.PauseTimer <= 0 {
				h.BurstTimer = 2.0
				h.FireTimer = 0
			}
		}
	}

	// --- Missiles (tank shells + heli missiles) ---
	for mi := len(ms.Missiles) - 1; mi >= 0; mi-- {
		m := &ms.Missiles[mi]
		m.Life -= dt

		// Homing: steer toward snake.
		if m.Homing {
			dx := hx - m.X
			dy := hy - m.Y
			d := math.Hypot(dx, dy)
			if d > 1.0 {
				targetAng := math.Atan2(dy, dx)
				curAng := math.Atan2(m.VY, m.VX)
				diff := angDiff(curAng, targetAng)
				maxTurn := 2.0 * dt
				if math.Abs(diff) <= maxTurn {
					curAng = targetAng
				} else if diff > 0 {
					curAng += maxTurn
				} else {
					curAng -= maxTurn
				}
				spd := math.Hypot(m.VX, m.VY)
				m.VX = math.Cos(curAng) * spd
				m.VY = math.Sin(curAng) * spd
			}
		}

		m.X += m.VX * dt
		m.Y += m.VY * dt
		ix := int(math.Round(m.X))
		iy := int(math.Round(m.Y))

		// Smoke trail: spawn behind missile each frame.
		if ps != nil {
			spd := math.Hypot(m.VX, m.VY)
			smokeX := m.X - (m.VX/spd)*1.5
			smokeY := m.Y - (m.VY/spd)*1.5
			smokeSize := float64(0.7)
			if m.Big {
				smokeSize = 1.2
			}
			ps.Add(Particle{
				X: smokeX, Y: smokeY,
				VX:   (m.VX/spd)*-2 + float64(NewRand(uint64(ix*7+iy*13)^uint64(m.Life*1000)).Range(-3, 3)),
				VY:   (m.VY/spd)*-2 + float64(NewRand(uint64(ix*11+iy*7)^uint64(m.Life*999)).Range(-3, 3)),
				Size: smokeSize, MaxLife: 0.35,
				Col: RGB{R: 120, G: 115, B: 110}, Kind: ParticleSmoke,
			})
		}

		hit := false
		if ix < 0 || iy < 0 || ix >= WorldWidth || iy >= WorldHeight {
			hit = true
		} else if world.HeightAt(ix, iy) > 0 {
			radius := 4
			if m.Big {
				radius = 7
			}
			ExplodeAt(ix, iy, radius, world, ps, nil, nil, cam, nil, nil)
			hit = true
		} else if math.Hypot(m.X-hx, m.Y-hy) < 3.0 {
			damage := 1.0
			if m.Big {
				damage = 3.0
			}
			if len(snake.Ghosts) == 0 {
				snake.HP.Damage(damage)
			}
			radius := 4
			if m.Big {
				radius = 7
			}
			ExplodeAt(ix, iy, radius, world, ps, nil, nil, cam, nil, nil)
			if cam != nil {
				cam.AddShake(0.4, 0.25)
			}
			hit = true
		} else if m.Life <= 0 {
			if ps != nil {
				SpawnExplosionWithShockwave(ix, iy, RGB{150, 110, 60}, 0.25, 0, world, ps)
			}
			hit = true
		}

		if hit {
			ms.Missiles[mi] = ms.Missiles[len(ms.Missiles)-1]
			ms.Missiles = ms.Missiles[:len(ms.Missiles)-1]
		}
	}

	// --- Mines ---
	for i := range ms.Mines {
		mine := &ms.Mines[i]
		if !mine.Alive {
			continue
		}
		if mine.ArmTimer < 1.0 {
			mine.ArmTimer += dt
			continue
		}
		if math.Hypot(mine.X-hx, mine.Y-hy) < 3.5 {
			mine.Alive = false
			ix := int(math.Round(mine.X))
			iy := int(math.Round(mine.Y))
			ExplodeAt(ix, iy, 5, world, ps, nil, nil, cam, nil, nil)
			if len(snake.Ghosts) == 0 {
				snake.HP.Damage(2.5)
			}
			if cam != nil {
				cam.AddShake(0.5, 0.3)
			}
		}
	}

	// --- Troops ---
	for i := range ms.Troops {
		t := &ms.Troops[i]
		if !t.Alive {
			continue
		}
		dx := hx - t.X
		dy := hy - t.Y
		dist := math.Hypot(dx, dy)

		// Waypoint-based marching: all troops in same group share the same
		// general direction, updated periodically to create a marching feel.
		t.WaypointTimer -= dt
		if t.WaypointTimer <= 0 || math.Hypot(t.WaypointX-t.X, t.WaypointY-t.Y) < 3.0 {
			t.WaypointTimer = 2.5
			r := NewRand(ms.seed ^ uint64(i)*0xD34D ^ uint64(now*100))
			offset := 10.0
			t.WaypointX = hx + r.RangeF(-offset, offset)
			t.WaypointY = hy + r.RangeF(-offset, offset)
		}

		// Movement: kind-specific.
		var moveX, moveY, spd float64
		switch t.Kind {
		case TroopSniper:
			// Hold ~35px range.
			const pref = 35.0
			if dist < pref-5 && dist > 0.1 {
				moveX, moveY = -dx/dist, -dy/dist
			} else if dist > pref+5 {
				wx := t.WaypointX - t.X
				wy := t.WaypointY - t.Y
				if wd := math.Hypot(wx, wy); wd > 1.0 {
					moveX, moveY = wx/wd, wy/wd
				}
			}
			spd = 12.0
		case TroopMinigun:
			// Advance to medium range via waypoint.
			const pref = 15.0
			if dist > pref+5 {
				wx := t.WaypointX - t.X
				wy := t.WaypointY - t.Y
				if wd := math.Hypot(wx, wy); wd > 1.0 {
					moveX, moveY = wx/wd, wy/wd
				}
			}
			spd = 14.0
		default: // TroopRegular — march in formation via waypoint
			wx := t.WaypointX - t.X
			wy := t.WaypointY - t.Y
			if wd := math.Hypot(wx, wy); wd > 1.5 {
				moveX, moveY = wx/wd, wy/wd
			}
			spd = 16.0
		}

		if moveX != 0 || moveY != 0 {
			nx := t.X + moveX*spd*dt
			ny := t.Y + moveY*spd*dt
			// Building collision: only move if destination is clear.
			if world.HeightAt(int(math.Round(nx)), int(math.Round(ny))) > 0 {
				// Blocked: pick a new walkable position.
				r := NewRand(ms.seed ^ uint64(i)*0xC0DE ^ uint64(t.StuckTimer*100))
				for range 12 {
					tx := int(math.Round(t.X)) + r.Range(-8, 8)
					ty := int(math.Round(t.Y)) + r.Range(-8, 8)
					if tx >= 0 && ty >= 0 && tx < WorldWidth && ty < WorldHeight && world.HeightAt(tx, ty) == 0 {
						t.X = float64(tx) + 0.5
						t.Y = float64(ty) + 0.5
						break
					}
				}
			} else {
				t.X = nx
				t.Y = ny
			}
			if moved := math.Hypot(t.X-t.PrevX, t.Y-t.PrevY); moved < 0.05*dt {
				t.StuckTimer += dt
				if t.StuckTimer > 2.0 {
					r := NewRand(ms.seed ^ uint64(i)*0xBEEF ^ uint64(t.StuckTimer*100))
					for range 20 {
						tx := int(math.Round(hx)) + r.Range(-20, 20)
						ty := int(math.Round(hy)) + r.Range(-20, 20)
						if tx >= 0 && ty >= 0 && tx < WorldWidth && ty < WorldHeight && world.HeightAt(tx, ty) == 0 {
							t.X = float64(tx)
							t.Y = float64(ty)
							t.StuckTimer = 0
							break
						}
					}
				}
			} else {
				t.StuckTimer = 0
			}
			t.PrevX, t.PrevY = t.X, t.Y
		}

		// Shoot at snake based on kind.
		var shootInterval, shootRange, shootDamage float64
		switch t.Kind {
		case TroopSniper:
			shootInterval, shootRange, shootDamage = 3.0, 60.0, 0.8
		case TroopMinigun:
			shootInterval, shootRange, shootDamage = 0.1, 22.0, 0.15
		default:
			shootInterval, shootRange, shootDamage = 1.0, 40.0, 0.3
		}
		t.ShootTimer -= dt
		if t.ShootTimer <= 0 && dist < shootRange && HasLineOfSight(t.X, t.Y, hx, hy, world) {
			t.ShootTimer = shootInterval
			if ps != nil {
				ang := math.Atan2(dy, dx)
				ps.Add(Particle{
					X: t.X, Y: t.Y,
					VX: math.Cos(ang) * 90, VY: math.Sin(ang) * 90,
					Size: 0.4, MaxLife: 0.3,
					Col: RGB{R: 255, G: 200, B: 50}, Kind: ParticleGlow,
				})
			}
			if dist < shootRange*0.9 && len(snake.Ghosts) == 0 {
				snake.HP.Damage(shootDamage)
			}
		}

		if dist < SnakeEatRadius {
			if len(snake.Ghosts) == 0 {
				snake.HP.Damage(0.3)
			}
			// Blood splatter when eaten.
			if ps != nil {
				ang := math.Atan2(t.Y-hy, t.X-hx)
				ps.SpawnBlood(t.X, t.Y, math.Cos(ang), math.Sin(ang), 18, 0.9)
				ps.SpawnBlood(t.X+0.3, t.Y+0.3, math.Cos(ang+math.Pi/2), math.Sin(ang+math.Pi/2), 9, 0.55)
			}
			t.Alive = false
		}
	}

	// Spawn reinforcements.
	ms.SpawnTimer -= dt
	if ms.SpawnTimer <= 0 {
		r := NewRand(ms.seed ^ uint64(now*1000) ^ uint64(len(ms.Troops)*0xFAB))
		ms.SpawnTimer = 5.0 + r.RangeF(0, 3.0)
		ms.spawnReinforcements(snake, world, r)
	}

	// Drop mines periodically.
	ms.MineTimer -= dt
	if ms.MineTimer <= 0 {
		r := NewRand(ms.seed ^ uint64(now*773) ^ uint64(len(ms.Mines)*0xA4E))
		ms.MineTimer = 6.0 + r.RangeF(0, 4.0)
		for range 30 {
			ang := r.RangeF(0, 2*math.Pi)
			d := r.RangeF(15, 40)
			mx := hx + math.Cos(ang)*d
			my := hy + math.Sin(ang)*d
			ix := int(math.Round(mx))
			iy := int(math.Round(my))
			if ix >= 0 && iy >= 0 && ix < WorldWidth && iy < WorldHeight && world.HeightAt(ix, iy) == 0 {
				ms.Mines = append(ms.Mines, Mine{X: mx, Y: my, Alive: true})
				break
			}
		}
	}
}

func (ms *MilitarySystem) spawnReinforcements(snake *Snake, world *World, r *Rand) {
	hx, hy := snake.Head()

	// Tanks: scale with how long military has been active.
	maxTanks := 1
	if ms.ActiveTimer > 60 {
		maxTanks = 2
	}
	if ms.ActiveTimer > 120 {
		maxTanks = 3
	}
	aliveTanks := 0
	for _, t := range ms.Tanks {
		if t.Alive {
			aliveTanks++
		}
	}
	if aliveTanks < maxTanks {
		sx, sy := edgeSpawnPos(hx, hy, r)
		ms.Tanks = append(ms.Tanks, Tank{
			X: sx, Y: sy,
			Heading:   math.Atan2(hy-sy, hx-sx),
			Speed:     12.0 + r.RangeF(0, 5.0),
			HP:        NewHealth(50.0),
			Alive:     true,
			Size:      float32(CarSize * 1.5),
			FireTimer: 2.0 + r.RangeF(0, 2.0),
		})
	}

	// Helis: 1-2.
	maxHelis := 1
	if ms.ActiveTimer > 60 {
		maxHelis = 2
	}
	aliveHelis := 0
	for _, h := range ms.Helis {
		if h.Alive {
			aliveHelis++
		}
	}
	if aliveHelis < maxHelis {
		radius := 32.0 + r.RangeF(0, 8.0)
		ang := r.RangeF(0, 2*math.Pi)
		ms.Helis = append(ms.Helis, MilHeli{
			X: hx + math.Cos(ang)*radius, Y: hy + math.Sin(ang)*radius,
			CenterX:      hx,
			CenterY:      hy,
			CircleAngle:  ang,
			CircleRadius: radius,
			HP:           NewHealth(30.0),
			Alive:        true,
			PauseTimer:   2.0 + r.RangeF(0, 2.0),
		})
	}

	// Troops: squads of 3-4, mix of kinds.
	maxTroops := 4
	if ms.ActiveTimer > 60 {
		maxTroops = 8
	}
	if ms.ActiveTimer > 120 {
		maxTroops = 12
	}
	aliveTroops := 0
	for _, t := range ms.Troops {
		if t.Alive {
			aliveTroops++
		}
	}
	if aliveTroops < maxTroops {
		sx, sy := edgeSpawnPos(hx, hy, r)
		ms.nextGroupID++

		kind := TroopRegular
		roll := r.Intn(5)
		if roll == 0 {
			kind = TroopSniper
		} else if roll == 1 {
			kind = TroopMinigun
		}

		groupSize := 3 + r.Intn(2)
		for i := range groupSize {
			for range 20 {
				tx := int(math.Round(sx)) + r.Range(-5, 5)
				ty := int(math.Round(sy)) + r.Range(-5, 5)
				if tx >= 0 && ty >= 0 && tx < WorldWidth && ty < WorldHeight && world.HeightAt(tx, ty) == 0 {
					hp := 8.0
					if kind == TroopMinigun {
						hp = 12.0
					}
					ms.Troops = append(ms.Troops, MilTroop{
						X: float64(tx), Y: float64(ty),
						HP:            NewHealth(hp),
						Alive:         true,
						ShootTimer:    float64(i)*0.2 + r.RangeF(0, 0.5),
						Kind:          kind,
						GroupID:       ms.nextGroupID,
						WaypointX:     hx,
						WaypointY:     hy,
						WaypointTimer: 0,
					})
					break
				}
			}
		}
	}
}

func (ms *MilitarySystem) RemoveDead() {
	for i := 0; i < len(ms.Tanks); {
		if !ms.Tanks[i].Alive {
			ms.Tanks[i] = ms.Tanks[len(ms.Tanks)-1]
			ms.Tanks = ms.Tanks[:len(ms.Tanks)-1]
		} else {
			i++
		}
	}
	for i := 0; i < len(ms.Helis); {
		if !ms.Helis[i].Alive {
			ms.Helis[i] = ms.Helis[len(ms.Helis)-1]
			ms.Helis = ms.Helis[:len(ms.Helis)-1]
		} else {
			i++
		}
	}
	for i := 0; i < len(ms.Mines); {
		if !ms.Mines[i].Alive {
			ms.Mines[i] = ms.Mines[len(ms.Mines)-1]
			ms.Mines = ms.Mines[:len(ms.Mines)-1]
		} else {
			i++
		}
	}
	for i := 0; i < len(ms.Troops); {
		if !ms.Troops[i].Alive {
			ms.Troops[i] = ms.Troops[len(ms.Troops)-1]
			ms.Troops = ms.Troops[:len(ms.Troops)-1]
		} else {
			i++
		}
	}
}

// RenderData returns point sprites for all military entities.
func (ms *MilitarySystem) RenderData(now float64) []float32 {
	buf := make([]float32, 0, 512)

	// Tanks: camo body + tracks + turret.
	for _, t := range ms.Tanks {
		if !t.Alive {
			continue
		}
		tx32, ty32 := float32(t.X), float32(t.Y)
		fwdX := math.Cos(t.Heading)
		fwdY := math.Sin(t.Heading)
		perpX := -math.Sin(t.Heading)
		perpY := math.Cos(t.Heading)

		buf = append(buf, tx32, ty32, 5.5, 0.25, 0.32, 0.18, 1.0, 0) // main body
		buf = append(buf,
			float32(t.X+fwdX*2.5), float32(t.Y+fwdY*2.5),
			3.5, 0.22, 0.28, 0.15, 1.0, 0) // front plate
		buf = append(buf,
			float32(t.X+perpX*2.8), float32(t.Y+perpY*2.8),
			2.2, 0.15, 0.18, 0.12, 1.0, 0) // left track
		buf = append(buf,
			float32(t.X-perpX*2.8), float32(t.Y-perpY*2.8),
			2.2, 0.15, 0.18, 0.12, 1.0, 0) // right track
		buf = append(buf, tx32, ty32, 2.5, 0.3, 0.38, 0.22, 1.0, 0) // turret dome
		// Barrel (line from turret toward target).
		buf = append(buf,
			float32(t.X+math.Cos(t.TurretAng)*3.8),
			float32(t.Y+math.Sin(t.TurretAng)*3.8),
			0.9, 0.22, 0.28, 0.16, 1.0, 0)
		buf = append(buf,
			float32(t.X+math.Cos(t.TurretAng)*2.0),
			float32(t.Y+math.Sin(t.TurretAng)*2.0),
			1.1, 0.22, 0.28, 0.16, 1.0, 0)
	}

	// Military helicopters: green body, same shape as cop helis.
	for _, h := range ms.Helis {
		if !h.Alive {
			continue
		}
		fwdX := math.Cos(h.Heading)
		fwdY := math.Sin(h.Heading)
		perpX := -math.Sin(h.Heading)
		perpY := math.Cos(h.Heading)
		hx32, hy32 := float32(h.X), float32(h.Y)

		buf = append(buf, hx32, hy32, 3.5, 0.25, 0.5, 0.22, 1.0, 0) // green body
		buf = append(buf,
			float32(h.X+fwdX*1.8), float32(h.Y+fwdY*1.8),
			2.2, 0.18, 0.42, 0.16, 1.0, 0) // cockpit
		for step := 1; step <= 3; step++ {
			sz := float32(1.6 - float32(step)*0.25)
			buf = append(buf,
				float32(h.X-fwdX*float64(step)*1.3),
				float32(h.Y-fwdY*float64(step)*1.3),
				sz, 0.2, 0.4, 0.18, 1.0, 0)
		}
		tailX := h.X - fwdX*4.5
		tailY := h.Y - fwdY*4.5
		buf = append(buf, float32(tailX+perpX*1.1), float32(tailY+perpY*1.1), 1.1, 0.22, 0.44, 0.2, 1.0, 0)
		buf = append(buf, float32(tailX-perpX*1.1), float32(tailY-perpY*1.1), 1.1, 0.22, 0.44, 0.2, 1.0, 0)
		buf = append(buf, float32(tailX), float32(tailY), 1.0, 0.18, 0.36, 0.16, 1.0, 0)

		const bladeR = 3.2
		rotX := math.Cos(h.RotorAngle)
		rotY := math.Sin(h.RotorAngle)
		buf = append(buf, float32(h.X+rotX*bladeR), float32(h.Y+rotY*bladeR), 1.0, 0.4, 0.7, 0.35, 0.85, 0)
		buf = append(buf, float32(h.X+rotX*bladeR*0.5), float32(h.Y+rotY*bladeR*0.5), 1.0, 0.38, 0.65, 0.32, 0.9, 0)
		buf = append(buf, float32(h.X-rotX*bladeR*0.5), float32(h.Y-rotY*bladeR*0.5), 1.0, 0.38, 0.65, 0.32, 0.9, 0)
		buf = append(buf, float32(h.X-rotX*bladeR), float32(h.Y-rotY*bladeR), 1.0, 0.4, 0.7, 0.35, 0.85, 0)
	}

	// Missiles: orange tracer.
	for _, m := range ms.Missiles {
		sz := float32(0.9)
		if m.Big {
			sz = 1.3
		}
		buf = append(buf, float32(m.X), float32(m.Y), sz, 1.0, 0.55, 0.1, 1.0, 0)
	}

	// Mines: small blinking dot.
	for _, mine := range ms.Mines {
		if !mine.Alive {
			continue
		}
		r, g, b, a := float32(0.2), float32(0.2), float32(0.15), float32(0.7)
		if mine.ArmTimer >= 1.0 && int(now*4)%2 == 0 {
			r, g, b, a = 0.85, 0.1, 0.1, 1.0
		}
		buf = append(buf, float32(mine.X), float32(mine.Y), 1.2, r, g, b, a, 0)
	}

	// Troops: green uniform, kind-coded shade.
	for _, t := range ms.Troops {
		if !t.Alive {
			continue
		}
		var ur, ug, ub float32
		switch t.Kind {
		case TroopSniper:
			ur, ug, ub = 0.35, 0.6, 0.28 // lighter olive
		case TroopMinigun:
			ur, ug, ub = 0.12, 0.32, 0.1 // dark forest green
		default:
			ur, ug, ub = 0.22, 0.48, 0.2 // standard camo green
		}
		buf = append(buf, float32(t.X), float32(t.Y), 1.5, ur, ug, ub, 1.0, 0)
		buf = append(buf, float32(t.X), float32(t.Y)-0.8, 0.7, 0.85, 0.8, 0.65, 1.0, 0) // tan head
	}

	return buf
}

// GlowData returns additive glow for military entities.
func (ms *MilitarySystem) GlowData(now float64) []float32 {
	buf := make([]float32, 0, 128)

	// Tank muzzle flash when freshly fired.
	for _, t := range ms.Tanks {
		if !t.Alive {
			continue
		}
		if t.FireTimer < 0.12 {
			buf = append(buf,
				float32(t.X+math.Cos(t.TurretAng)*4.0),
				float32(t.Y+math.Sin(t.TurretAng)*4.0),
				6.0, 0.7, 0.45, 0.05, 1, 0)
		}
	}

	// Heli green/red beacon (alternating).
	for _, h := range ms.Helis {
		if !h.Alive {
			continue
		}
		if int(now*8)%2 == 0 {
			buf = append(buf, float32(h.X), float32(h.Y)-1.2, 1.5, 0.1, 0.9, 0.1, 1, 0)
			buf = append(buf, float32(h.X), float32(h.Y)-1.2, 5.0, 0.02, 0.25, 0.02, 1, 0)
		} else {
			buf = append(buf, float32(h.X), float32(h.Y)-1.2, 1.5, 0.9, 0.1, 0.1, 1, 0)
			buf = append(buf, float32(h.X), float32(h.Y)-1.2, 5.0, 0.22, 0.02, 0.02, 1, 0)
		}
	}

	// Missile exhaust glow.
	for _, m := range ms.Missiles {
		sz := float32(3.5)
		if m.Big {
			sz = 5.0
		}
		buf = append(buf, float32(m.X), float32(m.Y), sz, 0.45, 0.22, 0.02, 1, 0)
	}

	// Armed mine blink.
	for _, mine := range ms.Mines {
		if !mine.Alive || mine.ArmTimer < 1.0 {
			continue
		}
		if int(now*4)%2 == 0 {
			buf = append(buf, float32(mine.X), float32(mine.Y), 3.5, 0.5, 0.02, 0.02, 1, 0)
		}
	}

	return buf
}

// ExplodeAffectMilitary damages military entities within an explosion radius.
func ExplodeAffectMilitary(wx, wy, radius int, w *World, ps *ParticleSystem, mil *MilitarySystem) {
	if mil == nil || radius <= 0 {
		return
	}
	fwx, fwy, fr := float64(wx), float64(wy), float64(radius)

	for i := range mil.Tanks {
		t := &mil.Tanks[i]
		if !t.Alive {
			continue
		}
		d := math.Hypot(t.X-fwx, t.Y-fwy)
		if d < fr*1.5 {
			t.HP.Damage((1.0 - d/(fr*1.5)) * 30.0)
			if t.HP.IsDead() {
				t.Alive = false
				if ps != nil {
					SpawnExplosionWithShockwave(int(t.X), int(t.Y), RGB{80, 100, 50}, 1.2, 0, w, ps)
				}
			}
		}
	}

	for i := range mil.Helis {
		h := &mil.Helis[i]
		if !h.Alive {
			continue
		}
		d := math.Hypot(h.X-fwx, h.Y-fwy)
		if d < fr*2.0 {
			h.HP.Damage((1.0 - d/(fr*2.0)) * 22.0)
			if h.HP.IsDead() && !h.Burning {
				h.Burning = true // starts burning; crashes after BurnTimer reaches 3s
			}
		}
	}

	for i := range mil.Troops {
		t := &mil.Troops[i]
		if !t.Alive {
			continue
		}
		if math.Hypot(t.X-fwx, t.Y-fwy) < fr {
			t.Alive = false
		}
	}

	// Chain-detonate nearby armed mines.
	for i := range mil.Mines {
		mine := &mil.Mines[i]
		if !mine.Alive || mine.ArmTimer < 1.0 {
			continue
		}
		if math.Hypot(mine.X-fwx, mine.Y-fwy) < fr*1.2 {
			mine.Alive = false
		}
	}
}
