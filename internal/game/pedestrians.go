package game

import "math"

var (
	pedUniformSize float32 = 1.1

	clothPalette = []RGB{
		{R: 230, G: 40, B: 60}, {R: 40, G: 120, B: 235}, {R: 255, G: 200, B: 60},
		{R: 60, G: 200, B: 90}, {R: 180, G: 60, B: 200}, {R: 255, G: 110, B: 50},
	}
	skinPalette = []RGB{
		{R: 160, G: 110, B: 75}, {R: 194, G: 134, B: 96},
		{R: 224, G: 172, B: 130}, {R: 245, G: 205, B: 170},
	}

	sickTint = RGB{R: 80, G: 220, B: 30} // vivid neon green for infected peds
)

type PedVariant int

const (
	PedVariantCivilian PedVariant = iota
	PedVariantArctic
	PedVariantDesert
	PedVariantAstronaut
	PedVariantDiver
)

type Pedestrian struct {
	X, Y             float64
	VX, VY           float64
	TargetX, TargetY float64
	FacingX, FacingY float64
	WalkCycle        float64
	Speed            float64
	BaseSpeed        float64
	Col              RGB
	OrigCol          RGB
	Skin             RGB
	Phase            float64
	Size             float32
	Crossing         bool
	Alive            bool
	HP               Health

	// Stuck detection.
	PrevX, PrevY float64
	StuckTimer   float64

	// Grouping.
	GroupID  uint64
	IsLeader bool
	OffsetX  float64
	OffsetY  float64

	// Snake awareness.
	Fleeing bool

	// Infection visual (10% spawn symptomatic — bad to eat, won't spread).
	Infection InfectionState

	// Armed pedestrian: shoots at snake.
	Armed         bool
	ShootCooldown float64

	// Themed entity variant (arctic, desert, space, underwater, etc.).
	Variant PedVariant
}

type PedestrianSystem struct {
	P    []Pedestrian
	seed uint64
	Env  string

	groupLeader map[uint64]int
	nextGroupID uint64

	// Spatial buckets for avoidance.
	bucketSize int
	bucketCols int
	bucketRows int
	buckets    [][]int
}

func NewPedestrianSystem(maxPeds int, seed uint64) *PedestrianSystem {
	if maxPeds <= 0 {
		maxPeds = 200
	}
	if seed == 0 {
		seed = 1
	}
	ps := &PedestrianSystem{
		P:           make([]Pedestrian, 0, maxPeds),
		seed:        seed,
		Env:         ThemeCity.Name,
		groupLeader: make(map[uint64]int),
		nextGroupID: seed + 1,
		bucketSize:  16,
	}
	ps.bucketCols = (WorldWidth + ps.bucketSize - 1) / ps.bucketSize
	ps.bucketRows = (WorldHeight + ps.bucketSize - 1) / ps.bucketSize
	ps.buckets = make([][]int, ps.bucketCols*ps.bucketRows)
	return ps
}

func (ps *PedestrianSystem) SetEnvironment(name string) {
	if name == "" {
		name = ThemeCity.Name
	}
	ps.Env = name
}

func (ps *PedestrianSystem) pickVariant(r *Rand) PedVariant {
	switch ps.Env {
	case ThemeArctic.Name, ThemeWinter.Name, ThemeForestWinter.Name:
		return PedVariantArctic
	case ThemeDesert.Name, ThemeSand.Name:
		return PedVariantDesert
	case ThemeSpace.Name:
		if r.Intn(100) < 65 {
			return PedVariantAstronaut
		}
		return PedVariantCivilian
	case ThemeUnderwater.Name:
		if r.Intn(100) < 70 {
			return PedVariantDiver
		}
		return PedVariantCivilian
	default:
		return PedVariantCivilian
	}
}

func colorFromPalette(p []RGB, r *Rand) RGB {
	if len(p) == 0 {
		return RGB{R: 200, G: 200, B: 200}
	}
	base := p[r.Intn(len(p))]
	return base.Add(r.Range(-16, 16), r.Range(-16, 16), r.Range(-16, 16))
}

