package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"

	"github.com/goccy/go-yaml"
)

type version struct {
	Folder string `yaml:"folder"`
}

type config struct {
	Versions map[string]version `yaml:"versions"`
}

type configEx struct {
	Versions  map[string]version `yaml:"versions"`
	SourceURL string             `yaml:"url"`
}

type conandata struct {
	Sources map[string]map[string]any `yaml:"sources"`
}

func getConanData(conandatas map[string]conandata, folder, localDir string) (ret conandata, err error) {
	if v, ok := conandatas[folder]; ok {
		return v, nil
	}
	file := localDir + folder + "/conandata.yml"
	b, err := os.ReadFile(file)
	if err != nil {
		return
	}
	if err = yaml.Unmarshal(b, &ret); err != nil {
		return
	}
	conandatas[folder] = ret
	return
}

// cpgithubpkg /7bitconf/config.yml
func main() {
	if len(os.Args) < 2 || !strings.HasPrefix(os.Args[1], "/") {
		fmt.Fprintln(os.Stderr, "Please provide the path to the conan config.yml file")
		os.Exit(1)
	}

	pkgName := strings.TrimSuffix(os.Args[1][1:], "/config.yml")
	localDir := conanRoot() + "recipes/" + pkgName + "/"

	confFile := localDir + "config.yml"
	b, err := os.ReadFile(confFile)
	check(err)

	var conf config
	err = yaml.Unmarshal(b, &conf)
	check(err)

	conandatas := make(map[string]conandata) // folder -> conandata
	for ver, v := range conf.Versions {
		cd, err := getConanData(conandatas, v.Folder, localDir)
		check(err)

		if src, ok := cd.Sources[ver]; ok {
			switch url := src["url"].(type) {
			case string:
				if pkgPath, urlPattern, ok := checkGithbPkg(url, ver); ok {
					cpGithubPkg(pkgPath, urlPattern, localDir, conf)
				}
			case []any:
				for _, u := range url {
					if url, ok := u.(string); ok {
						if pkgPath, urlPattern, ok := checkGithbPkg(url, ver); ok {
							cpGithubPkg(pkgPath, urlPattern, localDir, conf)
						}
					}
				}
			default:
				log.Println("[INFO] skip source:", src)
			}
		}
	}
}

func cpGithubPkg(pkgPath, urlPattern, srcDir string, conf config) {
	destDir := cppkgRoot() + pkgPath
	os.MkdirAll(destDir, os.ModePerm)

	err := exec.Command("cp", "-r", srcDir, destDir).Run()
	check(err)

	confex := &configEx{
		Versions:  conf.Versions,
		SourceURL: urlPattern,
	}
	b, err := yaml.Marshal(confex)
	check(err)

	err = os.WriteFile(destDir+"/config.yml", b, os.ModePerm)
	check(err)

	log.Println("[INFO] copy", pkgPath)
	os.Exit(0)
}

func checkGithbPkg(url, ver string) (pkgPath, urlPattern string, ok bool) {
	const githubPrefix = "https://github.com/"
	if strings.HasPrefix(url, githubPrefix) {
		path := url[len(githubPrefix):]
		if pos := strings.Index(path, ver); pos >= 0 {
			parts := strings.SplitN(path, "/", 3)
			if len(parts) == 3 {
				at := len(githubPrefix) + pos
				ending := url[at+len(ver):]
				if !strings.Contains(ending, ver) {
					urlPattern = url[:at] + "%s" + ending
					pkgPath, ok = strings.ToLower(parts[0]+"/"+parts[1]), true
				}
			}
		}
	}
	return
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
