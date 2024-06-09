package model

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

type GetSecretRequest struct {
	Meta GetSecretRequestMeta `meta:"meta" json:"-"`
}

type GetSecretRequestMeta struct {
	UserID int `meta:"user_id"`
}

type GetSecretResponse struct {
	Secret string `json:"secret"`
}

type HealthCheckRequest struct {
}

type HealthCheckResponse struct {
}

type ValidateAuthRequest struct {
	Meta ValidateAuthRequestMeta `meta:"meta" json:"-"`
}

type ValidateAuthRequestMeta struct {
	UserID int `meta:"user_id"`
}

type ValidateAuthResponse struct {
}

type State int

const (
	StateUntracked = State(iota)
	StateWorkFromHome
	StateWorkFromOffice
	StateOther
)

type DayState struct {
	State State `json:"state"`
}

type MonthState struct {
	Days map[int]DayState `json:"days"`
}

type YearState struct {
	Months map[int]MonthState `json:"months"`
}
