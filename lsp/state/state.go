package state

type State struct {
	docs map[string]string
}

func New() *State {
	return &State{docs: map[string]string{}}
}

func (s *State) UpdateDocument(key string, doc string) {
	s.docs[key] = doc
}

func (s *State) GetDocument(key string) string {
	return s.docs[key]
}
