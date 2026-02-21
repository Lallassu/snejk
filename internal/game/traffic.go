package game

import "math"

type NPCCar struct {
	X, Y           float64
	VX, VY         float64
	Heading        float64
	TurnTarget     float64
	Speed          float64
	TargetSpeed    float64
	TurnTimer      float64
	Aggression     float64
	WaitTimer      float64
	StuckTimer     float64
	TurnCount      int  // incremented each intersection decision, ensures varied routing
	InIntersection bool // true while traversing a junction; prevents re-rolling turns
	Alive          bool
	HP             Health
	OnFire         bool
	FireTimer      float64
	Parked         bool
	LotParked      bool

	// Visual.
	R, G, B float32
	Size    float32
}

type TrafficSystem struct {
	Cars        []NPCCar
	seed        uint64
	Env         string
	NightFactor float32 // 0=day, 1=midnight; set each frame from sun ambient

	// Spatial grid for neighbor queries.
	gridW, gridH int
	cellSize     int
	cells        [][]int
}

func NewTrafficSystem(seed uint64) *TrafficSystem {
	if seed == 0 {
		seed = 1
	}
	ts := &TrafficSystem{
		Cars:     make([]NPCCar, 0, 64),
		seed:     seed,
		Env:      ThemeCity.Name,
		cellSize: 32,
	}
	ts.gridW = (WorldWidth + ts.cellSize - 1) / ts.cellSize
	ts.gridH = (WorldHeight + ts.cellSize - 1) / ts.cellSize
	ts.cells = make([][]int, ts.gridW*ts.gridH)
	return ts
}

func (ts *TrafficSystem) SetEnvironment(name string) {
	if name == "" {
		name = ThemeCity.Name
	}
	ts.Env = name
}

func isRoadPixel(w *World, tp themePalette, x, y int) bool {
	if x < 0 || y < 0 || x >= WorldWidth || y >= WorldHeight {
		return false
	}
	if w.HeightAt(x, y) > 0 {
		return false
	}
	col := w.ColorAt(x, y)
	return rgbEq(col, tp.Road) || rgbEq(col, roadStripeColor(tp))
}

func isParkingLotPixel(w *World, tp themePalette, x, y int) bool {
	if x < 0 || y < 0 || x >= WorldWidth || y >= WorldHeight {
		return false
	}
	if w.HeightAt(x, y) > 0 {
		return false
	}
	col := w.ColorAt(x, y)
	if isRoadPixel(w, tp, x, y) {
		return false
	}
	return rgbEq(col, tp.Road.Add(-10, -10, -10)) ||
		rgbEq(col, tp.Road.Add(-4, -4, -3)) ||
		rgbEq(col, roadStripeColor(tp).Add(18, 18, 10))
}

func roadCardinalOptions(w *World, tp themePalette, x, y int) []float64 {
	type dir struct {
		h      float64
		dx, dy int
	}
	dirs := [4]dir{
		{h: 0, dx: 1, dy: 0},
		{h: math.Pi / 2, dx: 0, dy: 1},
		{h: math.Pi, dx: -1, dy: 0},
		{h: -math.Pi / 2, dx: 0, dy: -1},
	}
	opts := make([]float64, 0, 4)
	for _, d := range dirs {
		if isRoadPixel(w, tp, x+d.dx, y+d.dy) || isRoadPixel(w, tp, x+2*d.dx, y+2*d.dy) {
			opts = append(opts, d.h)
		}
	}
	return opts
}

func hasHeadingOption(opts []float64, h float64) bool {
	for _, o := range opts {
		if math.Abs(angDiff(o, h)) < 0.01 {
			return true
		}
	}
	return false
}

func chooseTurnTarget(r *Rand, current float64, opts []float64) float64 {
	if len(opts) == 0 {
		return current
	}
	forward := snapToCardinal(current)
	left := snapToCardinal(forward + math.Pi/2)
	right := snapToCardinal(forward - math.Pi/2)
	back := snapToCardinal(forward + math.Pi)

	type weighted struct {
		h float64
		w float64
	}
	list := make([]weighted, 0, len(opts))
	total := 0.0
	for _, h := range opts {
		w := 0.05
		switch {
		case math.Abs(angDiff(h, forward)) < 0.01:
			w = 0.50
		case math.Abs(angDiff(h, left)) < 0.01 || math.Abs(angDiff(h, right)) < 0.01:
			w = 0.22
		case math.Abs(angDiff(h, back)) < 0.01:
			w = 0.06
		}
		list = append(list, weighted{h: h, w: w})
		total += w
	}
	roll := r.RangeF(0, total)
	acc := 0.0
	for _, it := range list {
		acc += it.w
		if roll <= acc {
			return it.h
		}
	}
	return list[len(list)-1].h
}

