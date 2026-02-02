package venueselector

import (
	"errors"
	"math/rand"
	"testing"

	"github.com/hayashi-yaken/daily-paper-bot/internal/config"
)

func TestRandomVenueSelector_Select(t *testing.T) {
	venues := []config.VenueConfig{
		{Name: "ICLR", Venue: "ICLR.cc/2025/Conference"},
		{Name: "NeurIPS", Venue: "NeurIPS.cc/2025/Conference"},
		{Name: "ICML", Venue: "ICML.cc/2025/Conference"},
	}

	t.Run("selects one venue randomly", func(t *testing.T) {
		// 型アサーションで *RandomVenueSelector を取得
		selector := NewRandomVenueSelector().(*RandomVenueSelector)
		// 乱数生成器を固定したものに差し替え
		selector.rand = rand.New(rand.NewSource(10)) // seed 10 の場合, Intn(3) は 2 を返す

		selected, err := selector.Select(venues)
		if err != nil {
			t.Fatalf("Select() returned an error: %v", err)
		}

		expectedName := "ICML"
		if selected.Name != expectedName {
			t.Errorf("expected venue %s, but got %s", expectedName, selected.Name)
		}
	})

	t.Run("returns error if no venues", func(t *testing.T) {
		selector := NewRandomVenueSelector()
		_, err := selector.Select([]config.VenueConfig{})
		if !errors.Is(err, ErrNoVenues) {
			t.Errorf("expected ErrNoVenues, but got %v", err)
		}
	})
}
