package todo

type Item struct {
	SessionID string `json:"-"`
	ID        int64  `json:"id,omitempty"`
	Title     string `json:"title"`
	Completed bool   `json:"completed"`
}