func nearestRoadCenterCar(w *World, tp themePalette, x, y float64) (float64, float64, bool) {
	cx := int(math.Round(x))
	cy := int(math.Round(y))
	bestX, bestY := cx, cy
	bestD2 := math.MaxFloat64
	found := false
	for r := 0; r <= 20; r++ {
		for dy := -r; dy <= r; dy++ {
			for dx := -r; dx <= r; dx++ {
				wx := cx + dx
				wy := cy + dy
				if !isRoadPixel(w, tp, wx, wy) {
					continue
				}
				d2 := float64(dx*dx + dy*dy)
				if d2 < bestD2 {
					bestD2 = d2
					bestX = wx
					bestY = wy
					found = true
				}
			}
		}
		if found {
			break
		}
	}
	if !found {
		return x, y, false
	}
	return float64(bestX) + 0.5, float64(bestY) + 0.5, true
}

func roadAxisAt(w *World, tp themePalette, x, y int) (bool, bool) {
	if !isRoadPixel(w, tp, x, y) {
		return false, false
	}
	h := isRoadPixel(w, tp, x-1, y) || isRoadPixel(w, tp, x+1, y)
	v := isRoadPixel(w, tp, x, y-1) || isRoadPixel(w, tp, x, y+1)
	return h, v
}

// snapToCardinal snaps a heading to the nearest cardinal direction.
func snapToCardinal(h float64) float64 {
	for h > math.Pi {
		h -= 2 * math.Pi
	}
	for h < -math.Pi {
		h += 2 * math.Pi
	}
	cardinals := [4]float64{0, math.Pi / 2, math.Pi, -math.Pi / 2}
	best := cardinals[0]
	bestD := math.Abs(angDiff(h, cardinals[0]))
	for _, c := range cardinals[1:] {
		d := math.Abs(angDiff(h, c))
		if d < bestD {
			bestD = d
			best = c
		}
	}
	return best
}

