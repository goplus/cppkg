package main

import (
	"crypto/md5"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/goccy/go-yaml"
	"github.com/qiniu/x/httputil"
)

var conanCmd = NewCommand("conan", []string{
	"brew install conan",
	"apt-get install conan",
})

type conandata struct {
	Sources map[string]any `yaml:"sources"`
}

func replaceVer(src any, fromVer, toVer string) any {
	switch src := src.(type) {
	case map[string]any:
		doReplace(src, fromVer, toVer)
	case []any:
		for _, u := range src {
			doReplace(u.(map[string]any), fromVer, toVer)
		}
	}
	return src
}

func doReplace(src map[string]any, fromVer, toVer string) {
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
	os.MkdirAll(outDir, os.ModePerm)

	var rev string
	var gr *githubRelease
	var conandataYml, conanfilePy []byte

	conanfileDir := p.conanfileDir(pkg.Path, pkg.Folder)
	pkgVer := pkg.Version
	template := pkg.Template
	if template != nil {
		gr, err = githubReleaseGet(pkg.Path, "v"+pkg.Version)
		if err != nil {
			return
		}

		err = copyDirR(conanfileDir, outDir)
		if err != nil {
			return
		}

		conanfilePy, err = os.ReadFile(outDir + "/conanfile.py")
		if err != nil {
			return
		}

		conandataFile := outDir + "/conandata.yml"
		conandataYml, err = os.ReadFile(conandataFile)
		if err != nil {
			return
		}
		var cd conandata
		err = yaml.Unmarshal(conandataYml, &cd)
		if err != nil {
			return
		}
		fromVer := template.FromVer
		source, ok := cd.Sources[fromVer]
		if !ok {
			return ErrVersionNotFound
		}
		cd.Sources = map[string]any{
			pkgVer: replaceVer(source, fromVer, pkgVer),
		}
		conandataYml, err = yaml.Marshal(cd)
		if err != nil {
			return
		}
		err = os.WriteFile(conandataFile, conandataYml, os.ModePerm)
		if err != nil {
			return
		}
		rev = recipeRevision(pkg, gr, conandataYml)
		conanfileDir = outDir
	}

	outFile := outDir + "/out.json"
	out, err := os.Create(outFile)
	if err == nil {
		defer out.Close()
	} else {
		out = os.Stdout
	}

	nameAndVer := pkg.Name + "/" + pkgVer
	if template == nil {
		return conanInstall(nameAndVer, outDir, conanfileDir, out, flags)
	}

	mtime, err := unixTime(gr.PublishedAt)
	if err != nil {
		return
	}

	cmd := exec.Command("tar", "-czf", "conan_export.tgz", "conandata.yml")
	cmd.Dir = outDir
	err = cmd.Run()
	if err != nil {
		return
	}
	conanExportTgz, err := os.ReadFile(outDir + "/conan_export.tgz")
	if err != nil {
		return
	}

	logFile := outDir + "/rp.log"
	return remoteProxy(flags, logFile, func() error {
		return conanInstall(nameAndVer, outDir, conanfileDir, out, flags)
	}, func(mux *http.ServeMux) {
		base := "/v2/conans/" + nameAndVer
		revbase := base + "/_/_/revisions/" + rev
		mux.HandleFunc(base+"/_/_/latest", func(w http.ResponseWriter, r *http.Request) {
			h := w.Header()
			h.Set("Cache-Control", "public,max-age=300")
			httputil.Reply(w, http.StatusOK, map[string]any{
				"revision": rev,
				"time":     gr.PublishedAt,
			})
		})
		mux.HandleFunc(revbase+"/files", func(w http.ResponseWriter, r *http.Request) {
			h := w.Header()
			h.Set("Cache-Control", "public,max-age=3600")
			empty := map[string]any{}
			httputil.Reply(w, http.StatusOK, map[string]any{
				"files": map[string]any{
					"conan_export.tgz":  empty,
					"conanmanifest.txt": empty,
					"conanfile.py":      empty,
				},
			})
		})
		mux.HandleFunc(revbase+"/files/conanfile.py", func(w http.ResponseWriter, r *http.Request) {
			h := w.Header()
			h.Set("Cache-Control", "public,max-age=3600")
			h.Set("Content-Disposition", `attachment; filename="conanfile.py"`)
			httputil.ReplyWith(w, http.StatusOK, "text/x-python", conanfilePy)
		})
		const conanmanifest = "%d\nconandata.yml: %s\nconanfile.py: %s\n"
		mux.HandleFunc(revbase+"/files/conanmanifest.txt", func(w http.ResponseWriter, r *http.Request) {
			h := w.Header()
			h.Set("Cache-Control", "public,max-age=3600")
			h.Set("Content-Disposition", `attachment; filename="conanmanifest.txt"`)
			data := fmt.Sprintf(conanmanifest, mtime, md5Of(conandataYml), md5Of(conanfilePy))
			httputil.ReplyWithStream(w, http.StatusOK, "text/plain", strings.NewReader(data), int64(len(data)))
		})
		mux.HandleFunc(revbase+"/files/conan_export.tgz", func(w http.ResponseWriter, r *http.Request) {
			h := w.Header()
			h.Set("Cache-Control", "public,max-age=3600")
			h.Set("Content-Disposition", `attachment; filename="conan_export.tgz"`)
			httputil.ReplyWith(w, http.StatusOK, "application/x-gzip", conanExportTgz)
		})
	})
}

func (p *Manager) outDir(pkg *Package) string {
	return p.cacheDir + "/build/" + pkg.Name + "@" + pkg.Version
}

func (p *Manager) conanfileDir(pkgPath, pkgFolder string) string {
	root := p.indexRoot()
	return root + "/" + pkgPath + "/" + pkgFolder
}

func conanInstall(pkg, outDir, conanfileDir string, out io.Writer, flags int) (err error) {
	args := make([]string, 0, 12)
	args = append(args, "install",
		"--requires", pkg,
		"--generator", "PkgConfigDeps",
		"--build", "missing",
		"--format", "json",
		"--output-folder", outDir,
	)
	quietInstall := flags&ToolQuietInstall != 0
	cmd, err := conanCmd.New(quietInstall, args...)
	if err != nil {
		return
	}
	cmd.Dir = conanfileDir
	cmd.Stderr = os.Stderr
	cmd.Stdout = out
	err = cmd.Run()
	return
}

func recipeRevision(_ *Package, _ *githubRelease, conandataYml []byte) string {
	return md5Of(conandataYml)
}

func md5Of(data []byte) string {
	h := md5.New()
	h.Write(data)
	return hex.EncodeToString(h.Sum(nil))
}

func unixTime(tstr string) (ret int64, err error) {
	t, err := time.Parse(time.RFC3339, tstr)
	if err == nil {
		ret = t.Unix()
	}
	return
}

func copyDirR(srcDir, destDir string) error {
	if cp, err := exec.LookPath("cp"); err == nil {
		return exec.Command(cp, "-r", "-p", srcDir+"/", destDir).Run()
	}
	if cp, err := exec.LookPath("xcopy"); err == nil {
		// TODO(xsw): check xcopy
		return exec.Command(cp, "/E", "/I", "/Y", srcDir+"/", destDir).Run()
	}
	return errors.New("copy command not found")
}
