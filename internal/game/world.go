package game

import "math"

// World is a fixed-size 2D pixel world split into chunks.
type World struct {
	seed  uint64
	Theme ThemeConfig

	maxCx int
	maxCy int

	chunks []*Chunk

	spatial *QuadNode

	// Temporary pixel changes (restored after TTL).
	temp []TempPaint

	// Scheduled temporary paints: applied after Delay seconds.
	scheduled []ScheduledPaint

	// Burning trees and buildings.
	burningTrees     map[int64]*TreeBurn
	burningBuildings map[int64]*BuildingBurn

	// Dynamic sun parameters for shadows (continuous angle).
	sunAngle float64 // radians, 0=east, -π/2=north
	sunSlope float64 // height drop per pixel of sun-ray travel
	sunCosA  float64 // precomputed cos(sunAngle)
	sunSinA  float64 // precomputed sin(sunAngle)
}

type TempPaint struct {
	X, Y int
	Orig RGB
	TTL  float64
}

type ScheduledPaint struct {
	X, Y  int
	Col   RGB
	Delay float64
	TTL   float64
}

type TreeBurn struct {
	X, Y         int
	Pixels       []struct{ X, Y int }
	rng          uint64
	timer        float64
	dropInterval float64
}

type BuildingBurn struct {
	X0, Y0, X1, Y1  int
	Pixels          []struct{ X, Y int }
	rng             uint64
	timer           float64
	stepInterval    float64
	smolder         bool
	smolderTimer    float64
	smolderStep     float64
	smolderDuration float64
	smolderTotal    float64
	pendingBurst    int
}

func NewWorld(seed uint64) *World {
	if seed == 0 {
		seed = 1
	}
	maxCx := floorDiv(WorldWidth-1, ChunkSize)
	maxCy := floorDiv(WorldHeight-1, ChunkSize)
	count := (maxCx + 1) * (maxCy + 1)
	return &World{
		seed:             seed,
		maxCx:            maxCx,
		maxCy:            maxCy,
		chunks:           make([]*Chunk, count),
		burningTrees:     make(map[int64]*TreeBurn),
		burningBuildings: make(map[int64]*BuildingBurn),
		sunAngle:         math.Atan2(float64(SunDy), float64(SunDx)),
		sunSlope:         float64(SunSlope),
		sunCosA:          float64(SunDx),
		sunSinA:          float64(SunDy),
	}
}

// UpdateSun updates sun parameters and invalidates chunk shadows when the sun has
// moved enough to produce a visible change. Calling this every frame with a slowly
// moving sun is safe — shadows only recompute a few times per second.
func (w *World) UpdateSun(angle, slope float64) {
	// ~0.5px shadow shift at MaxShadowDist=48. Below this threshold the
	// change is sub-pixel and not worth recomputing the entire world.
	const angleThreshold = 0.01 // ~0.57 degrees
	const slopeThreshold = 0.04

	if math.Abs(angDiff(w.sunAngle, angle)) < angleThreshold &&
		math.Abs(slope-w.sunSlope) < slopeThreshold {
		return
	}

	w.sunAngle = angle
	w.sunSlope = slope
	w.sunCosA = math.Cos(angle)
	w.sunSinA = math.Sin(angle)
	for _, c := range w.chunks {
		if c != nil {
			c.NeedsShadow = true
		}
	}
}

func (w *World) chunkIndex(cx, cy int) int {
	return cy*(w.maxCx+1) + cx
}

func (w *World) GetChunk(cx, cy int) *Chunk {
	if cx < 0 || cy < 0 || cx > w.maxCx || cy > w.maxCy {
		return nil
	}
	idx := w.chunkIndex(cx, cy)
	if c := w.chunks[idx]; c != nil {
		return c
	}
	c := NewChunk(cx, cy)
	generateChunk(c, w.seed, w.Theme)
	w.chunks[idx] = c
	return c
}

func (w *World) GenerateAll() {
	for cy := 0; cy <= w.maxCy; cy++ {
		for cx := 0; cx <= w.maxCx; cx++ {
			_ = w.GetChunk(cx, cy)
		}
	}
	w.repairRoadNetwork()
}

