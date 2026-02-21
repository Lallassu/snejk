//go:build android

package game

import (
	"container/heap"
	_ "embed"
	"encoding/binary"
	"fmt"
	"math"
	"strings"
	"time"

	"golang.org/x/mobile/app"
	"golang.org/x/mobile/event/lifecycle"
	"golang.org/x/mobile/event/paint"
	"golang.org/x/mobile/event/size"
	"golang.org/x/mobile/event/touch"
	"golang.org/x/mobile/gl"
)

//go:embed font_alt.png
var fontPNGMobile []byte

type mobileGame struct {
	seed uint64

	world     *World
	peds      *PedestrianSystem
	traffic   *TrafficSystem
	particles *ParticleSystem
	weather   *WeatherSystem
	bonuses   *BonusSystem
	cops      *CopSystem
	mil       *MilitarySystem
	session   *GameSession
	snake     *Snake
	cam       Camera

	now       float64
	stateTime float64

	// touch state
	activeTouch touch.Sequence
	touchDown   bool
	moveTargets []moveTarget
	lastTouchX  float32
	lastTouchY  float32

	// software frame buffer (world pixel space)
	frame []byte

	// reusable sprite buffers
	pedBuf        []float32
	glowBuf       []float32
	normBuf       []float32
	carShadowBuf  []float32
	carHeadBuf    []float32
	targetGlowBuf []float32
	snakeLitBuf   []float32

	// GL blit resources
	prog     gl.Program
	tex      gl.Texture
	vbo      gl.Buffer
	aPos     gl.Attrib
	aUV      gl.Attrib
	uTex     gl.Uniform
	glReady  bool
	fbWidth  int
	fbHeight int

	// GL sprite resources (desktop-like render path).
	spriteProg    gl.Program
	glowProg      gl.Program
	bonusProg     gl.Program
	npcProg       gl.Program
	spriteVBO     gl.Buffer
	carTexBase    gl.Texture
	copCarTex     gl.Texture
	trafficCarBuf []float32
	copCarBuf     []float32

	spAPos     gl.Attrib
	spASize    gl.Attrib
	spAColor   gl.Attrib
	spARot     gl.Attrib
	spUCamera  gl.Uniform
	spUZoom    gl.Uniform
	spURes     gl.Uniform
	spUAmbient gl.Uniform
	spUSunTint gl.Uniform

	glowAPos    gl.Attrib
	glowASize   gl.Attrib
	glowAColor  gl.Attrib
	glowARot    gl.Attrib
	glowUCamera gl.Uniform
	glowUZoom   gl.Uniform
	glowURes    gl.Uniform

	bonusAPos     gl.Attrib
	bonusASize    gl.Attrib
	bonusAColor   gl.Attrib
	bonusARot     gl.Attrib
	bonusUCamera  gl.Uniform
	bonusUZoom    gl.Uniform
	bonusURes     gl.Uniform
	bonusUAmbient gl.Uniform
	bonusUSunTint gl.Uniform

	npcAPos       gl.Attrib
	npcASize      gl.Attrib
	npcAColor     gl.Attrib
	npcARot       gl.Attrib
	npcUCamera    gl.Uniform
	npcUZoom      gl.Uniform
	npcURes       gl.Uniform
	npcUCarTex    gl.Uniform
	npcUCarAspect gl.Uniform

	// HUD text (font atlas) resources.
	textProg  gl.Program
	textVBO   gl.Buffer
	textFont  gl.Texture
	textAPos  gl.Attrib
	textAUV   gl.Attrib
	textACol  gl.Attrib
	textURes  gl.Uniform
	textUFont gl.Uniform
	textBuf   []float32
}

type moveTarget struct {
	X   float64
	Y   float64
	TTL float64
}

type pathPoint struct {
	X int
	Y int
}

type pathNode struct {
	Idx int
	G   int
	F   int
}

type pathMinHeap []pathNode

func (h pathMinHeap) Len() int { return len(h) }

func (h pathMinHeap) Less(i, j int) bool {
	if h[i].F == h[j].F {
		return h[i].G > h[j].G
	}
	return h[i].F < h[j].F
}

func (h pathMinHeap) Swap(i, j int) { h[i], h[j] = h[j], h[i] }

func (h *pathMinHeap) Push(x any) { *h = append(*h, x.(pathNode)) }

func (h *pathMinHeap) Pop() any {
	old := *h
	n := len(old)
	item := old[n-1]
	*h = old[:n-1]
	return item
}

const (
	moveWaypointBaseTTL = 15.0
	moveWaypointMaxTTL  = 36.0
)

func gridIdx(x, y int) int {
	return y*WorldWidth + x
}

func idxToGrid(idx int) (int, int) {
	return idx % WorldWidth, idx / WorldWidth
}

func absInt(v int) int {
	if v < 0 {
		return -v
	}
	return v
}

func pathHeuristic(x0, y0, x1, y1 int) int {
	dx := absInt(x1 - x0)
	dy := absInt(y1 - y0)
	if dx < dy {
		dx, dy = dy, dx
	}
	return 14*dy + 10*(dx-dy)
}

func walkableForSnakePath(w *World, x, y int) bool {
	if w == nil {
		return false
	}
	const clearance = 1
	for yy := y - clearance; yy <= y+clearance; yy++ {
		for xx := x - clearance; xx <= x+clearance; xx++ {
			if xx < 0 || yy < 0 || xx >= WorldWidth || yy >= WorldHeight {
				return false
			}
			if w.HeightAt(xx, yy) > 0 {
				return false
			}
		}
	}
	return true
}

func nearestWalkableForSnakePath(w *World, x, y, maxRadius int) (int, int, bool) {
	if w == nil {
		return 0, 0, false
	}
	x = clamp(x, 0, WorldWidth-1)
	y = clamp(y, 0, WorldHeight-1)
	if walkableForSnakePath(w, x, y) {
		return x, y, true
	}
	for r := 1; r <= maxRadius; r++ {
		minX := clamp(x-r, 0, WorldWidth-1)
		maxX := clamp(x+r, 0, WorldWidth-1)
		minY := clamp(y-r, 0, WorldHeight-1)
		maxY := clamp(y+r, 0, WorldHeight-1)
		bestD2 := int(^uint(0) >> 1)
		bestX, bestY := 0, 0
		found := false

		for xx := minX; xx <= maxX; xx++ {
			for _, yy := range [...]int{minY, maxY} {
				if !walkableForSnakePath(w, xx, yy) {
					continue
				}
				dx := xx - x
				dy := yy - y
				d2 := dx*dx + dy*dy
				if d2 < bestD2 {
					bestD2 = d2
					bestX, bestY = xx, yy
					found = true
				}
			}
		}
		for yy := minY + 1; yy <= maxY-1; yy++ {
			for _, xx := range [...]int{minX, maxX} {
				if !walkableForSnakePath(w, xx, yy) {
					continue
				}
				dx := xx - x
				dy := yy - y
				d2 := dx*dx + dy*dy
				if d2 < bestD2 {
					bestD2 = d2
					bestX, bestY = xx, yy
					found = true
				}
			}
		}
		if found {
			return bestX, bestY, true
		}
	}
	return 0, 0, false
}

