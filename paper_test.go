package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

// fakeAPI stands in for fill.papermc.io. Responses mirror the real v3 payload
// shapes captured from the live API.
func fakeAPI(t *testing.T, jar []byte) *httptest.Server {
	t.Helper()

	sum := sha256.Sum256(jar)
	checksum := hex.EncodeToString(sum[:])

	mux := http.NewServeMux()
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	mux.HandleFunc("/v3/projects/paper", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(paperProject{Versions: map[string][]string{
			"26.2": {"26.2-rc-2", "26.2"},
			"26.1": {"26.1.1", "26.1.2"},
			"1.21": {"1.21.4", "1.21.11"},
		}})
	})

	mux.HandleFunc("/v3/projects/paper/versions/26.2/builds/latest", func(w http.ResponseWriter, r *http.Request) {
		if ua := r.Header.Get("User-Agent"); !strings.HasPrefix(ua, "mc/") {
			t.Errorf("User-Agent = %q, want it to identify mc", ua)
		}
		fmt.Fprintf(w, `{
			"id": 87,
			"channel": "STABLE",
			"downloads": {
				"server:default": {
					"name": "paper-26.2-87.jar",
					"url": %q,
					"size": %d,
					"checksums": {"sha256": %q}
				}
			}
		}`, server.URL+"/jar", len(jar), checksum)
	})

	// A version whose advertised checksum does not match the served bytes.
	mux.HandleFunc("/v3/projects/paper/versions/26.1.2/builds/latest", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, `{
			"id": 12,
			"channel": "STABLE",
			"downloads": {
				"server:default": {"name": "paper-26.1.2-12.jar", "url": %q, "size": %d,
					"checksums": {"sha256": "0000000000000000000000000000000000000000000000000000000000000000"}}
			}
		}`, server.URL+"/jar", len(jar))
	})

	mux.HandleFunc("/jar", func(w http.ResponseWriter, r *http.Request) {
		w.Write(jar)
	})

	// Anything else is an unknown version, exactly as the real API answers.
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprint(w, `{"ok":false,"error":"version_not_found","message":"No version was found with the given identifier."}`)
	})

	return server
}

func useFakeAPI(t *testing.T, jar []byte) {
	t.Helper()
	server := fakeAPI(t, jar)
	original := apiBaseURL
	apiBaseURL = server.URL + "/v3"
	t.Cleanup(func() { apiBaseURL = original })
}

func TestLatestBuild(t *testing.T) {
	useFakeAPI(t, []byte("jar contents"))

	build, err := latestBuild("26.2")
	if err != nil {
		t.Fatalf("latestBuild() error = %v", err)
	}
	if build.ID != 87 {
		t.Errorf("build.ID = %d, want 87", build.ID)
	}
	download, ok := build.Downloads[serverDownloadKey]
	if !ok {
		t.Fatalf("build has no %q download, got %v", serverDownloadKey, build.Downloads)
	}
	if download.Name != "paper-26.2-87.jar" {
		t.Errorf("download.Name = %q, want paper-26.2-87.jar", download.Name)
	}
}

func TestLatestBuildUnknownVersion(t *testing.T) {
	useFakeAPI(t, []byte("jar contents"))

	if _, err := latestBuild("99.9"); !errors.Is(err, errVersionNotFound) {
		t.Errorf("latestBuild() error = %v, want errVersionNotFound", err)
	}
}

func TestAvailableVersions(t *testing.T) {
	useFakeAPI(t, []byte("jar contents"))

	families, byFamily, err := availableVersions()
	if err != nil {
		t.Fatalf("availableVersions() error = %v", err)
	}

	wantFamilies := []string{"26.2", "26.1", "1.21"}
	if !reflect.DeepEqual(families, wantFamilies) {
		t.Errorf("families = %v, want %v", families, wantFamilies)
	}
	// Within a family the newest version comes first, and the release sorts
	// above its own release candidate.
	if want := []string{"26.2", "26.2-rc-2"}; !reflect.DeepEqual(byFamily["26.2"], want) {
		t.Errorf("byFamily[26.2] = %v, want %v", byFamily["26.2"], want)
	}
	if want := []string{"1.21.11", "1.21.4"}; !reflect.DeepEqual(byFamily["1.21"], want) {
		t.Errorf("byFamily[1.21] = %v, want %v", byFamily["1.21"], want)
	}
}

func TestDownloadPaper(t *testing.T) {
	jar := []byte("pretend this is a server jar")
	useFakeAPI(t, jar)

	dest := filepath.Join(t.TempDir(), "paper-26.2.jar")
	if err := downloadPaper("26.2", dest); err != nil {
		t.Fatalf("downloadPaper() error = %v", err)
	}

	got, err := os.ReadFile(dest)
	if err != nil {
		t.Fatalf("reading downloaded jar: %v", err)
	}
	if string(got) != string(jar) {
		t.Errorf("downloaded jar = %q, want %q", got, jar)
	}
}

func TestDownloadPaperChecksumMismatch(t *testing.T) {
	dir := t.TempDir()
	useFakeAPI(t, []byte("pretend this is a server jar"))

	dest := filepath.Join(dir, "paper-26.1.2.jar")
	err := downloadPaper("26.1.2", dest)
	if err == nil || !strings.Contains(err.Error(), "checksum mismatch") {
		t.Fatalf("downloadPaper() error = %v, want a checksum mismatch", err)
	}

	// A corrupt download must not be left behind: the jar's presence is what
	// tells a later run that the version is already cached.
	if _, err := os.Stat(dest); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("jar exists after a failed download, want it removed")
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Errorf("temp files left behind: %v", entries)
	}
}

func TestDownloadPaperUnknownVersion(t *testing.T) {
	useFakeAPI(t, []byte("jar contents"))

	err := downloadPaper("99.9", filepath.Join(t.TempDir(), "paper-99.9.jar"))
	if err == nil || !strings.Contains(err.Error(), "no builds for Minecraft 99.9") {
		t.Fatalf("downloadPaper() error = %v, want a version-not-found message", err)
	}
}

func TestHumanSize(t *testing.T) {
	tests := []struct {
		bytes int64
		want  string
	}{
		{512, "512 B"},
		{1024, "1.0 KiB"},
		{1536, "1.5 KiB"},
		{61756982, "58.9 MiB"},
	}

	for _, test := range tests {
		if got := humanSize(test.bytes); got != test.want {
			t.Errorf("humanSize(%d) = %q, want %q", test.bytes, got, test.want)
		}
	}
}
