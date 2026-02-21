package game

import "math"

// splitmix64 is a fast, high-quality 64-bit mixer.
func splitmix64(x uint64) uint64 {
	x += 0x9E3779B97F4A7C15
	z := x
	z = (z ^ (z >> 30)) * 0xBF58476D1CE4E5B9
	z = (z ^ (z >> 27)) * 0x94D049BB133111EB
	return z ^ (z >> 31)
}

// hash2D returns a deterministic 64-bit hash for (x,y) under the given seed.
func hash2D(seed uint64, x, y int) uint64 {
	ux := uint64(uint32(x))
	uy := uint64(uint32(y))
	h := seed
	h ^= ux * 0x9E3779B185EBCA87
	h ^= uy * 0xC2B2AE3D27D4EB4F
	return splitmix64(h)
}

// floorDiv performs mathematical floor division for integers.
func floorDiv(a, b int) int {
	q := a / b
	r := a % b
	if (r != 0) && ((r < 0) != (b < 0)) {
		q--
	}
	return q
}

func clamp(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func clampF(v, lo, hi float64) float64 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func abs(v int) int {
	if v < 0 {
		return -v
	}
	return v
}

func approach(cur, target, maxDelta float64) float64 {
	if cur < target {
		cur += maxDelta
		if cur > target {
			cur = target
		}
		return cur
	}
	if cur > target {
		cur -= maxDelta
		if cur < target {
			cur = target
		}
	}
	return cur
}

func angDiff(a, b float64) float64 {
	d := b - a
	for d <= -math.Pi {
		d += 2 * math.Pi
	}
	for d > math.Pi {
		d -= 2 * math.Pi
	}
	return d
}

func lerpU8(a, b uint8, t float64) uint8 {
	if t <= 0 {
		return a
	}
	if t >= 1 {
		return b
	}
	return uint8(float64(a) + (float64(b)-float64(a))*t)
}

func lerpRGB(a, b RGB, t float64) RGB {
	return RGB{R: lerpU8(a.R, b.R, t), G: lerpU8(a.G, b.G, t), B: lerpU8(a.B, b.B, t)}
}

// Rand is a tiny deterministic RNG (xorshift64*).
type Rand struct {
	s uint64
}

func NewRand(seed uint64) *Rand {
	if seed == 0 {
		seed = 1
	}
	return &Rand{s: seed}
}

func (r *Rand) NextU64() uint64 {
	x := r.s
	x ^= x >> 12
	x ^= x << 25
	x ^= x >> 27
	r.s = x
	return x * 2685821657736338717
}

func (r *Rand) Intn(n int) int {
	if n <= 0 {
		return 0
	}
	return int(r.NextU64() % uint64(n))
}

func (r *Rand) Range(min, max int) int {
	if max <= min {
		return min
	}
	return min + r.Intn(max-min+1)
}

func (r *Rand) Float64() float64 {
	return float64(r.NextU64()>>11) * (1.0 / (1 << 53))
}

func (r *Rand) RangeF(min, max float64) float64 {
	if max <= min {
		return min
	}
	return min + (max-min)*r.Float64()
}

func rgbEq(a, b RGB) bool { return a.R == b.R && a.G == b.G && a.B == b.B }