// repairRoadNetwork does a final global pass after all chunks are generated:
// 1) enforce striped center lines on perimeter roads and
// 2) fill tiny one-pixel road gaps so corners/crossings stay connected.
func (w *World) repairRoadNetwork() {
	if w.Theme.NoRoads {
		return
	}
	tp := buildThemePalette(w.Theme)
	roadCol := tp.Road

	isRoad := func(x, y int) bool {
		if x < 0 || y < 0 || x >= WorldWidth || y >= WorldHeight {
			return false
		}
		if w.HeightAt(x, y) > 0 {
			return false
		}
		return isRoadSurfaceColor(w.ColorAt(x, y), tp)
	}
	isCorridor := func(x, y int) bool {
		if isPerimeterRoad(x, y) {
			return true
		}
		xm := x % Pattern
		ym := y % Pattern
		return xm < RoadWidth || ym < RoadWidth
	}

	// Pass A: repaint perimeter ring roads with dashed center lines.
	for y := BorderThickness; y < WorldHeight-BorderThickness; y++ {
		for x := BorderThickness; x < WorldWidth-BorderThickness; x++ {
			if !isPerimeterRoad(x, y) || w.HeightAt(x, y) > 0 {
				continue
			}
			_ = w.PaintRGB(x, y, perimeterRoadColor(x, y, tp))
		}
	}

	// Pass B: bridge tiny missed links (including chunk seams).
	type pt struct{ X, Y int }
	for iter := 0; iter < 2; iter++ {
		bridges := make([]pt, 0, 256)
		for y := BorderThickness + 1; y < WorldHeight-BorderThickness-1; y++ {
			for x := BorderThickness + 1; x < WorldWidth-BorderThickness-1; x++ {
				if !isCorridor(x, y) || isRoad(x, y) || w.HeightAt(x, y) > 0 {
					continue
				}
				l := isRoad(x-1, y)
				r := isRoad(x+1, y)
				u := isRoad(x, y-1)
				d := isRoad(x, y+1)
				// Fill only likely single-pixel misses.
				if (l && r) || (u && d) || ((l || r) && (u || d)) {
					bridges = append(bridges, pt{X: x, Y: y})
				}
			}
		}
		if len(bridges) == 0 {
			break
		}
		for _, b := range bridges {
			_ = w.PaintRGB(b.X, b.Y, roadCol)
		}
	}
}

func (w *World) BuildSpatialIndex() {
	root := NewQuadNode(RectF{X0: 0, Y0: 0, X1: float64(WorldWidth), Y1: float64(WorldHeight)}, 0)
	for cy := 0; cy <= w.maxCy; cy++ {
		for cx := 0; cx <= w.maxCx; cx++ {
			x0 := float64(cx * ChunkSize)
			y0 := float64(cy * ChunkSize)
			x1 := x0 + float64(ChunkSize)
			y1 := y0 + float64(ChunkSize)
			if x1 > float64(WorldWidth) {
				x1 = float64(WorldWidth)
			}
			if y1 > float64(WorldHeight) {
				y1 = float64(WorldHeight)
			}
			root.Insert(ChunkKey{X: cx, Y: cy}, RectF{X0: x0, Y0: y0, X1: x1, Y1: y1})
		}
	}
	w.spatial = root
}

func (w *World) VisibleChunks(view RectF, out []ChunkKey) []ChunkKey {
	out = out[:0]
	if w.spatial == nil {
		minCx := clamp(int(view.X0)/ChunkSize, 0, w.maxCx)
		maxCx := clamp(int(view.X1)/ChunkSize, 0, w.maxCx)
		minCy := clamp(int(view.Y0)/ChunkSize, 0, w.maxCy)
		maxCy := clamp(int(view.Y1)/ChunkSize, 0, w.maxCy)
		for cy := minCy; cy <= maxCy; cy++ {
			for cx := minCx; cx <= maxCx; cx++ {
				out = append(out, ChunkKey{X: cx, Y: cy})
			}
		}
		return out
	}
	w.spatial.Query(view, &out)
	return out
}

