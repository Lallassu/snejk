package game

// InfectionState marks whether a pedestrian is visually infected (won't spread, just distinct & bad to eat).
type InfectionState int

const (
	StateHealthy     InfectionState = iota
	StateSymptomatic                // green tint — dangerous to eat (shrinks snake)
)
