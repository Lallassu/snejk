package game

import "math"

const (
	particleGravity    = 320.0
	particleBounce     = 0.18
	particleGroundFric = 0.35
	particleAirDrag    = 1.65
	particleSettleSpd  = 14.0
	particleSettleVZ   = 8.0
)

// particleDecays holds exponential drag factors precomputed once per frame.
// Avoids calling math.Exp() inside the per-particle hot loop.
type particleDecays struct {
	waveXY       float64 // exp(-2.4 * dt)
	smokeXY      float64 // exp(-1.2 * dt)
	smokeZ       float64 // exp(-0.8 * dt)
	fireXY       float64 // exp(-2.2 * dt)
	fireZ        float64 // exp(-1.0 * dt)
	debrisXY     float64 // exp(-particleAirDrag * dt)
	debrisBurnXY float64 // exp(-particleAirDrag * 0.9 * dt)
	rainXY       float64 // exp(-0.25 * dt)
	snowXY       float64 // exp(-0.4 * dt)
}

func computeDecays(dt float64) particleDecays {
	return particleDecays{
		waveXY:       math.Exp(-2.4 * dt),
		smokeXY:      math.Exp(-1.2 * dt),
		smokeZ:       math.Exp(-0.8 * dt),
		fireXY:       math.Exp(-2.2 * dt),
		fireZ:        math.Exp(-1.0 * dt),
		debrisXY:     math.Exp(-particleAirDrag * dt),
		debrisBurnXY: math.Exp(-particleAirDrag * 0.9 * dt),
		rainXY:       math.Exp(-0.25 * dt),
		snowXY:       math.Exp(-0.4 * dt),
	}
}

func (ps *ParticleSystem) Update(dt float64, w *World) {
	ps.UpdateWithShockwaveDamage(dt, w, nil, nil, nil)
}

// UpdateWithShockwaveDamage advances particles and lets large shockwaves kill
// nearby peds, cops, and military troops as the wavefront passes them.
func (ps *ParticleSystem) UpdateWithShockwaveDamage(dt float64, w *World, peds *PedestrianSystem, cops *CopSystem, mil *MilitarySystem) {
	if dt <= 0 {
		return
	}

	d := computeDecays(dt)

	for i := 0; i < len(ps.P); {
		p := &ps.P[i]

		// Decay bounce pulse.
		if p.Bounce > 0 {
			p.Bounce -= 8.0 * dt
			if p.Bounce < 0 {
				p.Bounce = 0
			}
		}

		p.Life += dt
		if p.Life >= p.MaxLife {
			ps.P[i] = ps.P[len(ps.P)-1]
			ps.P = ps.P[:len(ps.P)-1]
			continue
		}

		// Skip delayed particles.
		if p.Life < 0 {
			i++
			continue
		}

		switch p.Kind {
		case ParticleWave:
			ps.updateWave(p, dt, d.waveXY, w, peds, cops, mil)
		case ParticleSmoke:
			ps.updateSmoke(p, dt, d.smokeXY, d.smokeZ)
		case ParticleFire:
			ps.updateFire(p, dt, d.fireXY, d.fireZ, w)
		case ParticleBlood:
			if ps.updateBloodOrDebris(p, i, dt, d.debrisXY, d.debrisBurnXY, w) {
				continue // removed
			}
		case ParticleRain:
			ps.updateRain(p, dt, d.rainXY)
		case ParticleSnow:
			ps.updateSnow(p, dt, d.snowXY)
		default: // Debris, Glow
			if ps.updateBloodOrDebris(p, i, dt, d.debrisXY, d.debrisBurnXY, w) {
				continue // removed
			}
		}

		i++
	}
}

func (ps *ParticleSystem) updateWave(p *Particle, dt, decayXY float64, w *World, peds *PedestrianSystem, cops *CopSystem, mil *MilitarySystem) {
	p.VX *= decayXY
	p.VY *= decayXY
	p.X += p.VX * dt
	p.Y += p.VY * dt

	if p.MaxLife > 0 {
		frac := clampF(p.Life/p.MaxLife, 0, 1)
		p.Size = 0.6 + 2.2*frac
	}
	// One-shot hit test when this wave particle reaches its target location.
	if !p.Hit {
		p.Hit = true
		if p.Spread > 0 {
			ps.applyWaveImpactDamage(p, w, peds, cops, mil)
		}
	}
}

