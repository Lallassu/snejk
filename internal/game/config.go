package game

// World dimensions (in world pixels).
// Chosen as a 4:3 size aligned to the city pattern (33 px) so
// auto-fit camera fills the window without partial edge blocks.
const (
	WorldWidth  = 264
	WorldHeight = 198
)

// Window defaults.
const (
	WindowWidth  = 800
	WindowHeight = 600
	DefaultZoom  = 4.0
	MinZoom      = 1.5
	MaxZoom      = 12.0
)

// Chunking.
const ChunkSize = 128

// Road/city-block layout (in world pixels).
const (
	RoadWidth     = 5
	SidewalkWidth = 3
	BlockInner    = 22
	Pattern       = RoadWidth + 2*SidewalkWidth + BlockInner // 33
)

// World border (indestructible).
const (
	BorderThickness = 1
	BorderHeight    = 12
)

// Directional sun shadow settings.
const (
	SunDx         = -1
	SunDy         = -1
	SunSlope      = 1
	MaxShadowDist = 48
	ShadeLit      = 255
	ShadeDark     = 160
)

// Explosion defaults.
const (
	ExplosionRadius    = 10
	HitExplosionRadius = 5
)

// Car physics/visual.
const (
	CarSize         = 5.0
	CarVisualAspect = 0.68
	CarMaxSpeed     = 160.0
	CarCrashSpeed   = 150.0
	CarWheelBase    = 16.0
)

// Particles.
const (
	MaxParticles         = 15000
	MaxParticleRender    = 20000
	ParticleCullDistance = 220.0
)

// Spatial index.
const (
	QuadCapacity = 16
	QuadMaxDepth = 8
)

// Font atlas layout (font.png: 32 cols x 4 rows, ASCII 0-127).
const (
	FontCellW  = 18
	FontCellH  = 32
	FontCols   = 32
	FontRows   = 4
	FontAtlasW = FontCellW * FontCols // 576
	FontAtlasH = FontCellH * FontRows // 128
)

// Snake constants.
const (
	SnakeBaseSpeed    = 28.0
	SnakeMaxLength    = 100.0
	SnakeStartLength  = 12.0
	SnakeTurnRate     = 6.0 // rad/s
	SnakeHeadSize     = 3.0
	SnakeEatRadius    = 2.5
	SnakeCarEatRadius = 3.5
	SnakeBonusRadius  = 3.0
)
