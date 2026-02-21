package game

// Health tracks HP for any entity.
type Health struct {
	Current float64
	Max     float64
}

func NewHealth(max float64) Health {
	return Health{Current: max, Max: max}
}

func (h *Health) Damage(amount float64) {
	h.Current -= amount
	if h.Current < 0 {
		h.Current = 0
	}
}

func (h *Health) Heal(amount float64) {
	h.Current += amount
	if h.Current > h.Max {
		h.Current = h.Max
	}
}

func (h *Health) Fraction() float64 {
	if h.Max <= 0 {
		return 0
	}
	f := h.Current / h.Max
	if f < 0 {
		return 0
	}
	if f > 1 {
		return 1
	}
	return f
}

func (h *Health) IsDead() bool {
	return h.Current <= 0
}

func (h *Health) IsInjured() bool {
	return h.Current < h.Max
}

// HealthBarColor returns green/yellow/red based on fraction.
func HealthBarColor(frac float64) RGB {
	if frac > 0.6 {
		return RGB{R: 60, G: 220, B: 60}
	}
	if frac > 0.3 {
		return RGB{R: 220, G: 220, B: 60}
	}
	return RGB{R: 220, G: 60, B: 60}
}
