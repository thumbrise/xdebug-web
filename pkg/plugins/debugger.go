package plugins

import "context"

type Info struct {
	Name     string
	Language string
}

type Debugger interface {
	Info() Info
	Run(ctx context.Context) error
	Close() error
	Commands() chan<- Command
	State() <-chan *State
}

type Factory func(ctx context.Context) (Debugger, error)