func (ts *TrafficSystem) SpawnRandom(w *World, n int) {
	if n <= 0 || w == nil {
		return
	}
	r := NewRand(ts.seed ^ 0xBEEF)
	tp := buildThemePalette(w.Theme)
	roadPts := make([][2]int, 0, 1024)
	parkingPts := make([][2]int, 0, 512)
	for y := 1; y < WorldHeight-1; y++ {
		for x := 1; x < WorldWidth-1; x++ {
			if isRoadPixel(w, tp, x, y) {
				roadPts = append(roadPts, [2]int{x, y})
			}
			if isParkingLotPixel(w, tp, x, y) {
				parkingPts = append(parkingPts, [2]int{x, y})
			}
		}
	}
	if len(roadPts) == 0 {
		return
	}

	cardinals := [4]float64{0, math.Pi / 2, math.Pi, -math.Pi / 2}
	for i := 0; i < n; i++ {
		useParkingLot := len(parkingPts) > 0 && r.Intn(100) < 28
		pt := roadPts[r.Intn(len(roadPts))]
		jitter := 0.35
		if useParkingLot {
			pt = parkingPts[r.Intn(len(parkingPts))]
			jitter = 0.15
		}
		fx := float64(pt[0]) + 0.5 + r.RangeF(-jitter, jitter)
		fy := float64(pt[1]) + 0.5 + r.RangeF(-jitter, jitter)
		fx = clampF(fx, 2, float64(WorldWidth-3))
		fy = clampF(fy, 2, float64(WorldHeight-3))

		heading := cardinals[r.Intn(4)]
		if useParkingLot {
			if rx, ry, ok := nearestRoadCenterCar(w, tp, fx, fy); ok {
				heading = snapToCardinal(math.Atan2(ry-fy, rx-fx))
			}
		} else {
			horiz, vert := roadAxisAt(w, tp, pt[0], pt[1])
			switch {
			case horiz && !vert:
				if r.Intn(2) == 0 {
					heading = 0
				} else {
					heading = math.Pi
				}
			case vert && !horiz:
				if r.Intn(2) == 0 {
					heading = math.Pi / 2
				} else {
					heading = -math.Pi / 2
				}
			}
			if opts := roadCardinalOptions(w, tp, pt[0], pt[1]); len(opts) > 0 && !hasHeadingOption(opts, heading) {
				heading = opts[r.Intn(len(opts))]
			}
		}

		hp := 5.0 + r.RangeF(0, 10.0)
		// Reduced base speed so city traffic doesn't zip too fast.
		speedMul := 1.0
		sizeMul := 1.0
		palette := [][3]float64{
			{0.4, 0.2, 0.15},
			{0.6, 0.35, 0.25},
			{0.35, 0.45, 0.6},
		}
		switch ts.Env {
		case ThemeArctic.Name, ThemeWinter.Name, ThemeForestWinter.Name:
			speedMul = 0.82
			sizeMul = 1.10
			hp += 2.0
			palette = [][3]float64{
				{0.78, 0.86, 0.95},
				{0.65, 0.78, 0.92},
				{0.84, 0.92, 1.0},
			}
		case ThemeDesert.Name, ThemeSand.Name:
			speedMul = 1.18
			sizeMul = 0.92
			palette = [][3]float64{
				{0.78, 0.52, 0.26},
				{0.66, 0.40, 0.18},
				{0.86, 0.64, 0.34},
			}
		case ThemeBeach.Name:
			speedMul = 1.05
			sizeMul = 0.96
			palette = [][3]float64{
				{0.30, 0.62, 0.86},
				{0.90, 0.70, 0.38},
				{0.86, 0.42, 0.56},
			}
		case ThemeSpace.Name:
			speedMul = 1.22
			sizeMul = 1.06
			hp += 2.5
			palette = [][3]float64{
				{0.42, 0.92, 1.0},
				{0.82, 0.54, 1.0},
				{0.72, 1.0, 0.58},
			}
		case ThemeUnderwater.Name:
			speedMul = 0.76
			sizeMul = 1.18
			hp += 1.8
			palette = [][3]float64{
				{0.22, 0.62, 0.72},
				{0.16, 0.44, 0.64},
				{0.32, 0.72, 0.82},
			}
		}
		baseCol := palette[r.Intn(len(palette))]
		c := NPCCar{
			X: fx, Y: fy, Heading: heading, TurnTarget: heading, Alive: true,
			HP:          NewHealth(hp),
			TargetSpeed: CarMaxSpeed * 0.15 * (0.5 + r.RangeF(0, 0.9)) * speedMul,
			Aggression:  r.RangeF(0, 0.95),
			R:           float32(clampF(baseCol[0]+r.RangeF(-0.08, 0.08), 0.05, 1.0)),
			G:           float32(clampF(baseCol[1]+r.RangeF(-0.08, 0.08), 0.05, 1.0)),
			B:           float32(clampF(baseCol[2]+r.RangeF(-0.08, 0.08), 0.05, 1.0)),
			Size:        float32((3.0 + r.RangeF(0, 2.5)) * sizeMul),
		}
		// Some cars are parked; parking lots strongly bias parked cars.
		if useParkingLot || r.Intn(100) < 5 {
			c.Parked = true
			c.LotParked = useParkingLot
		} else {
			c.Speed = c.TargetSpeed * (0.3 + r.RangeF(0, 0.7))
			c.VX = math.Cos(c.Heading) * c.Speed
			c.VY = math.Sin(c.Heading) * c.Speed
		}
		ts.Cars = append(ts.Cars, c)
	}
}

func (ts *TrafficSystem) RebuildGrid() {
	for i := range ts.cells {
		ts.cells[i] = ts.cells[i][:0]
	}
	for idx := range ts.Cars {
		c := &ts.Cars[idx]
		gx := clamp(int(c.X)/ts.cellSize, 0, ts.gridW-1)
		gy := clamp(int(c.Y)/ts.cellSize, 0, ts.gridH-1)
		ts.cells[gy*ts.gridW+gx] = append(ts.cells[gy*ts.gridW+gx], idx)
	}
}

