package game

import "math"

func (ps *ParticleSystem) SpawnExplosion(wx, wy int, baseCol RGB, intensity float64) {
	if intensity <= 0 {
		return
	}

	r := NewRand(hash2D(ps.seed^0xA5A5A5A5, wx, wy))
	fx := float64(wx)
	fy := float64(wy)

	// Debris.
	for range int(65 * intensity) {
		ang := r.RangeF(0, math.Pi*2)
		spd := r.RangeF(45, 170) * intensity
		vz := r.RangeF(30, 95) * intensity
		col := baseCol.Add(r.Range(-14, 14), r.Range(-14, 14), r.Range(-14, 14))
		ps.Add(Particle{
			X: fx + r.RangeF(-2, 2), Y: fy + r.RangeF(-2, 2),
			VX: math.Cos(ang) * spd, VY: math.Sin(ang) * spd,
			Z: r.RangeF(0, 10), VZ: vz * 1.25,
			Size: 1.0, MaxLife: r.RangeF(0.5, 1.0),
			Col: col, Kind: ParticleDebris,
			Spread:  0.35 + 0.9*intensity,
			Burning: r.RangeF(0, 1) < 0.38*intensity*(0.92+0.7*intensity),
		})
	}

	// Fire.
	for range int(70 * intensity) {
		ang := r.RangeF(0, math.Pi*2)
		spd := r.RangeF(10, 38)
		ps.Add(Particle{
			X: fx + r.RangeF(-2, 2), Y: fy + r.RangeF(-2, 2),
			VX: math.Cos(ang) * spd, VY: math.Sin(ang) * spd,
			Z: r.RangeF(0, 6), VZ: r.RangeF(12, 48),
			Size: 0.45 + r.RangeF(0, 0.45), MaxLife: r.RangeF(0.12, 0.36),
			Col: Palette.FireHot, Kind: ParticleFire,
		})
	}

	// Glow.
	for range int(12 * intensity) {
		ang := r.RangeF(0, math.Pi*2)
		spd := r.RangeF(85, 240) * intensity
		ps.Add(Particle{
			X: fx + r.RangeF(-1.5, 1.5), Y: fy + r.RangeF(-1.5, 1.5),
			VX: math.Cos(ang) * spd, VY: math.Sin(ang) * spd,
			Z: r.RangeF(0, 8), VZ: r.RangeF(40, 120) * intensity,
			Size: 1.0, MaxLife: r.RangeF(0.12, 0.32),
			Col: Palette.Glow, Kind: ParticleGlow,
		})
	}

	// Smoke.
	for range int(48*intensity) + 16 {
		ang := r.RangeF(0, math.Pi*2)
		spd := r.RangeF(5, 22)
		ps.Add(Particle{
			X: fx + r.RangeF(-3, 3), Y: fy + r.RangeF(-3, 3),
			VX: math.Cos(ang) * spd, VY: math.Sin(ang) * spd,
			Z: r.RangeF(0, 10), VZ: r.RangeF(10, 24),
			Size: 1.0, MaxLife: r.RangeF(0.12, 0.36),
			Col: Palette.Smoke, Kind: ParticleSmoke,
		})
	}

	// White smoke plume.
	for range int(18 * intensity) {
		ps.Add(Particle{
			X: fx + r.RangeF(-3.5, 3.5), Y: fy + r.RangeF(-3.5, 3.5),
			VX: r.RangeF(-8, 8), VY: r.RangeF(-18, -6),
			Z: r.RangeF(2, 18), VZ: r.RangeF(40, 110) * intensity,
			Size: 1.0, MaxLife: r.RangeF(0.16, 0.45),
			Col: RGB{R: 245, G: 245, B: 250}, Kind: ParticleSmoke,
		})
	}
}

// ApplySuction pulls all live particles toward (cx, cy) within radius.
func (ps *ParticleSystem) ApplySuction(cx, cy, radius, strength, dt float64) {
	r2 := radius * radius
	for i := range ps.P {
		p := &ps.P[i]
		if p.Life < 0 {
			continue
		}
		dx := cx - p.X
		dy := cy - p.Y
		d2 := dx*dx + dy*dy
		if d2 > r2 || d2 < 0.01 {
			continue
		}
		d := math.Sqrt(d2)
		force := (1.0 - d/radius) * strength
		p.VX += dx / d * force * dt
		p.VY += dy / d * force * dt
	}
}

func (ps *ParticleSystem) SpawnBlood(x, y, dirX, dirY float64, count int, intensity float64) {
	r := NewRand(ps.seed ^ uint64(int(x)*31+int(y)*17))
	blood := RGB{R: 130, G: 20, B: 20}
	ang0 := math.Atan2(dirY, dirX)

	for range count {
		ang := ang0 + r.RangeF(-0.9, 0.9)
		spd := r.RangeF(30, 140) * (0.5 + 0.5*intensity)
		ps.Add(Particle{
			X: x + r.RangeF(-0.6, 0.6), Y: y + r.RangeF(-0.6, 0.6),
			VX: math.Cos(ang) * spd, VY: math.Sin(ang) * spd,
			Z: r.RangeF(0, 6), VZ: r.RangeF(30, 110),
			Size: 1.0, MaxLife: r.RangeF(0.8, 2.2),
			Col: blood, Kind: ParticleBlood,
		})
	}
}
