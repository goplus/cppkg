package main

import (
	"fmt"
	"os"

	"github.com/goplus/llgo/xtool/cppkg"
)

// cppkg 7bitcoder/7bitconf@1.2.0
func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "Usage: cppkg <package>[@<version>]")
		return
	}

	cppkg.Install(os.Args[1], cppkg.DefaultFlags)
}
