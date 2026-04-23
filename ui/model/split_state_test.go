package model

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"cliamp/playlist"
	"cliamp/resolve"
)

func TestSplitStateActivityText(t *testing.T) {
	tests := []struct {
		name string
		run  func(*splitState)
		want string
	}{
		{"inactive", func(*splitState) {}, ""},
		{"active", func(s *splitState) { s.start(func() {}) }, "Splitting chapters... [press x to cancel]"},
		{"finished", func(s *splitState) {
			s.start(func() {})
			s.finish()
		}, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var split splitState
			tt.run(&split)
			if got := split.activityText(); got != tt.want {
				t.Errorf("activityText() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestSplitStateLifecycle(t *testing.T) {
	tests := []struct {
		name            string
		action          string
		wantActive      bool
		wantNilCancel   bool
		wantCancelCalls int
	}{
		{
			name:            "finish clears active",
			action:          "finish",
			wantActive:      false,
			wantNilCancel:   true,
			wantCancelCalls: 0,
		},
		{
			name:            "cancel clears active and calls cancel func",
			action:          "cancel",
			wantActive:      false,
			wantNilCancel:   true,
			wantCancelCalls: 1,
		},
		{
			name:            "cancel is idempotent",
			action:          "cancel-cancel",
			wantActive:      false,
			wantNilCancel:   true,
			wantCancelCalls: 1,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var s splitState
			cancelCalls := 0
			s.start(func() { cancelCalls++ })

			switch tt.action {
			case "finish":
				s.finish()
			case "cancel":
				s.cancelSplit()
			case "cancel-cancel":
				s.cancelSplit()
				s.cancelSplit()
			}

			if s.active != tt.wantActive {
				t.Errorf("active = %v, want %v", s.active, tt.wantActive)
			}
			if (s.cancel == nil) != tt.wantNilCancel {
				t.Errorf("cancel is nil = %v, want %v", s.cancel == nil, tt.wantNilCancel)
			}
			if cancelCalls != tt.wantCancelCalls {
				t.Errorf("cancel calls = %d, want %d", cancelCalls, tt.wantCancelCalls)
			}
		})
	}
}

func TestYTDLSplitMsg(t *testing.T) {
	tests := []struct {
		name       string
		msg        ytdlSplitMsg
		wantStatus string
		wantActive bool
	}{
		{
			name:       "success",
			msg:        ytdlSplitMsg{outDir: "/tmp/album", count: 5, err: nil},
			wantStatus: "Split into 5 files in /tmp/album",
			wantActive: false,
		},
		{
			name:       "no chapters",
			msg:        ytdlSplitMsg{err: resolve.ErrNoChapters},
			wantStatus: "No chapters found. Use Ctrl+S to save the whole track.",
			wantActive: false,
		},
		{
			name:       "wrapped no chapters",
			msg:        ytdlSplitMsg{err: fmt.Errorf("probing chapters: %w", resolve.ErrNoChapters)},
			wantStatus: "No chapters found. Use Ctrl+S to save the whole track.",
			wantActive: false,
		},
		{
			name:       "generic error",
			msg:        ytdlSplitMsg{err: errors.New("network timeout")},
			wantStatus: "Split failed: network timeout",
			wantActive: false,
		},
		{
			name:       "cancelled",
			msg:        ytdlSplitMsg{err: context.Canceled},
			wantStatus: "",
			wantActive: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := Model{split: splitState{active: true}}
			nextModel, cmd := m.Update(tt.msg)
			if cmd != nil {
				t.Errorf("Update() cmd = %v, want nil", cmd)
			}
			next := nextModel.(Model)
			if next.split.active != tt.wantActive {
				t.Errorf("active = %v, want %v", next.split.active, tt.wantActive)
			}
			if next.status.text != tt.wantStatus {
				t.Errorf("status.text = %q, want %q", next.status.text, tt.wantStatus)
			}
		})
	}
}

func TestSplitTrackGuards(t *testing.T) {
	tests := []struct {
		name  string
		setup func() Model
	}{
		{
			name: "empty playlist",
			setup: func() Model {
				return Model{playlist: playlist.New()}
			},
		},
		{
			name: "non-streaming track",
			setup: func() Model {
				pl := playlist.New()
				pl.Add(playlist.Track{Path: "/local/file.mp3"})
				return Model{playlist: pl}
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := tt.setup()
			if cmd := m.splitTrack(); cmd != nil {
				t.Errorf("splitTrack() returned cmd, want nil")
			}
		})
	}
}
