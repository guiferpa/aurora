package evaluator

import (
	"bufio"
	"io"
)

type Player struct {
	scanner *bufio.Scanner
}

func NewPlayer(reader io.Reader) *Player {
	scanner := bufio.NewScanner(reader)
	return &Player{scanner}
}
