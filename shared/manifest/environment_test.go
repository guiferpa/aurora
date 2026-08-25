package manifest

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func projectWith(t *testing.T, files map[string]string) string {
	t.Helper()

	dir := t.TempDir()
	for name, content := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o600); err != nil {
			t.Fatalf("writing %s: %v", name, err)
		}
	}
	return dir
}

// A value the manifest should not hold is named instead, and read from beside it.
func TestAManifestReadsWhatItNames(t *testing.T) {
	dir := projectWith(t, map[string]string{
		EnvFilename: "PRIVKEY=abc123\n",
		Filename: `[project]
  name = "p"
[profiles]
  [profiles.main]
    source = "src/main.ar"
    privkey = "${{ PRIVKEY }}"
`,
	})

	m, err := Load(dir)
	if err != nil {
		t.Fatalf("loading: %v", err)
	}
	if got := m.Profiles["main"].Privkey; got != "abc123" {
		t.Errorf("the key is %q, want what .env set", got)
	}
}

// The project's own file comes first and the machine's environment second, so a project can be
// cloned and run — and a machine that knows better still gets to say so, for what the project
// did not write down.
func TestTheProjectComesFirstAndTheMachineSecond(t *testing.T) {
	t.Setenv("RPC", "from-the-machine")
	t.Setenv("PRIVKEY", "from-the-machine")

	dir := projectWith(t, map[string]string{
		EnvFilename: "PRIVKEY=from-the-project\n",
		Filename: `[project]
  name = "p"
[profiles]
  [profiles.main]
    source = "src/main.ar"
    rpc = "${{ RPC }}"
    privkey = "${{ PRIVKEY }}"
`,
	})

	m, err := Load(dir)
	if err != nil {
		t.Fatalf("loading: %v", err)
	}
	if got := m.Profiles["main"].Privkey; got != "from-the-project" {
		t.Errorf("the key is %q, want the project's own", got)
	}
	if got := m.Profiles["main"].RPC; got != "from-the-machine" {
		t.Errorf("the address is %q, want the machine's, since the project does not say", got)
	}
}

// A name nothing sets is refused. An empty value reaches a deploy as a key that is not a key,
// and the failure that follows says nothing about the manifest that caused it.
func TestANameNothingSetsIsRefused(t *testing.T) {
	dir := projectWith(t, map[string]string{
		Filename: `[project]
  name = "p"
[profiles]
  [profiles.main]
    source = "src/main.ar"
    privkey = "${{ NOBODY_SETS_THIS }}"
`,
	})

	_, err := Load(dir)
	if err == nil {
		t.Fatal("it loaded a manifest naming something nothing sets")
	}
	for _, want := range []string{"NOBODY_SETS_THIS", EnvFilename} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("it says %q, want it to mention %q", err, want)
		}
	}
}

// A project with nothing to hide needs neither file nor environment.
func TestAManifestThatNamesNothingNeedsNothing(t *testing.T) {
	dir := projectWith(t, map[string]string{
		Filename: `[project]
  name = "p"
[profiles]
  [profiles.main]
    source = "src/main.ar"
`,
	})

	if _, err := Load(dir); err != nil {
		t.Fatalf("loading: %v", err)
	}
}

// What a .env holds and what it does not. A value arrives as it was written, because it is a
// secret or an address and both mean their own bytes.
func TestWhatAnEnvFileHolds(t *testing.T) {
	dir := projectWith(t, map[string]string{
		EnvFilename: strings.Join([]string{
			"# a comment",
			"",
			"PLAIN=value",
			"  SPACED  =  around  ",
			`QUOTED="  kept  "`,
			"SINGLE='kept too'",
			"export EXPORTED=works",
			"WITH_EQUALS=a=b",
			"no equals here",
		}, "\n"),
	})

	env, err := LoadEnvironment(dir)
	if err != nil {
		t.Fatalf("reading: %v", err)
	}

	for name, want := range map[string]string{
		"PLAIN":       "value",
		"SPACED":      "around",
		"QUOTED":      "  kept  ",
		"SINGLE":      "kept too",
		"EXPORTED":    "works",
		"WITH_EQUALS": "a=b",
	} {
		if got, found := env.Lookup(name); !found || got != want {
			t.Errorf("%s is %q (found=%v), want %q", name, got, found, want)
		}
	}
	if _, found := env.Lookup("a comment"); found {
		t.Error("a comment was read as a setting")
	}
}

// A reference is written with doubled braces so that nothing a shell touches looks like one.
func TestOnlyDoubledBracesAreAReference(t *testing.T) {
	t.Setenv("NAME", "read")
	env, err := LoadEnvironment(t.TempDir())
	if err != nil {
		t.Fatalf("reading: %v", err)
	}

	got, err := env.Expand("test", `a = "${{ NAME }}" b = "${NAME}" c = "$NAME"`)
	if err != nil {
		t.Fatalf("expanding: %v", err)
	}
	if want := `a = "read" b = "${NAME}" c = "$NAME"`; got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

// A manifest may show the form in a comment beside the setting it explains, and a comment is
// not a value: nothing tries to resolve it.
//
// It is the reason the reading happens on the values and not on the text. Doing it on the text
// was shorter, and it asked a manifest to resolve its own example.
func TestACommentIsNotAValue(t *testing.T) {
	dir := projectWith(t, map[string]string{
		Filename: `[project]
  name = "p"
[profiles]
  [profiles.main]
    source = "src/main.ar"
    # A value written as ${{ AURORA_PRIVKEY }} is read from .env beside this file.
    # privkey = "${{ AURORA_PRIVKEY }}"
`,
	})

	if _, err := Load(dir); err != nil {
		t.Fatalf("loading: %v", err)
	}
}

// And a value that is not one of the two a deploy needs is left alone. A path is a path: it is
// not a thing a project keeps somewhere else, so it is not read from anywhere else.
func TestOnlyWhatADeployNeedsIsRead(t *testing.T) {
	t.Setenv("SOMETHING", "read")

	dir := projectWith(t, map[string]string{
		Filename: `[project]
  name = "${{ SOMETHING }}"
[profiles]
  [profiles.main]
    source = "${{ SOMETHING }}"
`,
	})

	m, err := Load(dir)
	if err != nil {
		t.Fatalf("loading: %v", err)
	}
	if got := m.Project.Name; got != "${{ SOMETHING }}" {
		t.Errorf("the project is named %q, want the reference left as it was written", got)
	}
	if got := m.Profiles["main"].Source; got != "${{ SOMETHING }}" {
		t.Errorf("the source is %q, want the reference left as it was written", got)
	}
}
