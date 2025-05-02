package main

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"syscall"

	"github.com/goccy/go-yaml"
	"golang.org/x/mod/semver"
)

type version struct {
	Folder string `yaml:"folder"`
}

type config struct {
	Versions map[string]version `yaml:"versions"`
}

type require struct {
	PkgName string `yaml:"name"`
	Cond    bool   `yaml:"cond,omitempty"`
}

type graph struct {
	Requires map[string][]require `yaml:"requires"`
}

// graphcppkg /7bitconf/config.yml
func main() {
	if len(os.Args) < 2 || !strings.HasPrefix(os.Args[1], "/") {
		fmt.Fprintln(os.Stderr, "Please provide the path to the conan config.yml file")
		os.Exit(1)
	}

	graphFile := cppkgRoot() + "graph.yml"
	file, err := os.OpenFile(graphFile, os.O_CREATE|os.O_RDWR, 0666)
	check(err)
	defer file.Close()

	err = syscall.Flock(int(file.Fd()), syscall.LOCK_EX|syscall.LOCK_NB)
	if err != nil {
		log.Println("[INFO] another instance is running")
		os.Exit(1)
	}
	defer syscall.Flock(int(file.Fd()), syscall.LOCK_UN)

	b, err := io.ReadAll(file)
	check(err)

	var g graph
	if len(b) == 0 {
		g.Requires = make(map[string][]require)
	} else {
		err = yaml.Unmarshal(b, &g)
		check(err)
	}

	pkgName := strings.TrimSuffix(os.Args[1][1:], "/config.yml")
	if _, ok := g.Requires[pkgName]; ok {
		log.Panicln("[ERROR] package already exists in the graph:", pkgName)
	}

	localDir := conanRoot() + "recipes/" + pkgName + "/"

	confFile := localDir + "config.yml"
	b, err = os.ReadFile(confFile)
	check(err)

	var conf config
	err = yaml.Unmarshal(b, &conf)
	check(err)

	latestVerDo(conf.Versions, func(ver string, v version) {
		conanFile := localDir + v.Folder + "/conanfile.py"
		f, err := os.Open(conanFile)
		check(err)
		defer f.Close()

		var fnIndent int
		var fn string
		var reqs []require
		s := bufio.NewScanner(f)
		for s.Scan() {
			line := s.Text()
			code := strings.TrimLeft(line, " \t")
			if strings.HasPrefix(code, "def ") {
				fn = strings.TrimSuffix(strings.TrimSpace(code[4:]), ":")
				fnIndent = indentOf(code, line)
			} else if fn == "requirements(self)" {
				if req, ok := checkRequire(code, line, fnIndent); ok {
					reqs = append(reqs, req)
				}
			}
		}
		g.Requires[pkgName] = reqs
		b, err := yaml.Marshal(g)
		check(err)

		_, err = file.Seek(0, io.SeekStart)
		check(err)

		_, err = file.Write(b)
		check(err)

		os.Stderr.Write(b)
	})
}

// self.requires("expat/[>=2.6.2 <3]")
func checkRequire(code, line string, fnIndent int) (_ require, ok bool) {
	const requirePrefix = `self.requires("`
	if strings.HasPrefix(code, requirePrefix) {
		left := code[len(requirePrefix):]
		if pos := strings.IndexAny(left, `/"`); pos >= 0 {
			name := left[:pos]
			cond := indentOf(code, line) > (fnIndent << 1)
			return require{name, cond}, true
		}
	}
	return
}

const (
	tabSpaces = 4
)

func indentOf(code, line string) int {
	indent := len(line) - len(code)
	return indent + strings.Count(line[:indent], "\t")*(tabSpaces-1)
}

func latestVerDo[V any](data map[string]V, yield func(string, V)) {
	keys := make([]string, 0, len(data))
	for k := range data {
		keys = append(keys, "v"+k)
	}
	semver.Sort(keys)
	k := keys[len(keys)-1][1:] // remove 'v'
	yield(k, data[k])
}

func conanRoot() string {
	home, _ := os.UserHomeDir()
	return home + "/conan-center-index/"
}

func cppkgRoot() string {
	home, _ := os.UserHomeDir()
	return home + "/cppkg/"
}

func check(err error) {
	if err != nil {
		panic(err)
	}
}