func hasLineClearance(w *World, ax, ay, bx, by int) bool {
	x0, y0 := ax, ay
	x1, y1 := bx, by
	dx := absInt(x1 - x0)
	dy := absInt(y1 - y0)
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
		if !walkableForSnakePath(w, x0, y0) {
			return false
		}
		if x0 == x1 && y0 == y1 {
			return true
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

func simplifyPathPoints(path []pathPoint, w *World) []pathPoint {
	if len(path) <= 2 || w == nil {
		out := make([]pathPoint, len(path))
		copy(out, path)
		return out
	}
	out := make([]pathPoint, 0, len(path))
	anchor := 0
	out = append(out, path[anchor])
	for anchor < len(path)-1 {
		far := anchor + 1
		for far+1 < len(path) {
			if !hasLineClearance(w, path[anchor].X, path[anchor].Y, path[far+1].X, path[far+1].Y) {
				break
			}
			far++
		}
		out = append(out, path[far])
		anchor = far
	}
	return out
}

func findPathPoints(w *World, sx, sy, gx, gy int) []pathPoint {
	startX, startY, okStart := nearestWalkableForSnakePath(w, sx, sy, 24)
	goalX, goalY, okGoal := nearestWalkableForSnakePath(w, gx, gy, 24)
	if !okStart || !okGoal {
		return nil
	}
	if startX == goalX && startY == goalY {
		return []pathPoint{{X: goalX, Y: goalY}}
	}
	if hasLineClearance(w, startX, startY, goalX, goalY) {
		return []pathPoint{
			{X: startX, Y: startY},
			{X: goalX, Y: goalY},
		}
	}

	const nodeCount = WorldWidth * WorldHeight
	const inf = int(^uint(0) >> 2)
	gScore := make([]int, nodeCount)
	parent := make([]int, nodeCount)
	closed := make([]bool, nodeCount)
	for i := 0; i < nodeCount; i++ {
		gScore[i] = inf
		parent[i] = -1
	}

	startIdx := gridIdx(startX, startY)
	goalIdx := gridIdx(goalX, goalY)
	gScore[startIdx] = 0
	open := &pathMinHeap{}
	heap.Init(open)
	heap.Push(open, pathNode{
		Idx: startIdx,
		G:   0,
		F:   pathHeuristic(startX, startY, goalX, goalY),
	})

	neighbors := [...]struct {
		DX   int
		DY   int
		Cost int
	}{
		{DX: 1, DY: 0, Cost: 10},
		{DX: -1, DY: 0, Cost: 10},
		{DX: 0, DY: 1, Cost: 10},
		{DX: 0, DY: -1, Cost: 10},
		{DX: 1, DY: 1, Cost: 14},
		{DX: 1, DY: -1, Cost: 14},
		{DX: -1, DY: 1, Cost: 14},
		{DX: -1, DY: -1, Cost: 14},
	}

	found := false
	for open.Len() > 0 {
		cur := heap.Pop(open).(pathNode)
		if cur.G != gScore[cur.Idx] {
			continue
		}
		if closed[cur.Idx] {
			continue
		}
		if cur.Idx == goalIdx {
			found = true
			break
		}
		closed[cur.Idx] = true

		cx, cy := idxToGrid(cur.Idx)
		for _, n := range neighbors {
			nx := cx + n.DX
			ny := cy + n.DY
			if nx < 0 || ny < 0 || nx >= WorldWidth || ny >= WorldHeight {
				continue
			}
			if !walkableForSnakePath(w, nx, ny) {
				continue
			}
			// Prevent corner-cutting through blocked corners.
			if n.DX != 0 && n.DY != 0 {
				if !walkableForSnakePath(w, cx+n.DX, cy) || !walkableForSnakePath(w, cx, cy+n.DY) {
					continue
				}
			}
			nidx := gridIdx(nx, ny)
			if closed[nidx] {
				continue
			}
			tentative := cur.G + n.Cost
			if tentative >= gScore[nidx] {
				continue
			}
			gScore[nidx] = tentative
			parent[nidx] = cur.Idx
			heap.Push(open, pathNode{
				Idx: nidx,
				G:   tentative,
				F:   tentative + pathHeuristic(nx, ny, goalX, goalY),
			})
		}
	}
	if !found {
		return nil
	}

	path := make([]pathPoint, 0, 128)
	for cur := goalIdx; cur >= 0; cur = parent[cur] {
		px, py := idxToGrid(cur)
		path = append(path, pathPoint{X: px, Y: py})
		if cur == startIdx {
			break
		}
	}
	for i, j := 0, len(path)-1; i < j; i, j = i+1, j-1 {
		path[i], path[j] = path[j], path[i]
	}
	return simplifyPathPoints(path, w)
}

const spriteVertSrcMobile = `
attribute vec2 aWorldPos;
attribute float aSize;
attribute vec4 aColor;
attribute float aRotation;
uniform vec2 uCamera;
uniform vec2 uZoom;
uniform vec2 uResolution;
varying vec4 vColor;
varying float vRotation;
void main() {
  vec2 screenPos = (aWorldPos - uCamera) * uZoom + uResolution * 0.5;
  vec2 ndc = (screenPos / uResolution) * 2.0 - 1.0;
  ndc.y = -ndc.y;
  gl_Position = vec4(ndc, 0.0, 1.0);
  float zoom = min(uZoom.x, uZoom.y);
  float ps = floor(aSize * zoom + 0.5);
  gl_PointSize = max(1.0, ps);
  vColor = aColor;
  vRotation = aRotation;
}`

const spriteFragSrcMobile = `
precision mediump float;
uniform float uAmbient;
uniform vec3 uSunTint;
varying vec4 vColor;
varying float vRotation;
void main() {
  float keep = vRotation * 0.0;
  gl_FragColor = vec4(vColor.rgb * uAmbient * uSunTint, vColor.a + keep);
}`

const glowFragSrcMobile = `
precision mediump float;
varying vec4 vColor;
varying float vRotation;
void main() {
  float dist = length(gl_PointCoord - vec2(0.5)) * 2.0;
  float falloff = clamp(1.0 - dist, 0.0, 1.0);
  falloff = falloff * falloff;
  float keep = vRotation * 0.0;
  gl_FragColor = vec4(vColor.rgb * falloff, 1.0 + keep);
}`

const bonusFragSrcMobile = `
precision mediump float;
uniform float uAmbient;
uniform vec3 uSunTint;
varying vec4 vColor;
varying float vRotation;
void main() {
  vec2 uv = gl_PointCoord - vec2(0.5);
  float c = cos(vRotation);
  float s = sin(vRotation);
  vec2 rot = vec2(c * uv.x - s * uv.y, s * uv.x + c * uv.y);
  float outer = 0.44;
  float inner = 0.34;
  float ax = abs(rot.x);
  float ay = abs(rot.y);
  if (ax > outer || ay > outer) discard;
  vec3 col;
  float alpha = vColor.a;
  if (ax > inner || ay > inner) {
    col = vec3(0.04, 0.04, 0.04);
  } else {
    col = vColor.rgb;
    float hiX = max(0.0, -rot.x - 0.04);
    float hiY = max(0.0, -rot.y - 0.04);
    float hi = clamp((hiX + hiY) * 2.2, 0.0, 0.5);
    col = mix(col, vec3(1.0), hi);
    float shX = max(0.0, rot.x - 0.04);
    float shY = max(0.0, rot.y - 0.04);
    float sh = clamp((shX + shY) * 1.8, 0.0, 0.35);
    col = mix(col, vec3(0.0), sh);
  }
  gl_FragColor = vec4(col * uAmbient * uSunTint, alpha);
}`

const npcFragSrcMobile = `
precision mediump float;
uniform sampler2D uCarTex;
uniform float uCarAspect;
varying vec4 vColor;
varying float vRotation;
void main() {
  vec2 uv = gl_PointCoord - vec2(0.5);
  float c = cos(vRotation);
  float s = sin(vRotation);
  vec2 rot = vec2(c * uv.x - s * uv.y, s * uv.x + c * uv.y);
  rot.x *= uCarAspect;
  uv = rot + vec2(0.5);
  if (uv.x < 0.0 || uv.x > 1.0 || uv.y < 0.0 || uv.y > 1.0) discard;
  vec4 t = texture2D(uCarTex, uv);
  vec3 col = t.rgb * vColor.rgb;
  float a = t.a * vColor.a;
  if (a < 0.01) discard;
  gl_FragColor = vec4(col, a);
}`

func newMobileGame(seed uint64) *mobileGame {
	g := &mobileGame{
		seed:  seed,
		frame: make([]byte, WorldWidth*WorldHeight*4),
		cam: Camera{
			X:    float64(WorldWidth) / 2,
			Y:    float64(WorldHeight) / 2,
			Zoom: DefaultZoom,
		},
	}
	g.resetSystems()
	return g
}

func (g *mobileGame) resetSystems() {
	seed := g.seed
	world := NewWorld(seed)
	startTheme, _ := PickLevelTheme(seed, 1, 0, -1)
	world.Theme = startTheme
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

	g.world = world
	g.peds = NewPedestrianSystem(400, seed^0xFED)
	g.traffic = NewTrafficSystem(seed ^ 0xCAFE)
	g.particles = NewParticleSystem(MaxParticles, seed^0xBEAD)
	g.weather = NewWeatherSystem(seed ^ 0x57A7)
	g.bonuses = NewBonusSystem(seed^0xB0B, 5)
	g.cops = NewCopSystem(seed ^ 0xC095)
	g.mil = NewMilitarySystem(seed ^ 0xA7A1)
	g.session = NewGameSession()
	g.snake = nil

	_ = NewEventBus()
	g.stateTime = 0
	g.clearMoveTarget()
}

func (g *mobileGame) handleTouch(e touch.Event) {
	switch e.Type {
	case touch.TypeBegin:
		if g.session != nil && g.session.State == StateMenu {
			PlaySound(SoundMenuSelect)
			StartLevelMusic(1)
			g.session.StartLevel(1, g.world, g.peds, g.traffic, g.bonuses, g.cops, g.mil, &g.snake, g.particles, g.seed)
			g.weather.Configure(g.session.Weather, g.session.WeatherSeed)
			g.stateTime = 0
			g.clearMoveTarget()
			g.touchDown = false
			return
		}
		if !g.touchDown {
			g.activeTouch = e.Sequence
			g.touchDown = true
		}
		if e.Sequence == g.activeTouch {
			g.lastTouchX = e.X
			g.lastTouchY = e.Y
			if g.activateTargetAbilityAtScreen(e.X, e.Y) {
				// This tap was consumed by tactical targeting; don't also queue movement.
				g.touchDown = false
				return
			}
			g.enqueueMoveTargetFromScreen(e.X, e.Y)
		}
	case touch.TypeMove:
		if g.touchDown && e.Sequence == g.activeTouch {
			// Movement is tap-to-set. Dragging should not continuously retarget.
			g.lastTouchX = e.X
			g.lastTouchY = e.Y
		}
	case touch.TypeEnd:
		if g.touchDown && e.Sequence == g.activeTouch {
			g.touchDown = false
		}
	}
}

func (g *mobileGame) targetSelectingActive() bool {
	return g.session != nil &&
		g.session.State == StatePlaying &&
		g.snake != nil &&
		g.snake.Alive &&
		g.snake.TargetNukeTimer > 0
}

func (g *mobileGame) setTargetingPrompt(rem float64) {
	if g.snake == nil {
		return
	}
	g.snake.PowerupTimer = 1.1
	msg, col := g.snake.TargetingPrompt(rem)
	g.snake.PowerupMsg = strings.ReplaceAll(msg, "CLICK", "TAP")
	g.snake.PowerupCol = col
}

func (g *mobileGame) setTargetingLockLostPrompt() {
	if g.snake == nil {
		return
	}
	msg, col := g.snake.TargetingLockLostPrompt()
	g.snake.PowerupMsg = msg
	g.snake.PowerupCol = col
	g.snake.PowerupTimer = 1.6
}

func (g *mobileGame) activateTargetAbilityAtScreen(sx, sy float32) bool {
	if !g.targetSelectingActive() {
		return false
	}
	wxF, wyF := g.screenToWorld(float64(sx), float64(sy))
	wx := clamp(int(math.Round(wxF)), 0, WorldWidth-1)
	wy := clamp(int(math.Round(wyF)), 0, WorldHeight-1)
	g.snake.ActivateTargetAbilityAt(wx, wy, g.world, g.peds, g.traffic, g.particles, &g.cam, g.cops, g.mil)
	g.clearMoveTarget()
	return true
}

func (g *mobileGame) clearMoveTarget() {
	g.moveTargets = g.moveTargets[:0]
}

func (g *mobileGame) setMoveRouteToWorld(goalX, goalY float64) {
	const maxTTLStep = 2.0
	g.moveTargets = g.moveTargets[:0]
	if g.world == nil {
		return
	}

	// Build an obstacle-aware route from current snake head to destination.
	startX, startY := goalX, goalY
	if g.snake != nil {
		startX, startY = g.snake.Head()
	}
	path := findPathPoints(
		g.world,
		clamp(int(math.Round(startX)), 0, WorldWidth-1),
		clamp(int(math.Round(startY)), 0, WorldHeight-1),
		clamp(int(math.Round(goalX)), 0, WorldWidth-1),
		clamp(int(math.Round(goalY)), 0, WorldHeight-1),
	)
	if len(path) == 0 {
		// Fallback: direct target, still clamped to world bounds.
		g.moveTargets = append(g.moveTargets, moveTarget{
			X:   clampF(goalX, 0, float64(WorldWidth-1)),
			Y:   clampF(goalY, 0, float64(WorldHeight-1)),
			TTL: moveWaypointBaseTTL,
		})
		return
	}

	// Skip the first point (current position) and keep waypoints to goal.
	for i := 1; i < len(path); i++ {
		ttl := moveWaypointBaseTTL + float64(i-1)*maxTTLStep
		if ttl > moveWaypointMaxTTL {
			ttl = moveWaypointMaxTTL
		}
		g.moveTargets = append(g.moveTargets, moveTarget{
			X:   float64(path[i].X) + 0.5,
			Y:   float64(path[i].Y) + 0.5,
			TTL: ttl,
		})
	}
	if len(g.moveTargets) == 0 {
		g.moveTargets = append(g.moveTargets, moveTarget{
			X:   float64(path[len(path)-1].X) + 0.5,
			Y:   float64(path[len(path)-1].Y) + 0.5,
			TTL: moveWaypointBaseTTL,
		})
	}
}

func (g *mobileGame) enqueueMoveTargetFromScreen(sx, sy float32) {
	wx, wy := g.screenToWorld(float64(sx), float64(sy))
	g.setMoveRouteToWorld(wx, wy)
}

func (g *mobileGame) popMoveTarget() {
	if len(g.moveTargets) == 0 {
		return
	}
	copy(g.moveTargets, g.moveTargets[1:])
	g.moveTargets = g.moveTargets[:len(g.moveTargets)-1]
}

func (g *mobileGame) decayMoveTargets(dt float64) {
	if len(g.moveTargets) == 0 || dt <= 0 {
		return
	}
	dst := g.moveTargets[:0]
	for i := range g.moveTargets {
		t := g.moveTargets[i]
		t.TTL -= dt
		if t.TTL > 0 {
			dst = append(dst, t)
		}
	}
	g.moveTargets = dst
}

func (g *mobileGame) step(dt float64) {
	if dt <= 0 || dt > 0.1 {
		dt = 0.016
	}
	g.now += dt
	g.decayMoveTargets(dt)

	// State transitions are touch-driven/automatic on mobile.
	switch g.session.State {
	case StateMenu:
		// Wait for tap-to-start from handleTouch.
	case StateLevelComplete:
		g.stateTime += dt
		if g.stateTime > 0.8 {
			nextLevel := g.session.CurrentLevel + 1
			StartLevelMusic(nextLevel)
			g.session.StartLevel(nextLevel, g.world, g.peds, g.traffic, g.bonuses, g.cops, g.mil, &g.snake, g.particles, g.seed)
			g.weather.Configure(g.session.Weather, g.session.WeatherSeed)
			g.stateTime = 0
			g.clearMoveTarget()
		}
	case StateLevelFailed:
		g.stateTime += dt
		if g.stateTime > 0.8 {
			StartLevelMusic(g.session.CurrentLevel)
			g.session.StartLevel(g.session.CurrentLevel, g.world, g.peds, g.traffic, g.bonuses, g.cops, g.mil, &g.snake, g.particles, g.seed)
			g.weather.Configure(g.session.Weather, g.session.WeatherSeed)
			g.stateTime = 0
			g.clearMoveTarget()
		}
	}

	if g.session.State == StatePlaying {
		targetSelecting := false
		if g.snake != nil && g.snake.Alive && g.snake.TargetNukeTimer > 0 {
			targetSelecting = true
			g.snake.TargetNukeTimer -= dt
			rem := g.snake.TargetNukeTimer
			if rem < 0 {
				rem = 0
			}
			g.setTargetingPrompt(rem)
			if g.snake.TargetNukeTimer <= 0 {
				g.snake.TargetNukeTimer = 0
				g.setTargetingLockLostPrompt()
			}
		}

		if !targetSelecting && g.snake != nil && g.snake.Alive {
			if g.snake.AITimer <= 0 {
				steer, idle := g.touchSteerTarget()
				hx, hy := g.snake.Head()
				if g.bonuses != nil {
					bestDist := 5.0
					for i := range g.bonuses.Boxes {
						b := &g.bonuses.Boxes[i]
						if !b.Alive {
							continue
						}
						d := math.Hypot(b.X-hx, b.Y-hy)
						if d < bestDist {
							bestDist = d
							steer = math.Atan2(b.Y-hy, b.X-hx)
							idle = false
						}
					}
				}
				if g.snake.BounceTimer > 0 {
					steer = g.snake.BounceDir
					idle = false
				} else if !idle && g.world != nil {
					steer = WallAvoidAngle(hx, hy, steer, g.world)
				}
				g.snake.Idle = idle
				if !idle {
					g.snake.Steer(steer, dt)
				}
			} else {
				g.snake.Idle = false
			}
			g.snake.Update(dt, g.world, g.peds, g.traffic, g.bonuses, g.particles, &g.cam, g.cops, g.mil)
		}

		g.session.Update(dt)
		g.cam.UpdateShake(dt, g.seed^uint64(g.now*1000))
		g.world.Update(dt)
		UpdateBurnVisuals(g.world, g.particles, dt)
		g.peds.Update(dt, g.world, g.snake, g.particles)
		sunAmbNow, _, _, _ := SunCycleLight(g.session.LevelTimer)
		g.traffic.NightFactor = NightIntensityFromAmbient(sunAmbNow)
		g.traffic.Update(dt, g.world, g.particles, g.peds, &g.cam)
		g.weather.UpdateAndSpawn(g.particles, dt)
		g.particles.UpdateWithShockwaveDamage(dt, g.world, g.peds, g.cops, g.mil)
		snakeHP := 1.0
		if g.snake != nil {
			snakeHP = g.snake.HP.Fraction()
		}
		g.bonuses.Update(dt, g.peds.AliveCount(), snakeHP)
		g.bonuses.SpawnSparks(g.particles, dt)
		g.cops.Update(dt, g.snake, g.world, g.particles, &g.cam, g.now)
		g.mil.Update(dt, g.snake, g.world, g.particles, &g.cam, g.now)

		g.peds.RemoveDead()
		g.traffic.RemoveDead()
		g.cops.RemoveDead()
		g.mil.RemoveDead()
		g.session.CheckLevelEnd(g.peds, g.snake)
	}

	_, _, _, _, zoomX, zoomY := g.renderViewport()
	zoom := math.Min(zoomX, zoomY)
	if zoom <= 0 {
		zoom = 1
	}
	g.cam.Zoom = zoom
	g.cam.X = float64(WorldWidth) * 0.5
	g.cam.Y = float64(WorldHeight) * 0.5
}

func (g *mobileGame) touchSteerTarget() (float64, bool) {
	if g.snake == nil {
		return 0, true
	}
	hx, hy := g.snake.Head()
	for len(g.moveTargets) > 0 {
		target := g.moveTargets[0]
		txi := clamp(int(math.Round(target.X)), 0, WorldWidth-1)
		tyi := clamp(int(math.Round(target.Y)), 0, WorldHeight-1)
		if g.world != nil && !walkableForSnakePath(g.world, txi, tyi) {
			g.popMoveTarget()
			continue
		}

		// If we already have clear line to a farther waypoint, skip intermediate one.
		if len(g.moveTargets) > 1 && g.world != nil {
			next := g.moveTargets[1]
			nxi := clamp(int(math.Round(next.X)), 0, WorldWidth-1)
			nyi := clamp(int(math.Round(next.Y)), 0, WorldHeight-1)
			hxi := clamp(int(math.Round(hx)), 0, WorldWidth-1)
			hyi := clamp(int(math.Round(hy)), 0, WorldHeight-1)
			if hasLineClearance(g.world, hxi, hyi, nxi, nyi) {
				g.popMoveTarget()
				continue
			}
		}

		dx := target.X - hx
		dy := target.Y - hy
		dist := math.Hypot(dx, dy)
		if dist < 1.5 {
			g.popMoveTarget()
			continue
		}
		return math.Atan2(dy, dx), false
	}
	return g.snake.Heading, true
}

func clampViewCenter(camX, camY, viewW, viewH float64) (float64, float64) {
	if viewW >= float64(WorldWidth) {
		camX = float64(WorldWidth) * 0.5
	} else {
		halfW := viewW * 0.5
		camX = clampF(camX, halfW, float64(WorldWidth)-halfW)
	}
	if viewH >= float64(WorldHeight) {
		camY = float64(WorldHeight) * 0.5
	} else {
		halfH := viewH * 0.5
		camY = clampF(camY, halfH, float64(WorldHeight)-halfH)
	}
	return camX, camY
}

func (g *mobileGame) renderViewport() (x, y, w, h int, zoomX, zoomY float64) {
	if g.fbWidth <= 0 || g.fbHeight <= 0 {
		return 0, 0, 0, 0, 1, 1
	}
	// Fill mode with full map visibility: stretch to full window so gameplay area
	// always occupies the whole screen (no crop, no letterbox).
	x, y = 0, 0
	w, h = g.fbWidth, g.fbHeight
	zoomX = float64(w) / float64(WorldWidth)
	zoomY = float64(h) / float64(WorldHeight)
	if zoomX <= 0 {
		zoomX = 1
	}
	if zoomY <= 0 {
		zoomY = 1
	}
	return
}

func (g *mobileGame) screenToWorld(sx, sy float64) (float64, float64) {
	vx, vy, vw, vh, zoomX, zoomY := g.renderViewport()
	if vw <= 0 || vh <= 0 || zoomX <= 0 || zoomY <= 0 {
		return float64(WorldWidth) * 0.5, float64(WorldHeight) * 0.5
	}
	lx := clampF(sx-float64(vx), 0, float64(vw))
	ly := clampF(sy-float64(vy), 0, float64(vh))
	// Use non-shaken camera for stable tap placement.
	camX, camY := g.cam.X, g.cam.Y
	viewW := float64(vw) / zoomX
	viewH := float64(vh) / zoomY
	camX, camY = clampViewCenter(camX, camY, viewW, viewH)
	wx := camX + (lx-float64(vw)*0.5)/zoomX
	wy := camY + (ly-float64(vh)*0.5)/zoomY
	return clampF(wx, 0, float64(WorldWidth-1)), clampF(wy, 0, float64(WorldHeight-1))
}

func clampByte(v float64) uint8 {
	if v < 0 {
		return 0
	}
	if v > 255 {
		return 255
	}
	return uint8(v + 0.5)
}

func (g *mobileGame) renderSoftware() {
	sunAmb, sunTR, sunTG, sunTB := SunCycleLight(g.session.LevelTimer)
	sunAngle, sunSlope := SunCycleShadow(g.session.LevelTimer)
	g.world.UpdateSun(sunAngle, sunSlope)
	for cy := 0; cy <= g.world.maxCy; cy++ {
		for cx := 0; cx <= g.world.maxCx; cx++ {
			c := g.world.GetChunk(cx, cy)
			if c == nil {
				continue
			}
			if c.NeedsShadow {
				c.RecomputeShadows(g.world)
			}
			baseX, baseY := c.WorldOrigin()
			for ly := 0; ly < ChunkSize; ly++ {
				wy := baseY + ly
				if wy < 0 || wy >= WorldHeight {
					continue
				}
				for lx := 0; lx < ChunkSize; lx++ {
					wx := baseX + lx
					if wx < 0 || wx >= WorldWidth {
						continue
					}
					src := (ly*ChunkSize + lx) * 4
					dst := (wy*WorldWidth + wx) * 4
					shade := float32(c.Pixels[src+3]) / 255.0
					g.frame[dst+0] = clampByte(float64(c.Pixels[src+0]) * float64(shade) * float64(sunAmb) * float64(sunTR))
					g.frame[dst+1] = clampByte(float64(c.Pixels[src+1]) * float64(shade) * float64(sunAmb) * float64(sunTG))
					g.frame[dst+2] = clampByte(float64(c.Pixels[src+2]) * float64(shade) * float64(sunAmb) * float64(sunTB))
					g.frame[dst+3] = 255
				}
			}
		}
	}
}

func (g *mobileGame) drawSprites(buf []float32, additive bool) {
	for i := 0; i+7 < len(buf); i += 8 {
		x := buf[i+0]
		y := buf[i+1]
		size := buf[i+2]
		r := buf[i+3]
		gg := buf[i+4]
		b := buf[i+5]
		a := buf[i+6]
		if a <= 0 || size <= 0 {
			continue
		}
		if additive {
			g.drawCircle(x, y, size*0.5, r, gg, b, a, true)
		} else {
			g.drawSquare(x, y, size, r, gg, b, a, false)
		}
	}
}

func (g *mobileGame) drawSquare(cx, cy, size float32, r, gg, b, a float32, additive bool) {
	if size < 0.5 {
		size = 0.5
	}
	half := size * 0.5
	minX := int(math.Floor(float64(cx - half)))
	maxX := int(math.Ceil(float64(cx + half)))
	minY := int(math.Floor(float64(cy - half)))
	maxY := int(math.Ceil(float64(cy + half)))
	if maxX < 0 || maxY < 0 || minX >= WorldWidth || minY >= WorldHeight {
		return
	}
	if minX < 0 {
		minX = 0
	}
	if minY < 0 {
		minY = 0
	}
	if maxX >= WorldWidth {
		maxX = WorldWidth - 1
	}
	if maxY >= WorldHeight {
		maxY = WorldHeight - 1
	}

	srcR := float64(r * 255)
	srcG := float64(gg * 255)
	srcB := float64(b * 255)
	srcA := float64(a)
	for py := minY; py <= maxY; py++ {
		for px := minX; px <= maxX; px++ {
			o := (py*WorldWidth + px) * 4
			if additive {
				g.frame[o+0] = clampByte(float64(g.frame[o+0]) + srcR*srcA)
				g.frame[o+1] = clampByte(float64(g.frame[o+1]) + srcG*srcA)
				g.frame[o+2] = clampByte(float64(g.frame[o+2]) + srcB*srcA)
				g.frame[o+3] = 255
			} else {
				invA := 1.0 - srcA
				g.frame[o+0] = clampByte(float64(g.frame[o+0])*invA + srcR*srcA)
				g.frame[o+1] = clampByte(float64(g.frame[o+1])*invA + srcG*srcA)
				g.frame[o+2] = clampByte(float64(g.frame[o+2])*invA + srcB*srcA)
				g.frame[o+3] = 255
			}
		}
	}
}

func (g *mobileGame) drawOrientedRect(cx, cy, heading, halfW, halfL float32, r, gg, b, a float32, additive bool) {
	if halfW <= 0 || halfL <= 0 {
		return
	}
	fwdX := float32(math.Cos(float64(heading)))
	fwdY := float32(math.Sin(float64(heading)))
	sideX := -fwdY
	sideY := fwdX

	extX := float32(math.Abs(float64(fwdX))*float64(halfL) + math.Abs(float64(sideX))*float64(halfW))
	extY := float32(math.Abs(float64(fwdY))*float64(halfL) + math.Abs(float64(sideY))*float64(halfW))
	minX := int(math.Floor(float64(cx - extX)))
	maxX := int(math.Ceil(float64(cx + extX)))
	minY := int(math.Floor(float64(cy - extY)))
	maxY := int(math.Ceil(float64(cy + extY)))
	if maxX < 0 || maxY < 0 || minX >= WorldWidth || minY >= WorldHeight {
		return
	}
	if minX < 0 {
		minX = 0
	}
	if minY < 0 {
		minY = 0
	}
	if maxX >= WorldWidth {
		maxX = WorldWidth - 1
	}
	if maxY >= WorldHeight {
		maxY = WorldHeight - 1
	}

	srcR := float64(r * 255)
	srcG := float64(gg * 255)
	srcB := float64(b * 255)
	srcA := float64(a)
	halfWF := float64(halfW)
	halfLF := float64(halfL)
	fwdXF := float64(fwdX)
	fwdYF := float64(fwdY)
	sideXF := float64(sideX)
	sideYF := float64(sideY)
	cxF := float64(cx)
	cyF := float64(cy)

	for py := minY; py <= maxY; py++ {
		yy := float64(py) + 0.5 - cyF
		for px := minX; px <= maxX; px++ {
			xx := float64(px) + 0.5 - cxF
			lf := xx*fwdXF + yy*fwdYF
			lp := xx*sideXF + yy*sideYF
			if math.Abs(lf) > halfLF || math.Abs(lp) > halfWF {
				continue
			}
			o := (py*WorldWidth + px) * 4
			if additive {
				g.frame[o+0] = clampByte(float64(g.frame[o+0]) + srcR*srcA)
				g.frame[o+1] = clampByte(float64(g.frame[o+1]) + srcG*srcA)
				g.frame[o+2] = clampByte(float64(g.frame[o+2]) + srcB*srcA)
				g.frame[o+3] = 255
			} else {
				invA := 1.0 - srcA
				g.frame[o+0] = clampByte(float64(g.frame[o+0])*invA + srcR*srcA)
				g.frame[o+1] = clampByte(float64(g.frame[o+1])*invA + srcG*srcA)
				g.frame[o+2] = clampByte(float64(g.frame[o+2])*invA + srcB*srcA)
				g.frame[o+3] = 255
			}
		}
	}
}

func (g *mobileGame) drawCarShadows() {
	addShadow := func(x, y, heading float64) {
		fwdX := float32(math.Cos(heading))
		fwdY := float32(math.Sin(heading))
		ox := float32(x) + 0.6
		oy := float32(y) + 1.0
		sz := float32(CarSize * 0.95)
		for _, t := range [3]float32{-1.6, 0, 1.6} {
			g.drawCircle(ox+fwdX*t, oy+fwdY*t, sz*0.5, 0, 0, 0, 0.22, false)
		}
	}
	for i := range g.traffic.Cars {
		c := &g.traffic.Cars[i]
		if c.Alive {
			addShadow(c.X, c.Y, c.Heading)
		}
	}
	for i := range g.cops.Cars {
		c := &g.cops.Cars[i]
		if c.Alive {
			addShadow(c.X, c.Y, c.Heading)
		}
	}
}

func (g *mobileGame) drawTrafficCar(c *NPCCar) {
	baseR, baseG, baseB := c.R, c.G, c.B
	halfL := c.Size * 0.5
	halfW := c.Size * 0.5 * CarVisualAspect
	fwdX := float32(math.Cos(c.Heading))
	fwdY := float32(math.Sin(c.Heading))
	cx := float32(c.X)
	cy := float32(c.Y)

	g.drawOrientedRect(cx, cy, float32(c.Heading), halfW, halfL, baseR, baseG, baseB, 1.0, false)
	g.drawOrientedRect(cx+fwdX*halfL*0.18, cy+fwdY*halfL*0.18, float32(c.Heading), halfW*0.76, halfL*0.20, 0.56, 0.58, 0.62, 0.96, false)
	g.drawOrientedRect(cx-fwdX*halfL*0.12, cy-fwdY*halfL*0.12, float32(c.Heading), halfW*0.72, halfL*0.20, baseR*0.74, baseG*0.74, baseB*0.74, 1.0, false)
	g.drawOrientedRect(cx-fwdX*halfL*0.46, cy-fwdY*halfL*0.46, float32(c.Heading), halfW*0.82, halfL*0.14, baseR*0.92, baseG*0.92, baseB*0.92, 1.0, false)
}

func (g *mobileGame) drawCopCar(c *CopCar) {
	halfL := c.Size * 0.5
	halfW := c.Size * 0.5 * CarVisualAspect
	fwdX := float32(math.Cos(c.Heading))
	fwdY := float32(math.Sin(c.Heading))
	cx := float32(c.X)
	cy := float32(c.Y)
	heading := float32(c.Heading)

	g.drawOrientedRect(cx, cy, heading, halfW, halfL, 0.22, 0.40, 0.88, 1.0, false)
	g.drawOrientedRect(cx+fwdX*halfL*0.45, cy+fwdY*halfL*0.45, heading, halfW*0.84, halfL*0.16, 0.92, 0.92, 0.95, 1.0, false)
	g.drawOrientedRect(cx-fwdX*halfL*0.45, cy-fwdY*halfL*0.45, heading, halfW*0.84, halfL*0.16, 0.92, 0.92, 0.95, 1.0, false)
	g.drawOrientedRect(cx+fwdX*halfL*0.12, cy+fwdY*halfL*0.12, heading, halfW*0.72, halfL*0.18, 0.16, 0.24, 0.52, 0.96, false)
	g.drawOrientedRect(cx-fwdX*halfL*0.10, cy-fwdY*halfL*0.10, heading, halfW*0.66, halfL*0.18, 0.20, 0.33, 0.78, 1.0, false)
}

func (g *mobileGame) drawCircle(cx, cy, rad float32, r, gg, b, a float32, additive bool) {
	if rad < 0.5 {
		rad = 0.5
	}
	minX := int(math.Floor(float64(cx - rad)))
	maxX := int(math.Ceil(float64(cx + rad)))
	minY := int(math.Floor(float64(cy - rad)))
	maxY := int(math.Ceil(float64(cy + rad)))
	if maxX < 0 || maxY < 0 || minX >= WorldWidth || minY >= WorldHeight {
		return
	}
	if minX < 0 {
		minX = 0
	}
	if minY < 0 {
		minY = 0
	}
	if maxX >= WorldWidth {
		maxX = WorldWidth - 1
	}
	if maxY >= WorldHeight {
		maxY = WorldHeight - 1
	}

	r2 := rad * rad
	srcR := float64(r * 255)
	srcG := float64(gg * 255)
	srcB := float64(b * 255)
	srcA := float64(a)
	for py := minY; py <= maxY; py++ {
		dy := (float32(py) + 0.5) - cy
		for px := minX; px <= maxX; px++ {
			dx := (float32(px) + 0.5) - cx
			if dx*dx+dy*dy > r2 {
				continue
			}
			o := (py*WorldWidth + px) * 4
			if additive {
				g.frame[o+0] = clampByte(float64(g.frame[o+0]) + srcR*srcA)
				g.frame[o+1] = clampByte(float64(g.frame[o+1]) + srcG*srcA)
				g.frame[o+2] = clampByte(float64(g.frame[o+2]) + srcB*srcA)
				g.frame[o+3] = 255
			} else {
				invA := 1.0 - srcA
				g.frame[o+0] = clampByte(float64(g.frame[o+0])*invA + srcR*srcA)
				g.frame[o+1] = clampByte(float64(g.frame[o+1])*invA + srcG*srcA)
				g.frame[o+2] = clampByte(float64(g.frame[o+2])*invA + srcB*srcA)
				g.frame[o+3] = 255
			}
		}
	}
}

func (g *mobileGame) drawRing(cx, cy, innerR, outerR float32, r, gg, b, a float32, additive bool) {
	if outerR <= innerR || outerR < 0.75 {
		return
	}
	minX := int(math.Floor(float64(cx - outerR)))
	maxX := int(math.Ceil(float64(cx + outerR)))
	minY := int(math.Floor(float64(cy - outerR)))
	maxY := int(math.Ceil(float64(cy + outerR)))
	if maxX < 0 || maxY < 0 || minX >= WorldWidth || minY >= WorldHeight {
		return
	}
	if minX < 0 {
		minX = 0
	}
	if minY < 0 {
		minY = 0
	}
	if maxX >= WorldWidth {
		maxX = WorldWidth - 1
	}
	if maxY >= WorldHeight {
		maxY = WorldHeight - 1
	}

	outer2 := outerR * outerR
	inner2 := innerR * innerR
	midR := (innerR + outerR) * 0.5
	halfW := (outerR - innerR) * 0.5
	if halfW < 0.001 {
		return
	}

	srcR := float64(r * 255)
	srcG := float64(gg * 255)
	srcB := float64(b * 255)
	baseA := float64(a)

	for py := minY; py <= maxY; py++ {
		dy := (float32(py) + 0.5) - cy
		for px := minX; px <= maxX; px++ {
			dx := (float32(px) + 0.5) - cx
			d2 := dx*dx + dy*dy
			if d2 < inner2 || d2 > outer2 {
				continue
			}
			dist := float32(math.Sqrt(float64(d2)))
			falloff := 1.0 - math.Abs(float64(dist-midR))/float64(halfW)
			if falloff <= 0 {
				continue
			}
			srcA := baseA * falloff
			o := (py*WorldWidth + px) * 4
			if additive {
				g.frame[o+0] = clampByte(float64(g.frame[o+0]) + srcR*srcA)
				g.frame[o+1] = clampByte(float64(g.frame[o+1]) + srcG*srcA)
				g.frame[o+2] = clampByte(float64(g.frame[o+2]) + srcB*srcA)
				g.frame[o+3] = 255
			} else {
				invA := 1.0 - srcA
				g.frame[o+0] = clampByte(float64(g.frame[o+0])*invA + srcR*srcA)
				g.frame[o+1] = clampByte(float64(g.frame[o+1])*invA + srcG*srcA)
				g.frame[o+2] = clampByte(float64(g.frame[o+2])*invA + srcB*srcA)
				g.frame[o+3] = 255
			}
		}
	}
}

func f32bytes(vals []float32) []byte {
	out := make([]byte, len(vals)*4)
	for i, v := range vals {
		binary.LittleEndian.PutUint32(out[i*4:], math.Float32bits(v))
	}
	return out
}

func compileShader(glctx gl.Context, kind gl.Enum, src string) (gl.Shader, error) {
	sh := glctx.CreateShader(kind)
	glctx.ShaderSource(sh, src)
	glctx.CompileShader(sh)
	if glctx.GetShaderi(sh, gl.COMPILE_STATUS) == 0 {
		log := glctx.GetShaderInfoLog(sh)
		glctx.DeleteShader(sh)
		return gl.Shader{}, fmt.Errorf("shader compile failed: %s", log)
	}
	return sh, nil
}

func linkProgram(glctx gl.Context, vertSrc, fragSrc string) (gl.Program, error) {
	vs, err := compileShader(glctx, gl.VERTEX_SHADER, vertSrc)
	if err != nil {
		return gl.Program{}, err
	}
	fs, err := compileShader(glctx, gl.FRAGMENT_SHADER, fragSrc)
	if err != nil {
		glctx.DeleteShader(vs)
		return gl.Program{}, err
	}
	prog := glctx.CreateProgram()
	glctx.AttachShader(prog, vs)
	glctx.AttachShader(prog, fs)
	glctx.LinkProgram(prog)
	glctx.DeleteShader(vs)
	glctx.DeleteShader(fs)
	if glctx.GetProgrami(prog, gl.LINK_STATUS) == 0 {
		log := glctx.GetProgramInfoLog(prog)
		glctx.DeleteProgram(prog)
		return gl.Program{}, fmt.Errorf("program link failed: %s", log)
	}
	return prog, nil
}

func makeCarTextureBaseMobile(glctx gl.Context) gl.Texture {
	const s = 8
	pix := make([]byte, s*s*4)
	set := func(x, y int, r, g, b uint8) {
		i := (y*s + x) * 4
		pix[i+0] = r
		pix[i+1] = g
		pix[i+2] = b
		pix[i+3] = 255
	}
	for y := 0; y < s; y++ {
		var r, g, b uint8
		switch y / 2 {
		case 0:
			r, g, b = 190, 90, 80
		case 1:
			r, g, b = 140, 140, 140
		case 2:
			r, g, b = 145, 70, 62
		default:
			r, g, b = 190, 90, 80
		}
		for x := 0; x < s; x++ {
			set(x, y, r, g, b)
		}
	}
	tex := glctx.CreateTexture()
	glctx.BindTexture(gl.TEXTURE_2D, tex)
	glctx.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.NEAREST)
	glctx.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.NEAREST)
	glctx.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_S, gl.CLAMP_TO_EDGE)
	glctx.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_T, gl.CLAMP_TO_EDGE)
	glctx.TexImage2D(gl.TEXTURE_2D, 0, int(gl.RGBA), s, s, gl.RGBA, gl.UNSIGNED_BYTE, pix)
	return tex
}

