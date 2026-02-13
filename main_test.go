package main

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestParsePreviewTrackFilename(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		filename    string
		wantPalette string
		wantTrack   string
		wantOK      bool
	}{
		{
			name:        "open phase",
			filename:    "WARM_OPEN-FirstLight.m4a",
			wantPalette: "warm",
			wantTrack:   "open",
			wantOK:      true,
		},
		{
			name:        "beacon case insensitive",
			filename:    "modern_sonare.m4a",
			wantPalette: "modern",
			wantTrack:   "beacon",
			wantOK:      true,
		},
		{
			name:        "offpeak with uppercase extension",
			filename:    "PREMIUM_OFFPEAK-DriftState.M4A",
			wantPalette: "premium",
			wantTrack:   "offpeak",
			wantOK:      true,
		},
		{
			name:     "invalid extension",
			filename: "WARM_OPEN-FirstLight.mp3",
			wantOK:   false,
		},
		{
			name:     "missing separator",
			filename: "WARMOPEN-FirstLight.m4a",
			wantOK:   false,
		},
		{
			name:     "unknown phase",
			filename: "WARM_TRANSITION-Rise.m4a",
			wantOK:   false,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			gotPalette, gotTrack, gotOK := parsePreviewTrackFilename(tc.filename)
			if gotOK != tc.wantOK {
				t.Fatalf("ok mismatch: got=%v want=%v", gotOK, tc.wantOK)
			}
			if !tc.wantOK {
				return
			}
			if gotPalette != tc.wantPalette {
				t.Fatalf("palette mismatch: got=%q want=%q", gotPalette, tc.wantPalette)
			}
			if gotTrack != tc.wantTrack {
				t.Fatalf("track mismatch: got=%q want=%q", gotTrack, tc.wantTrack)
			}
		})
	}
}

func TestPreviewSourcesForPalette(t *testing.T) {
	t.Parallel()

	musicDir := t.TempDir()
	files := []string{
		"WARM_OPEN-FirstLight.m4a",
		"WARM_PEAK-CoreFlow.m4a",
		"WARM_OFFPEAK-DriftState.m4a",
		"WARM_CLOSE-LastCall.m4a",
		"WARM_Sonare.m4a",
		"MODERN_Sonare.m4a",
		"README.txt",
	}

	for _, file := range files {
		err := os.WriteFile(filepath.Join(musicDir, file), []byte("x"), 0o644)
		if err != nil {
			t.Fatalf("write %q: %v", file, err)
		}
	}

	got, err := previewSourcesForPalette(musicDir, "warm")
	if err != nil {
		t.Fatalf("previewSourcesForPalette returned error: %v", err)
	}

	want := map[string]string{
		"open":    "/music/WARM_OPEN-FirstLight.m4a",
		"peak":    "/music/WARM_PEAK-CoreFlow.m4a",
		"offpeak": "/music/WARM_OFFPEAK-DriftState.m4a",
		"close":   "/music/WARM_CLOSE-LastCall.m4a",
		"beacon":  "/music/WARM_Sonare.m4a",
	}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("sources mismatch:\n got: %#v\nwant: %#v", got, want)
	}
}

func TestPreviewSourcesForPaletteUnknownPalette(t *testing.T) {
	t.Parallel()

	musicDir := t.TempDir()
	err := os.WriteFile(filepath.Join(musicDir, "WARM_OPEN-FirstLight.m4a"), []byte("x"), 0o644)
	if err != nil {
		t.Fatalf("write test file: %v", err)
	}

	got, err := previewSourcesForPalette(musicDir, "unknown")
	if err != nil {
		t.Fatalf("previewSourcesForPalette returned error: %v", err)
	}

	want := map[string]string{
		"open":    "",
		"peak":    "",
		"offpeak": "",
		"close":   "",
		"beacon":  "",
	}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("unknown palette mismatch:\n got: %#v\nwant: %#v", got, want)
	}
}
