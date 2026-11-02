//go:build !windows

package main

import (
	"fmt"
	"os"
)

func main() {
	fmt.Fprintln(os.Stderr, "Cursor fast proxy собран для Windows 10 и 11. На этой системе окно не запускается.")
	os.Exit(1)
}
