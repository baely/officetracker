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
		theme.TodayBorderColor = "#4CAF50" // Green today highlight
		theme.ExtraCSS = `
		main nav { 
			background-image: linear-gradient(to bottom, #d8eed8, #e8f8e8);
			border-bottom: 1px solid #c8d8c8;
		}
		main { 
			box-shadow: 0 0 12px rgba(120, 180, 120, 0.15);
		}
		/* Spring flower decoration in the corner */
		main::after {
			content: "";
			position: absolute;
			bottom: 24px;
			right: 24px;
			width: 120px;
			height: 120px;
			background-image: url("data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' viewBox='0 0 100 100'%3E%3Cg fill='%23aed581' opacity='0.3'%3E%3Ccircle cx='50' cy='50' r='15'/%3E%3Ccircle cx='30' cy='50' r='15'/%3E%3Ccircle cx='70' cy='50' r='15'/%3E%3Ccircle cx='50' cy='30' r='15'/%3E%3Ccircle cx='50' cy='70' r='15'/%3E%3C/g%3E%3Ccircle cx='50' cy='50' r='10' fill='%23ffeb3b'/%3E%3C/svg%3E");
			background-size: contain;
			opacity: 0.15;
			pointer-events: none;
			z-index: 1;
		}`
		
	case SeasonSummer:
		theme.MainBackground = "#fffdf7" // Slight yellow tint
		theme.MainNavBackground = "#e6e6da" // Warm nav color
		theme.TodayBorderColor = "#FF9800" // Orange today highlight
		theme.ExtraCSS = `
		main nav { 
			background-image: linear-gradient(to bottom, #e6e6da, #eeeee0);
			border-bottom: 1px solid #ddddcc;
		}
		main { 
			box-shadow: 0 0 12px rgba(200, 180, 100, 0.15);
		}
		/* Summer sun decoration */
		main::after {
			content: "";
			position: absolute;
			top: 24px;
			right: 24px;
			width: 100px;
			height: 100px;
			background-image: url("data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' viewBox='0 0 100 100'%3E%3Ccircle cx='50' cy='50' r='20' fill='%23ffeb3b' opacity='0.25'/%3E%3Cpath d='M50 0 L53 35 L50 40 L47 35 Z M50 100 L53 65 L50 60 L47 65 Z M0 50 L35 47 L40 50 L35 53 Z M100 50 L65 47 L60 50 L65 53 Z' fill='%23ffeb3b' opacity='0.2'/%3E%3Cpath d='M14 14 L38 38 M86 14 L62 38 M14 86 L38 62 M86 86 L62 62' stroke='%23ffeb3b' stroke-width='3' opacity='0.15'/%3E%3C/svg%3E");
			background-size: contain;
			opacity: 0.15;
			pointer-events: none;
			z-index: 1;
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
			box-shadow: 0 0 12px rgba(170, 140, 110, 0.2);
		}
		/* Autumn leaf decoration */
		main::after {
			content: "";
			position: absolute;
			bottom: 24px;
			left: 24px;
			width: 120px;
			height: 120px;
			background-image: url("data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' viewBox='0 0 100 100'%3E%3Cpath d='M50,10 C30,10 10,30 10,50 C10,70 60,90 50,90 C40,90 10,70 10,50 C10,30 30,10 50,10 Z' fill='%23e67e22' opacity='0.2'/%3E%3Cpath d='M50,10 C70,10 90,30 90,50 C90,70 40,90 50,90 C60,90 90,70 90,50 C90,30 70,10 50,10 Z' fill='%23d35400' opacity='0.2'/%3E%3Cpath d='M50,10 L50,90 M10,50 L90,50' stroke='%23c0392b' opacity='0.15' stroke-width='2'/%3E%3C/svg%3E");
			background-size: contain;
			opacity: 0.15;
			pointer-events: none;
			z-index: 1;
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
			box-shadow: 0 0 12px rgba(100, 140, 180, 0.15);
		}
		/* Winter snowflake decoration */
		main::after {
			content: "";
			position: absolute;
			top: 24px;
			left: 24px;
			width: 100px;
			height: 100px;
			background-image: url("data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' viewBox='0 0 100 100'%3E%3Cpath d='M50 0 L50 100 M0 50 L100 50 M15 15 L85 85 M15 85 L85 15' stroke='%233498db' opacity='0.15' stroke-width='3'/%3E%3Ccircle cx='50' cy='50' r='8' fill='%233498db' opacity='0.15'/%3E%3C/svg%3E");
			background-size: contain;
			opacity: 0.15;
			pointer-events: none;
			z-index: 1;
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