func pedVariantColor(v PedVariant, r *Rand) RGB {
	switch v {
	case PedVariantArctic:
		return colorFromPalette([]RGB{
			{R: 210, G: 235, B: 255},
			{R: 165, G: 205, B: 240},
			{R: 235, G: 240, B: 248},
		}, r)
	case PedVariantDesert:
		return colorFromPalette([]RGB{
			{R: 220, G: 165, B: 95},
			{R: 185, G: 125, B: 70},
			{R: 240, G: 195, B: 120},
		}, r)
	case PedVariantAstronaut:
		return colorFromPalette([]RGB{
			{R: 205, G: 220, B: 240},
			{R: 160, G: 210, B: 235},
			{R: 220, G: 180, B: 245},
		}, r)
	case PedVariantDiver:
		return colorFromPalette([]RGB{
			{R: 80, G: 160, B: 190},
			{R: 65, G: 130, B: 180},
			{R: 95, G: 190, B: 210},
		}, r)
	default:
		return clothPalette[r.Intn(len(clothPalette))].Add(int(r.RangeF(-18, 18)), int(r.RangeF(-18, 18)), int(r.RangeF(-18, 18)))
	}
}

func applyVariantTuning(p *Pedestrian) {
	switch p.Variant {
	case PedVariantArctic:
		p.Speed *= 0.95
		p.BaseSpeed = p.Speed
		p.HP.Heal(0.4)
	case PedVariantDesert:
		p.Speed *= 1.14
		p.BaseSpeed = p.Speed
	case PedVariantAstronaut:
		p.Speed *= 0.90
		p.BaseSpeed = p.Speed
		p.HP.Heal(1.1)
	case PedVariantDiver:
		p.Speed *= 0.82
		p.BaseSpeed = p.Speed
		p.HP.Heal(0.8)
	}
}

// pedWalkable returns true if a pedestrian can walk on this tile.
func pedWalkable(w *World, wx, wy int) bool {
	if wx < 0 || wy < 0 || wx >= WorldWidth || wy >= WorldHeight {
		return false
	}
	col := w.ColorAt(wx, wy)
	if rgbEq(col, Palette.Road) || rgbEq(col, Palette.Border) {
		return false
	}
	return w.HeightAt(wx, wy) == 0
}

func (ps *PedestrianSystem) SpawnRandom(w *World, n int) {
	if w == nil || n <= 0 {
		return
	}
	r := NewRand(ps.seed)
	attempts := 0
	for len(ps.P) < n && attempts < n*12 {
		attempts++
		x := r.Range(0, WorldWidth-1)
		y := r.Range(0, WorldHeight-1)
		if !pedWalkable(w, x, y) {
			continue
		}

		hp := 1.0 + r.RangeF(0, 2.0)
		spd := 10.0 + r.RangeF(0, 8.0)
		variant := ps.pickVariant(r)
		col := pedVariantColor(variant, r)
		p := Pedestrian{
			X:         float64(x) + 0.5,
			Y:         float64(y) + 0.5,
			Speed:     spd,
			BaseSpeed: spd,
			Col:       col,
			OrigCol:   col,
			Skin:      skinPalette[r.Intn(len(skinPalette))],
			Phase:     r.RangeF(0, 1),
			Size:      pedUniformSize,
			Alive:     true,
			HP:        NewHealth(hp),
			Infection: StateHealthy,
			Variant:   variant,
			FacingY:   1,
			WalkCycle: r.RangeF(0, 2),
			PrevX:     float64(x) + 0.5,
			PrevY:     float64(y) + 0.5,
		}
		applyVariantTuning(&p)

		// Pick a nearby walkable target biased towards sidewalks.
		bestScore := -1.0
		bestX, bestY := x, y
		for s := 0; s < 12; s++ {
			tx := x + r.Range(-8, 8)
			ty := y + r.Range(-8, 8)
			if !pedWalkable(w, tx, ty) {
				continue
			}
			score := 0.5
			tcol := w.ColorAt(tx, ty)
			if rgbEq(tcol, Palette.Sidewalk) {
				score += 1.2
			}
			if rgbEq(tcol, Palette.Grass) || rgbEq(tcol, Palette.GrassPatch) {
				score += 0.8
			}
			if score > bestScore {
				bestScore = score
				bestX, bestY = tx, ty
			}
		}
		p.TargetX = float64(bestX) + 0.5
		p.TargetY = float64(bestY) + 0.5

		// Occasionally spawn a small group.
		if len(ps.P) < n && r.RangeF(0, 1) < 0.14 {
			gid := ps.nextGroupID
			ps.nextGroupID++
			p.GroupID = gid
			p.IsLeader = true
			ps.P = append(ps.P, p)

			groupSize := 1 + r.Range(0, 3)
			for g := 0; g < groupSize && len(ps.P) < n; g++ {
				offx := r.RangeF(-1.6, 1.6)
				offy := r.RangeF(-1.6, 1.6)
				fp := p
				fp.X = p.X + offx
				fp.Y = p.Y + offy
				fp.GroupID = gid
				fp.IsLeader = false
				fp.OffsetX = -offx
				fp.OffsetY = -offy
				fp.Speed = fp.Speed * (0.9 + r.RangeF(0, 0.2))
				fp.BaseSpeed = fp.Speed
				fp.PrevX = fp.X
				fp.PrevY = fp.Y
				ps.P = append(ps.P, fp)
			}
		} else {
			ps.P = append(ps.P, p)
		}
	}
}

