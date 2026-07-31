package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// PaperMC retired the v2 API — api.papermc.io/v2 now answers HTTP 410 Gone.
// v3 ("Fill") is the replacement and is the only supported endpoint.
const fillAPIBase = "https://fill.papermc.io/v3"

// serverDownloadKey is the artifact a plain server jar is published under.
const serverDownloadKey = "server:default"

// errVersionNotFound signals that Paper publishes no builds for a version.
var errVersionNotFound = errors.New("version not found")

// httpClient deliberately has no overall timeout: server jars are ~60MB and a
// slow connection is not an error. Hangs are caught by the per-stage timeouts.
var httpClient = &http.Client{
	Transport: &http.Transport{
		Proxy:                 http.ProxyFromEnvironment,
		TLSHandshakeTimeout:   15 * time.Second,
		ResponseHeaderTimeout: 30 * time.Second,
	},
}

// apiBaseURL is overridden in tests to point at an httptest server.
var apiBaseURL = fillAPIBase

type paperBuild struct {
	ID        int                      `json:"id"`
	Channel   string                   `json:"channel"`
	Downloads map[string]paperDownload `json:"downloads"`
}

type paperDownload struct {
	Name      string `json:"name"`
	URL       string `json:"url"`
	Size      int64  `json:"size"`
	Checksums struct {
		SHA256 string `json:"sha256"`
	} `json:"checksums"`
}

// paperProject is the project index. Versions maps a version family ("1.21",
// "26.2") to the individual versions released within it.
type paperProject struct {
	Versions map[string][]string `json:"versions"`
}

type paperAPIError struct {
	Error   string `json:"error"`
	Message string `json:"message"`
}

func userAgent() string {
	return fmt.Sprintf("mc/%s (+https://github.com/Mindgamesnl/mc)", Version)
}

// get performs a JSON API request and decodes the response into out.
func get(url string, out any) error {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", userAgent())
	req.Header.Set("Accept", "application/json")

	resp, err := httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK:
	case http.StatusNotFound:
		return errVersionNotFound
	case http.StatusTooManyRequests:
		return errors.New("rate limited by the PaperMC API, try again in a moment")
	default:
		return fmt.Errorf("papermc API returned %s%s", resp.Status, apiErrorDetail(resp.Body))
	}

	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		return fmt.Errorf("decoding papermc API response: %w", err)
	}
	return nil
}

// apiErrorDetail extracts the API's own error message, if it sent one.
func apiErrorDetail(body io.Reader) string {
	var apiErr paperAPIError
	if err := json.NewDecoder(io.LimitReader(body, 4096)).Decode(&apiErr); err != nil {
		return ""
	}
	if apiErr.Message == "" {
		return ""
	}
	return ": " + apiErr.Message
}

// latestBuild resolves the newest build Paper published for a version.
func latestBuild(version string) (*paperBuild, error) {
	url := fmt.Sprintf("%s/projects/paper/versions/%s/builds/latest", apiBaseURL, version)

	var build paperBuild
	if err := get(url, &build); err != nil {
		return nil, err
	}
	return &build, nil
}

// availableVersions returns every version Paper publishes, grouped by family
// and sorted newest first within and across families.
func availableVersions() (families []string, byFamily map[string][]string, err error) {
	var project paperProject
	if err := get(apiBaseURL+"/projects/paper", &project); err != nil {
		return nil, nil, err
	}

	byFamily = make(map[string][]string, len(project.Versions))
	for family, versions := range project.Versions {
		sorted := append([]string(nil), versions...)
		sortVersionsDescending(sorted)
		byFamily[family] = sorted
		families = append(families, family)
	}
	sortVersionsDescending(families)

	return families, byFamily, nil
}

// downloadPaper fetches the latest server jar for a version to jarPath.
func downloadPaper(version, jarPath string) error {
	build, err := latestBuild(version)
	if err != nil {
		if errors.Is(err, errVersionNotFound) {
			return fmt.Errorf("Paper has no builds for Minecraft %s (run `mc versions` to see what is available)", version)
		}
		return err
	}

	download, ok := build.Downloads[serverDownloadKey]
	if !ok {
		return fmt.Errorf("build %d of Paper %s has no %q artifact", build.ID, version, serverDownloadKey)
	}

	channel := ""
	if build.Channel != "" && !strings.EqualFold(build.Channel, "STABLE") {
		channel = fmt.Sprintf(" [%s]", strings.ToLower(build.Channel))
	}
	fmt.Printf("Downloading Paper %s build %d%s (%s)...\n", version, build.ID, channel, humanSize(download.Size))

	return downloadJar(download, jarPath)
}

// downloadJar streams a download to a temporary file, verifies its checksum,
// and only then moves it into place, so an interrupted run never leaves a
// truncated jar that later looks like a valid cached download.
func downloadJar(download paperDownload, dest string) error {
	req, err := http.NewRequest(http.MethodGet, download.URL, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", userAgent())

	resp, err := httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download failed with status %s", resp.Status)
	}

	dir := filepath.Dir(dest)
	tmp, err := os.CreateTemp(dir, ".mc-download-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	moved := false
	defer func() {
		tmp.Close()
		if !moved {
			os.Remove(tmpName)
		}
	}()

	digest := sha256.New()
	progress := newProgressWriter(download.Size)
	if _, err := io.Copy(io.MultiWriter(tmp, digest, progress), resp.Body); err != nil {
		progress.abort()
		return err
	}
	progress.finish()

	if want := download.Checksums.SHA256; want != "" {
		got := hex.EncodeToString(digest.Sum(nil))
		if !strings.EqualFold(got, want) {
			return fmt.Errorf("checksum mismatch for %s: got %s, want %s", download.Name, got, want)
		}
	}

	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Rename(tmpName, dest); err != nil {
		return err
	}
	moved = true
	return nil
}

func humanSize(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %ciB", float64(bytes)/float64(div), "KMGTPE"[exp])
}