// HeightAt returns the height at a world coordinate.
func (w *World) HeightAt(wx, wy int) uint8 {
	if wx < 0 || wy < 0 || wx >= WorldWidth || wy >= WorldHeight {
		return 255
	}
	cx := wx / ChunkSize
	cy := wy / ChunkSize
	c := w.GetChunk(cx, cy)
	if c == nil {
		return 255
	}
	lx := wx - cx*ChunkSize
	ly := wy - cy*ChunkSize
	return c.Height[ly*ChunkSize+lx]
}

// ColorAt returns the current RGB at a world coordinate.
func (w *World) ColorAt(wx, wy int) RGB {
	if wx < 0 || wy < 0 || wx >= WorldWidth || wy >= WorldHeight {
		return Palette.Border
	}
	cx := wx / ChunkSize
	cy := wy / ChunkSize
	c := w.GetChunk(cx, cy)
	if c == nil {
		return Palette.Border
	}
	lx := wx - cx*ChunkSize
	ly := wy - cy*ChunkSize
	i := ly*ChunkSize + lx
	o := i * 4
	return RGB{R: c.Pixels[o+0], G: c.Pixels[o+1], B: c.Pixels[o+2]}
}

// IsBlocked returns true if the world pixel has height > 0.
func (w *World) IsBlocked(wx, wy int) bool {
	return w.HeightAt(wx, wy) > 0
}

// HasLineOfSight returns true if a straight line from (ax,ay) to (bx,by) is clear of obstacles.
// Uses Bresenham integer line algorithm to avoid per-step float arithmetic.
func HasLineOfSight(ax, ay, bx, by float64, w *World) bool {
	x0 := int(math.Round(ax))
	y0 := int(math.Round(ay))
	x1 := int(math.Round(bx))
	y1 := int(math.Round(by))

	dx := x1 - x0
	if dx < 0 {
		dx = -dx
	}
	dy := y1 - y0
	if dy < 0 {
		dy = -dy
	}
	sx := 1
	if x0 > x1 {
		sx = -1
	}
	sy := 1
	if y0 > y1 {
		sy = -1
	}
	err := dx - dy

	for {
		if x0 == x1 && y0 == y1 {
			return true
		}
		if w.IsBlocked(x0, y0) {
			return false
		}
		e2 := err * 2
		if e2 > -dy {
			err -= dy
			x0 += sx
		}
		if e2 < dx {
			err += dx
			y0 += sy
		}
	}
}

// PaintRGB overwrites only the RGB channels, keeping height and shade.
func (w *World) PaintRGB(wx, wy int, col RGB) bool {
	if wx < 0 || wy < 0 || wx >= WorldWidth || wy >= WorldHeight {
		return false
	}
	cx := wx / ChunkSize
	cy := wy / ChunkSize
	c := w.GetChunk(cx, cy)
	if c == nil {
		return false
	}
	lx := wx - cx*ChunkSize
	ly := wy - cy*ChunkSize
	i := ly*ChunkSize + lx
	if c.Unbreakable[i] != 0 {
		return false
	}
	c.setRGBKeepHeight(i, col)
	c.NeedsUpload = true
	return true
}

// BurnPixel permanently converts a pixel to rubble and clears height.
func (w *World) BurnPixel(wx, wy int) bool {
	return w.BurnPixelWithColor(wx, wy, Palette.Rubble)
}

// BurnPixelWithColor permanently clears collision height at a pixel and repaints it.
func (w *World) BurnPixelWithColor(wx, wy int, col RGB) bool {
	if wx < 0 || wy < 0 || wx >= WorldWidth || wy >= WorldHeight {
		return false
	}
	cx := wx / ChunkSize
	cy := wy / ChunkSize
	c := w.GetChunk(cx, cy)
	if c == nil {
		return false
	}
	lx := wx - cx*ChunkSize
	ly := wy - cy*ChunkSize
	i := ly*ChunkSize + lx
	if c.Unbreakable[i] != 0 {
		return false
	}
	c.set(i, col, 0, ShadeLit, 0)
	c.NeedsUpload = true
	return true
}

// burnedGroundColorAt returns a charred ground tone based on nearby walkable pixels.
func (w *World) burnedGroundColorAt(wx, wy int) RGB {
	// Uniform dark char with tiny noise variation so burn scars read as solid
	// aftermath instead of shaded checkerboard patches.
	n := int(hash2D(0xB0A04ED5, wx, wy)>>60) - 8 // [-8..7]
	base := RGB{R: 40, G: 34, B: 28}
	return base.Add(n/2, n/3, n/4)
}

