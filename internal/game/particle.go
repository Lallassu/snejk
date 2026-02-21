package game

import "math"

type ParticleKind uint8

const (
	ParticleDebris ParticleKind = iota
	ParticleFire
	ParticleGlow
	ParticleSmoke
	ParticleBlood
	ParticleWave
	ParticleRain
	ParticleSnow
)

type Particle struct {
	X, Y   float64
	VX, VY float64
	Z, VZ  float64

	Size   float64
	Bounce float64 // short-lived pulse on impact [0..1]

	Life    float64 // negative = delayed start
	MaxLife float64

	Col     RGB
	Kind    ParticleKind
	Hit     bool    // one-shot impact flag (used by shockwaves)
	Burning bool    // debris on fire
	Spread  float64 // incendiary potential [0..1]
}

type ParticleSystem struct {
	Max    int
	P      []Particle
	seed   uint64
	ovrIdx int // circular overwrite index when full
}

func NewParticleSystem(maxParticles int, seed uint64) *ParticleSystem {
	if maxParticles <= 0 {
		maxParticles = MaxParticles
	}
	if seed == 0 {
		seed = 1
	}
	return &ParticleSystem{
		Max:  maxParticles,
		P:    make([]Particle, 0, maxParticles),
		seed: seed,
	}
}

func (ps *ParticleSystem) Clear() {
	ps.P = ps.P[:0]
	ps.ovrIdx = 0
}

func (ps *ParticleSystem) Add(p Particle) {
	if len(ps.P) < ps.Max {
		ps.P = append(ps.P, p)
		return
	}
	// Circular overwrite.
	if ps.ovrIdx >= ps.Max {
		ps.ovrIdx = 0
	}
	ps.P[ps.ovrIdx] = p
	ps.ovrIdx++
}

// ParticleRenderData splits particles into glow (additive) and normal (alpha blend) buffers.
// Format: [x, y, size, r, g, b, a, rotation] * N.
func (ps *ParticleSystem) ParticleRenderData(glowBuf, normBuf []float32) ([]float32, []float32) {
	glowBuf = glowBuf[:0]
	normBuf = normBuf[:0]

	for _, p := range ps.P {
		if p.Life < 0 {
			continue
		}
		t := p.Life / p.MaxLife
		if t < 0 {
			t = 0
		}
		if t > 1 {
			t = 1
		}

		col := p.Col
		a := 1.0 - t

		switch p.Kind {
		case ParticleDebris, ParticleWave:
			a = 1.0
		case ParticleSmoke:
			fadeIn := t / 0.18
			if fadeIn > 1 {
				fadeIn = 1
			}
			a = (1.0 - t) * fadeIn * 0.85
		case ParticleGlow:
			a = (1.0 - t) * 1.15
		case ParticleFire:
			fadeIn := t / 0.08
			if fadeIn > 1 {
				fadeIn = 1
			}
			a = (1.0 - t) * fadeIn * 1.25
			if t < 0.5 {
				col = lerpRGB(Palette.FireHot, Palette.FireMid, t*2.0)
			} else {
				col = lerpRGB(Palette.FireMid, Palette.FireCool, (t-0.5)*2.0)
			}
		case ParticleBlood:
			a = 1.0 - t*0.3
		case ParticleRain:
			a = (1.0 - t) * 0.75
		case ParticleSnow:
			a = (1.0 - t) * 0.95
		}
		if a <= 0 {
			continue
		}

		// Size: base + height scale + bounce pulse.
		zScale := 0.0
		if p.Z > 0 {
			zScale = p.Z * 0.02
			if zScale > 2.0 {
				zScale = 2.0
			}
		}
		visSize := 1.0 + zScale
		if p.Bounce > 0 {
			visSize += 0.75 * p.Bounce
		}
		if p.Kind == ParticleSmoke {
			visSize *= 1.0 + t*1.6
		}
		if p.Kind == ParticleRain {
			visSize *= 0.72
		}
		if p.Kind == ParticleSnow {
			visSize *= 1.10 + t*0.20
		}

		rc := float32(col.R) / 255.0
		gc := float32(col.G) / 255.0
		bc := float32(col.B) / 255.0
		ac := float32(a)
		if ac < 0 {
			ac = 0
		}
		if ac > 1 {
			ac = 1
		}

		// Additive: pre-multiply color by alpha.
		if p.Kind == ParticleGlow || p.Kind == ParticleFire {
			rc *= ac
			gc *= ac
			bc *= ac
		}

		sx := float32(math.Round(p.X))
		sy := float32(math.Round(p.Y))
		sz := float32(visSize)

		if p.Kind == ParticleGlow || p.Kind == ParticleFire {
			glowBuf = append(glowBuf, sx, sy, sz, rc, gc, bc, ac, 0)
		} else {
			normBuf = append(normBuf, sx, sy, sz, rc, gc, bc, ac, 0)
		}
	}
	return glowBuf, normBuf
}
