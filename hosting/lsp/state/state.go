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

// Documents answers every open document, keyed by the URI it arrived under.
//
// Whoever wants to find one by something other than that key does the translating: a URI is
// the client's idea of a name, and this package keeps what it was handed. The one caller
// today is the module resolution, which has a path and needs the buffer rather than the file
// on disk — an editor that answered from the disk would be answering about a version the
// person editing has already moved past.
func (s *State) Documents() map[string]string {
	open := make(map[string]string, len(s.docs))
	for uri, text := range s.docs {
		open[uri] = text
	}
	return open
}

// DeleteDocument drops a document the client closed.
func (s *State) DeleteDocument(key string) {
	delete(s.docs, key)
}
