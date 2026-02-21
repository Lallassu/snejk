package game

import "math"

// ExplodeAt performs a full explosion: carves world, spawns particles, affects entities.
// Returns the number of pedestrians killed by this explosion.
func ExplodeAt(wx, wy, radius int, w *World, ps *ParticleSystem, peds *PedestrianSystem, traffic *TrafficSystem, cam *Camera, cops *CopSystem, mil *MilitarySystem) int {
	PlayExplosionSound(float64(radius))
	if cam != nil {
		intensity := float64(radius) * 0.25
		cam.AddShake(intensity, 0.3)
	}
	col := w.ColorAt(wx, wy)
	w.Explode(wx, wy, radius)
	if ps != nil {
		SpawnExplosionWithShockwave(wx, wy, col, 1.0, radius, w, ps)
	}
	pedKills := 0
	if peds != nil {
		pedKills = ExplodeAffectPeds(wx, wy, radius, w, ps, peds)
	}
	if traffic != nil {
		ExplodeAffectCars(wx, wy, radius, w, ps, traffic)
	}
	if cops != nil {
		ExplodeAffectCops(wx, wy, radius, w, ps, cops)
	}
	if mil != nil {
		ExplodeAffectMilitary(wx, wy, radius, w, ps, mil)
	}
	return pedKills
}

// SpawnExplosionWithShockwave spawns explosion particles and a matching shockwave.
func SpawnExplosionWithShockwave(wx, wy int, col RGB, intensity float64, shockRadius int, w *World, ps *ParticleSystem) {
	if ps == nil || intensity <= 0 {
		return
	}
	ps.SpawnExplosion(wx, wy, col, intensity)

	// Ensure visible shockwaves even for tiny impacts.
	r := shockRadius
	if r <= 0 {
		r = int(math.Round(float64(ExplosionRadius) * intensity))
	}
	if r < 2 {
		r = 2
	}
	SpawnShockwave(wx, wy, r, w, ps)
}

// SpawnNukeAftermath layers heavy smoke/fire particles over a nuke detonation.
func SpawnNukeAftermath(wx, wy int, w *World, ps *ParticleSystem, scale float64) {
	if ps == nil || scale <= 0 {
		return
	}

	fx := float64(wx)
	fy := float64(wy)
	r := NewRand(hash2D(0x4E554B45, wx, wy) ^ uint64(int(scale*1000)))

	if w != nil {
		// Extra-wide pressure wave to sell the blast size.
		shock := int(math.Round(40.0 * scale))
		if shock < 10 {
			shock = 10
		}
		SpawnShockwave(wx, wy, shock, w, ps)
	}

	// Core fireball.
	for range int(240*scale) + 80 {
		ang := r.RangeF(0, 2*math.Pi)
		dist := r.RangeF(0, 4.5*scale)
		spd := r.RangeF(18, 88) * (0.75 + 0.5*scale)
		ps.Add(Particle{
			X: fx + math.Cos(ang)*dist, Y: fy + math.Sin(ang)*dist,
			VX: math.Cos(ang) * spd, VY: math.Sin(ang) * spd,
			Z: r.RangeF(0, 10), VZ: r.RangeF(36, 130) * scale,
			Size: 1.2, MaxLife: r.RangeF(0.45, 1.15),
			Col: Palette.FireHot, Kind: ParticleFire,
			Spread: 0.9,
		})
	}

	// Thick black smoke stem around the epicenter.
	for range int(320*scale) + 120 {
		ang := r.RangeF(0, 2*math.Pi)
		dist := r.RangeF(0, 14.0*scale)
		drift := r.RangeF(4, 20) * (0.8 + 0.5*scale)
		ps.Add(Particle{
			X: fx + math.Cos(ang)*dist, Y: fy + math.Sin(ang)*dist,
			VX: math.Cos(ang) * drift, VY: math.Sin(ang) * drift,
			Z: r.RangeF(2, 18), VZ: r.RangeF(22, 95) * scale,
			Size: 1.3, MaxLife: r.RangeF(1.0, 2.8),
			Col: RGB{R: uint8(r.Range(26, 58)), G: uint8(r.Range(24, 54)), B: uint8(r.Range(24, 52))}, Kind: ParticleSmoke,
		})
	}

	// Higher ash cloud with lighter tones.
	for range int(220*scale) + 90 {
		ang := r.RangeF(0, 2*math.Pi)
		dist := r.RangeF(5.0*scale, 24.0*scale)
		ps.Add(Particle{
			X: fx + math.Cos(ang)*dist, Y: fy + math.Sin(ang)*dist,
			VX: r.RangeF(-18, 18), VY: r.RangeF(-18, 18),
			Z: r.RangeF(12, 36), VZ: r.RangeF(26, 110) * scale,
			Size: 1.1, MaxLife: r.RangeF(1.2, 3.4),
			Col: RGB{R: uint8(r.Range(84, 136)), G: uint8(r.Range(78, 128)), B: uint8(r.Range(74, 124))}, Kind: ParticleSmoke,
		})
	}

	// Ember spray for bright lingering highlights.
	for range int(150*scale) + 60 {
		ang := r.RangeF(0, 2*math.Pi)
		spd := r.RangeF(35, 170) * (0.7 + 0.6*scale)
		ps.Add(Particle{
			X: fx + r.RangeF(-4, 4), Y: fy + r.RangeF(-4, 4),
			VX: math.Cos(ang) * spd, VY: math.Sin(ang) * spd,
			Z: r.RangeF(4, 20), VZ: r.RangeF(45, 170) * scale,
			Size: 0.9, MaxLife: r.RangeF(0.25, 0.8),
			Col: RGB{R: 255, G: uint8(r.Range(130, 220)), B: uint8(r.Range(35, 95))}, Kind: ParticleGlow,
		})
	}
}

