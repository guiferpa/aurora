package state

type State struct {
	docs map[string]string
	// snippets is what the client said about itself at initialize. A client that does not
	// expand snippets gets plain keywords: the placeholders would land in the buffer as
	// the literal text they are.
	snippets bool
}

func New() *State {
	return &State{docs: map[string]string{}}
}

func (s *State) SetSnippetSupport(supported bool) {
	s.snippets = supported
}

func (s *State) SnippetSupport() bool {
	return s.snippets
}

func (s *State) UpdateDocument(key string, doc string) {
	s.docs[key] = doc
}

func (s *State) GetDocument(key string) string {
	return s.docs[key]
}

// DeleteDocument drops a document the client closed.
func (s *State) DeleteDocument(key string) {
	delete(s.docs, key)
}
