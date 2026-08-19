package environ

import (
	"bytes"
	"testing"
)

func TestAhead(t *testing.T) {
	env0 := NewEnviron(NewEnvironOptions{
		Args: []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 10},
	})
	env1 := NewEnviron(NewEnvironOptions{
		Args: []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 20},
	})

	env3 := env1.Ahead(env0)

	got := env3.GetArgument(0)
	expected := env0.GetArgument(0)
	if !bytes.Equal(got, expected) {
		t.Errorf("unexpected result: got: %v, expected: %v", got, expected)
	}
}

func TestGetPrevious(t *testing.T) {
	env0 := NewEnviron(NewEnvironOptions{
		Args: []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 10},
	})
	env1 := NewEnviron(NewEnvironOptions{
		Prev: env0,
	})

	env3 := env1.GetPrevious()

	got := env3.GetArgument(0)
	expected := env0.GetArgument(0)
	if !bytes.Equal(got, expected) {
		t.Errorf("unexpected result: got: %v, expected: %v", got, expected)
	}
}

func TestGetIdent(t *testing.T) {
	env0 := NewEnviron(NewEnvironOptions{
		Idents: map[string][]byte{
			"A": []byte("B"),
			"E": []byte("Y"),
		},
	})
	env1 := NewEnviron(NewEnvironOptions{
		Idents: map[string][]byte{
			"C": []byte("D"),
		},
		Prev: env0,
	})
	env2 := NewEnviron(NewEnvironOptions{
		Idents: map[string][]byte{
			"E": []byte("F"),
		},
		Prev: env1,
	})

	t.Run("not_exists", func(t *testing.T) {
		got := env1.GetIdent("Z")
		if got != nil {
			t.Errorf("ident Z should not exists, got: %v", got)
		}
	})

	t.Run("exists", func(t *testing.T) {
		got := env2.GetIdent("A") // from environ 0
		expected := []byte("B")
		if !bytes.Equal(got, expected) {
			t.Errorf("unexpected result: got: %v, expected: %v", got, expected)
		}
	})

	t.Run("exists_priority", func(t *testing.T) {
		got := env2.GetIdent("E") // from environ 2
		expected := []byte("F")
		if !bytes.Equal(got, expected) {
			t.Errorf("unexpected result: got: %v, expected: %v", got, expected)
		}
	})
}

func TestSetIdent(t *testing.T) {
	env0 := NewEnviron(NewEnvironOptions{
		Idents: map[string][]byte{},
	})

	t.Run("TestSetIdent", func(t *testing.T) {
		env0.SetIdent("A", []byte("G"))
		got := env0.GetIdent("A")
		expected := []byte("G")
		if !bytes.Equal(got, expected) {
			t.Errorf("unexpected result: got: %v, expected: %v", got, expected)
		}
	})
}

func TestSetArgument(t *testing.T) {
	env0 := NewEnviron(NewEnvironOptions{
		Args: make([]byte, 0),
	})
	env0.SetArgument(0, []byte("G"))
	got := env0.GetArgument(0)
	expected := []byte("G")
	if !bytes.Equal(got, expected) {
		t.Errorf("unexpected result: got: %v, expected: %v", got, expected)
	}
}

func TestGetArgument(t *testing.T) {
	env0 := NewEnviron(NewEnvironOptions{
		Args: []byte{
			0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 10,
			0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 20,
			0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 30,
		},
	})

	t.Run("TestGetArgument", func(t *testing.T) {
		got := env0.GetArgument(1)
		// Calldata words are 32 bytes; the environ narrows each one to a tape.
		expected := []byte{0, 0, 0, 0, 0, 0, 0, 20}
		if !bytes.Equal(got, expected) {
			t.Errorf("unexpected result: got: %v, expected: %v", got, expected)
		}
	})
}

func TestSetTemp(t *testing.T) {
	env0 := NewEnviron(NewEnvironOptions{})
	env0.SetTemp("A", []byte("G"))

	t.Run("exists", func(t *testing.T) {
		got := env0.GetTemp("A")
		expected := []byte("G")
		if !bytes.Equal(got, expected) {
			t.Errorf("unexpected result: got: %v, expected: %v", got, expected)
		}
	})

	t.Run("not_exists", func(t *testing.T) {
		got := env0.GetTemp("B")
		if got != nil {
			t.Errorf("temp B should not exists, got: %v", got)
		}
	})
}

