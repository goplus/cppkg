package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"slices"
	"strings"

	"github.com/goccy/go-yaml"
	"golang.org/x/mod/semver"
)

type version struct {
	Folder string `yaml:"folder"`
}

type config struct {
	Versions map[string]version `yaml:"versions"`
}

type template struct {
	Folder string `yaml:"folder"`
	URL    string `yaml:"url"`
}

type configEx struct {
	Versions map[string]version `yaml:"versions"`
	Template template           `yaml:"template"`
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

	tryCp := func(src map[string]any, ver string, v version) {
		switch url := src["url"].(type) {
		case string:
			if pkgPath, urlPattern, ok := checkGithbPkg(url, ver); ok {
				cpGithubPkg(pkgPath, urlPattern, localDir, conf, v)
			}
		case []any:
			for _, u := range url {
				url := u.(string)
				if pkgPath, urlPattern, ok := checkGithbPkg(url, ver); ok {
					cpGithubPkg(pkgPath, urlPattern, localDir, conf, v)
				}
			}
		default:
			log.Println("[INFO] skip source:", src)
		}
	}

	conandatas := make(map[string]conandata) // folder -> conandata
	rangeVerDesc(conf.Versions, func(ver string, v version) {
		cd, err := getConanData(conandatas, v.Folder, localDir)
		if err != nil {
			if os.IsNotExist(err) {
				return
			}
			check(err)
		}

		if src, ok := cd.Sources[ver]; ok {
			switch src := src.(type) {
			case map[string]any:
				tryCp(src, ver, v)
			case []any:
				for _, u := range src {
					tryCp(u.(map[string]any), ver, v)
				}
			default:
				log.Panicln("[FATAL] source:", src)
			}
		}
	})
}

func cpGithubPkg(pkgPath, urlPattern, srcDir string, conf config, v version) {
	destDir := cppkgRoot() + pkgPath
	os.MkdirAll(destDir, os.ModePerm)

	err := exec.Command("cp", "-r", srcDir, destDir).Run()
	check(err)

	confex := &configEx{
		Versions: conf.Versions,
		Template: template{
			Folder: v.Folder,
			URL:    urlPattern,
		},
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
	const pattern = "${version}"
	if strings.HasPrefix(url, githubPrefix) {
		path := url[len(githubPrefix):]
		if pos := strings.Index(path, ver); pos >= 0 {
			parts := strings.SplitN(path, "/", 3)
			if len(parts) == 3 {
				at := len(githubPrefix) + pos
				ending := url[at+len(ver):]
				urlPattern = url[:at] + pattern + strings.ReplaceAll(ending, ver, pattern)
				pkgPath, ok = strings.ToLower(parts[0]+"/"+parts[1]), true
			}
		}
	}
	return
}

type conandata struct {
	Sources map[string]any `yaml:"sources"`
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

func rangeVerDesc[V any](data map[string]V, f func(string, V)) {
	keys := make([]string, 0, len(data))
	for k := range data {
		keys = append(keys, "v"+k)
	}
	semver.Sort(keys)
	for _, k := range slices.Backward(keys) {
		k = k[1:] // remove 'v'
		f(k, data[k])
	}
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
