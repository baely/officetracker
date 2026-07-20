// Package dbtest provides an in-memory implementation of database.Databaser for
// use in unit tests. It stores day/note/preference data in maps so that
// higher-level code (report generation, the v1 service, auth) can be exercised
// end-to-end without a real database, and it supports per-method error
// injection and call recording for testing failure paths.
//
// It is imported only from _test.go files, so it is never linked into the
// production binaries.
package dbtest

import (
	"time"

	"github.com/baely/officetracker/internal/database"
	"github.com/baely/officetracker/internal/util"
	"github.com/baely/officetracker/pkg/model"
)

// compile-time assertion that Fake satisfies the interface.
var _ database.Databaser = (*Fake)(nil)

type dayKey struct {
	year, month, day int
}

type monthKey struct {
	year, month int
}

// Fake is a configurable in-memory Databaser.
//
// Construct one with New(): it starts with no entries, default theme
// preferences and an October tracking-year start. Callers may seed data with
// the Save* methods and inject errors via Errs.
type Fake struct {
	days  map[dayKey]model.DayState
	notes map[monthKey]model.Note

	theme  model.ThemePreferences
	sched  model.SchedulePreferences
	cal    model.CalendarPreferences
	target model.TargetPreferences

	// LinkedAccounts is returned verbatim by GetUserLinkedAccounts.
	LinkedAccounts []model.LinkedAccount
	// Tokens is returned verbatim by ListActiveTokens.
	Tokens []database.TokenMetadata
	// Snapshot / statsTime are returned by GetLatestStatsSnapshot.
	Snapshot  []model.StatWidget
	statsTime time.Time

	// Suspended is returned by IsUserSuspended.
	Suspended bool

	// ExportTables is returned verbatim by ExportUserData.
	ExportTables []model.ExportTable

	// User-resolution hooks. When nil a sensible default is used
	// (see the individual methods).
	GetUserBySecretFn    func(secret string) (int, error)
	GetUserByGHIDFn      func(ghID string) (int, error)
	GetUserByAuth0SubFn  func(sub string) (int, error)
	SaveUserByAuth0SubFn func(sub, profile string) (int, error)

	// Recorded mutating calls, for assertions.
	SavedSecrets   []SavedSecret
	SavedSnapshots [][]model.StatWidget
	RevokedTokens  []RevokedToken
	LinkedAuth0    []LinkedAuth0Call
	UpdatedAuth0   []UpdatedAuth0Call

	// Errs injects an error for the named method (the exact Go method name,
	// e.g. "GetDay"). When present the method returns its zero value and the
	// error before doing any work.
	Errs map[string]error
}

// SavedSecret records a SaveSecret call.
type SavedSecret struct {
	UserID int
	Secret string
	Name   string
}

// RevokedToken records a RevokeToken call.
type RevokedToken struct {
	UserID  int
	TokenID int
}

// LinkedAuth0Call records a LinkAuth0Account call.
type LinkedAuth0Call struct {
	UserID  int
	Sub     string
	Profile string
}

// UpdatedAuth0Call records an UpdateAuth0Profile call.
type UpdatedAuth0Call struct {
	Sub     string
	Profile string
}

// New returns an empty Fake ready for use.
func New() *Fake {
	return &Fake{
		days:  make(map[dayKey]model.DayState),
		notes: make(map[monthKey]model.Note),
		theme: model.ThemePreferences{Theme: "default"},
		cal:   model.CalendarPreferences{TrackingYearStartMonth: model.DefaultTrackingYearStartMonth},
	}
}

func (f *Fake) fail(method string) error {
	if f.Errs == nil {
		return nil
	}
	return f.Errs[method]
}

// SetStatsTime sets the timestamp returned by GetLatestStatsSnapshot.
func (f *Fake) SetStatsTime(t time.Time) { f.statsTime = t }

func (f *Fake) SaveDay(_ int, day, month, year int, state model.DayState) error {
	if err := f.fail("SaveDay"); err != nil {
		return err
	}
	f.days[dayKey{year, month, day}] = state
	return nil
}

func (f *Fake) GetDay(_ int, day, month, year int) (model.DayState, error) {
	if err := f.fail("GetDay"); err != nil {
		return model.DayState{}, err
	}
	return f.days[dayKey{year, month, day}], nil
}

func (f *Fake) SaveMonth(_ int, month, year int, state model.MonthState) error {
	if err := f.fail("SaveMonth"); err != nil {
		return err
	}
	for day, ds := range state.Days {
		f.days[dayKey{year, month, day}] = ds
	}
	return nil
}

