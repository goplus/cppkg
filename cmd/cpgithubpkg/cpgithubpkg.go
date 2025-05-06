package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/goplus/llgo/xtool/cpgithubpkg"
)

// cpgithubpkg /7bitconf/config.yml
func main() {
	if len(os.Args) < 2 || !strings.HasPrefix(os.Args[1], "/") {
		fmt.Fprintln(os.Stderr, "Please provide the path to the conan config.yml file")
		os.Exit(1)
	}

	pkgName := strings.TrimSuffix(os.Args[1][1:], "/config.yml")
	cpgithubpkg.Main(pkgName)
}