func TestGetArgumentsLength(t *testing.T) {
	t.Run("empty_args", func(t *testing.T) {
		env := NewEnviron(NewEnvironOptions{Args: make([]byte, 0)})
		got := env.GetArgumentsLength()
		if got != 0 {
			t.Errorf("expected 0 arguments, got: %d", got)
		}
	})

	t.Run("one_arg_from_opts", func(t *testing.T) {
		args := make([]byte, 32)
		env := NewEnviron(NewEnvironOptions{Args: args})
		got := env.GetArgumentsLength()
		if got != 1 {
			t.Errorf("expected 1 argument, got: %d", got)
		}
	})

	t.Run("two_args_from_opts", func(t *testing.T) {
		args := make([]byte, 64)
		env := NewEnviron(NewEnvironOptions{Args: args})
		got := env.GetArgumentsLength()
		if got != 2 {
			t.Errorf("expected 2 arguments, got: %d", got)
		}
	})

	t.Run("args_set_via_SetArgument", func(t *testing.T) {
		env := NewEnviron(NewEnvironOptions{Args: make([]byte, 0)})
		env.SetArgument(0, []byte("a"))
		env.SetArgument(1, []byte("b"))
		got := env.GetArgumentsLength()
		if got != 2 {
			t.Errorf("expected 2 arguments after SetArgument(0) and SetArgument(1), got: %d", got)
		}
	})
}

func TestDefersLength(t *testing.T) {
	env := NewEnviron(NewEnvironOptions{})

	t.Run("empty", func(t *testing.T) {
		if got := env.DefersLength(); got != 0 {
			t.Errorf("expected 0 defers, got: %d", got)
		}
	})

	t.Run("after_one_SetDefer", func(t *testing.T) {
		env.SetDefer("0", []byte("blob1"))
		if got := env.DefersLength(); got != 1 {
			t.Errorf("expected 1 defer, got: %d", got)
		}
	})

	t.Run("after_two_SetDefer", func(t *testing.T) {
		env.SetDefer("1", []byte("blob2"))
		if got := env.DefersLength(); got != 2 {
			t.Errorf("expected 2 defers, got: %d", got)
		}
	})
}

func TestSetDefer(t *testing.T) {
	env := NewEnviron(NewEnvironOptions{})

	t.Run("store_and_retrieve", func(t *testing.T) {
		blob := []byte("defer-blob-data")
		env.SetDefer("0", blob)
		got := env.GetDefer("0")
		if !bytes.Equal(got, blob) {
			t.Errorf("unexpected result: got: %v, expected: %v", got, blob)
		}
	})

	t.Run("empty_blob_not_stored", func(t *testing.T) {
		env.SetDefer("empty", []byte{})
		got := env.GetDefer("empty")
		if got != nil {
			t.Errorf("empty blob should not be stored, got: %v", got)
		}
	})
}

func TestGetDefer(t *testing.T) {
	env0 := NewEnviron(NewEnvironOptions{})
	env0.SetDefer("0", []byte("from-env0"))
	env1 := NewEnviron(NewEnvironOptions{
		Prev: env0,
	})
	env1.SetDefer("1", []byte("from-env1"))
	env2 := NewEnviron(NewEnvironOptions{
		Prev: env1,
	})
	env2.SetDefer("0", []byte("from-env2-override"))

	t.Run("key_not_exists", func(t *testing.T) {
		got := env1.GetDefer("missing")
		if got != nil {
			t.Errorf("expected nil for missing key, got: %v", got)
		}
	})

	t.Run("key_in_current_env", func(t *testing.T) {
		got := env1.GetDefer("1")
		expected := []byte("from-env1")
		if !bytes.Equal(got, expected) {
			t.Errorf("unexpected result: got: %v, expected: %v", got, expected)
		}
	})

	t.Run("key_in_prev_env", func(t *testing.T) {
		got := env1.GetDefer("0")
		expected := []byte("from-env0")
		if !bytes.Equal(got, expected) {
			t.Errorf("unexpected result: got: %v, expected: %v", got, expected)
		}
	})

	t.Run("inner_shadows_outer", func(t *testing.T) {
		got := env2.GetDefer("0")
		expected := []byte("from-env2-override")
		if !bytes.Equal(got, expected) {
			t.Errorf("inner env should shadow outer: got: %v, expected: %v", got, expected)
		}
	})
}

