package main

import (
	"errors"
	"os"
	"strings"

	"github.com/goccy/go-yaml"
	"golang.org/x/mod/semver"
)

var (
	gitCmd = NewCommand("git", []string{
		"brew install git",
		"apt-get install git",
	})

	conanCmd = NewCommand("conan", []string{
		"brew install conan",
		"apt-get install conan",
	})
)

// Manager represents a package manager for C/C++ packages.
type Manager struct {
	cacheDir string
}

func New(cacheDir string) (ret *Manager, err error) {
	if cacheDir == "" {
		cacheDir, err = os.UserCacheDir()
		if err != nil {
			return
		}
		cacheDir += "/cppkg"
	}
	os.MkdirAll(cacheDir, 0755)
	ret = &Manager{
		cacheDir: cacheDir,
	}
	return
}

type version struct {
	Folder string `yaml:"folder"`
}

type template struct {
	FromVer string `yaml:"from"`
	Folder  string `yaml:"folder"`
	URL     string `yaml:"url"`
}

type config struct {
	Versions map[string]version `yaml:"versions"`
	Template template           `yaml:"template"`
}

type Package struct {
	Path    string
	Version string
	Folder  string
	URL     string
}

func (p *Package) isTemplate() bool {
	return p.URL != ""
}

var (
	ErrVersionNotFound = errors.New("version not found")
)

const (
	IndexAutoUpdate = 1 << iota
	ToolQuietInstall
)

func (p *Manager) Lookup(pkgPath, ver string, flags int) (_ *Package, err error) {
	root := p.indexRoot()
	err = indexUpate(root, flags)
	if err != nil {
		return
	}
	pkgDir := root + "/" + pkgPath
	confFile := pkgDir + "/config.yml"
	b, err := os.ReadFile(confFile)
	if err != nil {
		return
	}
	var conf config
	err = yaml.Unmarshal(b, &conf)
	if err != nil {
		return
	}
	if v, ok := conf.Versions[ver]; ok {
		return &Package{Path: pkgPath, Version: ver, Folder: v.Folder}, nil
	}
	if compareVer(ver, conf.Template.FromVer) < 0 {
		err = ErrVersionNotFound
		return
	}
	folder := conf.Template.Folder
	url := strings.ReplaceAll(conf.Template.URL, "${version}", ver)
	return &Package{Path: pkgPath, Version: ver, Folder: folder, URL: url}, nil
}

func (p *Manager) Install(pkg *Package, flags int) (err error) {
	if pkg.isTemplate() {
		panic("TODO: install by template")
	}
	quietInstall := flags&ToolQuietInstall != 0
	_, err = conanCmd.New(quietInstall)
	if err != nil {
		return
	}
	panic("TODO: install")
}

func (p *Manager) indexRoot() string {
	return p.cacheDir + "/index"
}

func indexUpate(root string, flags int) (err error) {
	if _, err = os.Stat(root + "/.git"); os.IsNotExist(err) {
		os.RemoveAll(root)
		return indexInit(root, flags)
	}
	if flags&IndexAutoUpdate != 0 {
		quietInstall := flags&ToolQuietInstall != 0
		git, e := gitCmd.New(quietInstall, "pull", "--ff-only", "origin", "main")
		if e != nil {
			return e
		}
		git.Dir = root
		git.Stdout = os.Stdout
		git.Stderr = os.Stderr
		err = git.Run()
	}
	return
}

func indexInit(root string, flags int) (err error) {
	quietInstall := flags&ToolQuietInstall != 0
	git, err := gitCmd.New(quietInstall, "clone", "https://github.com/goplus/cppkg.git", root)
	if err != nil {
		return
	}
	git.Stdout = os.Stdout
	git.Stderr = os.Stderr
	err = git.Run()
	return
}

func compareVer(v1, v2 string) int {
	return semver.Compare("v"+v1, "v"+v2)
}
