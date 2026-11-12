package jsonx

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// GetSeasonByDate returns the season based on the given date.
func GetSeasonByDate(date SimpleDate) Season {
	month := date.ToTime().Month()

	switch month {
	case time.December, time.January, time.February:
		return Winter
	case time.March, time.April, time.May:
		return Spring
	case time.June, time.July, time.August:
		return Summer
	case time.September, time.October, time.November:
		return Autumn
	default:
		return 0 // Should never happen.
	}
}

// Season is a bitmask of seasons
// Winter, Spring, Summer, Autumn.
// It is used to represent the seasons of a year.
// It is a bitmask so that multiple seasons can be set at once.
// It has methods to check if a season is set, add a season,
// remove a season, and get a string representation of the seasons.
type Season int

const (
	// December, January, February.
	Winter Season = 1 << iota // 1 << 0 = 1
	// March, April, May.
	Spring // 1 << 1 = 2
	// June, July, August.
	Summer // 1 << 2 = 4
	// September, October, November.
	Autumn // 1 << 3 = 8

	// AllSeasons is a bitmask of all seasons.
	// It's the number 15 because it's the sum of all seasons.
	AllSeasons = Winter | Spring | Summer | Autumn // 1 + 2 + 4 + 8 = 15
)

// IsValid checks if the season is valid.
func (s Season) IsValid() bool {
	return s&AllSeasons == s
}

// Is checks if the season is set in the bitmask.
func (s Season) Is(season Season) bool {
	return s&season != 0
}

// Add adds a season to the bitmask.
func (s *Season) Add(season Season) {
	*s |= season
}

// Remove removes a season from the bitmask.
func (s *Season) Remove(season Season) {
	*s &= ^season
}

// String returns the string representation of the season(s).
func (s Season) String() string {
	var seasons []string

	if s.Is(Winter) {
		seasons = append(seasons, "Winter")
	}
	if s.Is(Spring) {
		seasons = append(seasons, "Spring")
	}
	if s.Is(Summer) {
		seasons = append(seasons, "Summer")
	}
	if s.Is(Autumn) {
		seasons = append(seasons, "Autumn")
	}

	if len(seasons) == 0 {
		return "None"
	}

	return strings.Join(seasons, ", ")
}

// MarshalJSON marshals the season to JSON.
// It marshals the season as a string.
func (s *Season) UnmarshalJSON(data []byte) error {
	if isNull(data) {
		return nil
	}

	data = trimQuotes(data)
	if len(data) == 0 {
		return nil
	}

	str := string(data)
	constantAsInt, err := strconv.Atoi(str)
	if err != nil {
		return err
	}

	if constantAsInt == 0 { // if 0 is passed, it means All seasons.
		constantAsInt = int(AllSeasons)
	}

	*s = Season(constantAsInt)
	if !s.IsValid() {
		return fmt.Errorf("%w: season: %s", ErrInvalid, str)
	}

	return nil
}