// AddTempPaint temporarily paints a pixel for ttl seconds then restores original.
func (w *World) AddTempPaint(wx, wy int, col RGB, ttl float64) bool {
	if wx < 0 || wy < 0 || wx >= WorldWidth || wy >= WorldHeight {
		return false
	}
	cx := wx / ChunkSize
	cy := wy / ChunkSize
	c := w.GetChunk(cx, cy)
	if c == nil {
		return false
	}
	lx := wx - cx*ChunkSize
	ly := wy - cy*ChunkSize
	i := ly*ChunkSize + lx
	orig := RGB{R: c.Pixels[i*4+0], G: c.Pixels[i*4+1], B: c.Pixels[i*4+2]}
	c.setRGBKeepHeight(i, col)
	c.NeedsUpload = true
	w.temp = append(w.temp, TempPaint{X: wx, Y: wy, Orig: orig, TTL: ttl})
	return true
}

// AddScheduledPaint schedules a temp paint to be applied after delay seconds.
func (w *World) AddScheduledPaint(wx, wy int, col RGB, ttl, delay float64) bool {
	if wx < 0 || wy < 0 || wx >= WorldWidth || wy >= WorldHeight {
		return false
	}
	w.scheduled = append(w.scheduled, ScheduledPaint{X: wx, Y: wy, Col: col, Delay: delay, TTL: ttl})
	return true
}

// Explode destroys a circular area of the world.
func (w *World) Explode(wx, wy, radius int) {
	if radius <= 0 {
		return
	}

	minX := clamp(wx-radius, 0, WorldWidth-1)
	maxX := clamp(wx+radius, 0, WorldWidth-1)
	minY := clamp(wy-radius, 0, WorldHeight-1)
	maxY := clamp(wy+radius, 0, WorldHeight-1)

	cx0 := floorDiv(minX, ChunkSize)
	cx1 := floorDiv(maxX, ChunkSize)
	cy0 := floorDiv(minY, ChunkSize)
	cy1 := floorDiv(maxY, ChunkSize)

	r2 := radius * radius
	inner2 := (radius / 2) * (radius / 2)

	for cy := cy0; cy <= cy1; cy++ {
		for cx := cx0; cx <= cx1; cx++ {
			c := w.GetChunk(cx, cy)
			if c == nil {
				continue
			}
			baseX := cx * ChunkSize
			baseY := cy * ChunkSize

			lx0 := clamp(minX-baseX, 0, ChunkSize)
			lx1 := clamp(maxX-baseX+1, 0, ChunkSize)
			ly0 := clamp(minY-baseY, 0, ChunkSize)
			ly1 := clamp(maxY-baseY+1, 0, ChunkSize)

			changed := false
			for ly := ly0; ly < ly1; ly++ {
				gy := baseY + ly
				dy := gy - wy
				for lx := lx0; lx < lx1; lx++ {
					gx := baseX + lx
					dx := gx - wx
					if dx*dx+dy*dy > r2 {
						continue
					}
					i := ly*ChunkSize + lx
					if c.Unbreakable[i] != 0 {
						continue
					}
					col := Palette.Lot
					if dx*dx+dy*dy <= inner2 {
						col = Palette.Rubble
					}
					o := i * 4
					c.Pixels[o+0] = col.R
					c.Pixels[o+1] = col.G
					c.Pixels[o+2] = col.B
					c.Pixels[o+3] = ShadeLit
					c.Height[i] = 0
					changed = true
				}
			}
			if changed {
				c.NeedsShadow = true
				c.NeedsUpload = true
			}
		}
	}

	// Invalidate shadow chunks in the down-shadow direction.
	shadowDirX := -w.sunCosA
	shadowDirY := -w.sunSinA
	reach := radius + MaxShadowDist
	fx0, fx1 := wx-radius, wx+radius
	fy0, fy1 := wy-radius, wy+radius
	if shadowDirX > 0 {
		fx1 += reach
	} else if shadowDirX < 0 {
		fx0 -= reach
	}
	if shadowDirY > 0 {
		fy1 += reach
	} else if shadowDirY < 0 {
		fy0 -= reach
	}
	fx0 = clamp(fx0, 0, WorldWidth-1)
	fx1 = clamp(fx1, 0, WorldWidth-1)
	fy0 = clamp(fy0, 0, WorldHeight-1)
	fy1 = clamp(fy1, 0, WorldHeight-1)

	dcx0 := floorDiv(fx0, ChunkSize)
	dcx1 := floorDiv(fx1, ChunkSize)
	dcy0 := floorDiv(fy0, ChunkSize)
	dcy1 := floorDiv(fy1, ChunkSize)
	for cy := dcy0; cy <= dcy1; cy++ {
		for cx := dcx0; cx <= dcx1; cx++ {
			c := w.GetChunk(cx, cy)
			if c == nil {
				continue
			}
			c.NeedsShadow = true
		}
	}
}