// SpawnArmed spawns n armed pedestrians who can shoot at the snake.
func (ps *PedestrianSystem) SpawnArmed(w *World, n int) {
	if w == nil || n <= 0 {
		return
	}
	r := NewRand(ps.seed ^ 0xA33D)
	attempts := 0
	spawned := 0
	for spawned < n && attempts < n*12 {
		attempts++
		x := r.Range(0, WorldWidth-1)
		y := r.Range(0, WorldHeight-1)
		if !pedWalkable(w, x, y) {
			continue
		}
		hp := 1.5 + r.RangeF(0, 2.0)
		spd := 9.0 + r.RangeF(0, 6.0)
		variant := ps.pickVariant(r)
		col := pedVariantColor(variant, r)
		p := Pedestrian{
			X: float64(x) + 0.5, Y: float64(y) + 0.5,
			TargetX: float64(x) + 0.5, TargetY: float64(y) + 0.5,
			Speed: spd, BaseSpeed: spd,
			Col: col, OrigCol: col,
			Skin:      skinPalette[r.Intn(len(skinPalette))],
			Phase:     r.RangeF(0, 1),
			Size:      pedUniformSize,
			Alive:     true,
			HP:        NewHealth(hp),
			Infection: StateHealthy,
			Armed:     true,
			Variant:   variant,
			FacingY:   1,
			WalkCycle: r.RangeF(0, 2),
			PrevX:     float64(x) + 0.5,
			PrevY:     float64(y) + 0.5,
		}
		applyVariantTuning(&p)
		if p.Variant == PedVariantDesert {
			p.ShootCooldown = 0.6 + r.RangeF(0, 0.4)
		}
		ps.P = append(ps.P, p)
		spawned++
	}
}

// SpawnInfected spawns n peds pre-marked as symptomatic (green, bad to eat).
func (ps *PedestrianSystem) SpawnInfected(w *World, n int) {
	if w == nil || n <= 0 {
		return
	}
	r := NewRand(ps.seed ^ 0x1AFEC7)
	attempts := 0
	spawned := 0
	for spawned < n && attempts < n*12 {
		attempts++
		x := r.Range(0, WorldWidth-1)
		y := r.Range(0, WorldHeight-1)
		if !pedWalkable(w, x, y) {
			continue
		}
		hp := 1.0 + r.RangeF(0, 1.5)
		spd := 8.0 + r.RangeF(0, 5.0)
		variant := ps.pickVariant(r)
		origCol := pedVariantColor(variant, r)
		col := lerpRGB(origCol, sickTint, 0.85)
		p := Pedestrian{
			X: float64(x) + 0.5, Y: float64(y) + 0.5,
			TargetX: float64(x) + 0.5, TargetY: float64(y) + 0.5,
			Speed: spd, BaseSpeed: spd,
			Col: col, OrigCol: origCol,
			Skin:      skinPalette[r.Intn(len(skinPalette))],
			Phase:     r.RangeF(0, 1),
			Size:      pedUniformSize,
			Alive:     true,
			HP:        NewHealth(hp),
			Infection: StateSymptomatic,
			Variant:   variant,
			FacingY:   1,
			WalkCycle: r.RangeF(0, 2),
			PrevX:     float64(x) + 0.5,
			PrevY:     float64(y) + 0.5,
		}
		applyVariantTuning(&p)
		ps.P = append(ps.P, p)
		spawned++
	}
}