func (f *Fake) GetMonth(_ int, month, year int) (model.MonthState, error) {
	if err := f.fail("GetMonth"); err != nil {
		return model.MonthState{}, err
	}
	ms := model.MonthState{Days: make(map[int]model.DayState)}
	for k, v := range f.days {
		if k.year == year && k.month == month {
			ms.Days[k.day] = v
		}
	}
	return ms, nil
}

func inWindow(entryYear, entryMonth, firstYear, secondYear, startMonth int) bool {
	if entryYear == firstYear && entryMonth >= startMonth {
		return true
	}
	if entryYear == secondYear && entryMonth < startMonth {
		return true
	}
	return false
}

func (f *Fake) GetYear(_ int, year, startMonth int) (model.YearState, error) {
	if err := f.fail("GetYear"); err != nil {
		return model.YearState{}, err
	}
	startMonth = util.NormaliseStartMonth(startMonth)
	firstYear, secondYear := util.TrackingYearCalendarYears(year, startMonth)
	ys := model.YearState{Months: make(map[int]model.MonthState)}
	for k, v := range f.days {
		if !inWindow(k.year, k.month, firstYear, secondYear, startMonth) {
			continue
		}
		ms, ok := ys.Months[k.month]
		if !ok {
			ms = model.MonthState{Days: make(map[int]model.DayState)}
			ys.Months[k.month] = ms
		}
		ms.Days[k.day] = v
	}
	return ys, nil
}

func (f *Fake) SaveNote(_ int, month, year int, note string) error {
	if err := f.fail("SaveNote"); err != nil {
		return err
	}
	f.notes[monthKey{year, month}] = model.Note{Note: note}
	return nil
}

func (f *Fake) GetNote(_ int, month, year int) (model.Note, error) {
	if err := f.fail("GetNote"); err != nil {
		return model.Note{}, err
	}
	return f.notes[monthKey{year, month}], nil
}

func (f *Fake) GetNotes(_ int, year, startMonth int) (map[int]model.Note, error) {
	if err := f.fail("GetNotes"); err != nil {
		return nil, err
	}
	startMonth = util.NormaliseStartMonth(startMonth)
	firstYear, secondYear := util.TrackingYearCalendarYears(year, startMonth)
	out := make(map[int]model.Note)
	for k, v := range f.notes {
		if inWindow(k.year, k.month, firstYear, secondYear, startMonth) {
			out[k.month] = v
		}
	}
	return out, nil
}

func (f *Fake) GetUserByGHID(ghID string) (int, error) {
	if err := f.fail("GetUserByGHID"); err != nil {
		return 0, err
	}
	if f.GetUserByGHIDFn != nil {
		return f.GetUserByGHIDFn(ghID)
	}
	return 0, database.ErrNoUser
}

func (f *Fake) GetUserBySecret(secret string) (int, error) {
	if err := f.fail("GetUserBySecret"); err != nil {
		return 0, err
	}
	if f.GetUserBySecretFn != nil {
		return f.GetUserBySecretFn(secret)
	}
	return 0, database.ErrNoUser
}

func (f *Fake) GetUserLinkedAccounts(_ int) ([]model.LinkedAccount, error) {
	if err := f.fail("GetUserLinkedAccounts"); err != nil {
		return nil, err
	}
	return f.LinkedAccounts, nil
}

func (f *Fake) GetUserByAuth0Sub(sub string) (int, error) {
	if err := f.fail("GetUserByAuth0Sub"); err != nil {
		return 0, err
	}
	if f.GetUserByAuth0SubFn != nil {
		return f.GetUserByAuth0SubFn(sub)
	}
	return 0, database.ErrNoUser
}

func (f *Fake) SaveUserByAuth0Sub(sub, profile string) (int, error) {
	if err := f.fail("SaveUserByAuth0Sub"); err != nil {
		return 0, err
	}
	if f.SaveUserByAuth0SubFn != nil {
		return f.SaveUserByAuth0SubFn(sub, profile)
	}
	return 0, nil
}

func (f *Fake) UpdateAuth0Profile(sub, profile string) error {
	if err := f.fail("UpdateAuth0Profile"); err != nil {
		return err
	}
	f.UpdatedAuth0 = append(f.UpdatedAuth0, UpdatedAuth0Call{Sub: sub, Profile: profile})
	return nil
}

