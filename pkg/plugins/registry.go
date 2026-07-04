package plugins

import "sync"

type Descriptor struct {
	Name       string
	Language   string
	Factory    Factory
}

type Registry struct {
	mu   sync.RWMutex
	descs map[string]Descriptor
}

func NewRegistry() *Registry {
	return &Registry{descs: make(map[string]Descriptor)}
}

func (r *Registry) Register(d Descriptor) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.descs[d.Name] = d
}

func (r *Registry) Get(name string) (Descriptor, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	d, ok := r.descs[name]
	return d, ok
}

func (r *Registry) All() []Descriptor {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]Descriptor, 0, len(r.descs))
	for _, d := range r.descs {
		out = append(out, d)
	}
	return out
}