// Update advances all pedestrian AI.
func (ps *PedestrianSystem) Update(dt float64, w *World, snake *Snake, particles *ParticleSystem) {
	if dt <= 0 || w == nil {
		return
	}

	// Rebuild spatial buckets.
	cols := ps.bucketCols
	rows := ps.bucketRows
	buckets := ps.buckets
	for bi := range buckets {
		buckets[bi] = buckets[bi][:0]
	}
	for j := range ps.P {
		if !ps.P[j].Alive {
			continue
		}
		bx := clamp(int(ps.P[j].X)/ps.bucketSize, 0, cols-1)
		by := clamp(int(ps.P[j].Y)/ps.bucketSize, 0, rows-1)
		buckets[by*cols+bx] = append(buckets[by*cols+bx], j)
	}

	// Refresh group leaders.
	for gid := range ps.groupLeader {
		delete(ps.groupLeader, gid)
	}
	for idx := range ps.P {
		if ps.P[idx].IsLeader && ps.P[idx].GroupID != 0 {
			ps.groupLeader[ps.P[idx].GroupID] = idx
		}
	}

	var snakeHX, snakeHY float64
	if snake != nil {
		snakeHX, snakeHY = snake.Head()
	}

	for i := range ps.P {
		p := &ps.P[i]
		if !p.Alive {
			continue
		}
		startX, startY := p.X, p.Y

		// Armed ped: shoot at snake if close and has LOS.
		if p.Armed && snake != nil && snake.Alive {
			p.ShootCooldown -= dt
			dist := math.Hypot(snakeHX-p.X, snakeHY-p.Y)
			if dist < 30.0 && p.ShootCooldown <= 0 && HasLineOfSight(p.X, p.Y, snakeHX, snakeHY, w) {
				p.ShootCooldown = 1.5
				// Bullet particle toward snake.
				ang := math.Atan2(snakeHY-p.Y, snakeHX-p.X)
				rr := NewRand(ps.seed ^ uint64(i)*0x5B007)
				particles.Add(Particle{
					X: p.X, Y: p.Y,
					VX: math.Cos(ang) * 80, VY: math.Sin(ang) * 80,
					Size: 0.8, MaxLife: 0.4,
					Col: RGB{R: 255, G: 240, B: 100}, Kind: ParticleGlow,
				})
				// Check if bullet hits snake (approximate: if within 2px of head).
				if dist < 2.5 {
					snake.HP.Damage(0.5)
					snake.Length -= 1
					_ = rr
				}
			}
		}

		// Flee from snake head if close.
		p.Fleeing = false
		if snake != nil && snake.Alive {
			dist := math.Hypot(snakeHX-p.X, snakeHY-p.Y)
			if dist < 15.0 && dist > 0.1 {
				p.Fleeing = true
				fdx := (p.X - snakeHX) / dist
				fdy := (p.Y - snakeHY) / dist
				p.TargetX = p.X + fdx*20.0
				p.TargetY = p.Y + fdy*20.0
			}
		}

		// Followers steer towards leader.
		if !p.IsLeader && p.GroupID != 0 {
			if lid, ok := ps.groupLeader[p.GroupID]; ok && lid < len(ps.P) {
				leader := &ps.P[lid]
				p.TargetX = leader.X + p.OffsetX
				p.TargetY = leader.Y + p.OffsetY
			}
		}

		// Wander: move towards target, pick new when close.
		dx := p.TargetX - p.X
		dy := p.TargetY - p.Y
		dist := math.Hypot(dx, dy)

		if dist < 0.8 || dist > 200 {
			r := NewRand(ps.seed ^ uint64(i)*0xC0FFEE)
			px0 := int(math.Round(p.X))
			py0 := int(math.Round(p.Y))
			bestScore := -1.0
			bestX, bestY := px0, py0
			for tries := 0; tries < 40; tries++ {
				tx := px0 + r.Range(-12, 12)
				ty := py0 + r.Range(-12, 12)
				if !pedWalkable(w, tx, ty) {
					continue
				}
				score := 0.5
				tcol := w.ColorAt(tx, ty)
				if rgbEq(tcol, Palette.Sidewalk) {
					score += 1.5
				}
				if rgbEq(tcol, Palette.Grass) || rgbEq(tcol, Palette.GrassPatch) {
					score += 0.8
				}
				if dist > 0.1 {
					vx := float64(tx - px0)
					vy := float64(ty - py0)
					vd := math.Hypot(vx, vy)
					if vd > 0.1 {
						dot := (dx*vx + dy*vy) / (dist * vd)
						score += 0.3 * (dot + 1.0) / 2.0
					}
				}
				if score > bestScore {
					bestScore = score
					bestX, bestY = tx, ty
				}
			}
			p.TargetX = float64(bestX) + 0.5
			p.TargetY = float64(bestY) + 0.5
			dx = p.TargetX - p.X
			dy = p.TargetY - p.Y
			dist = math.Hypot(dx, dy)
		}

		if dist > 0.001 {
			nx := dx / dist
			ny := dy / dist
			spd := p.Speed
			if p.Fleeing {
				spd *= 1.4
			}
			if dist < 3.0 && !p.Fleeing {
				spd *= 0.35
			}
			newX := p.X + nx*spd*dt
			newY := p.Y + ny*spd*dt

			wx := int(math.Round(newX))
			wy := int(math.Round(newY))
			if !pedWalkable(w, wx, wy) && !p.Crossing {
				r := NewRand(ps.seed ^ uint64(i)*0xBEEF)
				for tries := 0; tries < 12; tries++ {
					tx := int(math.Round(p.X)) + r.Range(-8, 8)
					ty := int(math.Round(p.Y)) + r.Range(-8, 8)
					if pedWalkable(w, tx, ty) {
						p.TargetX = float64(tx) + 0.5
						p.TargetY = float64(ty) + 0.5
						break
					}
				}
			} else {
				p.X = newX
				p.Y = newY
			}
		}

		// Occasional road crossing.
		iwx := int(math.Round(p.X))
		iwy := int(math.Round(p.Y))
		dirs := [4][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}}
		for _, d := range dirs {
			nwx := iwx + d[0]
			nwy := iwy + d[1]
			if nwx < 0 || nwy < 0 || nwx >= WorldWidth || nwy >= WorldHeight {
				continue
			}
			if rgbEq(w.ColorAt(nwx, nwy), Palette.Road) {
				r := NewRand(ps.seed ^ uint64(i)*0xDEADBEEF)
				if r.Intn(1000) < 8 {
					sx := nwx + d[0]
					sy := nwy + d[1]
					for steps := 0; steps < 12; steps++ {
						if sx < 0 || sy < 0 || sx >= WorldWidth || sy >= WorldHeight {
							break
						}
						if pedWalkable(w, sx, sy) {
							p.TargetX = float64(sx) + 0.5
							p.TargetY = float64(sy) + 0.5
							p.Crossing = true
							break
						}
						sx += d[0]
						sy += d[1]
					}
				}
			}
		}

		if p.Crossing {
			if math.Hypot(p.TargetX-p.X, p.TargetY-p.Y) < 0.6 {
				p.Crossing = false
			}
		}

		// Local avoidance.
		avBx := clamp(int(p.X)/ps.bucketSize, 0, cols-1)
		avBy := clamp(int(p.Y)/ps.bucketSize, 0, rows-1)
		avoidRadius := 2.6
		for yy := avBy - 1; yy <= avBy+1; yy++ {
			if yy < 0 || yy >= rows {
				continue
			}
			for xx := avBx - 1; xx <= avBx+1; xx++ {
				if xx < 0 || xx >= cols {
					continue
				}
				for _, j := range buckets[yy*cols+xx] {
					if j == i {
						continue
					}
					other := &ps.P[j]
					odx := p.X - other.X
					ody := p.Y - other.Y
					od := math.Hypot(odx, ody)
					if od > 0 && od < avoidRadius {
						push := (avoidRadius - od) * 0.14
						ux := odx / od
						uy := ody / od
						half := push * 0.5
						p.X += ux * half
						p.Y += uy * half
						other.X -= ux * half
						other.Y -= uy * half
					}
				}
			}
		}

		// Stuck detection.
		moved := math.Hypot(p.X-p.PrevX, p.Y-p.PrevY)
		p.PrevX = p.X
		p.PrevY = p.Y
		if moved < 0.1*dt {
			p.StuckTimer += dt
			if p.StuckTimer > 4.0 {
				r := NewRand(ps.seed ^ uint64(i)*0xDEAD ^ uint64(p.StuckTimer*100))
				for tries := 0; tries < 30; tries++ {
					tx := r.Range(0, WorldWidth-1)
					ty := r.Range(0, WorldHeight-1)
					if pedWalkable(w, tx, ty) {
						p.X = float64(tx) + 0.5
						p.Y = float64(ty) + 0.5
						p.PrevX = p.X
						p.PrevY = p.Y
						p.StuckTimer = 0
						break
					}
				}
			} else if p.StuckTimer > 2.0 {
				r := NewRand(ps.seed ^ uint64(i)*0xFACE ^ uint64(p.StuckTimer*100))
				for tries := 0; tries < 20; tries++ {
					tx := int(math.Round(p.X)) + r.Range(-15, 15)
					ty := int(math.Round(p.Y)) + r.Range(-15, 15)
					if pedWalkable(w, tx, ty) {
						p.TargetX = float64(tx) + 0.5
						p.TargetY = float64(ty) + 0.5
						break
					}
				}
			}
		} else {
			p.StuckTimer = 0
		}

		// Drive facing and walk-cycle from real movement this frame.
		moveDX := p.X - startX
		moveDY := p.Y - startY
		moveDist := math.Hypot(moveDX, moveDY)
		maxReasonableStep := math.Max(8.0, p.BaseSpeed*dt*6.0)
		if moveDist > 0.001 && moveDist <= maxReasonableStep {
			p.VX = moveDX / dt
			p.VY = moveDY / dt
			p.FacingX = moveDX / moveDist
			p.FacingY = moveDY / moveDist
			p.WalkCycle += moveDist * 0.5
		} else {
			p.VX = 0
			p.VY = 0
		}
	}
}