func (f *Fake) LinkAuth0Account(userID int, sub, profile string) error {
	if err := f.fail("LinkAuth0Account"); err != nil {
		return err
	}
	f.LinkedAuth0 = append(f.LinkedAuth0, LinkedAuth0Call{UserID: userID, Sub: sub, Profile: profile})
	return nil
}

func (f *Fake) GetThemePreferences(_ int) (model.ThemePreferences, error) {
	if err := f.fail("GetThemePreferences"); err != nil {
		return model.ThemePreferences{}, err
	}
	return f.theme, nil
}

func (f *Fake) SaveThemePreferences(_ int, prefs model.ThemePreferences) error {
	if err := f.fail("SaveThemePreferences"); err != nil {
		return err
	}
	f.theme = prefs
	return nil
}

func (f *Fake) GetSchedulePreferences(_ int) (model.SchedulePreferences, error) {
	if err := f.fail("GetSchedulePreferences"); err != nil {
		return model.SchedulePreferences{}, err
	}
	return f.sched, nil
}

func (f *Fake) SaveSchedulePreferences(_ int, prefs model.SchedulePreferences) error {
	if err := f.fail("SaveSchedulePreferences"); err != nil {
		return err
	}
	f.sched = prefs
	return nil
}

func (f *Fake) GetCalendarPreferences(_ int) (model.CalendarPreferences, error) {
	if err := f.fail("GetCalendarPreferences"); err != nil {
		return model.CalendarPreferences{}, err
	}
	return f.cal, nil
}

func (f *Fake) SaveCalendarPreferences(_ int, prefs model.CalendarPreferences) error {
	if err := f.fail("SaveCalendarPreferences"); err != nil {
		return err
	}
	f.cal = prefs
	return nil
}

func (f *Fake) GetTargetPreferences(_ int) (model.TargetPreferences, error) {
	if err := f.fail("GetTargetPreferences"); err != nil {
		return model.TargetPreferences{}, err
	}
	return f.target, nil
}

func (f *Fake) SaveTargetPreferences(_ int, prefs model.TargetPreferences) error {
	if err := f.fail("SaveTargetPreferences"); err != nil {
		return err
	}
	f.target = prefs
	return nil
}

func (f *Fake) SaveSecret(userID int, secret, name string) error {
	if err := f.fail("SaveSecret"); err != nil {
		return err
	}
	f.SavedSecrets = append(f.SavedSecrets, SavedSecret{UserID: userID, Secret: secret, Name: name})
	return nil
}

func (f *Fake) ListActiveTokens(_ int) ([]database.TokenMetadata, error) {
	if err := f.fail("ListActiveTokens"); err != nil {
		return nil, err
	}
	return f.Tokens, nil
}

func (f *Fake) RevokeToken(userID, tokenID int) error {
	if err := f.fail("RevokeToken"); err != nil {
		return err
	}
	f.RevokedTokens = append(f.RevokedTokens, RevokedToken{UserID: userID, TokenID: tokenID})
	return nil
}

func (f *Fake) RevokeSecretByValue(_ string) error {
	return f.fail("RevokeSecretByValue")
}

func (f *Fake) IsUserSuspended(_ int) (bool, error) {
	if err := f.fail("IsUserSuspended"); err != nil {
		return false, err
	}
	return f.Suspended, nil
}

func (f *Fake) ExportUserData(_ int) ([]model.ExportTable, error) {
	if err := f.fail("ExportUserData"); err != nil {
		return nil, err
	}
	return f.ExportTables, nil
}

func (f *Fake) SaveStatsSnapshot(widgets []model.StatWidget) error {
	if err := f.fail("SaveStatsSnapshot"); err != nil {
		return err
	}
	f.SavedSnapshots = append(f.SavedSnapshots, widgets)
	return nil
}

func (f *Fake) GetLatestStatsSnapshot() ([]model.StatWidget, time.Time, error) {
	if err := f.fail("GetLatestStatsSnapshot"); err != nil {
		return nil, time.Time{}, err
	}
	return f.Snapshot, f.statsTime, nil
}

func (f *Fake) CountTrackedDays() (int, error) {
	if err := f.fail("CountTrackedDays"); err != nil {
		return 0, err
	}
	count := 0
	for _, v := range f.days {
		if v.State != model.StateUntracked {
			count++
		}
	}
	return count, nil
}

func (f *Fake) CountEntriesByState() (map[model.State]int, error) {
	if err := f.fail("CountEntriesByState"); err != nil {
		return nil, err
	}
	out := make(map[model.State]int)
	for _, v := range f.days {
		if v.State != model.StateUntracked {
			out[v.State]++
		}
	}
	return out, nil
}
