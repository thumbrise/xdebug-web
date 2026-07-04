package plugins

import "errors"

type CommandType string

const (
	CmdStepInto         CommandType = "step_into"
	CmdStepOver         CommandType = "step_over"
	CmdStepOut          CommandType = "step_out"
	CmdRun              CommandType = "run"
	CmdStop             CommandType = "stop"
	CmdBreakpointSet    CommandType = "breakpoint_set"
	CmdBreakpointRemove CommandType = "breakpoint_remove"
	CmdPropertyGet      CommandType = "property_get"
)

type Command struct {
	Type  CommandType
	Args  map[string]string
	Reply chan<- any
}

type Status string

const (
	StatusStarting Status = "starting"
	StatusRunning  Status = "running"
	StatusBreak    Status = "break"
	StatusStopping Status = "stopping"
	StatusStopped  Status = "stopped"
)

type Frame struct {
	Level    int    `json:"level"`
	Filename string `json:"filename"`
	Lineno   int    `json:"lineno"`
	Where    string `json:"where"`
}

type Variable struct {
	Name        string     `json:"name"`
	FullName    string     `json:"fullName,omitempty"`
	Type        string     `json:"type"`
	Value       string     `json:"value"`
	ClassName   string     `json:"className,omitempty"`
	NumChildren int        `json:"numChildren"`
	Children    []Variable `json:"children,omitempty"`
}

type Breakpoint struct {
	ID   string `json:"id"`
	File string `json:"file"`
	Line int    `json:"line"`
}

type Capabilities struct {
	HasStack   bool `json:"hasStack"`
	HasLocals  bool `json:"hasLocals"`
	HasGlobals bool `json:"hasGlobals"`
}

type State struct {
	Status       Status       `json:"status"`
	CurrentFile  string       `json:"currentFile"`
	CurrentLine  int          `json:"currentLine"`
	Stack        []Frame      `json:"stack"`
	Locals       []Variable   `json:"locals"`
	Globals      []Variable   `json:"globals"`
	Breakpoints  []Breakpoint `json:"breakpoints"`
	Capabilities Capabilities `json:"capabilities"`
}

var (
	ErrUnknownCommand      = errors.New("unknown command")
	ErrMissingBreakpointID = errors.New("breakpoint response missing id")
)