func makeCopCarTextureMobile(glctx gl.Context) gl.Texture {
	const s = 8
	pix := make([]byte, s*s*4)
	white := [3]uint8{230, 230, 235}
	blue := [3]uint8{60, 100, 220}
	window := [3]uint8{40, 60, 140}
	roof := [3]uint8{50, 85, 200}
	bands := [8][3]uint8{
		white, white, window, window, roof, roof, blue, white,
	}
	for y := 0; y < s; y++ {
		col := bands[y]
		for x := 0; x < s; x++ {
			i := (y*s + x) * 4
			pix[i+0] = col[0]
			pix[i+1] = col[1]
			pix[i+2] = col[2]
			pix[i+3] = 255
		}
	}
	tex := glctx.CreateTexture()
	glctx.BindTexture(gl.TEXTURE_2D, tex)
	glctx.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.NEAREST)
	glctx.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.NEAREST)
	glctx.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_S, gl.CLAMP_TO_EDGE)
	glctx.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_T, gl.CLAMP_TO_EDGE)
	glctx.TexImage2D(gl.TEXTURE_2D, 0, int(gl.RGBA), s, s, gl.RGBA, gl.UNSIGNED_BYTE, pix)
	return tex
}

func (g *mobileGame) initGL(glctx gl.Context) error {
	if g.glReady {
		return nil
	}
	vertSrc := `
attribute vec2 aPos;
attribute vec2 aUV;
varying vec2 vUV;
void main() {
  vUV = aUV;
  gl_Position = vec4(aPos, 0.0, 1.0);
}`
	fragSrc := `
precision mediump float;
varying vec2 vUV;
uniform sampler2D uTex;
void main() {
  gl_FragColor = texture2D(uTex, vUV);
}`
	prog, err := linkProgram(glctx, vertSrc, fragSrc)
	if err != nil {
		return err
	}

	verts := []float32{
		-1, -1, 0, 1,
		1, -1, 1, 1,
		-1, 1, 0, 0,
		1, 1, 1, 0,
	}
	vbo := glctx.CreateBuffer()
	glctx.BindBuffer(gl.ARRAY_BUFFER, vbo)
	glctx.BufferData(gl.ARRAY_BUFFER, f32bytes(verts), gl.STATIC_DRAW)

	tex := glctx.CreateTexture()
	glctx.ActiveTexture(gl.TEXTURE0)
	glctx.BindTexture(gl.TEXTURE_2D, tex)
	glctx.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.NEAREST)
	glctx.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.NEAREST)
	glctx.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_S, gl.CLAMP_TO_EDGE)
	glctx.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_T, gl.CLAMP_TO_EDGE)
	glctx.TexImage2D(gl.TEXTURE_2D, 0, int(gl.RGBA), WorldWidth, WorldHeight, gl.RGBA, gl.UNSIGNED_BYTE, nil)

	g.prog = prog
	g.tex = tex
	g.vbo = vbo
	g.aPos = glctx.GetAttribLocation(prog, "aPos")
	g.aUV = glctx.GetAttribLocation(prog, "aUV")
	g.uTex = glctx.GetUniformLocation(prog, "uTex")

	g.spriteProg, err = linkProgram(glctx, spriteVertSrcMobile, spriteFragSrcMobile)
	if err != nil {
		return err
	}
	g.glowProg, err = linkProgram(glctx, spriteVertSrcMobile, glowFragSrcMobile)
	if err != nil {
		return err
	}
	g.bonusProg, err = linkProgram(glctx, spriteVertSrcMobile, bonusFragSrcMobile)
	if err != nil {
		return err
	}
	g.npcProg, err = linkProgram(glctx, spriteVertSrcMobile, npcFragSrcMobile)
	if err != nil {
		return err
	}

	g.spriteVBO = glctx.CreateBuffer()
	glctx.BindBuffer(gl.ARRAY_BUFFER, g.spriteVBO)
	glctx.BufferData(gl.ARRAY_BUFFER, nil, gl.STREAM_DRAW)

	g.spAPos = glctx.GetAttribLocation(g.spriteProg, "aWorldPos")
	g.spASize = glctx.GetAttribLocation(g.spriteProg, "aSize")
	g.spAColor = glctx.GetAttribLocation(g.spriteProg, "aColor")
	g.spARot = glctx.GetAttribLocation(g.spriteProg, "aRotation")
	g.spUCamera = glctx.GetUniformLocation(g.spriteProg, "uCamera")
	g.spUZoom = glctx.GetUniformLocation(g.spriteProg, "uZoom")
	g.spURes = glctx.GetUniformLocation(g.spriteProg, "uResolution")
	g.spUAmbient = glctx.GetUniformLocation(g.spriteProg, "uAmbient")
	g.spUSunTint = glctx.GetUniformLocation(g.spriteProg, "uSunTint")

	g.glowAPos = glctx.GetAttribLocation(g.glowProg, "aWorldPos")
	g.glowASize = glctx.GetAttribLocation(g.glowProg, "aSize")
	g.glowAColor = glctx.GetAttribLocation(g.glowProg, "aColor")
	g.glowARot = glctx.GetAttribLocation(g.glowProg, "aRotation")
	g.glowUCamera = glctx.GetUniformLocation(g.glowProg, "uCamera")
	g.glowUZoom = glctx.GetUniformLocation(g.glowProg, "uZoom")
	g.glowURes = glctx.GetUniformLocation(g.glowProg, "uResolution")

	g.bonusAPos = glctx.GetAttribLocation(g.bonusProg, "aWorldPos")
	g.bonusASize = glctx.GetAttribLocation(g.bonusProg, "aSize")
	g.bonusAColor = glctx.GetAttribLocation(g.bonusProg, "aColor")
	g.bonusARot = glctx.GetAttribLocation(g.bonusProg, "aRotation")
	g.bonusUCamera = glctx.GetUniformLocation(g.bonusProg, "uCamera")
	g.bonusUZoom = glctx.GetUniformLocation(g.bonusProg, "uZoom")
	g.bonusURes = glctx.GetUniformLocation(g.bonusProg, "uResolution")
	g.bonusUAmbient = glctx.GetUniformLocation(g.bonusProg, "uAmbient")
	g.bonusUSunTint = glctx.GetUniformLocation(g.bonusProg, "uSunTint")

	g.npcAPos = glctx.GetAttribLocation(g.npcProg, "aWorldPos")
	g.npcASize = glctx.GetAttribLocation(g.npcProg, "aSize")
	g.npcAColor = glctx.GetAttribLocation(g.npcProg, "aColor")
	g.npcARot = glctx.GetAttribLocation(g.npcProg, "aRotation")
	g.npcUCamera = glctx.GetUniformLocation(g.npcProg, "uCamera")
	g.npcUZoom = glctx.GetUniformLocation(g.npcProg, "uZoom")
	g.npcURes = glctx.GetUniformLocation(g.npcProg, "uResolution")
	g.npcUCarTex = glctx.GetUniformLocation(g.npcProg, "uCarTex")
	g.npcUCarAspect = glctx.GetUniformLocation(g.npcProg, "uCarAspect")

	g.carTexBase = makeCarTextureBaseMobile(glctx)
	g.copCarTex = makeCopCarTextureMobile(glctx)

	glctx.UseProgram(g.npcProg)
	glctx.Uniform1i(g.npcUCarTex, 1)
	if err := g.initTextGL(glctx); err != nil {
		return err
	}
	g.glReady = true
	return nil
}

