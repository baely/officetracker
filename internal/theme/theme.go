package theme

import (
	"fmt"
	"html/template"
	"time"
)

// Season represents different seasons of the year
type Season string

const (
	SeasonSpring  Season = "spring"
	SeasonSummer  Season = "summer"
	SeasonAutumn  Season = "autumn"
	SeasonWinter  Season = "winter"
)

// Theme represents a seasonal theme with colors and styles
type Theme struct {
	// Season identifier
	Season Season

	// Main colors
	MainBackground   string
	MainNavBackground string
	MainTextColor    string

	// Calendar colors
	CalendarBorderColor     string
	CalendarBackgroundColor string
	CalendarHeaderColor     string
	
	// Present state colors
	PresentColor    string
	NotPresentColor string
	OtherColor      string
	TodayBorderColor string

	// Additional season-specific CSS
	ExtraCSS string
}

// DefaultTheme returns the default theme with no seasonal adjustments
func DefaultTheme() *Theme {
	return &Theme{
		Season:          "",
		MainBackground:  "#f7f8f8",
		MainNavBackground: "#dee",
		MainTextColor:   "#333333",
		
		CalendarBorderColor:     "#dee2e6",
		CalendarBackgroundColor: "#f8f9fa",
		CalendarHeaderColor:     "#495057",
		
		PresentColor:    "#4CAF50",
		NotPresentColor: "#F44336",
		OtherColor:      "#2196F3",
		TodayBorderColor: "#FFC107",
		
		ExtraCSS:        "",
	}
}

// GetSeasonalTheme returns a theme with subtle adjustments based on the current season
func GetSeasonalTheme() *Theme {
	now := time.Now()
	season := getCurrentSeason(now)
	
	// Start with default theme
	theme := DefaultTheme()
	theme.Season = season
	
	// Apply seasonal adjustments
	switch season {
	case SeasonSpring:
		theme.MainBackground = "#f8fff8" // Slight green tint
		theme.MainNavBackground = "#d8eed8" // Lighter green nav
		theme.ExtraCSS = `
		main nav { 
			background-image: linear-gradient(to bottom, #d8eed8, #e8f8e8);
			border-bottom: 1px solid #c8d8c8;
		}
		main { 
			box-shadow: 0 0 5px rgba(120, 180, 120, 0.1);
		}`
		
	case SeasonSummer:
		theme.MainBackground = "#fffdf7" // Slight yellow tint
		theme.MainNavBackground = "#e6e6da" // Warm nav color
		theme.TodayBorderColor = "#FFB107" // Slightly darker today highlight
		theme.ExtraCSS = `
		main nav { 
			background-image: linear-gradient(to bottom, #e6e6da, #eeeee0);
			border-bottom: 1px solid #ddddcc;
		}
		main { 
			box-shadow: 0 2px 8px rgba(200, 180, 100, 0.1);
		}`
		
	case SeasonAutumn:
		theme.MainBackground = "#faf8f5" // Warm beige background
		theme.MainNavBackground = "#e5dbd0" // Warm brown nav
		theme.CalendarBorderColor = "#d8cfc6" // Warmer borders
		theme.TodayBorderColor = "#e67e22" // Orange today highlight
		theme.ExtraCSS = `
		main nav { 
			background-image: linear-gradient(to bottom, #e5dbd0, #ebe4dd);
			border-bottom: 1px solid #d5cbc0;
		}
		main { 
			box-shadow: 0 2px 8px rgba(170, 140, 110, 0.15);
		}`
		
	case SeasonWinter:
		theme.MainBackground = "#f7f9fc" // Slight blue tint
		theme.MainNavBackground = "#dee8ee" // Cooler blue nav
		theme.CalendarBorderColor = "#d1dbe4" // Cooler borders
		theme.TodayBorderColor = "#3498db" // Blue today highlight
		theme.ExtraCSS = `
		main nav { 
			background-image: linear-gradient(to bottom, #dee8ee, #e8eef2);
			border-bottom: 1px solid #cfd8e0;
		}
		main { 
			box-shadow: 0 2px 8px rgba(100, 140, 180, 0.1);
		}`
	}
	
	return theme
}

// getCurrentSeason determines the season based on the current date
// This uses Northern Hemisphere seasons - could be adjusted for location
func getCurrentSeason(t time.Time) Season {
	month := t.Month()
	
	switch {
	case month >= 3 && month <= 5:
		return SeasonSpring
	case month >= 6 && month <= 8:
		return SeasonSummer
	case month >= 9 && month <= 11:
		return SeasonAutumn
	default:
		return SeasonWinter
	}
}

// ToCSS returns CSS variables for the theme
func (t *Theme) ToCSS() template.CSS {
	css := fmt.Sprintf(`
	:root {
		--main-background: %s;
		--main-nav-background: %s;
		--main-text-color: %s;
		--calendar-border-color: %s;
		--calendar-background-color: %s;
		--calendar-header-color: %s;
		--present-color: %s;
		--not-present-color: %s;
		--other-color: %s;
		--today-border-color: %s;
	}
	
	/* Small season indicator in the bottom right corner */
	body::after {
		content: "Theme: %s";
		position: fixed;
		bottom: 8px;
		right: 8px;
		font-size: 10px;
		color: #888;
		opacity: 0.7;
		pointer-events: none;
	}
	
	%s
	`, 
	t.MainBackground,
	t.MainNavBackground,
	t.MainTextColor,
	t.CalendarBorderColor,
	t.CalendarBackgroundColor,
	t.CalendarHeaderColor,
	t.PresentColor,
	t.NotPresentColor,
	t.OtherColor,
	t.TodayBorderColor,
	t.Season,
	t.ExtraCSS)
	
	return template.CSS(css)
}