func (ps *ParticleSystem) applyWaveImpactDamage(p *Particle, w *World, peds *PedestrianSystem, cops *CopSystem, mil *MilitarySystem) {
	if w == nil {
		return
	}
	dirX, dirY := p.VX, p.VY
	dirLen := math.Hypot(dirX, dirY)
	if dirLen > 0.001 {
		dirX /= dirLen
		dirY /= dirLen
	} else {
		dirX, dirY = 1, 0
	}

	// Larger shockwaves get slightly more forgiving hit radius.
	hitR := 1.0 + 1.2*clampF(p.Spread, 0, 1)
	hitR2 := hitR * hitR

	makeBlood := func(x, y float64, amount int, intensity float64) {
		if ps == nil {
			return
		}
		ps.SpawnBlood(x, y, dirX, dirY, amount, intensity)
		ps.SpawnBlood(x+0.25, y+0.25, dirX*0.85, dirY*0.85, max(5, amount/2), intensity*0.7)
	}

	if peds != nil {
		for i := range peds.P {
			ped := &peds.P[i]
			if !ped.Alive {
				continue
			}
			dx := ped.X - p.X
			dy := ped.Y - p.Y
			if dx*dx+dy*dy > hitR2 {
				continue
			}
			ped.Alive = false
			bx := clamp(int(math.Round(ped.X)), 0, WorldWidth-1)
			by := clamp(int(math.Round(ped.Y)), 0, WorldHeight-1)
			stain := RGB{R: 130, G: 20, B: 20}
			_ = w.PaintRGB(bx, by, stain)
			_ = w.PaintRGB(clamp(bx+1, 0, WorldWidth-1), by, stain)
			paintFallenPed(w, bx, by, ped.Skin, ped.Col)
			makeBlood(ped.X, ped.Y, 18, 0.95)
		}
	}

	if cops != nil {
		for i := range cops.Peds {
			cop := &cops.Peds[i]
			if !cop.Alive {
				continue
			}
			dx := cop.X - p.X
			dy := cop.Y - p.Y
			if dx*dx+dy*dy > hitR2 {
				continue
			}
			cop.Alive = false
			bx := clamp(int(math.Round(cop.X)), 0, WorldWidth-1)
			by := clamp(int(math.Round(cop.Y)), 0, WorldHeight-1)
			_ = w.PaintRGB(bx, by, RGB{R: 120, G: 18, B: 18})
			makeBlood(cop.X, cop.Y, 16, 0.9)
		}
	}

	if mil != nil {
		for i := range mil.Troops {
			troop := &mil.Troops[i]
			if !troop.Alive {
				continue
			}
			dx := troop.X - p.X
			dy := troop.Y - p.Y
			if dx*dx+dy*dy > hitR2 {
				continue
			}
			troop.Alive = false
			bx := clamp(int(math.Round(troop.X)), 0, WorldWidth-1)
			by := clamp(int(math.Round(troop.Y)), 0, WorldHeight-1)
			_ = w.PaintRGB(bx, by, RGB{R: 122, G: 18, B: 18})
			makeBlood(troop.X, troop.Y, 17, 0.92)
		}
	}
}

func (ps *ParticleSystem) updateSmoke(p *Particle, dt, decayXY, decayZ float64) {
	p.VX *= decayXY
	p.VY *= decayXY
	p.VZ *= decayZ
	p.X += p.VX * dt
	p.Y += p.VY * dt
	p.Z += p.VZ * dt
}

