package session

import (
	"sync"

	"github.com/thumbrise/xdebug-web/internal/dbgp"
)

type State struct {
	mu          sync.RWMutex
	Status      dbgp.Status     `json:"status"`
	InitInfo    *dbgp.InitInfo  `json:"initInfo,omitempty"`
	CurrentFile string          `json:"currentFile"`
	CurrentLine int             `json:"currentLine"`
	Stack       []dbgp.Frame    `json:"stack"`
	Locals      []dbgp.Variable `json:"locals"`
	Globals     []dbgp.Variable `json:"globals"`
}

func NewState() *State {
	return &State{Status: dbgp.StatusStarting}
}

func (s *State) SetInit(info *dbgp.InitInfo) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.InitInfo = info
	s.CurrentFile = info.FileURI
}

func (s *State) SetStepStatus(status dbgp.Status, filename string, lineno int) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.Status = status
	s.CurrentFile = filename
	s.CurrentLine = lineno
}

func (s *State) SetBreakStatus(filename string, lineno int) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.Status = dbgp.StatusBreak
	s.CurrentFile = filename
	s.CurrentLine = lineno
}

func (s *State) SetStopped() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.Status = dbgp.StatusStopped
}

func (s *State) SetRunning() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.Status = dbgp.StatusRunning
}

func (s *State) UpdateStack(frames []dbgp.Frame) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.Stack = frames
}

func (s *State) UpdateLocals(vars []dbgp.Variable) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.Locals = vars
}

func (s *State) UpdateGlobals(vars []dbgp.Variable) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.Globals = vars
}

func (s *State) Snapshot() *State {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return &State{
		Status:      s.Status,
		InitInfo:    s.InitInfo,
		CurrentFile: s.CurrentFile,
		CurrentLine: s.CurrentLine,
		Stack:       copyFrames(s.Stack),
		Locals:      copyVariables(s.Locals),
		Globals:     copyVariables(s.Globals),
	}
}

func copyFrames(in []dbgp.Frame) []dbgp.Frame {
	out := make([]dbgp.Frame, len(in))
	copy(out, in)

	return out
}

func copyVariables(in []dbgp.Variable) []dbgp.Variable {
	out := make([]dbgp.Variable, len(in))
	copy(out, in)

	return out
}
