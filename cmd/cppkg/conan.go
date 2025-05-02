package main

import (
	"os"
)

var conanCmd = NewCommand("conan", []string{
	"brew install conan",
	"apt-get install conan",
})

func (p *Package) isTemplate() bool {
	return p.URL != ""
}

func (p *Manager) Install(pkg *Package, flags int) (err error) {
	if pkg.isTemplate() {
		panic("TODO: install by template")
	}
	conanfileDir := p.conanfileDir(pkg.Path, pkg.Folder)
	return conanInstall(pkg, flags, p.outDir(pkg), conanfileDir)
}

func (p *Manager) outDir(pkg *Package) string {
	return p.cacheDir + "/build/" + pkg.Name + "@" + pkg.Version
}

func (p *Manager) conanfileDir(pkgPath, pkgFolder string) string {
	root := p.indexRoot()
	return root + "/" + pkgPath + "/" + pkgFolder
}

func conanInstall(pkg *Package, flags int, outDir, conanfileDir string) (err error) {
	args := make([]string, 0, 10)
	args = append(args, "install",
		"--build", "missing",
		// "--format", "json",
		"--version", pkg.Version,
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