func coordKey(x, y int) int64 {
	return (int64(x) << 32) | int64(uint32(y))
}

// StartTreeBurn initializes burn for a tree canopy near (wx,wy).
func (w *World) StartTreeBurn(wx, wy int) {
	key := coordKey(wx, wy)
	if _, ok := w.burningTrees[key]; ok {
		return
	}
	pixels := make([]struct{ X, Y int }, 0, 64)
	r := 5
	for yy := wy - r; yy <= wy+r; yy++ {
		for xx := wx - r; xx <= wx+r; xx++ {
			if xx < 0 || yy < 0 || xx >= WorldWidth || yy >= WorldHeight {
				continue
			}
			dx := xx - wx
			dy := yy - wy
			d2 := dx*dx + dy*dy
			r2 := r*r + int(hash2D(0x7AEE5A11, xx, yy)&7) - 3
			if d2 > r2 {
				continue
			}
			col := w.ColorAt(xx, yy)
			if col.G > col.R && col.G > col.B {
				pixels = append(pixels, struct{ X, Y int }{X: xx, Y: yy})
			}
		}
	}
	if len(pixels) == 0 {
		return
	}
	w.burningTrees[key] = &TreeBurn{
		X: wx, Y: wy, Pixels: pixels,
		rng: (uint64(uint32(wx)) * 1315423911) ^ (uint64(uint32(wy)) * 2654435761), timer: 0.08, dropInterval: 0.08,
	}
}

// StartBuildingBurn begins destructively burning a building region.
func (w *World) StartBuildingBurn(wx, wy int) {
	key := coordKey(wx, wy)
	if _, ok := w.burningBuildings[key]; ok {
		return
	}
	r := 8
	pixels := make([]struct{ X, Y int }, 0, 256)
	for yy := wy - r; yy <= wy+r; yy++ {
		for xx := wx - r; xx <= wx+r; xx++ {
			if xx < 0 || yy < 0 || xx >= WorldWidth || yy >= WorldHeight {
				continue
			}
			dx := xx - wx
			dy := yy - wy
			d2 := dx*dx + dy*dy
			r2 := r*r + int(hash2D(0xB17D5A11, xx, yy)&15) - 6
			if d2 > r2 {
				continue
			}
			col := w.ColorAt(xx, yy)
			if (col.R >= 90 && col.R <= 210) && (col.G >= 80 && col.G <= 180) {
				pixels = append(pixels, struct{ X, Y int }{X: xx, Y: yy})
			}
		}
	}
	if len(pixels) == 0 {
		return
	}
	w.burningBuildings[key] = &BuildingBurn{
		X0: wx - r, Y0: wy - r, X1: wx + r, Y1: wy + r,
		Pixels: pixels, rng: uint64(wx*97531 ^ wy*53197),
		timer: 0.35, stepInterval: 0.28,
		smolder: true, smolderTimer: 0.8, smolderStep: 0.8,
		smolderDuration: 14.0, smolderTotal: 14.0,
	}
}