func (ts *TrafficSystem) QueryNeighbors(x, y, radius float64, fn func(int)) {
	minGX := clamp(int(math.Floor((x-radius)/float64(ts.cellSize))), 0, ts.gridW-1)
	minGY := clamp(int(math.Floor((y-radius)/float64(ts.cellSize))), 0, ts.gridH-1)
	maxGX := clamp(int(math.Floor((x+radius)/float64(ts.cellSize))), 0, ts.gridW-1)
	maxGY := clamp(int(math.Floor((y+radius)/float64(ts.cellSize))), 0, ts.gridH-1)
	r2 := radius * radius
	for gy := minGY; gy <= maxGY; gy++ {
		for gx := minGX; gx <= maxGX; gx++ {
			for _, idx := range ts.cells[gy*ts.gridW+gx] {
				if idx >= len(ts.Cars) {
					continue
				}
				dx := ts.Cars[idx].X - x
				dy := ts.Cars[idx].Y - y
				if dx*dx+dy*dy <= r2 {
					fn(idx)
				}
			}
		}
	}
}

func npcCarCollides(w *World, x, y float64) bool {
	h := int(math.Round(CarSize * 0.5))
	offs := [8][2]int{{-h, -h}, {0, -h}, {h, -h}, {-h, 0}, {h, 0}, {-h, h}, {0, h}, {h, h}}
	for _, o := range offs {
		if w.IsBlocked(int(math.Round(x))+o[0], int(math.Round(y))+o[1]) {
			return true
		}
	}
	return false
}

