package github

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

var (
	ErrBreak    = errors.New("break")
	ErrNotFound = errors.New("not found")
)

// Commit represents a commit in a GitHub repository.
type Commit struct {
	SHA string `json:"sha"`
	URL string `json:"url"`
}

// Tag represents a GitHub tag.
type Tag struct {
	Name       string `json:"name"`
	ZipballURL string `json:"zipball_url"`
	TarballURL string `json:"tarball_url"`
	Commit     Commit `json:"commit"`
	NodeID     string `json:"node_id"`
}

// tagsURL constructs the URL for fetching tags from a GitHub repository.
func tagsURL(pkgPath string) string {
	return "https://api.github.com/repos/" + pkgPath + "/tags"
}

// GetTag retrieves a specific tag from a GitHub repository.
func GetTag(pkgPath, ver string) (tag *Tag, err error) {
	err = ErrNotFound
	EnumTags(pkgPath, 0, func(tags []*Tag, page, total int) error {
		for _, t := range tags {
			if t.Name == ver {
				tag = t
				err = nil
				return ErrBreak
			}
		}
		return nil
	})
	return
}

// EnumTags enumerates the tags of a GitHub repository.
func EnumTags(pkgPath string, page int, pager func(tags []*Tag, page, total int) error) (err error) {
	total := 0
	ubase := tagsURL(pkgPath)

loop:
	u := ubase
	if page > 0 {
		vals := url.Values{"page": []string{strconv.Itoa(page + 1)}}
		u += "?" + vals.Encode()
	}
	resp, err := http.Get(u)
	if err != nil {
		return
	}

	var tags []*Tag
	err = json.NewDecoder(resp.Body).Decode(&tags)
	if err != nil {
		return
	}

	// Link: <https://api.github.com/repositories/47859258/tags?page=2>; rel="next",
	//       <https://api.github.com/repositories/47859258/tags?page=5>; rel="last"
	if total == 0 {
		const relLast = `rel="last"`
		total = page + 1
		link := resp.Header.Get("Link")
		for _, part := range strings.Split(link, ",") {
			if strings.HasSuffix(part, relLast) {
				left := strings.TrimSpace(part[:len(part)-len(relLast)])
				lastUrl := strings.TrimSuffix(strings.TrimPrefix(left, "<"), ">;")
				if pos := strings.LastIndexByte(lastUrl, '?'); pos >= 0 {
					if vals, e := url.ParseQuery(lastUrl[pos+1:]); e == nil {
						if n, e := strconv.Atoi(vals.Get("page")); e == nil {
							total = n
						}
					}
				}
				break
			}
		}
	}
	err = pager(tags, page, total)
	if err != nil {
		if err == ErrBreak {
			err = nil
		}
		return
	}
	page++
	if page < total {
		goto loop
	}
	return
}
