package model

type Response struct {
	ContentType string
	Data        interface{}
}

type Service interface {
	// GetDay returns the state of a day
	GetDay(GetDayRequest) (GetDayResponse, error)
	// PutDay saves the state of a day
	PutDay(PutDayRequest) (PutDayResponse, error)
	// GetMonth returns the state of a month
	GetMonth(GetMonthRequest) (GetMonthResponse, error)
	// PutMonth saves the state of a month
	PutMonth(PutMonthRequest) (PutMonthResponse, error)
	// GetYear returns the state of a year
	GetYear(GetYearRequest) (GetYearResponse, error)
	// GetNote returns the note for a month
	GetNote(GetNoteRequest) (GetNoteResponse, error)
	// PutNote saves the note for a month
	PutNote(PutNoteRequest) (PutNoteResponse, error)
	// GetNotes returns the notes for a year
	GetNotes(request GetNotesRequest) (GetNotesResponse, error)

	// GetSecret returns a new secret
	GetSecret(GetSecretRequest) (GetSecretResponse, error)

	// GetReport returns a PDF report for the specified period
	GetReport(GetReportRequest) (Response, error)
	// GetReportCSV returns a CSV report for the specified period
	GetReportCSV(GetReportCSVRequest) (Response, error)

	// Healthcheck returns the status of the service
	Healthcheck(HealthCheckRequest) (HealthCheckResponse, error)
	// ValidateAuth validates the auth method
	ValidateAuth(ValidateAuthRequest) (ValidateAuthResponse, error)
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

type GetSecretRequest struct {
	Meta GetSecretRequestMeta `meta:"meta" json:"-"`
}

type GetSecretRequestMeta struct {
	UserID int `meta:"user_id"`
}

type GetSecretResponse struct {
	Secret string `json:"secret"`
}

type GetReportRequest struct {
	Meta GetReportRequestMeta `meta:"meta" json:"-"`
	Name string               `schema:"name"`
}

type GetReportRequestMeta struct {
	UserID int `meta:"user_id"`
}

type GetReportCSVRequest struct {
	Meta GetReportCSVRequestMeta `meta:"meta" json:"-"`
}

type GetReportCSVRequestMeta struct {
	UserID int `meta:"user_id"`
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
