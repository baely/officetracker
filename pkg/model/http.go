package model

type Response struct {
	ContentType string
	Data        interface{}
}

type GetYearRequest struct {
	Meta GetYearRequestMeta `meta:"meta" json:"-"`
}

type GetYearRequestMeta struct {
	UserID int `meta:"user_id"`
	Year   int `meta:"year"`
}

type GetYearResponse struct {
	Data YearState `json:"data"`
}

type GetMonthRequest struct {
	Meta GetMonthRequestMeta `meta:"meta" json:"-"`
}

type GetMonthRequestMeta struct {
	UserID int `meta:"user_id"`
	Year   int `meta:"year"`
	Month  int `meta:"month"`
}

type GetMonthResponse struct {
	Data MonthState `json:"data"`
}

type PutMonthRequest struct {
	Meta PutMonthRequestMeta `meta:"meta" json:"-"`
	Data MonthState          `json:"data"`
}

type PutMonthRequestMeta struct {
	UserID int `meta:"user_id"`
	Year   int `meta:"year"`
	Month  int `meta:"month"`
}

type PutMonthResponse struct {
}

type GetDayRequest struct {
	Meta GetDayRequestMeta `meta:"meta" json:"-"`
}

type GetDayRequestMeta struct {
	UserID int `meta:"user_id"`
	Year   int `meta:"year"`
	Month  int `meta:"month"`
	Day    int `meta:"day"`
}

type GetDayResponse struct {
	Data DayState `json:"data"`
}

type PutDayRequest struct {
	Meta PutDayRequestMeta `meta:"meta"`
	Data DayState          `json:"data"`
}

type PutDayRequestMeta struct {
	UserID int `meta:"user_id"`
	Year   int `meta:"year"`
	Month  int `meta:"month"`
	Day    int `meta:"day"`
}

type PutDayResponse struct {
}

type McpGetMonthRequest struct {
	Year  int
	Month int
}

type McpGetMonthResponse struct {
	Dates []struct {
		Date  int
		State string
	}
}

type McpPutDayRequest struct {
	Year  int
	Month int
	Date  int
	State string
}

type McpPutDayResponse struct{}

type GetNoteRequest struct {
	Meta GetNoteRequestMeta `meta:"meta" json:"-"`
}

type GetNoteRequestMeta struct {
	UserID int `meta:"user_id"`
	Year   int `meta:"year"`
	Month  int `meta:"month"`
}

type GetNoteResponse struct {
	Data Note `json:"data"`
}

type PutNoteRequest struct {
	Meta PutNoteRequestMeta `meta:"meta" json:"-"`
	Data Note               `json:"data"`
}

type PutNoteRequestMeta struct {
	UserID int `meta:"user_id"`
	Year   int `meta:"year"`
	Month  int `meta:"month"`
}

type PutNoteResponse struct {
}

type GetNotesRequest struct {
	Meta GetNotesRequestMeta `meta:"meta" json:"-"`
}

type GetNotesRequestMeta struct {
	UserID int `meta:"user_id"`
	Year   int `meta:"year"`
}

type GetNotesResponse struct {
	Data map[int]Note `json:"data"`
}

type GetSettingsRequest struct {
	Meta GetSettingsRequestMeta `meta:"meta" json:"-"`
}

type GetSettingsRequestMeta struct {
	UserID int `meta:"user_id"`
}

type ThemePreferences struct {
	Theme            string `json:"theme"`
	WeatherEnabled   bool   `json:"weather_enabled"`
	TimeBasedEnabled bool   `json:"time_based_enabled"`
	Location         string `json:"location,omitempty"`
}

type SchedulePreferences struct {
	Monday    State `json:"monday"`
	Tuesday   State `json:"tuesday"`
	Wednesday State `json:"wednesday"`
	Thursday  State `json:"thursday"`
	Friday    State `json:"friday"`
	Saturday  State `json:"saturday"`
	Sunday    State `json:"sunday"`
}

type LinkedAccount struct {
	Provider        string `json:"provider"`
	ProviderDisplay string `json:"provider_display"`
	Nickname        string `json:"nickname"`
}

type GetSettingsResponse struct {
	LinkedAccounts      []LinkedAccount     `json:"linked_accounts"`
	ThemePreferences    ThemePreferences    `json:"theme_preferences"`
	SchedulePreferences SchedulePreferences `json:"schedule_preferences"`
}

type UpdateThemePreferencesRequest struct {
	Meta UpdateThemePreferencesRequestMeta `meta:"meta" json:"-"`
	Data ThemePreferences                  `json:"data"`
}

type UpdateThemePreferencesRequestMeta struct {
	UserID int `meta:"user_id"`
}

type UpdateThemePreferencesResponse struct{}

type UpdateSchedulePreferencesRequest struct {
	Meta UpdateSchedulePreferencesRequestMeta `meta:"meta" json:"-"`
	Data SchedulePreferences                  `json:"data"`
}

type UpdateSchedulePreferencesRequestMeta struct {
	UserID int `meta:"user_id"`
}

type UpdateSchedulePreferencesResponse struct{}

// Token management models
type PostSecretRequest struct {
	Meta PostSecretRequestMeta `meta:"meta" json:"-"`
	Data PostSecretRequestData `json:"data"`
}

type PostSecretRequestMeta struct {
	UserID int `meta:"user_id"`
}

type PostSecretRequestData struct {
	Name string `json:"name"`
}

type PostSecretResponse struct {
	Secret string `json:"secret"`
}

type ListTokensRequest struct {
	Meta ListTokensRequestMeta `meta:"meta" json:"-"`
}

type ListTokensRequestMeta struct {
	UserID int `meta:"user_id"`
}

type TokenInfo struct {
	TokenID   int    `json:"token_id"`
	Name      string `json:"name"`
	CreatedAt string `json:"created_at"`
}

type ListTokensResponse struct {
	Tokens []TokenInfo `json:"tokens"`
}

type RevokeTokenRequest struct {
	Meta RevokeTokenRequestMeta `meta:"meta" json:"-"`
}

type RevokeTokenRequestMeta struct {
	UserID  int `meta:"user_id"`
	TokenID int `meta:"token_id"`
}

type RevokeTokenResponse struct {
	Success bool `json:"success"`
}

type GetReportRequest struct {
	Meta GetReportRequestMeta `meta:"meta" json:"-"`
	Name string               `schema:"name"`
}

type GetReportRequestMeta struct {
	UserID int `meta:"user_id"`
	Year   int `meta:"year"`
}

type GetReportCSVRequest struct {
	Meta GetReportCSVRequestMeta `meta:"meta" json:"-"`
}

type GetReportCSVRequestMeta struct {
	UserID int `meta:"user_id"`
	Year   int `meta:"year"`
}

type HealthCheckRequest struct {
}

type HealthCheckResponse struct {
	Status string `json:"status"`
}

type ValidateAuthRequest struct {
	Meta ValidateAuthRequestMeta `meta:"meta" json:"-"`
}

type ValidateAuthRequestMeta struct {
	UserID int `meta:"user_id"`
}

type ValidateAuthResponse struct {
	Status string `json:"status"`
}

type Error struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}