// Update processes temp paints and burning entities.
func (w *World) Update(dt float64) {
	if dt <= 0 {
		return
	}

	// Process scheduled paints.
	if len(w.scheduled) > 0 {
		out := w.scheduled[:0]
		for _, s := range w.scheduled {
			s.Delay -= dt
			if s.Delay <= 0 {
				cx := s.X / ChunkSize
				cy := s.Y / ChunkSize
				c := w.GetChunk(cx, cy)
				if c != nil {
					lx := s.X - cx*ChunkSize
					ly := s.Y - cy*ChunkSize
					i := ly*ChunkSize + lx
					orig := RGB{R: c.Pixels[i*4+0], G: c.Pixels[i*4+1], B: c.Pixels[i*4+2]}
					c.setRGBKeepHeight(i, s.Col)
					c.NeedsUpload = true
					w.temp = append(w.temp, TempPaint{X: s.X, Y: s.Y, Orig: orig, TTL: s.TTL})
				}
			} else {
				out = append(out, s)
			}
		}
		w.scheduled = out
	}

	// Process temp paints.
	if len(w.temp) > 0 {
		out := w.temp[:0]
		for _, t := range w.temp {
			t.TTL -= dt
			if t.TTL <= 0 {
				cx := t.X / ChunkSize
				cy := t.Y / ChunkSize
				c := w.GetChunk(cx, cy)
				if c != nil {
					lx := t.X - cx*ChunkSize
					ly := t.Y - cy*ChunkSize
					i := ly*ChunkSize + lx
					c.setRGBKeepHeight(i, t.Orig)
					c.NeedsUpload = true
				}
				continue
			}
			out = append(out, t)
		}
		w.temp = out
	}

	// Process burning trees.
	for key, tb := range w.burningTrees {
		tb.timer -= dt
		if tb.timer <= 0 {
			dropCount := 1
			if len(tb.Pixels) > 20 {
				dropCount = 2
			}
			for d := 0; d < dropCount && len(tb.Pixels) > 0; d++ {
				idx := int(tb.rng % uint64(len(tb.Pixels)))
				px := tb.Pixels[idx]
				w.BurnPixelWithColor(px.X, px.Y, w.burnedGroundColorAt(px.X, px.Y))
				tb.Pixels = append(tb.Pixels[:idx], tb.Pixels[idx+1:]...)
				tb.rng = hash2D(tb.rng, px.X, px.Y)
			}
			tb.timer = tb.dropInterval
		}
		if len(tb.Pixels) == 0 {
			delete(w.burningTrees, key)
		}
	}

	// Process burning buildings.
	for key, bb := range w.burningBuildings {
		if bb.smolder {
			bb.smolderTimer -= dt
			bb.smolderDuration -= dt
			if bb.smolderTimer <= 0 {
				if len(bb.Pixels) > 0 {
					idx := int(bb.rng % uint64(len(bb.Pixels)))
					px := bb.Pixels[idx]
					dirs := [][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}}
					d := int(bb.rng & 3)
					nx := px.X + dirs[d][0]
					ny := px.Y + dirs[d][1]
					if nx >= 0 && ny >= 0 && nx < WorldWidth && ny < WorldHeight {
						col := w.ColorAt(nx, ny)
						if (col.R >= 90 && col.R <= 210) && (col.G >= 80 && col.G <= 180) {
							if w.BurnPixelWithColor(nx, ny, w.burnedGroundColorAt(nx, ny)) {
								for j := 0; j < len(bb.Pixels); j++ {
									if bb.Pixels[j].X == nx && bb.Pixels[j].Y == ny {
										bb.Pixels = append(bb.Pixels[:j], bb.Pixels[j+1:]...)
										break
									}
								}
							}
						}
					}
					bb.rng = hash2D(bb.rng, px.X, px.Y)
				}
				bb.smolderTimer = bb.smolderStep
			}
			if bb.smolderDuration <= 0 {
				bb.smolder = false
				bb.timer = 0.08
			}
			continue
		}

		// Collapse phase.
		bb.timer -= dt
		if bb.timer <= 0 {
			if len(bb.Pixels) > 0 {
				idx := int(bb.rng % uint64(len(bb.Pixels)))
				px := bb.Pixels[idx]
				w.BurnPixelWithColor(px.X, px.Y, w.burnedGroundColorAt(px.X, px.Y))
				bb.Pixels = append(bb.Pixels[:idx], bb.Pixels[idx+1:]...)
				bb.rng = hash2D(bb.rng, px.X, px.Y)
			}
			bb.timer = bb.stepInterval
		}
		if len(bb.Pixels) == 0 {
			delete(w.burningBuildings, key)
		}
	}
}
