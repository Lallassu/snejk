package game

type Camera struct {
	X, Y float64 // world-pixel space, camera centre
	Zoom float64 // screen pixels per world pixel

	// Screen shake.
	ShakeX, ShakeY float64 // current offset in world pixels
	ShakeTimer     float64 // remaining shake time
	ShakeIntensity float64 // max offset magnitude
}

// AddShake triggers screen shake with given intensity and duration.
func (c *Camera) AddShake(intensity, duration float64) {
	if intensity > c.ShakeIntensity {
		c.ShakeIntensity = intensity
	}
	if duration > c.ShakeTimer {
		c.ShakeTimer = duration
	}
}

// UpdateShake decays shake and computes random offsets.
func (c *Camera) UpdateShake(dt float64, seed uint64) {
	if c.ShakeTimer <= 0 {
		c.ShakeX = 0
		c.ShakeY = 0
		c.ShakeIntensity = 0
		return
	}
	c.ShakeTimer -= dt
	if c.ShakeTimer < 0 {
		c.ShakeTimer = 0
	}
	// Decaying intensity.
	t := c.ShakeTimer
	rr := NewRand(seed ^ uint64(t*10000))
	mag := c.ShakeIntensity * (t / (t + 0.08))
	c.ShakeX = rr.RangeF(-mag, mag)
	c.ShakeY = rr.RangeF(-mag, mag)
}

// EffectivePos returns camera position with shake applied.
func (c *Camera) EffectivePos() (float64, float64) {
	return c.X + c.ShakeX, c.Y + c.ShakeY
}

func (c *Camera) Clamp(fbW, fbH int) {
	if c.Zoom < MinZoom {
		c.Zoom = MinZoom
	}
	if c.Zoom > MaxZoom {
		c.Zoom = MaxZoom
	}

	halfW := float64(fbW) / (2.0 * c.Zoom)
	halfH := float64(fbH) / (2.0 * c.Zoom)

	minX := halfW
	maxX := float64(WorldWidth) - halfW
	minY := halfH
	maxY := float64(WorldHeight) - halfH

	if minX > maxX {
		c.X = float64(WorldWidth) * 0.5
	} else {
		if c.X < minX {
			c.X = minX
		}
		if c.X > maxX {
			c.X = maxX
		}
	}

	if minY > maxY {
		c.Y = float64(WorldHeight) * 0.5
	} else {
		if c.Y < minY {
			c.Y = minY
		}
		if c.Y > maxY {
			c.Y = maxY
		}
	}
}
