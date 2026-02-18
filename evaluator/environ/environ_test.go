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
		expected := []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 20}
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
