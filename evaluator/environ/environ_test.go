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

	t.Run("TestGetIdentNotExists", func(t *testing.T) {
		got := env1.GetIdent("Z")
		if got != nil {
			t.Errorf("ident Z should not exists, got: %v", got)
		}
	})

	t.Run("TestGetIdentExists", func(t *testing.T) {
		got := env2.GetIdent("A") // from environ 0
		expected := []byte("B")
		if !bytes.Equal(got, expected) {
			t.Errorf("unexpected result: got: %v, expected: %v", got, expected)
		}
	})

	t.Run("TestGetIdentExistsPriority", func(t *testing.T) {
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

	t.Run("TestSetArgument", func(t *testing.T) {
		env0.SetArgument(0, []byte("G"))
		got := env0.GetArgument(0)
		expected := []byte("G")
		if !bytes.Equal(got, expected) {
			t.Errorf("unexpected result: got: %v, expected: %v", got, expected)
		}
	})
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
	got := env0.GetTemp("A")
	expected := []byte("G")
	if !bytes.Equal(got, expected) {
		t.Errorf("unexpected result: got: %v, expected: %v", got, expected)
	}
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
