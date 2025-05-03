package main

import (
	"errors"
	"os"

	"github.com/goccy/go-yaml"
	"golang.org/x/mod/semver"
)

var gitCmd = NewCommand("git", []string{
	"brew install git",
	"apt-get install git",
})

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
	os.MkdirAll(cacheDir, os.ModePerm)
	ret = &Manager{
		cacheDir: cacheDir,
	}
	return
}

type version struct {
	Folder string `yaml:"folder"`
}

type Template struct {
	FromVer string `yaml:"from"`
	Folder  string `yaml:"folder"`
	URL     string `yaml:"url"`
}

type config struct {
	PkgName  string             `yaml:"name"`
	Versions map[string]version `yaml:"versions"`
	Template Template           `yaml:"template"`
}

type Package struct {
	Name     string
	Path     string
	Version  string
	Folder   string
	Template *Template
}

var (
	ErrVersionNotFound = errors.New("version not found")
)

const (
	IndexAutoUpdate = 1 << iota
	ToolQuietInstall
	LogRevertProxy
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
		return &Package{conf.PkgName, pkgPath, ver, v.Folder, nil}, nil
	}
	if compareVer(ver, conf.Template.FromVer) < 0 {
		err = ErrVersionNotFound
		return
	}
	folder := conf.Template.Folder
	return &Package{conf.PkgName, pkgPath, ver, folder, &conf.Template}, nil
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
