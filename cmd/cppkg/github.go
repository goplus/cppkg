package main

import (
	"encoding/json"
	"net/http"
	"strings"
)

type githubReleaseAsset struct {
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

type githubReleaseAuthor struct {
	Login     string `json:"login"`      // github-actions[bot]
	ID        int    `json:"id"`         // 41898282
	NodeID    string `json:"node_id"`    // MDM6Qm90NDE4OTgyODI=
	AvatarURL string `json:"avatar_url"` // https://avatars.githubusercontent.com/in/15368?v=4
	URL       string `json:"url"`        // https://api.github.com/users/github-actions%5Bbot%5D
	HtmlURL   string `json:"html_url"`   // https://github.com/apps/github-actions
	Type      string `json:"type"`       // Bot
	SiteAdmin bool   `json:"site_admin"` // false
}

type githubRelease struct {
	URL             string                `json:"url"`              // https://api.github.com/repos/flintlib/flint/releases/209285187
	ID              int                   `json:"id"`               // 209285187
	NodeID          string                `json:"node_id"`          // RE_kwDOAC8YHs4MeXBD
	TagName         string                `json:"tag_name"`         // v3.2.2
	TargetCommitish string                `json:"target_commitish"` // b8223680e38ad048355a421bf7f617bb6c5d5e12
	Name            string                `json:"name"`             // FLINT v3.2.2
	PublishedAt     string                `json:"published_at"`     // 2025-03-31T08:54:16Z
	Body            string                `json:"body"`             // Release Notes
	TarballURL      string                `json:"tarball_url"`      // https://api.github.com/repos/flintlib/flint/tarball/v3.2.2
	ZipballURL      string                `json:"zipball_url"`      // https://api.github.com/repos/flintlib/flint/zipball/v3.2.2
	Author          githubReleaseAuthor   `json:"author"`
	Assets          []*githubReleaseAsset `json:"assets"`
	Prerelease      bool                  `json:"prerelease"`
}

func (p *githubRelease) asset(suffix string) *githubReleaseAsset {
	for _, asset := range p.Assets {
		if strings.HasSuffix(asset.BrowserDownloadURL, suffix) {
			return asset
		}
	}
	return nil
}

func githubReleaseURL(pkgPath, ver string) string {
	if ver == "" || ver == "latest" {
		return "https://api.github.com/repos/" + pkgPath + "/releases/latest"
	}
	return "https://api.github.com/repos/" + pkgPath + "/releases/tags/" + ver
}

func githubReleaseGet(pkgPath, ver string) (ret *githubRelease, err error) {
	url := githubReleaseURL(pkgPath, ver)
	resp, err := http.Get(url)
	if err != nil {
		return
	}
	ret = new(githubRelease)
	err = json.NewDecoder(resp.Body).Decode(ret)
	return
}
