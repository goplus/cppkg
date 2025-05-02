package main

import (
	"errors"
	"os"
	"os/exec"
	"strings"

	"github.com/goccy/go-yaml"
)

var conanCmd = NewCommand("conan", []string{
	"brew install conan",
	"apt-get install conan",
})

type conandata struct {
	Sources map[string]any `yaml:"sources"`
}

func replaceVer(src any, fromVer, toVer, hash string) any {
	switch src := src.(type) {
	case map[string]any:
		doReplace(src, fromVer, toVer, hash)
	case []any:
		for _, u := range src {
			doReplace(u.(map[string]any), fromVer, toVer, hash)
		}
	}
	return src
}

func doReplace(src map[string]any, fromVer, toVer, _ string) {
	switch url := src["url"].(type) {
	case string:
		src["url"] = strings.ReplaceAll(url, fromVer, toVer)
		delete(src, "sha256")
		// TODO(xsw): src["sha256"] = hash
	case []any:
		for i, u := range url {
			url[i] = strings.ReplaceAll(u.(string), fromVer, toVer)
		}
		delete(src, "sha256")
		// TODO(xsw): src["sha256"] = hash
	}
}

func (p *Manager) Install(pkg *Package, flags int) (err error) {
	outDir := p.outDir(pkg)
	conanfileDir := p.conanfileDir(pkg.Path, pkg.Folder)
	pkgVer := pkg.Version
	template := pkg.Template
	if template != nil {
		err = copyDirR(conanfileDir, outDir)
		if err != nil {
			return
		}
		conandataFile := outDir + "/conandata.yml"
		b, e := os.ReadFile(conandataFile)
		if e != nil {
			return e
		}
		var cd conandata
		err = yaml.Unmarshal(b, &cd)
		if err != nil {
			return
		}
		fromVer := template.FromVer
		source, ok := cd.Sources[fromVer]
		if !ok {
			return ErrVersionNotFound
		}
		hash := "" // TODO(xsw): hash
		cd.Sources = map[string]any{
			pkgVer: replaceVer(source, fromVer, pkgVer, hash),
		}
		b, err = yaml.Marshal(cd)
		if err != nil {
			return
		}
		err = os.WriteFile(conandataFile, b, os.ModePerm)
		if err != nil {
			return
		}
		conanfileDir = outDir
	}
	return conanInstall(pkgVer, flags, outDir, conanfileDir)
}

func (p *Manager) outDir(pkg *Package) string {
	return p.cacheDir + "/build/" + pkg.Name + "@" + pkg.Version
}

func (p *Manager) conanfileDir(pkgPath, pkgFolder string) string {
	root := p.indexRoot()
	return root + "/" + pkgPath + "/" + pkgFolder
}

func conanInstall(pkgVer string, flags int, outDir, conanfileDir string) (err error) {
	args := make([]string, 0, 10)
	args = append(args, "install",
		"--build", "missing",
		// "--format", "json",
		"--version", pkgVer,
		"--output-folder", outDir,
		"./conanfile.py",
	)
	quietInstall := flags&ToolQuietInstall != 0
	cmd, err := conanCmd.New(quietInstall, args...)
	if err != nil {
		return
	}
	cmd.Dir = conanfileDir
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
	err = cmd.Run()
	return
}

func copyDirR(srcDir, destDir string) error {
	if cp, err := exec.LookPath("cp"); err == nil {
		return exec.Command(cp, "-r", "-p", srcDir, destDir).Run()
	}
	if cp, err := exec.LookPath("xcopy"); err == nil {
		return exec.Command(cp, "/E", "/I", "/Y", srcDir, destDir).Run()
	}
	return errors.New("copy command not found")
}