func (ts *TrafficSystem) Update(dt float64, w *World, ps *ParticleSystem, peds *PedestrianSystem, cam *Camera) {
	if dt <= 0 || w == nil {
		return
	}
	tp := buildThemePalette(w.Theme)

	ts.RebuildGrid()

	for i := range ts.Cars {
		c := &ts.Cars[i]
		if !c.Alive {
			continue
		}

		// Parked cars: at night they gradually rejoin traffic.
		if c.Parked {
			if c.LotParked {
				continue
			}
			if ts.NightFactor > 0.15 {
				r := NewRand(ts.seed ^ uint64(i)*0xF33D ^ uint64(dt*1e9+float64(i)))
				if r.RangeF(0, 1) < float64(ts.NightFactor)*0.10*dt {
					c.Parked = false
					c.Speed = c.TargetSpeed * 0.4
					c.VX = math.Cos(c.Heading) * c.Speed
					c.VY = math.Sin(c.Heading) * c.Speed
				}
			}
			continue
		}

		// Fire damage over time; dying car explodes.
		if c.OnFire {
			c.FireTimer += dt
			c.HP.Damage(dt * 2.0)
			if c.HP.IsDead() {
				c.Alive = false
				ExplodeAt(int(math.Round(c.X)), int(math.Round(c.Y)), 8, w, ps, peds, ts, cam, nil, nil)
				continue
			}
		}

		// Approach target speed; night cars drive slightly faster.
		nightMult := 1.0 + float64(ts.NightFactor)*0.4
		c.Speed = approach(c.Speed, c.TargetSpeed*nightMult, 30.0*dt)
		c.VX = math.Cos(c.Heading) * c.Speed
		c.VY = math.Sin(c.Heading) * c.Speed

		// Forward avoidance: find the closest car ahead and brake proportionally.
		minAvoidSpeed := c.Speed
		ts.QueryNeighbors(c.X, c.Y, 12.0, func(j int) {
			if j == i {
				return
			}
			other := &ts.Cars[j]
			if !other.Alive || other.Parked {
				return
			}
			dx := other.X - c.X
			dy := other.Y - c.Y
			d := math.Hypot(dx, dy)
			if d < 0.5 {
				return
			}
			fwd := (dx*math.Cos(c.Heading) + dy*math.Sin(c.Heading)) / d
			if fwd > 0.4 {
				// Scale brake by how directly ahead and how close.
				closeness := 1.0 - d/12.0
				target := c.Speed * (1.0 - fwd*closeness*0.85)
				if target < minAvoidSpeed {
					minAvoidSpeed = target
				}
			}
		})
		if minAvoidSpeed < c.Speed {
			c.Speed = max(0, minAvoidSpeed)
			c.VX = math.Cos(c.Heading) * c.Speed
			c.VY = math.Sin(c.Heading) * c.Speed
		}

		nx := c.X + c.VX*dt
		ny := c.Y + c.VY*dt

		// Building collision: try 5 alternative headings before giving up.
		if npcCarCollides(w, nx, ny) {
			alts := [5]float64{
				c.Heading + math.Pi/4, // +45°
				c.Heading - math.Pi/4, // -45°
				c.Heading + math.Pi/2, // right
				c.Heading - math.Pi/2, // left
				c.Heading + math.Pi,   // reverse
			}
			moved := false
			for _, nh := range alts {
				hx := c.X + math.Cos(nh)*c.Speed*dt
				hy := c.Y + math.Sin(nh)*c.Speed*dt
				if !npcCarCollides(w, hx, hy) {
					c.TurnTarget = snapToCardinal(nh)
					nx = hx
					ny = hy
					moved = true
					break
				}
			}
			if !moved {
				// Completely boxed in — hold position; stuck detection below handles recovery.
				nx = c.X
				ny = c.Y
			}
		}

		// Direction-aware lane offset on actual generated roads.
		// Fast path: if already on a road pixel, skip the spiral search.
		snappedH := snapToCardinal(c.Heading)
		off := 1.6
		nxi, nyi := int(math.Round(nx)), int(math.Round(ny))
		var roadX, roadY float64
		var roadOk bool
		if isRoadPixel(w, tp, nxi, nyi) {
			roadX, roadY, roadOk = float64(nxi)+0.5, float64(nyi)+0.5, true
		} else {
			roadX, roadY, roadOk = nearestRoadCenterCar(w, tp, nx, ny)
		}
		if roadOk {
			horiz, vert := roadAxisAt(w, tp, int(math.Round(roadX)), int(math.Round(roadY)))
			switch {
			case horiz && !vert:
				desiredY := roadY + off
				if snappedH > math.Pi/2 || snappedH < -math.Pi/2 {
					desiredY = roadY - off
				}
				ny += clampF((desiredY-ny)*0.35, -2.0, 2.0)
			case vert && !horiz:
				desiredX := roadX + off
				if snappedH > 0 {
					desiredX = roadX - off
				}
				nx += clampF((desiredX-nx)*0.35, -2.0, 2.0)
			default:
				// At intersections, softly recentre.
				nx += clampF((roadX-nx)*0.15, -1.5, 1.5)
				ny += clampF((roadY-ny)*0.15, -1.5, 1.5)
			}
		}

		// Lateral repulsion from neighbors.
		repelX, repelY := 0.0, 0.0
		ts.QueryNeighbors(c.X, c.Y, 13.0, func(j int) {
			if j == i {
				return
			}
			dx := ts.Cars[j].X - c.X
			dy := ts.Cars[j].Y - c.Y
			d2 := dx*dx + dy*dy
			if d2 > 0 {
				inv := 1.0 / d2
				repelX -= dx * inv
				repelY -= dy * inv
			}
		})
		latAxisX := -math.Sin(c.Heading)
		latAxisY := math.Cos(c.Heading)
		latRepel := repelX*latAxisX + repelY*latAxisY

		avoidScale := 1.0 - c.Aggression*0.9
		lateralTarget := latRepel * 8.0 * avoidScale
		steerAngle := clampF(math.Atan2(lateralTarget, max(1e-3, c.Speed)), -0.75, 0.75)

		if math.Abs(c.Speed) > 1.0 {
			yaw := clampF((c.Speed/CarWheelBase)*math.Tan(steerAngle), -4.0, 4.0)
			c.Heading += yaw * dt
		}

		// Intersection detection: choose one direction when entering, then commit.
		c.TurnTimer -= dt
		ix := int(math.Round(c.X))
		iy := int(math.Round(c.Y))
		horizRoad, vertRoad := roadAxisAt(w, tp, ix, iy)
		atIntersection := isRoadPixel(w, tp, ix, iy) && horizRoad && vertRoad

		if atIntersection && !c.InIntersection && c.TurnTimer <= 0 {
			c.InIntersection = true
			c.TurnCount++

			if c.WaitTimer <= 0 {
				r := NewRand(ts.seed ^ uint64(i)*0x1234 ^ uint64(c.TurnCount)*0xAB01)
				c.WaitTimer = 0.05 + r.RangeF(0, 0.12)
				if c.Aggression > 0.7 && r.RangeF(0, 1) < c.Aggression*0.6 {
					c.WaitTimer = 0
				}
			}

			currentCardinal := snapToCardinal(c.Heading)
			r := NewRand(ts.seed ^ uint64(i)*0x5678 ^ uint64(c.TurnCount)*0xF00D)
			if opts := roadCardinalOptions(w, tp, ix, iy); len(opts) > 0 {
				c.TurnTarget = chooseTurnTarget(r, currentCardinal, opts)
			} else {
				roll := r.RangeF(0, 1)
				if roll < 0.45 {
					c.TurnTarget = currentCardinal
				} else if roll < 0.70 {
					c.TurnTarget = currentCardinal + math.Pi/2
				} else if roll < 0.95 {
					c.TurnTarget = currentCardinal - math.Pi/2
				} else {
					c.TurnTarget = currentCardinal + math.Pi // rare U-turn
				}
			}
		} else if !atIntersection && c.InIntersection {
			c.InIntersection = false
			// Short cooldown avoids re-rolling when straddling junction edges.
			c.TurnTimer = 0.35
		}

		if c.WaitTimer > 0 {
			c.WaitTimer -= dt
			minSpeed := c.TargetSpeed * 0.35
			c.Speed = max(minSpeed, c.Speed*0.85)
			c.VX = math.Cos(c.Heading) * c.Speed
			c.VY = math.Sin(c.Heading) * c.Speed
			nx = c.X + c.VX*dt
			ny = c.Y + c.VY*dt
		}

		// Heading blend toward TurnTarget with a capped turn rate.
		diff := angDiff(c.Heading, c.TurnTarget)
		if math.Abs(diff) > 0.001 {
			maxTurnRate := 3.8
			if c.InIntersection {
				maxTurnRate = 5.0
			}
			maxTurn := maxTurnRate * dt
			if math.Abs(diff) <= maxTurn {
				c.Heading = c.TurnTarget
			} else if diff > 0 {
				c.Heading += maxTurn
			} else {
				c.Heading -= maxTurn
			}
			for c.Heading > math.Pi {
				c.Heading -= 2 * math.Pi
			}
			for c.Heading < -math.Pi {
				c.Heading += 2 * math.Pi
			}
			if math.Abs(angDiff(c.Heading, c.TurnTarget)) < 0.035 {
				c.Heading = snapToCardinal(c.TurnTarget)
			}
		}

		// Stuck / off-road recovery.
		// Use displacement (proposed move this frame) to detect true blockage.
		// This catches building-blocked cars regardless of speed value.
		wx := int(math.Round(c.X))
		wy := int(math.Round(c.Y))
		offRoad := !isRoadPixel(w, tp, wx, wy)
		displacement := math.Hypot(nx-c.X, ny-c.Y)
		blockedWhileTryingToMove := displacement < 0.03 && c.WaitTimer <= 0 && c.Speed > c.TargetSpeed*0.45
		isStuck := offRoad || blockedWhileTryingToMove

		if isStuck {
			c.StuckTimer += dt
			rx, ry, foundRoad := nearestRoadCenterCar(w, tp, c.X, c.Y)
			if offRoad && foundRoad {
				c.TurnTarget = math.Atan2(ry-c.Y, rx-c.X)
			}
			if c.StuckTimer > 1.2 && foundRoad {
				// Teleport to nearest road center with a fresh random heading.
				r := NewRand(ts.seed ^ uint64(i)*0xDEAD ^ uint64(c.StuckTimer*1000))
				cardinals := [4]float64{0, math.Pi / 2, math.Pi, -math.Pi / 2}
				c.X = rx
				c.Y = ry
				c.Speed = c.TargetSpeed * 0.6
				rxi := int(math.Round(rx))
				ryi := int(math.Round(ry))
				opts := roadCardinalOptions(w, tp, rxi, ryi)
				if len(opts) > 0 {
					c.Heading = opts[r.Intn(len(opts))]
				} else {
					c.Heading = cardinals[r.Intn(4)]
				}
				c.TurnTarget = c.Heading
				c.TurnCount++
				c.StuckTimer = 0
				// Recompute next position after teleport.
				nx = c.X + math.Cos(c.Heading)*c.Speed*dt
				ny = c.Y + math.Sin(c.Heading)*c.Speed*dt
			} else if offRoad && c.StuckTimer > 0.5 {
				// Nudge heading toward nearest open road ahead.
				if rx2, ry2, ok2 := nearestRoadCenterCar(w, tp, c.X+math.Cos(c.Heading)*5, c.Y+math.Sin(c.Heading)*5); ok2 {
					c.TurnTarget = snapToCardinal(math.Atan2(ry2-c.Y, rx2-c.X))
					c.Speed = max(c.Speed, c.TargetSpeed*0.3)
				}
			}
		} else {
			c.StuckTimer = max(0, c.StuckTimer-dt*3)
		}

		// Normalize heading.
		for c.Heading > math.Pi {
			c.Heading -= 2 * math.Pi
		}
		for c.Heading < -math.Pi {
			c.Heading += 2 * math.Pi
		}

		c.X = clampF(nx, 0, float64(WorldWidth-1))
		c.Y = clampF(ny, 0, float64(WorldHeight-1))
	}

	// Car-car overlap pass: gently separate and slow down so cars can pass.
	// Grid reflects positions from start of frame; per-frame displacements are
	// sub-pixel so cells are still accurate for finding nearby pairs.
	for i := range ts.Cars {
		ci := &ts.Cars[i]
		if !ci.Alive || ci.Parked {
			continue
		}
		ts.QueryNeighbors(ci.X, ci.Y, float64(CarSize)*1.2, func(j int) {
			if j <= i {
				return
			}
			cj := &ts.Cars[j]
			if !cj.Alive || cj.Parked {
				return
			}
			dx := cj.X - ci.X
			dy := cj.Y - ci.Y
			d := math.Hypot(dx, dy)
			if d < 0.1 {
				d = 0.1
			}
			minDist := float64(ci.Size+cj.Size) * 0.5
			if d >= minDist {
				return
			}

			// Closing speed along collision normal.
			nx := dx / d
			ny := dy / d
			rvx := ci.VX - cj.VX
			rvy := ci.VY - cj.VY
			closing := rvx*nx + rvy*ny

			// Always separate to resolve overlap.
			overlap := (minDist - d) * 0.5
			ci.X -= nx * overlap
			ci.Y -= ny * overlap
			cj.X += nx * overlap
			cj.Y += ny * overlap

			// Slow both cars rather than crashing/exploding.
			// More head-on overlap means stronger slowdown.
			slowFactor := clampF(0.9-closing*0.08, 0.28, 0.9)
			ci.Speed = max(ci.TargetSpeed*0.25, ci.Speed*slowFactor)
			cj.Speed = max(cj.TargetSpeed*0.25, cj.Speed*slowFactor)
			ci.VX = math.Cos(ci.Heading) * ci.Speed
			ci.VY = math.Sin(ci.Heading) * ci.Speed
			cj.VX = math.Cos(cj.Heading) * cj.Speed
			cj.VY = math.Sin(cj.Heading) * cj.Speed

			// Hold reduced pace briefly so they visibly pass slowly.
			ci.WaitTimer = max(ci.WaitTimer, 0.18)
			cj.WaitTimer = max(cj.WaitTimer, 0.18)
		})
	}
}

// AliveCount returns the number of living cars.
func (ts *TrafficSystem) AliveCount() int {
	n := 0
	for i := range ts.Cars {
		if ts.Cars[i].Alive {
			n++
		}
	}
	return n
}

// RemoveDead removes dead cars using swap-remove.
func (ts *TrafficSystem) RemoveDead() {
	for i := 0; i < len(ts.Cars); {
		if !ts.Cars[i].Alive {
			ts.Cars[i] = ts.Cars[len(ts.Cars)-1]
			ts.Cars = ts.Cars[:len(ts.Cars)-1]
		} else {
			i++
		}
	}
}
