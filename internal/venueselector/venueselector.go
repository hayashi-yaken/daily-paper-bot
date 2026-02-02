package venueselector

import (
	"errors"
	"math/rand"
	"time"

	"github.com/hayashi-yaken/daily-paper-bot/internal/config"
)

var ErrNoVenues = errors.New("no venues to select from")

// VenueSelector は実行対象の学会を選定するインターフェースです。
type VenueSelector interface {
	Select(venues []config.VenueConfig) (config.VenueConfig, error)
}

// RandomVenueSelector はランダムに学会を選定します。
type RandomVenueSelector struct {
	rand *rand.Rand
}

// NewRandomVenueSelector は新しいRandomVenueSelectorを生成します。
func NewRandomVenueSelector() VenueSelector {
	return &RandomVenueSelector{
		rand: rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

// Select は学会のリストからランダムに1つを選びます。
func (s *RandomVenueSelector) Select(venues []config.VenueConfig) (config.VenueConfig, error) {
	if len(venues) == 0 {
		return config.VenueConfig{}, ErrNoVenues
	}

	selectedIndex := s.rand.Intn(len(venues))
	return venues[selectedIndex], nil
}