func (g *mobileGame) destroyGL(glctx gl.Context) {
	if !g.glReady {
		return
	}
	glctx.DeleteBuffer(g.vbo)
	glctx.DeleteBuffer(g.spriteVBO)
	glctx.DeleteTexture(g.tex)
	glctx.DeleteTexture(g.carTexBase)
	glctx.DeleteTexture(g.copCarTex)
	g.destroyTextGL(glctx)
	glctx.DeleteProgram(g.prog)
	glctx.DeleteProgram(g.spriteProg)
	glctx.DeleteProgram(g.glowProg)
	glctx.DeleteProgram(g.bonusProg)
	glctx.DeleteProgram(g.npcProg)
	g.glReady = false
}

func setSpriteAttribs(glctx gl.Context, pos, size, color, rot gl.Attrib) {
	const stride = 8 * 4
	glctx.EnableVertexAttribArray(pos)
	glctx.EnableVertexAttribArray(size)
	glctx.EnableVertexAttribArray(color)
	glctx.EnableVertexAttribArray(rot)
	glctx.VertexAttribPointer(pos, 2, gl.FLOAT, false, stride, 0)
	glctx.VertexAttribPointer(size, 1, gl.FLOAT, false, stride, 8)
	glctx.VertexAttribPointer(color, 4, gl.FLOAT, false, stride, 12)
	glctx.VertexAttribPointer(rot, 1, gl.FLOAT, false, stride, 28)
}