// ExplodeAffectPeds kills or pushes pedestrians caught in an explosion.
// Returns the number of pedestrians killed.
func ExplodeAffectPeds(wx, wy, radius int, w *World, ps *ParticleSystem, peds *PedestrianSystem) int {
	if peds == nil || radius <= 0 {
		return 0
	}
	kills := 0
	r2 := float64(radius * radius)
	fwx := float64(wx)
	fwy := float64(wy)

	for i := 0; i < len(peds.P); i++ {
		p := &peds.P[i]
		if !p.Alive {
			continue
		}

		dx := p.X - fwx
		dy := p.Y - fwy
		d2 := dx*dx + dy*dy
		if d2 > r2 {
			continue
		}

		d := math.Sqrt(d2)
		nd := d / float64(radius)

		damage := (1.0 - nd*nd) * 5.0
		p.HP.Damage(damage)

		if p.HP.IsDead() {
			p.Alive = false
			kills++
			bx := int(math.Round(p.X))
			by := int(math.Round(p.Y))

			blood := RGB{R: 130, G: 20, B: 20}
			w.PaintRGB(bx, by, blood)
			w.PaintRGB(clamp(bx+1, 0, WorldWidth-1), by, blood)
			w.PaintRGB(clamp(bx-1, 0, WorldWidth-1), by, blood)

			paintFallenPed(w, bx, by, p.Skin, p.Col)

			if ps != nil {
				dirX := dx
				dirY := dy
				if d < 0.1 {
					dirX, dirY = 1, 0
				}
				ps.SpawnBlood(p.X, p.Y, dirX, dirY, 20, 0.7+0.3*(1.0-nd))
				ps.SpawnBlood(p.X+0.4, p.Y+0.4, -dirX, -dirY, 10, 0.5)
			}
		} else {
			if d > 0.1 {
				force := (1.0 - nd) * float64(radius) * 18.0
				pushX := dx / d
				pushY := dy / d
				maxStep := max(1, int(math.Ceil(force/8.0)))

				nx, ny := p.X, p.Y
				for range maxStep {
					tryX := nx + pushX*1.2
					tryY := ny + pushY*1.2
					tx := int(math.Round(tryX))
					ty := int(math.Round(tryY))
					if tx < 0 || ty < 0 || tx >= WorldWidth || ty >= WorldHeight {
						break
					}
					if pedWalkable(w, tx, ty) {
						nx = tryX
						ny = tryY
					} else {
						break
					}
				}
				p.X = nx
				p.Y = ny
				p.TargetX = p.X + pushX*2.2
				p.TargetY = p.Y + pushY*2.2
			}
		}
	}
	return kills
}

