package manifest

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// EnvFilename is where a project keeps what it does not want written down in its manifest.
const EnvFilename = ".env"

// reference is what a manifest writes where a value comes from the environment.
//
// The braces are doubled so that it cannot be mistaken for anything a shell would expand: a
// manifest is read by Aurora and never by a shell, and a form a shell would touch is a form
// somebody will one day pipe through one.
var reference = regexp.MustCompile(`\$\{\{\s*([A-Za-z_][A-Za-z0-9_]*)\s*\}\}`)

// Environment is what a project's values are read from, when the manifest says to read them.
type Environment struct {
	// file is what .env held, and it is looked at first.
	file map[string]string
}

// LoadEnvironment reads the .env of a project, if it has one. A project without one is not an
// error: the system's environment is still there, and a manifest that names nothing needs
// neither.
func LoadEnvironment(projectRoot string) (Environment, error) {
	path := filepath.Join(projectRoot, EnvFilename)
	f, err := os.Open(path)
	if os.IsNotExist(err) {
		return Environment{}, nil
	}
	if err != nil {
		return Environment{}, fmt.Errorf("open %s: %w", path, err)
	}
	defer func() { _ = f.Close() }()

	values := make(map[string]string)
	scanner := bufio.NewScanner(f)
	for line := 1; scanner.Scan(); line++ {
		name, value, ok := readSetting(scanner.Text())
		if !ok {
			continue
		}
		if name == "" {
			return Environment{}, fmt.Errorf("%s:%d: a setting with no name", path, line)
		}
		values[name] = value
	}
	if err := scanner.Err(); err != nil {
		return Environment{}, fmt.Errorf("read %s: %w", path, err)
	}
	return Environment{file: values}, nil
}

// readSetting reads one line of a .env: a name, an equals, and the rest. A blank line and a
// comment are neither a setting nor a mistake, and say so by answering false.
//
// The value may be quoted, which is how a value with spaces at either end says it means them.
// Nothing else is interpreted — no escapes and no expansion — because a value here is a secret
// or an address, and both are meant to arrive exactly as they were written.
func readSetting(line string) (name, value string, ok bool) {
	line = strings.TrimSpace(line)
	if line == "" || strings.HasPrefix(line, "#") {
		return "", "", false
	}
	line = strings.TrimPrefix(line, "export ")

	name, value, found := strings.Cut(line, "=")
	if !found {
		return "", "", false
	}
	name = strings.TrimSpace(name)
	value = strings.TrimSpace(value)

	for _, quote := range []string{`"`, `'`} {
		if len(value) >= 2 && strings.HasPrefix(value, quote) && strings.HasSuffix(value, quote) {
			value = value[1 : len(value)-1]
			break
		}
	}
	return name, value, true
}

// Lookup answers what a name is set to, and where it was found.
//
// The project's own .env comes first and the system's environment second. A project says what
// it needs in a file beside itself, and a machine may say otherwise — a build server holding
// the real key, a person running against a different chain — so the outer one is the fallback
// and not the override. It is the order that lets a project be cloned and run.
func (e Environment) Lookup(name string) (string, bool) {
	if value, found := e.file[name]; found {
		return value, true
	}
	return os.LookupEnv(name)
}

// Expand answers the text with every reference to the environment replaced by what it names.
//
// A name nothing sets is refused rather than replaced with nothing. An empty value reaches a
// deploy as a key that is not a key and an address that is not an address, and the failure
// that follows says nothing about the manifest that caused it — which is the whole reason a
// manifest may name the environment at all.
func (e Environment) Expand(where, text string) (string, error) {
	var missing []string

	expanded := reference.ReplaceAllStringFunc(text, func(match string) string {
		name := reference.FindStringSubmatch(match)[1]
		value, found := e.Lookup(name)
		if !found {
			missing = append(missing, name)
			return match
		}
		return value
	})

	if len(missing) > 0 {
		return "", fmt.Errorf(
			"%s names %s, and nothing sets %s: put it in %s beside the manifest, or in the environment the command runs in",
			where, strings.Join(quoted(missing), ", "), plural(len(missing)), EnvFilename)
	}
	return expanded, nil
}

func quoted(names []string) []string {
	out := make([]string, 0, len(names))
	for _, name := range names {
		out = append(out, fmt.Sprintf("%q", name))
	}
	return out
}

func plural(n int) string {
	if n == 1 {
		return "it"
	}
	return "them"
}