func (g *mobileGame) drawLitSpritesGL(glctx gl.Context, buf []float32, additive bool, camX, camY, zoomX, zoomY float32, vw, vh int, sunAmb, sunTR, sunTG, sunTB float32) {
	if len(buf) == 0 {
		return
	}
	glctx.UseProgram(g.spriteProg)
	glctx.BindBuffer(gl.ARRAY_BUFFER, g.spriteVBO)
	glctx.BufferData(gl.ARRAY_BUFFER, f32bytes(buf), gl.STREAM_DRAW)
	setSpriteAttribs(glctx, g.spAPos, g.spASize, g.spAColor, g.spARot)
	glctx.Uniform2f(g.spUCamera, camX, camY)
	glctx.Uniform2f(g.spUZoom, zoomX, zoomY)
	glctx.Uniform2f(g.spURes, float32(vw), float32(vh))
	glctx.Uniform1f(g.spUAmbient, sunAmb)
	glctx.Uniform3f(g.spUSunTint, sunTR, sunTG, sunTB)
	glctx.Enable(gl.BLEND)
	if additive {
		glctx.BlendFunc(gl.ONE, gl.ONE)
	} else {
		glctx.BlendFunc(gl.SRC_ALPHA, gl.ONE_MINUS_SRC_ALPHA)
	}
	glctx.DrawArrays(gl.POINTS, 0, len(buf)/8)
	glctx.Disable(gl.BLEND)
}

