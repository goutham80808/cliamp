package resolve

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestArgsTreatsXiaoyuzhouEpisodeAsPending(t *testing.T) {
	url := "https://www.xiaoyuzhoufm.com/episode/69a13b07a22480add648dd03?s=eyJ1IjogIjYxODEzNmZiZTBmNWU3MjNiYjk2MmE5MiJ9"

	got, err := Args([]string{url})
	if err != nil {
		t.Fatalf("Args returned error: %v", err)
	}
	if len(got.Tracks) != 0 {
		t.Fatalf("Args returned %d immediate tracks, want 0", len(got.Tracks))
	}
	if len(got.Pending) != 1 || got.Pending[0] != url {
		t.Fatalf("Args pending = %#v, want [%q]", got.Pending, url)
	}
}

func TestRemoteResolvesXiaoyuzhouEpisodeHTML(t *testing.T) {
	const episodeURL = "https://www.xiaoyuzhoufm.com/episode/69a13b07a22480add648dd03?s=eyJ1IjogIjYxODEzNmZiZTBmNWU3MjNiYjk2MmE5MiJ9"
	const audioURL = "https://media.xyzcdn.net/65d322815c5cc49b4db454a8/lqbqTgipk04QFSwIMACyGNK655rR.m4a"
	const title = "周轶君对话张艾嘉：我从不刻意标榜“女性”"
	const podcast = "山下声"

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/episode/69a13b07a22480add648dd03" {
			t.Errorf("unexpected path: %s", r.URL.Path)
			http.Error(w, "unexpected path", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte(`<!DOCTYPE html>
<html><head>
<script name="schema:podcast-show" type="application/ld+json">{
  "@context":"https://schema.org/",
  "@type":"PodcastEpisode",
  "url":"https://www.xiaoyuzhoufm.com/episode/69a13b07a22480add648dd03",
  "name":"` + title + `",
  "timeRequired":"PT106M",
  "associatedMedia":{"@type":"MediaObject","contentUrl":"` + audioURL + `"},
  "partOfSeries":{"@type":"PodcastSeries","name":"` + podcast + `","url":"https://www.xiaoyuzhoufm.com/podcast/65d322815c5cc49b4db454a8"}
}</script>
</head><body></body></html>`))
	}))
	defer srv.Close()

	target, err := url.Parse(srv.URL)
	if err != nil {
		t.Fatalf("parsing test server URL: %v", err)
	}

	oldClient := httpClient
	httpClient = &http.Client{
		Timeout:   30 * time.Second,
		Transport: rewriteHostTransport{target: target, rt: http.DefaultTransport},
	}
	defer func() {
		httpClient = oldClient
	}()

	tracks, err := Remote([]string{episodeURL})
	if err != nil {
		t.Fatalf("Remote returned error: %v", err)
	}
	if len(tracks) != 1 {
		t.Fatalf("Remote returned %d tracks, want 1", len(tracks))
	}
	track := tracks[0]
	if track.Path != audioURL {
		t.Fatalf("track.Path = %q, want %q", track.Path, audioURL)
	}
	if track.Title != title {
		t.Fatalf("track.Title = %q, want %q", track.Title, title)
	}
	if track.Artist != podcast {
		t.Fatalf("track.Artist = %q, want %q", track.Artist, podcast)
	}
	if !track.Stream {
		t.Fatalf("track.Stream = false, want true")
	}
	if track.DurationSecs != 106*60 {
		t.Fatalf("track.DurationSecs = %d, want %d", track.DurationSecs, 106*60)
	}
}

func TestParseXiaoyuzhouOgAudioTakesPrecedence(t *testing.T) {
	const audioURL = "https://media.xyzcdn.net/audio.m4a"
	const title = "Test Episode"

	doc := `<!DOCTYPE html>
<html><head>
<meta property="og:audio" content="` + audioURL + `">
<meta property="og:title" content="` + title + `">
</head><body></body></html>`

	track, err := parseXiaoyuzhouEpisodeHTML("https://www.xiaoyuzhoufm.com/episode/abc", doc)
	if err != nil {
		t.Fatalf("parseXiaoyuzhouEpisodeHTML returned error: %v", err)
	}
	if track.Path != audioURL {
		t.Fatalf("track.Path = %q, want %q", track.Path, audioURL)
	}
	if track.Title != title {
		t.Fatalf("track.Title = %q, want %q", track.Title, title)
	}
}

