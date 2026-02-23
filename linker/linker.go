package linker

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"

	"github.com/guiferpa/aurora/emitter"
	"github.com/guiferpa/aurora/fileutil"
	"github.com/guiferpa/aurora/lexer"
	"github.com/guiferpa/aurora/parser"
)

const RootNamespace = "."

type Linker interface {
	Resolve() ([]emitter.Instruction, error)
}

type lkr struct {
	source     string
	namespaces map[string]parser.Namespace
	processing []string
	loggers    []string
	order      []string
}

type NewLinkerOptions struct {
	Source  string
	Loggers []string
}

func NewLinker(opts NewLinkerOptions) (Linker, error) {
	abs, err := filepath.Abs(opts.Source)
	if err != nil {
		return nil, err
	}
	return &lkr{
		source:     filepath.Dir(abs),
		namespaces: make(map[string]parser.Namespace),
		processing: make([]string, 0),
		loggers:    opts.Loggers,
		order:      make([]string, 0),
	}, nil
}

func (l *lkr) GetTokens(filename string) ([]lexer.Token, error) {
	bs, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	opts := lexer.NewLexerOptions{
		EnableLogging: slices.Contains(l.loggers, "lexer"),
	}
	return lexer.New(opts).GetFilledTokens(bs)
}

func (l *lkr) GetUnits(dir string) ([]parser.ParserUnit, error) {
	files, err := fileutil.ListFilesByExtension(dir, ".ar")
	if err != nil {
		return nil, err
	}
	units := make([]parser.ParserUnit, 0)
	for _, filename := range files {
		abs, err := filepath.Abs(filename)
		if err != nil {
			return nil, err
		}
		tokens, err := l.GetTokens(abs)
		if err != nil {
			return nil, err
		}
		units = append(units, parser.ParserUnit{
			Filename: filename,
			Tokens:   tokens,
		})
	}
	return units, nil
}

func (l *lkr) GetNamespace(name string) (parser.Namespace, error) {
	if ns, ok := l.namespaces[name]; ok {
		return ns, nil
	}

	units, err := l.GetUnits(l.ResolveNamespaceDir(name))
	if err != nil {
		return parser.Namespace{}, err
	}

	opts := parser.NewParserOptions{
		Namespace:     name,
		Units:         units,
		EnableLogging: slices.Contains(l.loggers, "parser"),
	}
	return parser.New(opts).Parse()
}

func (l *lkr) CheckDependency(namespace parser.Namespace, dep string) error {
	if slices.Contains(l.processing, dep) {
		return fmt.Errorf("dependency cycle detected: %s depends on %s", namespace.Name, dep)
	}
	return nil
}

func (l *lkr) ResolveNamespaceName(name string) string {
	if name == RootNamespace {
		return filepath.Base(l.source)
	}
	return name
}

func (l *lkr) ResolveNamespaceDir(name string) string {
	dir := filepath.Join(l.source, name)
	if dir == filepath.Join(l.source, filepath.Base(l.source)) {
		dir = l.source
	}
	return dir
}

func (l *lkr) ProcessNamespace(namespace parser.Namespace) error {
	l.processing = append(l.processing, namespace.Name)
	l.namespaces[namespace.Name] = namespace

	for _, dep := range namespace.Dependencies {
		if err := l.CheckDependency(namespace, dep); err != nil {
			return err
		}
		ns, err := l.GetNamespace(dep)
		if err != nil {
			return err
		}
		if err := l.ProcessNamespace(ns); err != nil {
			return err
		}
	}

	l.processing = slices.DeleteFunc(l.processing, func(s string) bool {
		return s == namespace.Name
	})

	l.order = append(l.order, namespace.Name)
	return nil
}

func (l *lkr) Link() error {
	namespace, err := l.GetNamespace(l.ResolveNamespaceName(RootNamespace))
	if err != nil {
		return err
	}
	if err := l.ProcessNamespace(namespace); err != nil {
		return err
	}
	return nil
}

func (l *lkr) Resolve() ([]emitter.Instruction, error) {
	// We need to call Link to ensure that all dependencies between namespaces are properly resolved
	// before emitting the instructions. Link performs the analysis and processing of dependencies,
	// detects cycles, and prepares the correct order of namespaces to be emitted. Without this call,
	// we could generate incomplete or inconsistent instructions due to unresolved namespaces or dependencies.
	if err := l.Link(); err != nil {
		return nil, err
	}

	insts := make([]emitter.Instruction, 0)

	for _, order := range l.order {
		namespace := l.namespaces[order]
		nsinsts, err := emitter.New(emitter.NewEmitterOptions{
			EnableLogging: slices.Contains(l.loggers, "emitter"),
		}).Emit(namespace)
		if err != nil {
			return nil, err
		}
		insts = append(insts, nsinsts...)
	}

	return insts, nil
}