// ExplodeAffectCars destroys or pushes cars caught in an explosion.
func ExplodeAffectCars(wx, wy, radius int, w *World, ps *ParticleSystem, traffic *TrafficSystem) {
	if traffic == nil || radius <= 0 {
		return
	}
	fwx := float64(wx)
	fwy := float64(wy)
	fr := float64(radius)

	for i := range traffic.Cars {
		c := &traffic.Cars[i]
		if !c.Alive {
			continue
		}
		dx := c.X - fwx
		dy := c.Y - fwy
		d := math.Hypot(dx, dy)
		if d > fr*1.5 {
			continue
		}

		nd := d / (fr * 1.5)
		damage := (1.0 - nd) * 10.0

		if d < fr*0.6 {
			damage = 15.0
		}
		c.HP.Damage(damage)
		c.OnFire = true

		if c.HP.IsDead() {
			c.Alive = false
			cx := int(math.Round(c.X))
			cy := int(math.Round(c.Y))
			col := w.ColorAt(cx, cy)
			if ps != nil {
				SpawnExplosionWithShockwave(cx, cy, col, 0.5, 0, w, ps)
			}
		} else {
			force := (1.0 - nd) * 120.0
			if d > 0.1 {
				c.VX += (dx / d) * force
				c.VY += (dy / d) * force
			}
			c.Speed *= 0.4
		}
	}
}

// SpawnShockwave creates visual wave particles radiating from an explosion.
func SpawnShockwave(wx, wy, radius int, w *World, ps *ParticleSystem) {
	if ps == nil || radius <= 0 {
		return
	}

	visRadius := int(math.Round(float64(radius) * 2.5))
	// Only large blasts produce lethal wavefront damage.
	waveDamage := 0.0
	if radius >= 10 {
		waveDamage = clampF((float64(radius)-10.0)/20.0, 0.3, 1.0)
	}
	fwx := float64(wx)
	fwy := float64(wy)
	maxDelay := float64(visRadius) * 0.018
	rr := NewRand(uint64(wx*31+wy*17) ^ 0xDEC0DE)

	minX := clamp(wx-visRadius, 0, WorldWidth-1)
	maxX := clamp(wx+visRadius, 0, WorldWidth-1)
	minY := clamp(wy-visRadius, 0, WorldHeight-1)
	maxY := clamp(wy+visRadius, 0, WorldHeight-1)

	spawned := 0
	maxWave := 2000

	for ty := minY; ty <= maxY; ty++ {
		for tx := minX; tx <= maxX; tx++ {
			ddx := float64(tx) - fwx
			ddy := float64(ty) - fwy
			d := math.Hypot(ddx, ddy)
			if d > float64(visRadius) {
				continue
			}

			nd := d / float64(visRadius)
			intensity := (1.0 - nd)
			intensity *= intensity

			if rr.RangeF(0, 1) > 0.35 {
				continue
			}

			orig := w.ColorAt(tx, ty)

			brightness := 0.35 + 0.65*intensity
			col := lerpRGB(orig, RGB{R: 255, G: 255, B: 255}, brightness*intensity)

			ang := math.Atan2(ddy, ddx) + rr.RangeF(-0.18, 0.18)
			spd := rr.RangeF(8, 42) * (0.6 + 0.4*(1.0-nd))
			delay := nd*maxDelay + rr.RangeF(-0.015, 0.015)
			if delay < 0.02 {
				delay = 0.02
			}
			ttl := rr.RangeF(0.12, 0.40) * (0.6 + 0.6*intensity)

			ps.Add(Particle{
				X:  float64(tx) + rr.RangeF(-0.12, 0.12),
				Y:  float64(ty) + rr.RangeF(-0.12, 0.12),
				VX: math.Cos(ang) * spd, VY: math.Sin(ang) * spd,
				Size: 0.6, MaxLife: ttl, Life: -delay,
				Col: col, Kind: ParticleWave,
				Bounce: 0.18 + 0.5*(1.0-nd),
				Spread: waveDamage,
			})

			spawned++
			if spawned >= maxWave {
				return
			}
		}
	}
}

