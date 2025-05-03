package github

import (
	"encoding/json"
	"net/http"
)

// ReleaseAsset represents a GitHub release asset.
type ReleaseAsset struct {
	URL                string `json:"url"`                  // https://api.github.com/repos/flintlib/flint/releases/assets/242245930
	ID                 int    `json:"id"`                   // 242245930
	NodeID             string `json:"node_id"`              // RA_kwDOAC8YHs4OcGEq
	Name               string `json:"name"`                 // flint-3.2.2.tar.gz
	ContentType        string `json:"content_type"`         // application/x-gtar
	State              string `json:"state"`                // uploaded
	Size               int64  `json:"size"`                 // 123456
	DownloadCount      int    `json:"download_count"`       // 176
	UpdatedAt          string `json:"updated_at"`           // 2025-03-31T08:54:16Z
	BrowserDownloadURL string `json:"browser_download_url"` // https://github.com/flintlib/flint/releases/download/v3.2.2/flint-3.2.2.tar.gz
}

// ReleaseAuthor represents the author of a GitHub release.
type ReleaseAuthor struct {
	Login     string `json:"login"`      // github-actions[bot]
	ID        int    `json:"id"`         // 41898282
	NodeID    string `json:"node_id"`    // MDM6Qm90NDE4OTgyODI=
	AvatarURL string `json:"avatar_url"` // https://avatars.githubusercontent.com/in/15368?v=4
	URL       string `json:"url"`        // https://api.github.com/users/github-actions%5Bbot%5D
	HtmlURL   string `json:"html_url"`   // https://github.com/apps/github-actions
	Type      string `json:"type"`       // Bot
	SiteAdmin bool   `json:"site_admin"` // false
}

// Release represents a GitHub release.
type Release struct {
	URL             string          `json:"url"`              // https://api.github.com/repos/flintlib/flint/releases/209285187
	ID              int             `json:"id"`               // 209285187
	NodeID          string          `json:"node_id"`          // RE_kwDOAC8YHs4MeXBD
	TagName         string          `json:"tag_name"`         // v3.2.2
	TargetCommitish string          `json:"target_commitish"` // b8223680e38ad048355a421bf7f617bb6c5d5e12
	Name            string          `json:"name"`             // FLINT v3.2.2
	PublishedAt     string          `json:"published_at"`     // 2025-03-31T08:54:16Z
	Body            string          `json:"body"`             // Release Notes
	TarballURL      string          `json:"tarball_url"`      // https://api.github.com/repos/flintlib/flint/tarball/v3.2.2
	ZipballURL      string          `json:"zipball_url"`      // https://api.github.com/repos/flintlib/flint/zipball/v3.2.2
	Author          ReleaseAuthor   `json:"author"`
	Assets          []*ReleaseAsset `json:"assets"`
	Prerelease      bool            `json:"prerelease"`
}

// ReleaseURL constructs the URL for a GitHub release.
func ReleaseURL(pkgPath, ver string) string {
	if ver == "" || ver == "latest" {
		return "https://api.github.com/repos/" + pkgPath + "/releases/latest"
	}
	return "https://api.github.com/repos/" + pkgPath + "/releases/tags/" + ver
}

// GetRelease fetches the release information from GitHub.
func GetRelease(pkgPath, ver string) (ret *Release, err error) {
	url := ReleaseURL(pkgPath, ver)
	resp, err := http.Get(url)
	if err != nil {
		return
	}
	ret = new(Release)
	err = json.NewDecoder(resp.Body).Decode(ret)
	return
}

/*
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

// TagsURL constructs the URL for fetching tags from a GitHub repository.
func TagsURL(pkgPath string) string {
	return "https://api.github.com/repos/" + pkgPath + "/tags"
}

// GetTags fetches the tags from a GitHub repository.
func GetTags(pkgPath string, page string) (tags []*Tag, err error) {
	u := TagsURL(pkgPath)
	if page != "" {
		vals := url.Values{"page": []string{page}}
		u += "?" + vals.Encode()
	}
	resp, err := http.Get(u)
	if err != nil {
		return
	}
	err = json.NewDecoder(resp.Body).Decode(&tags)
	return
}
*/
