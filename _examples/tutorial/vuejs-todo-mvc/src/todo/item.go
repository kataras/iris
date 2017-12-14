package todo

type State uint32

const (
	StateActive State = iota
	StateCompleted
)

func ParseState(s string) State {
	switch s {
	case "completed":
		return StateCompleted
	default:
		return StateActive
	}
}

type Item struct {
	OwnerID      string
	ID           int64
	Body         string
	CurrentState State
}
