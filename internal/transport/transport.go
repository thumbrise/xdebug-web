package transport

type OutgoingMessage struct {
	Type string `json:"type"`
	Data any    `json:"data,omitempty"`
}

type IncomingCommand struct {
	Type string            `json:"type"`
	Args map[string]string `json:"args,omitempty"`
}
