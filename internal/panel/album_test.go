package panel

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/kyros-software/claude-code-themes/internal/pet"
	"github.com/kyros-software/claude-code-themes/internal/theme"
)

func albumFor(t *testing.T, s *pet.State) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "pet.json")
	if !pet.Save(s, path) {
		t.Fatal("could not write the state")
	}
	var out bytes.Buffer
	if code := Run([]string{"album"}, &out, &out, path, time.Now()); code != 0 {
		t.Fatalf("album returned %d", code)
	}
	return theme.Strip(out.String())
}

func TestTheAlbumShowsEveryLifeAndWhatIsLeft(t *testing.T) {
	s := pet.New()
	s.Generation = 2
	s.Album = []pet.AlbumEntry{
		{Form: "spark", Gen: 1}, {Form: "pattern", Gen: 1}, {Form: "refactor", Gen: 1},
		{Form: "surgeon", Gen: 1}, {Form: "scalpel", Gen: 1}, {Form: "spark", Gen: 2},
	}
	got := albumFor(t, s)
	for _, want := range []string{
		"5/41 formas · generación 2",
		"generación 1   chispa › pauta › refactor › cirujano › bisturí",
		"generación 2   chispa",
		"por ver 36",
		"sabueso (reproducir antes de arreglar)",
		"lobo (de sabueso)",
		"2 sin descubrir",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("the album is missing %q:\n%s", want, got)
		}
	}
	// What has been seen is not listed as still to see.
	if strings.Contains(got, "bisturí (") {
		t.Errorf("a form already seen is listed as to see:\n%s", got)
	}
}

func TestThePanelSaysWhenTheEggComes(t *testing.T) {
	got := panelFor(t, func(s *pet.State) {
		s.XP = pet.Levels[len(pet.Levels)-1].XP
	})
	if !strings.Contains(got, "huevo en 3 días de uso") {
		t.Errorf("a pet that just reached its title does not say when the egg comes:\n%s", got)
	}
	if !strings.Contains(got, "/pet album") {
		t.Errorf("the panel does not point at the album:\n%s", got)
	}
}
