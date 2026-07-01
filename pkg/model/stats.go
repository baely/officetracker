package model

// StatWidget is a generic, self-describing statistic intended for rendering on
// the public stats dashboard. The frontend builds a card purely from these
// fields, so new widgets can be added without any UI changes.
type StatWidget struct {
	// Key is a stable machine identifier for the widget (e.g. "mau_30d").
	Key string `json:"key"`
	// Title is the human-readable label (e.g. "Monthly Active Users").
	Title string `json:"title"`
	// Value is the formatted display value (e.g. "1,024" or "0.50").
	Value string `json:"value"`
	// Prefix is an optional string rendered immediately before the value with no
	// space (e.g. "$"). May be empty.
	Prefix string `json:"prefix,omitempty"`
	// Unit is an optional unit/suffix rendered after the value (e.g. "users",
	// "req"). May be empty.
	Unit string `json:"unit,omitempty"`
	// Group is an optional grouping label used to visually cluster widgets
	// (e.g. "Usage", "Cost"). May be empty.
	Group string `json:"group,omitempty"`
	// Order controls display ordering; lower values render first.
	Order int `json:"order"`
}

// GetStatsRequest is the (empty) request for the public stats endpoint.
type GetStatsRequest struct{}

// GetStatsResponse is the response for the public stats endpoint.
type GetStatsResponse struct {
	// Widgets is the ordered list of statistics to render.
	Widgets []StatWidget `json:"widgets"`
	// ComputedAt is the RFC3339 timestamp the snapshot was generated. Empty if
	// no snapshot exists yet.
	ComputedAt string `json:"computedAt,omitempty"`
}