func (g *mobileGame) drawGlowSpritesGL(glctx gl.Context, buf []float32, camX, camY, zoomX, zoomY float32, vw, vh int) {
	if len(buf) == 0 {
		return
	}
	glctx.UseProgram(g.glowProg)
	glctx.BindBuffer(gl.ARRAY_BUFFER, g.spriteVBO)
	glctx.BufferData(gl.ARRAY_BUFFER, f32bytes(buf), gl.STREAM_DRAW)
	setSpriteAttribs(glctx, g.glowAPos, g.glowASize, g.glowAColor, g.glowARot)
	glctx.Uniform2f(g.glowUCamera, camX, camY)
	glctx.Uniform2f(g.glowUZoom, zoomX, zoomY)
	glctx.Uniform2f(g.glowURes, float32(vw), float32(vh))
	glctx.Enable(gl.BLEND)
	glctx.BlendFunc(gl.ONE, gl.ONE)
	glctx.DrawArrays(gl.POINTS, 0, len(buf)/8)
	glctx.Disable(gl.BLEND)
}

func (g *mobileGame) drawBonusSpritesGL(glctx gl.Context, buf []float32, camX, camY, zoomX, zoomY float32, vw, vh int, sunAmb, sunTR, sunTG, sunTB float32) {
	if len(buf) == 0 {
		return
	}
	glctx.UseProgram(g.bonusProg)
	glctx.BindBuffer(gl.ARRAY_BUFFER, g.spriteVBO)
	glctx.BufferData(gl.ARRAY_BUFFER, f32bytes(buf), gl.STREAM_DRAW)
	setSpriteAttribs(glctx, g.bonusAPos, g.bonusASize, g.bonusAColor, g.bonusARot)
	glctx.Uniform2f(g.bonusUCamera, camX, camY)
	glctx.Uniform2f(g.bonusUZoom, zoomX, zoomY)
	glctx.Uniform2f(g.bonusURes, float32(vw), float32(vh))
	glctx.Uniform1f(g.bonusUAmbient, sunAmb)
	glctx.Uniform3f(g.bonusUSunTint, sunTR, sunTG, sunTB)
	glctx.Enable(gl.BLEND)
	glctx.BlendFunc(gl.SRC_ALPHA, gl.ONE_MINUS_SRC_ALPHA)
	glctx.DrawArrays(gl.POINTS, 0, len(buf)/8)
	glctx.Disable(gl.BLEND)
}

