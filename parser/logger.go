package parser

import (
	"encoding/json"
	"fmt"
	"reflect"
	"regexp"

	"github.com/fatih/color"
)

type NodeLogging struct {
	Type string `json:"type"`
	Data any    `json:"data"`
}

type Logger struct {
	enableLogging bool
}

func NewLogger(enableLogging bool) *Logger {
	return &Logger{enableLogging: enableLogging}
}

var (
	colorizeKey    = color.New(color.FgHiCyan).SprintFunc()
	colorizeString = color.New(color.FgHiYellow).SprintFunc()
	colorizeValue  = color.New(color.FgHiMagenta).SprintFunc()
)

func colorizeJSON(s string) string {
	keyRe := regexp.MustCompile(`"([^"]+)"\s*:`)
	s = keyRe.ReplaceAllString(s, colorizeKey(`"$1"`)+":")

	// "string"
	strRe := regexp.MustCompile(`:\s*"([^"]*)"`)
	s = strRe.ReplaceAllString(s, ": "+colorizeString(`"$1"`))

	// números, bool, null
	valRe := regexp.MustCompile(`:\s*(\d+|true|false|null)`)
	s = valRe.ReplaceAllString(s, ": "+colorizeValue(`$1`))

	return s
}

func WrapNodeLogging(n Node) any {
	if n == nil {
		return nil
	}

	v := reflect.ValueOf(n)
	t := reflect.TypeOf(n)

	// unwrap ponteiro
	if t.Kind() == reflect.Pointer {
		t = t.Elem()
		v = v.Elem()
		if !v.IsValid() {
			return nil
		}
	}

	// percorre campos e reembrulha Nodes internos
	m := make(map[string]interface{})
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		value := v.Field(i).Interface()

		switch v := value.(type) {
		case Node:
			m[field.Tag.Get("json")] = WrapNodeLogging(v)

		case []Node:
			arr := make([]interface{}, 0, len(v))
			for _, n := range v {
				arr = append(arr, WrapNodeLogging(n))
			}
			m[field.Tag.Get("json")] = arr

		default:
			if tag := field.Tag.Get("json"); tag != "-" && tag != "" {
				m[tag] = value
			}
		}
	}

	return NodeLogging{
		Type: t.Name(),
		Data: m,
	}
}

func (l *Logger) JSON(m Module) (int, error) {
	if l.enableLogging {
		bs, err := json.MarshalIndent(WrapNodeLogging(m), "", "  ")
		if err != nil {
			return 0, err
		}
		return fmt.Println(colorizeJSON(string(bs)))
	}
	return 0, nil
}