func (ps *ParticleSystem) updateFire(p *Particle, dt, decayXY, decayZ float64, w *World) {
	p.VX *= decayXY
	p.VY *= decayXY
	p.VZ *= decayZ
	p.VZ += 260.0 * dt

	// Sideways jitter.
	j := float64(int(hash2D(ps.seed^0xF17E, int(p.X), int(p.Y))>>56)-128) / 128.0
	p.VX += j * 28.0 * dt
	p.VY -= j * 18.0 * dt

	prevX := p.X
	prevY := p.Y
	p.X += p.VX * dt
	p.Y += p.VY * dt
	p.Z += p.VZ * dt

	// Swept XY collision.
	if w == nil {
		return
	}
	dx := p.X - prevX
	dy := p.Y - prevY
	steps := max(1, int(math.Ceil(math.Hypot(dx, dy))))

	for s := 1; s <= steps; s++ {
		t := float64(s) / float64(steps)
		sx := int(math.Round(prevX + dx*t))
		sy := int(math.Round(prevY + dy*t))
		if !w.IsBlocked(sx, sy) {
			continue
		}
		h := float64(w.HeightAt(sx, sy))
		if p.Z > h+1.0 {
			continue
		}

		// Hit: bounce off wall.
		prevFrac := float64(s-1) / float64(steps)
		p.X = prevX + dx*prevFrac
		p.Y = prevY + dy*prevFrac
		p.VX = -p.VX * 0.4
		p.VY = -p.VY * 0.4
		p.Bounce = 0.6
		p.MaxLife *= 0.92

		// Ignite tree or building at hit point.
		col := w.ColorAt(sx, sy)
		if col.G > col.R && col.G > col.B {
			chance := 0.18 + 0.6*clampF(p.Spread, 0, 1)
			if NewRand(ps.seed^uint64(sx*13+sy)).RangeF(0, 1) < chance {
				w.StartTreeBurn(sx, sy)
			}
		}
		if col.R >= 90 && col.R <= 210 && col.G >= 80 && col.G <= 180 {
			chance := 0.28 + 0.6*clampF(p.Spread, 0, 1)
			if NewRand(ps.seed^uint64(sx*31+sy)).RangeF(0, 1) < chance {
				w.StartBuildingBurn(sx, sy)
			}
		}
		break
	}
}

func (ps *ParticleSystem) updateRain(p *Particle, dt, decayXY float64) {
	p.VX *= decayXY
	p.X += p.VX * dt
	p.Y += p.VY * dt

	// Cull quickly once drops move outside an expanded world rectangle.
	if p.X < -20 || p.X > float64(WorldWidth)+20 || p.Y < -20 || p.Y > float64(WorldHeight)+20 {
		p.Life = p.MaxLife
	}
}

func (ps *ParticleSystem) updateSnow(p *Particle, dt, decayXY float64) {
	wobble := math.Sin((p.X*0.06)+(p.Y*0.04)+(p.Life*3.0)) * 8.0
	p.VX = p.VX*decayXY + wobble*dt
	p.X += p.VX * dt
	p.Y += p.VY * dt

	// Cull quickly once flakes move outside an expanded world rectangle.
	if p.X < -24 || p.X > float64(WorldWidth)+24 || p.Y < -24 || p.Y > float64(WorldHeight)+24 {
		p.Life = p.MaxLife
	}
}

