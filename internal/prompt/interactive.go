package prompt

import (
	"os"

	"github.com/mattn/go-isatty"
)

var isInteractive = isTerminal(os.Stdin) && isTerminal(os.Stdout)

func isTerminal(f *os.File) bool {
	return isatty.IsTerminal(f.Fd()) || isatty.IsCygwinTerminal(f.Fd())
}