func TestParseItunesDuration(t *testing.T) {
	tests := []struct {
		input string
		want  int
	}{
		// Plain seconds
		{"3600", 3600},
		{"90", 90},
		{"0", 0},
		// Fractional seconds
		{"3661.5", 3661},
		{"90.9", 90},
		// MM:SS
		{"1:30", 90},
		{"87:05", 5225},
		// HH:MM:SS
		{"1:27:05", 5225},
		{"0:01:30", 90},
		// Whitespace
		{" 3600 ", 3600},
		// Empty
		{"", 0},
		// Invalid — return 0
		{"abc", 0},
		{"12:xx", 0},
		{"1:2:xx", 0},
		// Negative — clamp to 0
		{"-1", 0},
		{"0:-10", 0},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := parseItunesDuration(tt.input)
			if got != tt.want {
				t.Errorf("parseItunesDuration(%q) = %d, want %d", tt.input, got, tt.want)
			}
		})
	}
}

func TestSanitizeFilename(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"hello/world", "hello-world"},
		{"foo\\bar", "foo-bar"},
		{"a:b*c?d\"e<f>g|h", "a-b-c-d-e-f-g-h"},
		{"  trim me  ", "trim me"},
		{"normal name", "normal name"},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := sanitizeFilename(tt.input)
			if got != tt.want {
				t.Errorf("sanitizeFilename(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestUniqueDir(t *testing.T) {
	tmp := t.TempDir()
	base := filepath.Join(tmp, "testdir")

	// First call: dir does not exist yet.
	got := uniqueDir(base)
	if got != base {
		t.Fatalf("uniqueDir(base) = %q, want %q", got, base)
	}

	// Create the directory.
	if err := os.MkdirAll(base, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	// Second call: dir exists, should get " (1)".
	got = uniqueDir(base)
	want := base + " (1)"
	if got != want {
		t.Fatalf("uniqueDir(base) = %q, want %q", got, want)
	}

	// Create the " (1)" directory.
	if err := os.MkdirAll(want, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	// Third call: should get " (2)".
	got = uniqueDir(base)
	want2 := base + " (2)"
	if got != want2 {
		t.Fatalf("uniqueDir(base) = %q, want %q", got, want2)
	}
}

func TestIsChapterFile(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		// Valid chapter files
		{"001 - Intro.mp3", true},
		{"12 - Verse Two.m4a", true},
		{"1 - A.mp3", true},
		{"999 - Last Track.webm", true},
		// Invalid — too short
		{"1.mp3", false},
		{"1 - ", false},
		// Invalid — no separator
		{"001_Intro.mp3", false},
		{"001 -Intro.mp3", false},
		{"001  -  Intro.mp3", false}, // two spaces before hyphen
		// Invalid — no digits at start
		{"Intro - 001.mp3", false},
		{"A - 001.mp3", false},
		// Invalid — empty
		{"", false},
		// Invalid — digits but no separator or extension
		{"001", false},
		{"001 ", false},
		// Invalid — full file name (no chapter pattern)
		{"Pushpa 2 The Rule Telugu Audio Jukebox.mp3", false},
		{"Some Full Song.m4a", false},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := isChapterFile(tt.input)
			if got != tt.want {
				t.Errorf("isChapterFile(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestSplitChaptersYTDLCleanupRemovesFullFile(t *testing.T) {
	// Simulate the directory cleanup step: create a directory with a full
	// file and several chapter files, then call the cleanup logic.
	// This tests the filter behavior without requiring yt-dlp on the path.
	tmp := t.TempDir()
	outDir := filepath.Join(tmp, "splits")
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		t.Fatal(err)
	}

	// Simulate what yt-dlp produces: one full file + chapter files.
	files := []string{
		"Pushpa 2 The Rule - Full Audio Jukebox.mp3",
		"001 - Intro.mp3",
		"002 - Versio.mp3",
		"003 - Salaar Remix.mp3",
	}
	for _, f := range files {
		if err := os.WriteFile(filepath.Join(outDir, f), []byte("audio"), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	// Run the cleanup logic (same logic as SplitChaptersYTDL step 4).
	entries, err := os.ReadDir(outDir)
	if err != nil {
		t.Fatal(err)
	}
	count := 0
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		if isChapterFile(e.Name()) {
			count++
		} else {
			_ = os.Remove(filepath.Join(outDir, e.Name()))
		}
	}

	if count != 3 {
		t.Fatalf("expected 3 chapter files, got %d", count)
	}

	// Verify the full file was deleted.
	if _, err := os.Stat(filepath.Join(outDir, "Pushpa 2 The Rule - Full Audio Jukebox.mp3")); !os.IsNotExist(err) {
		t.Fatal("expected full file to be deleted")
	}

	// Verify chapter files still exist.
	for _, f := range files[1:] {
		if _, err := os.Stat(filepath.Join(outDir, f)); err != nil {
			t.Fatalf("chapter file %q was deleted: %v", f, err)
		}
	}
}

type rewriteHostTransport struct {
	target *url.URL
	rt     http.RoundTripper
}

func (t rewriteHostTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	clone := req.Clone(req.Context())
	clone.URL.Scheme = t.target.Scheme
	clone.URL.Host = t.target.Host
	clone.Host = t.target.Host
	return t.rt.RoundTrip(clone)
}