// updateBloodOrDebris handles physics for Debris, Blood, Glow.
// Returns true if the particle was removed (caller should not increment i).
func (ps *ParticleSystem) updateBloodOrDebris(p *Particle, idx int, dt, decayXY, decayBurnXY float64, w *World) bool {
	// Gravity and drag.
	if p.Burning {
		p.VZ -= particleGravity * 0.60 * dt
		p.VZ += 40.0 * dt // buoyancy
		p.VX *= decayBurnXY
		p.VY *= decayBurnXY
	} else {
		p.VZ -= particleGravity * dt
		p.VX *= decayXY
		p.VY *= decayXY
	}

	prevX := p.X
	prevY := p.Y
	p.X += p.VX * dt
	p.Y += p.VY * dt
	p.Z += p.VZ * dt

	// Spawn fire trail from burning debris.
	if p.Burning && p.Z > 1.5 {
		rr := NewRand(ps.seed ^ uint64(idx+1) ^ uint64(int(p.Life*1000)))
		if rr.RangeF(0, 1) < 3.0*dt {
			fp := Particle{
				X: p.X + rr.RangeF(-0.35, 0.35), Y: p.Y + rr.RangeF(-0.35, 0.35),
				VX: p.VX*0.12 + rr.RangeF(-5, 5), VY: p.VY*0.12 + rr.RangeF(-5, 5),
				Z: p.Z + rr.RangeF(0, 0.8), VZ: rr.RangeF(18, 60),
				Size: 0.36 + rr.RangeF(0, 0.36), MaxLife: rr.RangeF(0.6, 1.2),
				Col: Palette.FireHot, Kind: ParticleFire,
			}
			ps.Add(fp)
		}
	}

	if w == nil {
		return false
	}

	// Ground/height contact.
	wx := int(math.Round(p.X))
	wy := int(math.Round(p.Y))
	th := float64(w.HeightAt(wx, wy))

	if p.Z <= th+1.0 {
		p.Z = th + 1.0
		p.VZ = -p.VZ * particleBounce
		p.VX *= particleGroundFric
		p.VY *= particleGroundFric
		p.Bounce = 1.0
		p.MaxLife *= 0.96

		// Debris settle.
		if p.Kind == ParticleDebris {
			spd := math.Hypot(p.VX, p.VY)
			if spd < particleSettleSpd && math.Abs(p.VZ) < particleSettleVZ {
				// Paint debris color into world.
				painted := false
				if w.HeightAt(wx, wy) == 0 {
					painted = w.PaintRGB(wx, wy, p.Col)
				}
				if !painted {
					for r := 1; r <= 3 && !painted; r++ {
						for oy := -r; oy <= r && !painted; oy++ {
							for ox := -r; ox <= r; ox++ {
								tx := wx + ox
								ty := wy + oy
								if tx >= 0 && ty >= 0 && tx < WorldWidth && ty < WorldHeight && w.HeightAt(tx, ty) == 0 {
									painted = w.PaintRGB(tx, ty, p.Col)
									if painted {
										break
									}
								}
							}
						}
					}
				}
				// Burning debris spawn ground fires.
				if p.Burning {
					rr := NewRand(ps.seed ^ uint64(idx+1))
					for fi := 0; fi < 2+rr.Range(0, 3); fi++ {
						fp := Particle{
							X: float64(wx) + rr.RangeF(-0.9, 0.9), Y: float64(wy) + rr.RangeF(-0.9, 0.9),
							VX: rr.RangeF(-8, 8), VY: rr.RangeF(-8, 8),
							Z: rr.RangeF(0, 6), VZ: rr.RangeF(20, 80),
							MaxLife: rr.RangeF(0.7, 1.6), Col: Palette.FireHot, Kind: ParticleFire,
						}
						ps.Add(fp)
					}
				}
				// Remove settled debris.
				ps.P[idx] = ps.P[len(ps.P)-1]
				ps.P = ps.P[:len(ps.P)-1]
				return true
			}
		}

		// Blood settle.
		if p.Kind == ParticleBlood {
			spd := math.Hypot(p.VX, p.VY)
			if spd < 28.0 || math.Abs(p.VZ) < 5.0 {
				stain := RGB{R: 130, G: 20, B: 20}
				brush := 1 + int(min(4, int(spd/28.0)))
				for oy := -brush; oy <= brush; oy++ {
					for ox := -brush; ox <= brush; ox++ {
						if math.Hypot(float64(ox), float64(oy)) > float64(brush) {
							continue
						}
						sx := wx + ox
						sy := wy + oy
						if sx >= 0 && sy >= 0 && sx < WorldWidth && sy < WorldHeight {
							dst := w.ColorAt(sx, sy)
							w.PaintRGB(sx, sy, lerpRGB(dst, stain, 0.55))
						}
					}
				}
				ps.P[idx] = ps.P[len(ps.P)-1]
				ps.P = ps.P[:len(ps.P)-1]
				return true
			}
		}
	}

	// Swept XY collision.
	dx := p.X - prevX
	dy := p.Y - prevY
	steps := max(1, int(math.Ceil(math.Hypot(dx, dy))))
	for s := 1; s <= steps; s++ {
		t := float64(s) / float64(steps)
		sx := int(math.Round(prevX + dx*t))
		sy := int(math.Round(prevY + dy*t))
		if !w.IsBlocked(sx, sy) {
			continue
		}
		h := float64(w.HeightAt(sx, sy))
		if p.Z > h+1.0 {
			continue
		}
		prevFrac := float64(s-1) / float64(steps)
		hitPrevX := prevX + dx*prevFrac
		hitPrevY := prevY + dy*prevFrac
		prevWX := int(math.Round(hitPrevX))
		prevWY := int(math.Round(hitPrevY))
		blockedX := w.IsBlocked(sx, prevWY)
		blockedY := w.IsBlocked(prevWX, sy)

		if !blockedX && blockedY {
			p.X = hitPrevX
			p.VX = -p.VX * 0.4
			p.VY *= 0.75
		} else if blockedX && !blockedY {
			p.Y = hitPrevY
			p.VY = -p.VY * 0.4
			p.VX *= 0.75
		} else {
			p.X = hitPrevX
			p.Y = hitPrevY
			p.VX = -p.VX * 0.32
			p.VY = -p.VY * 0.32
		}
		p.Bounce = 0.6
		p.MaxLife *= 0.92
		break
	}

	return false
}
