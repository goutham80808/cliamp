package model

import (
	"errors"
	"testing"

	"cliamp/playlist"
	"cliamp/resolve"
)

func TestSplitStateActivityTextTracksPending(t *testing.T) {
	var split splitState

	if got := split.activityText(); got != "" {
		t.Fatalf("activityText() = %q, want empty", got)
	}

	split.start()
	if got := split.activityText(); got != "Splitting chapters..." {
		t.Fatalf("activityText() after first start = %q, want %q", got, "Splitting chapters...")
	}

	split.start()
	if got := split.activityText(); got != "Splitting chapters... (2)" {
		t.Fatalf("activityText() after second start = %q, want %q", got, "Splitting chapters... (2)")
	}

	split.finish()
	if got := split.activityText(); got != "Splitting chapters..." {
		t.Fatalf("activityText() after first finish = %q, want %q", got, "Splitting chapters...")
	}

	split.finish()
	if got := split.activityText(); got != "" {
		t.Fatalf("activityText() after second finish = %q, want empty", got)
	}
}

func TestYTDLSplitMsgUpdatesStatus(t *testing.T) {
	m := Model{
		split: splitState{pending: 1},
	}

	nextModel, cmd := m.Update(ytdlSplitMsg{outDir: "/tmp/album", count: 5, err: nil})
	if cmd != nil {
		t.Fatalf("Update() cmd = %v, want nil", cmd)
	}

	next, ok := nextModel.(Model)
	if !ok {
		t.Fatalf("Update() model = %T, want ui.Model", nextModel)
	}
	if next.split.pending != 0 {
		t.Fatalf("pending after ytdlSplitMsg = %d, want 0", next.split.pending)
	}
	want := "Split into 5 files in /tmp/album"
	if next.status.text != want {
		t.Fatalf("status.text = %q, want %q", next.status.text, want)
	}
}

func TestYTDLSplitMsgNoChaptersShowsFallback(t *testing.T) {
	m := Model{
		split: splitState{pending: 1},
	}

	nextModel, _ := m.Update(ytdlSplitMsg{err: resolve.ErrNoChapters})
	next := nextModel.(Model)

	want := "No chapters found. Use Ctrl+S to save the whole track."
	if next.status.text != want {
		t.Fatalf("status.text = %q, want %q", next.status.text, want)
	}
}

func TestYTDLSplitMsgGenericErrorShowsFailure(t *testing.T) {
	m := Model{
		split: splitState{pending: 1},
	}

	nextModel, _ := m.Update(ytdlSplitMsg{err: errors.New("network timeout")})
	next := nextModel.(Model)

	want := "Split failed: network timeout"
	if next.status.text != want {
		t.Fatalf("status.text = %q, want %q", next.status.text, want)
	}
}

func TestSplitTrackGuards(t *testing.T) {
	// No track in playlist.
	pl := playlist.New()
	m := Model{playlist: pl}
	if cmd := m.splitTrack(); cmd != nil {
		t.Fatalf("splitTrack() with empty playlist returned cmd, want nil")
	}

	// Non-streaming track.
	pl.Add(playlist.Track{Path: "/local/file.mp3"})
	m = Model{playlist: pl}
	if cmd := m.splitTrack(); cmd != nil {
		t.Fatalf("splitTrack() with local track returned cmd, want nil")
	}
}