// RemoveDead removes dead pedestrians using swap-remove.
func (ps *PedestrianSystem) RemoveDead() {
	for i := 0; i < len(ps.P); {
		if !ps.P[i].Alive {
			PlaySound(SoundSplatter)
			ps.P[i] = ps.P[len(ps.P)-1]
			ps.P = ps.P[:len(ps.P)-1]
		} else {
			i++
		}
	}
}

// AliveCount returns the number of living pedestrians.
func (ps *PedestrianSystem) AliveCount() int {
	n := 0
	for i := range ps.P {
		if ps.P[i].Alive {
			n++
		}
	}
	return n
}

func pedForwardDir(p *Pedestrian) (fx, fy, speed float32) {
	speed = float32(math.Hypot(p.VX, p.VY))
	if speed >= 0.25 {
		return float32(p.VX) / speed, float32(p.VY) / speed, speed
	}
	fx = float32(p.FacingX)
	fy = float32(p.FacingY)
	mag := float32(math.Hypot(float64(fx), float64(fy)))
	if mag < 0.001 {
		return 0, 1, speed
	}
	return fx / mag, fy / mag, speed
}

// PedRenderData builds point sprite data for all alive peds.
// buf is reset and reused to avoid per-frame allocations (same pattern as ParticleRenderData).
func (ps *PedestrianSystem) PedRenderData(buf []float32, now float64) []float32 {
	_ = now
	buf = buf[:0]

	for i := range ps.P {
		p := &ps.P[i]
		if !p.Alive {
			continue
		}
		frame := int(p.WalkCycle) & 1

		hx := float32(math.Round(p.X))
		hy := float32(math.Round(p.Y))

		fx, fy, speed := pedForwardDir(p)

		px := -fy
		py := fx

		sizeScale := p.Size
		shoulderOff := float32(0.7) * sizeScale
		handOff := float32(0.95) * sizeScale
		partSize := float32(1.0) * sizeScale

		hr := float32(p.Skin.R) / 255.0
		hg := float32(p.Skin.G) / 255.0
		hb := float32(p.Skin.B) / 255.0
		cr := float32(p.Col.R) / 255.0
		cg := float32(p.Col.G) / 255.0
		cb := float32(p.Col.B) / 255.0
		sr := cr * 0.92
		sg := cg * 0.92
		sb := cb * 0.92

		// Shadow: offset south-east, soft dark blob.
		buf = append(buf, hx+0.3, hy+0.9, 1.8*sizeScale, 0, 0, 0, 0.30, 0)
		// Head.
		buf = append(buf, hx, hy, partSize, hr, hg, hb, 1, 0)
		// Torso.
		buf = append(buf, hx, hy+1, partSize, cr, cg, cb, 1, 0)
		// Left shoulder.
		lx := float32(math.Round(float64(hx + px*shoulderOff)))
		ly := float32(math.Round(float64(hy + py*shoulderOff)))
		buf = append(buf, lx, ly, partSize, sr, sg, sb, 1, 0)
		// Right shoulder.
		rx := float32(math.Round(float64(hx - px*shoulderOff)))
		ry := float32(math.Round(float64(hy - py*shoulderOff)))
		buf = append(buf, rx, ry, partSize, sr, sg, sb, 1, 0)
		// Hand + foot swing along travel direction.
		stride := float32(0.8) * sizeScale
		if frame == 0 {
			stride = -stride
		}
		if speed < 0.25 {
			stride = 0
		}
		handX := float32(math.Round(float64(hx + fx*stride + px*handOff*0.35)))
		handY := float32(math.Round(float64(hy + fy*stride + py*handOff*0.35)))
		buf = append(buf, handX, handY, partSize*0.9, hr, hg, hb, 1, 0)
		footX := float32(math.Round(float64(hx - fx*stride - px*0.35*sizeScale)))
		footY := float32(math.Round(float64(hy - fy*stride - py*0.35*sizeScale)))
		buf = append(buf, footX, footY, partSize*0.85, sr*0.75, sg*0.75, sb*0.75, 1, 0)

		// Themed gear accents.
		switch p.Variant {
		case PedVariantArctic:
			// Knit-cap style top highlight.
			buf = append(buf, hx, hy-1.8*sizeScale, 0.85*sizeScale, 0.92, 0.96, 1.0, 1, 0)
		case PedVariantDesert:
			// Scarf/keffiyeh cloth accent.
			buf = append(buf, hx, hy-0.3*sizeScale, 0.75*sizeScale, 0.92, 0.66, 0.25, 1, 0)
		case PedVariantAstronaut:
			// Helmet halo + visor.
			buf = append(buf, hx, hy-0.1*sizeScale, 2.15*sizeScale, 0.72, 0.92, 1.0, 0.55, 0)
			buf = append(buf, hx, hy-0.45*sizeScale, 0.8*sizeScale, 0.18, 0.30, 0.42, 0.9, 0)
		case PedVariantDiver:
			// Bubble helmet + rear tank.
			buf = append(buf, hx, hy-0.1*sizeScale, 2.1*sizeScale, 0.35, 0.92, 1.0, 0.45, 0)
			buf = append(buf, hx-px*0.85*sizeScale, hy-py*0.85*sizeScale, 0.7*sizeScale, 0.28, 0.40, 0.48, 0.95, 0)
		}

		// Armed marker: small red dot above.
		if p.Armed {
			buf = append(buf, hx, hy-2.5, 0.7, 1, 0.1, 0.1, 1, 0)
		}
	}
	return buf
}
