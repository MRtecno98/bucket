package cli

import (
	"fmt"
	"io"
	"os"

	"golang.org/x/term"
)

const SEQUENCE = '\x1b'
const CTRLC = '\x03'
const ENTER = -13

const (
	UP = iota + 1
	DOWN
	RIGHT
	LEFT
)

func ReadKeycode(in io.Reader) (int, error) {
	b := make([]byte, 2)
	_, err := in.Read(b)
	if err != nil {
		return 0, err
	}

	switch b[1] {
	case 'A':
		return UP, nil
	case 'B':
		return DOWN, nil
	case 'C':
		return RIGHT, nil
	case 'D':
		return LEFT, nil
	case '1':
		in.Read([]byte{0})
		return ReadKeycode(in)
	default:
		return 0, nil
	}
}

func ReadSequence(in io.Reader) (int, error) {
	b := make([]byte, 1)
	_, err := in.Read(b)
	if err != nil {
		return 0, err
	}

	if b[0] == CTRLC {
		return 0, fmt.Errorf("ctrl-c: terminated")
	}

	if b[0] == byte(-ENTER) {
		return ENTER, nil
	}

	if b[0] != SEQUENCE {
		return 0, nil
	}

	return ReadKeycode(in)
}

type Table struct {
	Options  []string
	Selected int

	Writer io.Writer
}

func TableSelect(options []string, out io.Writer) (int, error) {
	// TODO: Better way to handle this, without env vars
	if _, b := os.LookupEnv("bucket.plain"); b {
		return TableSelectPlain(options, out)
	}

	table := &Table{
		Options: options,
		Writer:  out,
	}

	oldState, err := term.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		return -1, err
	}

	defer term.Restore(int(os.Stdin.Fd()), oldState)

	for {
		n, err := func() (int, error) {
			table.Render(true, false)
			defer table.Rollback()

			k, err := ReadSequence(os.Stdin)
			if err != nil {
				return -1, err
			}

			switch k {
			case UP:
				table.MoveUp()
			case DOWN:
				table.MoveDown()
			case ENTER:
				return table.Selected, nil
			}

			return -1, nil
		}()

		if err != nil {
			return -1, err
		} else if n != -1 {
			return n, nil
		}
	}
}

func TableSelectPlain(options []string, out io.Writer) (int, error) {
	table := &Table{
		Options: options,
		Writer:  out,
	}

	table.Render(false, true)

	for {
		fmt.Fprint(out, "\n> ")

		var n int
		_, err := fmt.Fscanf(os.Stdin, "%d", &n)
		if err != nil {
			return -1, err
		}

		n = n - 1

		if n >= 0 && n < len(options) {
			fmt.Fprintln(out)
			return n, nil
		}
	}
}

func (t *Table) MoveUp() {
	if t.Selected > 0 {
		t.Selected--
	}
}

func (t *Table) MoveDown() {
	if t.Selected < len(t.Options)-1 {
		t.Selected++
	}
}

func (t *Table) Render(cursor bool, indexes bool) {
	for i, v := range t.Options {
		if indexes {
			fmt.Fprintf(t.Writer, "[%d] ", i+1)
		}

		if i == t.Selected && cursor {
			fmt.Fprintf(t.Writer, "\033[1m> %s\033[0m\n", v)
		} else {
			fmt.Fprintln(t.Writer, v)
		}
	}
}

func (t *Table) Rollback() {
	for i := 0; i < len(t.Options); i++ {
		fmt.Fprint(os.Stdout, "\033[1A\r\033[K")
	}
}