func (g *mobileGame) drawNPCSpritesGL(glctx gl.Context, buf []float32, tex gl.Texture, aspect float32, camX, camY, zoomX, zoomY float32, vw, vh int) {
	if len(buf) == 0 {
		return
	}
	glctx.ActiveTexture(gl.TEXTURE1)
	glctx.BindTexture(gl.TEXTURE_2D, tex)
	glctx.UseProgram(g.npcProg)
	glctx.BindBuffer(gl.ARRAY_BUFFER, g.spriteVBO)
	glctx.BufferData(gl.ARRAY_BUFFER, f32bytes(buf), gl.STREAM_DRAW)
	setSpriteAttribs(glctx, g.npcAPos, g.npcASize, g.npcAColor, g.npcARot)
	glctx.Uniform2f(g.npcUCamera, camX, camY)
	glctx.Uniform2f(g.npcUZoom, zoomX, zoomY)
	glctx.Uniform2f(g.npcURes, float32(vw), float32(vh))
	glctx.Uniform1f(g.npcUCarAspect, aspect)
	glctx.Enable(gl.BLEND)
	glctx.BlendFunc(gl.SRC_ALPHA, gl.ONE_MINUS_SRC_ALPHA)
	glctx.DrawArrays(gl.POINTS, 0, len(buf)/8)
	glctx.Disable(gl.BLEND)
	glctx.ActiveTexture(gl.TEXTURE0)
}

func carShadowSpritesMobile(ts *TrafficSystem, cs *CopSystem, buf []float32) []float32 {
	buf = buf[:0]
	addShadow := func(x, y, heading float64) {
		fwdX := math.Cos(heading)
		fwdY := math.Sin(heading)
		ox := x + 0.6
		oy := y + 1.0
		sz := float32(CarSize * 0.95)
		for _, t := range [3]float64{-1.6, 0, 1.6} {
			buf = append(buf,
				float32(ox+fwdX*t), float32(oy+fwdY*t),
				sz, 0, 0, 0, 0.22, 0)
		}
	}
	for i := range ts.Cars {
		c := &ts.Cars[i]
		if c.Alive {
			addShadow(c.X, c.Y, c.Heading)
		}
	}
	for i := range cs.Cars {
		c := &cs.Cars[i]
		if c.Alive {
			addShadow(c.X, c.Y, c.Heading)
		}
	}
	return buf
}

func carHeadlightSpritesMobile(ts *TrafficSystem, brightness float32, buf []float32) []float32 {
	buf = buf[:0]
	if ts == nil || brightness <= 0.01 {
		return buf
	}
	for i := range ts.Cars {
		c := &ts.Cars[i]
		if !c.Alive {
			continue
		}
		offset := float64(c.Size) * 0.5
		perpX := float32(-math.Sin(c.Heading) * 0.9)
		perpY := float32(math.Cos(c.Heading) * 0.9)
		frontX := float32(c.X + math.Cos(c.Heading)*offset)
		frontY := float32(c.Y + math.Sin(c.Heading)*offset)
		rearX := float32(c.X - math.Cos(c.Heading)*offset)
		rearY := float32(c.Y - math.Sin(c.Heading)*offset)
		sz := 2.5 * brightness

		hw := 1.0 * brightness
		buf = append(buf, frontX+perpX, frontY+perpY, sz, hw, 0.95*brightness, 0.65*brightness, 1, 0)
		buf = append(buf, frontX-perpX, frontY-perpY, sz, hw, 0.95*brightness, 0.65*brightness, 1, 0)
		rw := 0.7 * brightness
		buf = append(buf, rearX+perpX, rearY+perpY, sz*0.7, rw, 0.03*brightness, 0.03*brightness, 1, 0)
		buf = append(buf, rearX-perpX, rearY-perpY, sz*0.7, rw, 0.03*brightness, 0.03*brightness, 1, 0)
	}
	return buf
}

func (g *mobileGame) buildCarSpriteBuffers() {
	g.trafficCarBuf = g.trafficCarBuf[:0]
	for i := range g.traffic.Cars {
		c := &g.traffic.Cars[i]
		if !c.Alive {
			continue
		}
		g.trafficCarBuf = append(g.trafficCarBuf,
			float32(c.X), float32(c.Y), float32(CarSize),
			1, 1, 1, 1, float32(c.Heading+math.Pi*0.5))
	}
	g.copCarBuf = g.copCarBuf[:0]
	for i := range g.cops.Cars {
		c := &g.cops.Cars[i]
		if !c.Alive {
			continue
		}
		g.copCarBuf = append(g.copCarBuf,
			float32(c.X), float32(c.Y), float32(CarSize),
			1, 1, 1, 1, float32(c.Heading+math.Pi*0.5))
	}
}

