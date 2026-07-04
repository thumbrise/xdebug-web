package transport

type OutgoingMessage struct {
	Type   string `json:"type"`
	Data   any    `json:"data,omitempty"`
	Source string `json:"source,omitempty"`
}

type IncomingCommand struct {
	Type   string            `json:"type"`
	Args   map[string]string `json:"args,omitempty"`
	Target string            `json:"target,omitempty"`
}