// UpdateBurnVisuals spawns fire/smoke particles from actively burning trees and buildings.
func UpdateBurnVisuals(w *World, ps *ParticleSystem, dt float64) {
	if ps == nil || w == nil {
		return
	}

	for _, tb := range w.burningTrees {
		rr := NewRand(tb.rng ^ uint64(int(tb.timer*1000)))
		if rr.RangeF(0, 1) < 4.0*dt {
			fx := float64(tb.X) + rr.RangeF(-2, 2)
			fy := float64(tb.Y) + rr.RangeF(-2, 2)
			ps.Add(Particle{
				X: fx, Y: fy,
				VX: rr.RangeF(-6, 6), VY: rr.RangeF(-6, 6),
				Z: rr.RangeF(2, 8), VZ: rr.RangeF(20, 60),
				Size: 0.4 + rr.RangeF(0, 0.3), MaxLife: rr.RangeF(0.15, 0.4),
				Col: Palette.FireHot, Kind: ParticleFire,
			})
		}
		if rr.RangeF(0, 1) < 2.0*dt {
			fx := float64(tb.X) + rr.RangeF(-2.5, 2.5)
			fy := float64(tb.Y) + rr.RangeF(-2.5, 2.5)
			ps.Add(Particle{
				X: fx, Y: fy,
				VX: rr.RangeF(-4, 4), VY: rr.RangeF(-8, -2),
				Z: rr.RangeF(4, 12), VZ: rr.RangeF(15, 40),
				Size: 1.0, MaxLife: rr.RangeF(0.2, 0.5),
				Col: Palette.Smoke, Kind: ParticleSmoke,
			})
		}
	}

	for _, bb := range w.burningBuildings {
		if !bb.smolder || len(bb.Pixels) == 0 {
			continue
		}
		rr := NewRand(bb.rng ^ uint64(int(bb.smolderTimer*1000)))
		idx := int(rr.NextU64() % uint64(len(bb.Pixels)))
		px := bb.Pixels[idx]

		if rr.RangeF(0, 1) < 3.0*dt {
			ps.Add(Particle{
				X:  float64(px.X) + rr.RangeF(-0.5, 0.5),
				Y:  float64(px.Y) + rr.RangeF(-0.5, 0.5),
				VX: rr.RangeF(-8, 8), VY: rr.RangeF(-8, 8),
				Z: rr.RangeF(1, 6), VZ: rr.RangeF(25, 70),
				Size: 0.35 + rr.RangeF(0, 0.3), MaxLife: rr.RangeF(0.2, 0.5),
				Col: Palette.FireHot, Kind: ParticleFire,
			})
		}

		if rr.RangeF(0, 1) < 5.0*dt {
			ps.Add(Particle{
				X:  float64(px.X) + rr.RangeF(-1, 1),
				Y:  float64(px.Y) + rr.RangeF(-1, 1),
				VX: rr.RangeF(-3, 3), VY: rr.RangeF(-12, -3),
				Z: rr.RangeF(6, 16), VZ: rr.RangeF(20, 50),
				Size: 1.0, MaxLife: rr.RangeF(0.25, 0.55),
				Col: RGB{R: 80, G: 80, B: 85}, Kind: ParticleSmoke,
			})
		}
	}
}

// paintFallenPed paints a small body decal on the ground.
func paintFallenPed(w *World, bx, by int, skin, cloth RGB) {
	if bx < 0 || by < 0 || bx >= WorldWidth || by >= WorldHeight {
		return
	}
	cx, cy := bx, by
	if !pedWalkable(w, cx, cy) {
		found := false
		for oy := -1; oy <= 1 && !found; oy++ {
			for ox := -1; ox <= 1; ox++ {
				nx := cx + ox
				ny := cy + oy
				if nx >= 0 && ny >= 0 && nx < WorldWidth && ny < WorldHeight && pedWalkable(w, nx, ny) {
					cx, cy = nx, ny
					found = true
					break
				}
			}
		}
		if !found {
			return
		}
	}
	w.PaintRGB(cx, cy, skin)
	w.PaintRGB(cx, cy+1, cloth)
	w.PaintRGB(cx-1, cy+1, cloth)
	w.PaintRGB(cx+1, cy+1, cloth)
	w.PaintRGB(cx, cy+2, cloth)
}
