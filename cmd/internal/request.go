package internal

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"charm.land/log/v2"
)

const (
	TypstPackageEndpoint string = "https://api.github.com/repos/typst/packages/contents/packages/preview/"
	TypstPackageIndexURL string = "https://packages.typst.org/preview/index.json"
)

type ResponseModel struct {
	Name string `json:"name" validate:"semver"`
}

type TypstIndexEntry struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

func FetchDataFromGitHub(url string, ctx context.Context) ([]*ResponseModel, error) {
	client := &http.Client{Timeout: 5 * time.Second}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer closeResponse(resp)

	var result []*ResponseModel
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return result, nil
}

func GetLatestVersion(versions []*ResponseModel) (string, error) {
	candidate := NewVersion()
	for _, v := range versions {
		currentVersion, err := ParseVersion(v.Name)
		if err != nil {
			return "", err
		}
		res := CompareVersions(candidate, currentVersion)
		switch res {
		case -1:
			candidate = currentVersion
		default:
			continue
		}
	}
	return candidate.String(), nil
}

func closeResponse(resp *http.Response) {
	err := resp.Body.Close()
	if err != nil {
		log.Fatal(err)
	}
}

// FetchTypstIndex fetches the full package index from packages.typst.org.
func FetchTypstIndex(ctx context.Context) ([]TypstIndexEntry, error) {
	client := &http.Client{Timeout: 10 * time.Second}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, TypstPackageIndexURL, nil)
	if err != nil {
		return nil, err
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer closeResponse(resp)
	var entries []TypstIndexEntry
	if err := json.NewDecoder(resp.Body).Decode(&entries); err != nil {
		return nil, err
	}
	return entries, nil
}

// BuildVersionIndex reduces a list of index entries to a map of package name
// to its latest version string.
func BuildVersionIndex(entries []TypstIndexEntry) map[string]string {
	index := make(map[string]string)
	for _, entry := range entries {
		current, exists := index[entry.Name]
		if !exists {
			index[entry.Name] = entry.Version
			continue
		}
		currentV, err := ParseVersion(current)
		if err != nil {
			continue
		}
		entryV, err := ParseVersion(entry.Version)
		if err != nil {
			continue
		}
		if CompareVersions(entryV, currentV) > 0 {
			index[entry.Name] = entry.Version
		}
	}
	return index
}
