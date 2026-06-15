package readmodel

type Response struct {
	GeneratedAt   string  `json:"generated_at"`
	Result        Result  `json:"result"`
	WindowSeconds int     `json:"window_seconds"`
	Views         Views   `json:"views"`
	Unavailable   []Group `json:"unavailable"`
}

func (r Response) Available() bool {
	return len(r.Unavailable) < len(groups())
}
