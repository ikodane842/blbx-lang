package interpreter

type Scope struct {
	parent *Scope
	values map[string]Value
}

func NewScope(parent *Scope) *Scope {
	return &Scope{
		parent: parent,
		values: map[string]Value{},
	}
}

func (s *Scope) Define(name string, value Value) {
	s.values[name] = value
}

func (s *Scope) Set(name string, value Value) {
	if _, ok := s.values[name]; ok {
		s.values[name] = value
		return
	}

	if s.parent != nil {
		s.parent.Set(name, value)
		return
	}

	s.values[name] = value
}

func (s *Scope) Get(name string) (Value, bool) {
	if value, ok := s.values[name]; ok {
		return value, true
	}

	if s.parent != nil {
		return s.parent.Get(name)
	}

	return Null(), false
}

func (s *Scope) Snapshot() map[string]Value {
	out := map[string]Value{}
	for name, value := range s.values {
		out[name] = value
	}
	return out
}
