package dbgp

import "errors"

type Status string

const (
	StatusStarting Status = "starting"
	StatusBreak    Status = "break"
	StatusStopping Status = "stopping"
	StatusStopped  Status = "stopped"
	StatusRunning  Status = "running"
)

type Frame struct {
	Level    int    `json:"level"`
	Filename string `json:"filename"`
	Lineno   int    `json:"lineno"`
	Where    string `json:"where"`
}

type Variable struct {
	Name        string     `json:"name"`
	Type        string     `json:"type"`
	Value       string     `json:"value"`
	ClassName   string     `json:"className,omitempty"`
	NumChildren int        `json:"numChildren"`
	Children    []Variable `json:"children,omitempty"`
}

type InitInfo struct {
	Language string `json:"language"`
	FileURI  string `json:"fileUri"`
	AppID    string `json:"appId"`
	IdeKey   string `json:"ideKey"`
}

type StepResult struct {
	Status   Status `json:"status"`
	Filename string `json:"filename,omitempty"`
	Lineno   int    `json:"lineno,omitempty"`
	Reason   string `json:"reason,omitempty"`
}

type CommandType string

const (
	CmdStepInto CommandType = "step_into"
	CmdStepOver CommandType = "step_over"
	CmdStepOut  CommandType = "step_out"
	CmdRun      CommandType = "run"
	CmdStop     CommandType = "stop"
	CmdBreak    CommandType = "break"

	CmdBreakpointSet    CommandType = "breakpoint_set"
	CmdBreakpointRemove CommandType = "breakpoint_remove"
)

type Breakpoint struct {
	ID   string `json:"id"`
	File string `json:"file"`
	Line int    `json:"line"`
}

type Command struct {
	Type  CommandType
	Args  map[string]string
	Reply chan<- any
}

var (
	ErrUnknownCommand      = errors.New("unknown dbgp command")
	ErrMissingBreakpointID = errors.New("breakpoint_set response missing id")
)