// A module keeps its names in an environ of its own, indexed by the module's name, and asking
// for it twice is asking for the same one — a module's body runs once and what it bound has
// to still be there when somebody calls into it.
func TestOpenModuleAnswersTheSameEnviron(t *testing.T) {
	root := NewEnviron(NewEnvironOptions{})

	first := root.OpenModule("a/b/c")
	first.SetIdent("k", []byte{7})

	if again := root.OpenModule("a/b/c"); again != first {
		t.Fatal("opening a module twice made two environs")
	}
	if got := root.Module("a/b/c").GetLocalIdent("k"); !bytes.Equal(got, []byte{7}) {
		t.Errorf("the module holds %v, want the value it bound", got)
	}
}

// A module nobody ran is not there, and asking is not a way of making one.
func TestAModuleThatNeverRan(t *testing.T) {
	root := NewEnviron(NewEnvironOptions{})
	if got := root.Module("a/b/c"); got != nil {
		t.Errorf("a module that never ran answered %v", got)
	}
}

// The index is reachable from anywhere on a chain, which is what a scope called from another
// module needs: it runs at the head of its caller's chain, and still has to find its own file.
func TestTheModuleIndexIsReachableFromAScope(t *testing.T) {
	root := NewEnviron(NewEnvironOptions{})
	root.OpenModule("a/b/c").SetIdent("k", []byte{7})

	inner := root.Ahead(NewEnviron(NewEnvironOptions{})).Ahead(NewEnviron(NewEnvironOptions{}))

	home := inner.Module("a/b/c")
	if home == nil {
		t.Fatal("a scope two levels down cannot reach the module index")
	}
	if got := home.GetLocalIdent("k"); !bytes.Equal(got, []byte{7}) {
		t.Errorf("the module holds %v, want the value it bound", got)
	}
}

// A module sees what it declared, and nothing of whoever imported it: its environ stands on
// its own rather than hanging off the program's.
func TestAModuleHasNoChainBehindIt(t *testing.T) {
	root := NewEnviron(NewEnvironOptions{})
	root.SetIdent("outside", []byte{1})

	if got := root.OpenModule("m").GetIdent("outside"); got != nil {
		t.Errorf("a module read %v from the program that uses it", got)
	}
}

// Holder is which environ has the name, which is what a call needs: a deferred scope is an
// index counted where it was created, so the body is looked for exactly there.
func TestHolderAnswersWhereANameLives(t *testing.T) {
	root := NewEnviron(NewEnvironOptions{})
	root.SetIdent("k", []byte{1})
	inner := root.Ahead(NewEnviron(NewEnvironOptions{}))

	if got := inner.Holder("k"); got != root {
		t.Error("the name was not found in the environ that has it")
	}
	if got := inner.Holder("nothing"); got != nil {
		t.Errorf("a name nobody bound was found in %v", got)
	}
}

// And a deferred scope is not looked for outwards: index 0 of one environ is a different
// scope from index 0 of another.
func TestGetLocalDeferDoesNotWalkOutwards(t *testing.T) {
	root := NewEnviron(NewEnvironOptions{})
	root.SetDefer("0", []byte{9})
	inner := root.Ahead(NewEnviron(NewEnvironOptions{}))

	if got := inner.GetLocalDefer("0"); got != nil {
		t.Errorf("a scope of the environ outside was found: %v", got)
	}
	if got := inner.GetDefer("0"); !bytes.Equal(got, []byte{9}) {
		t.Errorf("walking outwards answered %v, want the scope that is there", got)
	}
}
