package main

import (
	"fmt"
	"os"
	"strings"
)

// cppkg 7bitcoder/7bitconf@1.2.0
func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "Usage: cppkg <package>[@<version>]")
		return
	}

	pkgPath, ver := parsePkgVer(os.Args[1])
	if ver == "" {
		panic("TODO: get latest version")
	}

	m, err := New("")
	check(err)

	const flags = IndexAutoUpdate | ToolQuietInstall
	pkg, err := m.Lookup(pkgPath, ver, flags)
	check(err)

	err = m.Install(pkg, flags)
	check(err)
}

func parsePkgVer(pkg string) (string, string) {
	parts := strings.SplitN(pkg, "@", 2)
	if len(parts) == 1 {
		return parts[0], ""
	}
	return parts[0], parts[1]
}

func check(err error) {
	if err != nil {
		panic(err)
	}
}
