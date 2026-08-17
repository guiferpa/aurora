package state

import "testing"

// State holds the documents the editor has open, which is what every request works against.
func TestDocumentLifecycle(t *testing.T) {
	s := New()

	if got := s.GetDocument("file:///a.ar"); got != "" {
		t.Errorf("an unknown document should be empty, got %q", got)
	}

	s.UpdateDocument("file:///a.ar", "ident a = 1;")
	if got, want := s.GetDocument("file:///a.ar"), "ident a = 1;"; got != want {
		t.Errorf("got %q, want %q", got, want)
	}

	// didChange sends the whole document, so an update replaces what was there.
	s.UpdateDocument("file:///a.ar", "ident a = 2;")
	if got, want := s.GetDocument("file:///a.ar"), "ident a = 2;"; got != want {
		t.Errorf("after update: got %q, want %q", got, want)
	}

	s.DeleteDocument("file:///a.ar")
	if got := s.GetDocument("file:///a.ar"); got != "" {
		t.Errorf("after close: got %q, want empty", got)
	}
}

func TestDocumentsAreIndependent(t *testing.T) {
	s := New()
	s.UpdateDocument("file:///a.ar", "a")
	s.UpdateDocument("file:///b.ar", "b")

	s.DeleteDocument("file:///a.ar")

	if got := s.GetDocument("file:///b.ar"); got != "b" {
		t.Errorf("closing one document affected another: got %q", got)
	}
}

func TestDeleteUnknownDocumentIsHarmless(t *testing.T) {
	s := New()
	s.DeleteDocument("file:///never-opened.ar")
}
