package game

type EventType int

const (
	EventEntityDied EventType = iota
	EventExplosion
	EventAlienSpawned
	EventLevelComplete
)

type Event struct {
	Type EventType
	X, Y float64
	Data int // Generic payload (e.g. radius for explosion).
}

type EventHandler func(Event)

type EventBus struct {
	handlers map[EventType][]EventHandler
}

func NewEventBus() *EventBus {
	return &EventBus{
		handlers: make(map[EventType][]EventHandler),
	}
}

func (eb *EventBus) Subscribe(t EventType, fn EventHandler) {
	eb.handlers[t] = append(eb.handlers[t], fn)
}

func (eb *EventBus) Emit(e Event) {
	for _, fn := range eb.handlers[e.Type] {
		fn(e)
	}
}