func brightenSpriteColors(src, dst []float32, gain float32) []float32 {
	if cap(dst) < len(src) {
		dst = make([]float32, len(src))
	} else {
		dst = dst[:len(src)]
	}
	copy(dst, src)
	for i := 0; i+7 < len(dst); i += 8 {
		dst[i+3] = clamp01(dst[i+3] * gain)
		dst[i+4] = clamp01(dst[i+4] * gain)
		dst[i+5] = clamp01(dst[i+5] * gain)
	}
	return dst
}

func clamp01(v float32) float32 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}

func (g *mobileGame) drawGL(glctx gl.Context) {
	if !g.glReady {
		return
	}
	if g.fbWidth <= 0 || g.fbHeight <= 0 {
		return
	}

	glctx.ClearColor(0, 0, 0, 1)
	glctx.Clear(gl.COLOR_BUFFER_BIT)
	vx, vy, vw, vh, zoomX64, zoomY64 := g.renderViewport()
	if vw <= 0 || vh <= 0 || zoomX64 <= 0 || zoomY64 <= 0 {
		return
	}
	glctx.Viewport(vx, vy, vw, vh)

	camX, camY := g.cam.EffectivePos()
	viewW := float64(vw) / zoomX64
	viewH := float64(vh) / zoomY64
	u0 := (camX - viewW*0.5) / float64(WorldWidth)
	v0 := (camY - viewH*0.5) / float64(WorldHeight)
	u1 := (camX + viewW*0.5) / float64(WorldWidth)
	v1 := (camY + viewH*0.5) / float64(WorldHeight)
	verts := []float32{
		-1, -1, float32(u0), float32(v1),
		1, -1, float32(u1), float32(v1),
		-1, 1, float32(u0), float32(v0),
		1, 1, float32(u1), float32(v0),
	}

	glctx.ActiveTexture(gl.TEXTURE0)
	glctx.BindTexture(gl.TEXTURE_2D, g.tex)
	glctx.TexSubImage2D(gl.TEXTURE_2D, 0, 0, 0, WorldWidth, WorldHeight, gl.RGBA, gl.UNSIGNED_BYTE, g.frame)

	glctx.UseProgram(g.prog)
	glctx.BindBuffer(gl.ARRAY_BUFFER, g.vbo)
	glctx.BufferData(gl.ARRAY_BUFFER, f32bytes(verts), gl.STREAM_DRAW)
	glctx.EnableVertexAttribArray(g.aPos)
	glctx.EnableVertexAttribArray(g.aUV)
	glctx.VertexAttribPointer(g.aPos, 2, gl.FLOAT, false, 16, 0)
	glctx.VertexAttribPointer(g.aUV, 2, gl.FLOAT, false, 16, 8)
	glctx.Uniform1i(g.uTex, 0)
	glctx.DrawArrays(gl.TRIANGLE_STRIP, 0, 4)

	sunAmb, sunTR, sunTG, sunTB := SunCycleLight(g.session.LevelTimer)
	zoomX := float32(zoomX64)
	zoomY := float32(zoomY64)

	g.carShadowBuf = carShadowSpritesMobile(g.traffic, g.cops, g.carShadowBuf)
	g.drawLitSpritesGL(glctx, g.carShadowBuf, false, float32(camX), float32(camY), zoomX, zoomY, vw, vh, sunAmb, sunTR, sunTG, sunTB)

	g.buildCarSpriteBuffers()
	g.drawNPCSpritesGL(glctx, g.trafficCarBuf, g.carTexBase, CarVisualAspect, float32(camX), float32(camY), zoomX, zoomY, vw, vh)
	g.drawNPCSpritesGL(glctx, g.copCarBuf, g.copCarTex, CarVisualAspect, float32(camX), float32(camY), zoomX, zoomY, vw, vh)

	g.pedBuf = g.peds.PedRenderData(g.pedBuf, g.now)
	g.drawLitSpritesGL(glctx, g.pedBuf, false, float32(camX), float32(camY), zoomX, zoomY, vw, vh, sunAmb, sunTR, sunTG, sunTB)

	g.drawLitSpritesGL(glctx, g.cops.CopRenderData(g.now), false, float32(camX), float32(camY), zoomX, zoomY, vw, vh, sunAmb, sunTR, sunTG, sunTB)
	g.drawGlowSpritesGL(glctx, g.cops.CopGlowData(g.now), float32(camX), float32(camY), zoomX, zoomY, vw, vh)
	g.drawLitSpritesGL(glctx, g.mil.RenderData(g.now), false, float32(camX), float32(camY), zoomX, zoomY, vw, vh, sunAmb, sunTR, sunTG, sunTB)
	g.drawGlowSpritesGL(glctx, g.mil.GlowData(g.now), float32(camX), float32(camY), zoomX, zoomY, vw, vh)

	if g.snake != nil && g.snake.Alive {
		g.snakeLitBuf = brightenSpriteColors(g.snake.SnakeRenderData(), g.snakeLitBuf, 1.22)
		g.drawLitSpritesGL(glctx, g.snakeLitBuf, false, float32(camX), float32(camY), zoomX, zoomY, vw, vh, sunAmb, sunTR, sunTG, sunTB)
		g.drawGlowSpritesGL(glctx, g.snake.GlowData(), float32(camX), float32(camY), zoomX, zoomY, vw, vh)
	}

	if len(g.moveTargets) > 0 && g.snake != nil && g.snake.Alive && g.snake.TargetNukeTimer <= 0 {
		g.targetGlowBuf = g.targetGlowBuf[:0]
		t := g.moveTargets[len(g.moveTargets)-1]
		pulse := float32(1.0 + 0.14*math.Sin(g.now*2.5))
		baseOuter := float32(11.0)
		baseInner := baseOuter * 0.45
		cx := float32(t.X)
		cy := float32(t.Y)
		g.targetGlowBuf = append(g.targetGlowBuf,
			cx, cy, baseOuter*pulse, 0.24, 0.95, 0.40, 1.0, 0,
			cx, cy, baseInner*pulse, 0.62, 1.00, 0.72, 1.0, 0,
		)
		g.drawGlowSpritesGL(glctx, g.targetGlowBuf, float32(camX), float32(camY), zoomX, zoomY, vw, vh)
	}

	g.drawBonusSpritesGL(glctx, g.bonuses.RenderData(), float32(camX), float32(camY), zoomX, zoomY, vw, vh, sunAmb, sunTR, sunTG, sunTB)
	g.drawGlowSpritesGL(glctx, g.bonuses.GlowData(), float32(camX), float32(camY), zoomX, zoomY, vw, vh)

	lightBrightness := NightIntensityFromAmbient(sunAmb)
	if lightBrightness > 0.01 {
		if !g.world.Theme.NoRoads {
			g.drawGlowSpritesGL(glctx, streetlightSprites(lightBrightness), float32(camX), float32(camY), zoomX, zoomY, vw, vh)
		}
		g.carHeadBuf = carHeadlightSpritesMobile(g.traffic, lightBrightness, g.carHeadBuf)
		g.drawGlowSpritesGL(glctx, g.carHeadBuf, float32(camX), float32(camY), zoomX, zoomY, vw, vh)
	}

	g.glowBuf, g.normBuf = g.particles.ParticleRenderData(g.glowBuf, g.normBuf)
	g.drawLitSpritesGL(glctx, g.normBuf, false, float32(camX), float32(camY), zoomX, zoomY, vw, vh, sunAmb, sunTR, sunTG, sunTB)
	g.drawLitSpritesGL(glctx, g.glowBuf, true, float32(camX), float32(camY), zoomX, zoomY, vw, vh, sunAmb, sunTR, sunTG, sunTB)

	g.renderHUDMobile(glctx, g.fbWidth, g.fbHeight)
}

func RunAndroid() {
	seed := uint64(time.Now().UnixNano())
	game := newMobileGame(seed)
	if err := InitAudio(); err != nil {
		fmt.Printf("audio init failed (continuing without sound): %v\n", err)
	} else {
		go func() {
			time.Sleep(100 * time.Millisecond)
			StartMenuMusic()
		}()
	}

	app.Main(func(a app.App) {
		var glctx gl.Context
		var last time.Time

		for e := range a.Events() {
			switch e := a.Filter(e).(type) {
			case lifecycle.Event:
				switch e.Crosses(lifecycle.StageVisible) {
				case lifecycle.CrossOn:
					ctx, ok := e.DrawContext.(gl.Context)
					if !ok {
						continue
					}
					glctx = ctx
					if err := game.initGL(glctx); err != nil {
						panic(err)
					}
					last = time.Now()
					a.Send(paint.Event{})
				case lifecycle.CrossOff:
					if glctx != nil {
						game.destroyGL(glctx)
						glctx = nil
					}
				}
				if e.To == lifecycle.StageDead {
					return
				}

			case size.Event:
				game.fbWidth = e.WidthPx
				game.fbHeight = e.HeightPx

			case touch.Event:
				game.handleTouch(e)

			case paint.Event:
				if glctx == nil || game.fbWidth <= 0 || game.fbHeight <= 0 {
					continue
				}
				now := time.Now()
				dt := now.Sub(last).Seconds()
				last = now
				game.step(dt)
				game.renderSoftware()
				game.drawGL(glctx)
				a.Publish()
				a.Send(paint.Event{})
			}
		}
	})
}
