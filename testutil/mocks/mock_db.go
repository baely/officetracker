package mocks

import (
	"fmt"
	"sync"

	"github.com/baely/officetracker/internal/database"
	"github.com/baely/officetracker/pkg/model"
)

// MockDB is a full in-memory implementation of the Databaser interface for testing
type MockDB struct {
	mu sync.RWMutex

	// In-memory storage
	days               map[string]model.DayState              // key: "userID:day:month:year"
	months             map[string]model.MonthState            // key: "userID:month:year"
	years              map[string]model.YearState             // key: "userID:year"
	notes              map[string]model.Note                  // key: "userID:month:year"
	auth0Users         map[string]int                         // auth0 sub → userID
	ghUsers            map[string]int                         // GitHub ID → userID
	secrets            map[string]int                         // secret → userID
	linkedAccounts     map[int][]model.LinkedAccount          // userID → accounts
	themePrefs         map[int]model.ThemePreferences         // userID → theme
	schedulePrefs      map[int]model.SchedulePreferences      // userID → schedule
	suspendedUsers     map[int]bool                           // userID → suspended
	auth0Profiles      map[string]string                      // auth0 sub → profile JSON

	// Auto-increment for new users
	nextUserID int

	// Error injection - set to trigger errors on specific operations
	ErrorOnSaveDay               error
	ErrorOnGetDay                error
	ErrorOnSaveMonth             error
	ErrorOnGetMonth              error
	ErrorOnGetYear               error
	ErrorOnSaveNote              error
	ErrorOnGetNote               error
	ErrorOnGetNotes              error
	ErrorOnGetUserByGHID         error
	ErrorOnGetUserBySecret       error
	ErrorOnGetUserLinkedAccounts error
	ErrorOnGetUserByAuth0Sub     error
	ErrorOnSaveUserByAuth0Sub    error
	ErrorOnUpdateAuth0Profile    error
	ErrorOnLinkAuth0Account      error
	ErrorOnGetThemePreferences   error
	ErrorOnSaveThemePreferences  error
	ErrorOnGetSchedulePreferences error
	ErrorOnSaveSchedulePreferences error
	ErrorOnSaveSecret            error
	ErrorOnIsUserSuspended       error

	// Call tracking
	SaveDayCalls                []SaveDayCall
	GetDayCalls                 []GetDayCall
	SaveMonthCalls              []SaveMonthCall
	GetMonthCalls               []GetMonthCall
	GetYearCalls                []GetYearCall
	SaveNoteCalls               []SaveNoteCall
	GetNoteCalls                []GetNoteCall
	GetNotesCalls               []GetNotesCall
	GetUserByGHIDCalls          []GetUserByGHIDCall
	GetUserBySecretCalls        []GetUserBySecretCall
	GetUserLinkedAccountsCalls  []GetUserLinkedAccountsCall
	GetUserByAuth0SubCalls      []GetUserByAuth0SubCall
	SaveUserByAuth0SubCalls     []SaveUserByAuth0SubCall
	UpdateAuth0ProfileCalls     []UpdateAuth0ProfileCall
	LinkAuth0AccountCalls       []LinkAuth0AccountCall
	GetThemePreferencesCalls    []GetThemePreferencesCall
	SaveThemePreferencesCalls   []SaveThemePreferencesCall
	GetSchedulePreferencesCalls []GetSchedulePreferencesCall
	SaveSchedulePreferencesCalls []SaveSchedulePreferencesCall
	SaveSecretCalls             []SaveSecretCall
	IsUserSuspendedCalls        []IsUserSuspendedCall
}

// Call tracking structures
type SaveDayCall struct {
	UserID int
	Day    int
	Month  int
	Year   int
	State  model.DayState
}

type GetDayCall struct {
	UserID int
	Day    int
	Month  int
	Year   int
}

type SaveMonthCall struct {
	UserID int
	Month  int
	Year   int
	State  model.MonthState
}

type GetMonthCall struct {
	UserID int
	Month  int
	Year   int
}

type GetYearCall struct {
	UserID int
	Year   int
}

type SaveNoteCall struct {
	UserID int
	Month  int
	Year   int
	Note   string
}

type GetNoteCall struct {
	UserID int
	Month  int
	Year   int
}

type GetNotesCall struct {
	UserID int
	Year   int
}

type GetUserByGHIDCall struct {
	GHID string
}

type GetUserBySecretCall struct {
	Secret string
}

type GetUserLinkedAccountsCall struct {
	UserID int
}

type GetUserByAuth0SubCall struct {
	Sub string
}

type SaveUserByAuth0SubCall struct {
	Sub     string
	Profile string
}

type UpdateAuth0ProfileCall struct {
	Sub     string
	Profile string
}

type LinkAuth0AccountCall struct {
	UserID  int
	Sub     string
	Profile string
}

type GetThemePreferencesCall struct {
	UserID int
}

type SaveThemePreferencesCall struct {
	UserID int
	Prefs  model.ThemePreferences
}

type GetSchedulePreferencesCall struct {
	UserID int
}

type SaveSchedulePreferencesCall struct {
	UserID int
	Prefs  model.SchedulePreferences
}

type SaveSecretCall struct {
	UserID int
	Secret string
}

type IsUserSuspendedCall struct {
	UserID int
}

// NewMockDB creates a new MockDB instance
func NewMockDB() *MockDB {
	return &MockDB{
		days:            make(map[string]model.DayState),
		months:          make(map[string]model.MonthState),
		years:           make(map[string]model.YearState),
		notes:           make(map[string]model.Note),
		auth0Users:      make(map[string]int),
		ghUsers:         make(map[string]int),
		secrets:         make(map[string]int),
		linkedAccounts:  make(map[int][]model.LinkedAccount),
		themePrefs:      make(map[int]model.ThemePreferences),
		schedulePrefs:   make(map[int]model.SchedulePreferences),
		suspendedUsers:  make(map[int]bool),
		auth0Profiles:   make(map[string]string),
		nextUserID:      1,
	}
}

// Helper methods for test setup
func (m *MockDB) SetNextUserID(id int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.nextUserID = id
}

func (m *MockDB) AddUser(userID int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	// Just ensure user exists in our tracking
	if userID >= m.nextUserID {
		m.nextUserID = userID + 1
	}
}

func (m *MockDB) AddAuth0User(sub string, userID int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.auth0Users[sub] = userID
	if userID >= m.nextUserID {
		m.nextUserID = userID + 1
	}
}

func (m *MockDB) AddGitHubUser(ghID string, userID int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.ghUsers[ghID] = userID
	if userID >= m.nextUserID {
		m.nextUserID = userID + 1
	}
}

func (m *MockDB) SetUserSuspended(userID int, suspended bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.suspendedUsers[userID] = suspended
}

func (m *MockDB) SetDay(userID, day, month, year int, state model.DayState) {
	m.mu.Lock()
	defer m.mu.Unlock()
	key := fmt.Sprintf("%d:%d:%d:%d", userID, day, month, year)
	m.days[key] = state
}

func (m *MockDB) SetSchedulePreferences(userID int, prefs model.SchedulePreferences) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.schedulePrefs[userID] = prefs
}

// Reset clears all data and call tracking
func (m *MockDB) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.days = make(map[string]model.DayState)
	m.months = make(map[string]model.MonthState)
	m.years = make(map[string]model.YearState)
	m.notes = make(map[string]model.Note)
	m.auth0Users = make(map[string]int)
	m.ghUsers = make(map[string]int)
	m.secrets = make(map[string]int)
	m.linkedAccounts = make(map[int][]model.LinkedAccount)
	m.themePrefs = make(map[int]model.ThemePreferences)
	m.schedulePrefs = make(map[int]model.SchedulePreferences)
	m.suspendedUsers = make(map[int]bool)
	m.auth0Profiles = make(map[string]string)
	m.nextUserID = 1

	// Clear all errors
	m.ErrorOnSaveDay = nil
	m.ErrorOnGetDay = nil
	m.ErrorOnSaveMonth = nil
	m.ErrorOnGetMonth = nil
	m.ErrorOnGetYear = nil
	m.ErrorOnSaveNote = nil
	m.ErrorOnGetNote = nil
	m.ErrorOnGetNotes = nil
	m.ErrorOnGetUserByGHID = nil
	m.ErrorOnGetUserBySecret = nil
	m.ErrorOnGetUserLinkedAccounts = nil
	m.ErrorOnGetUserByAuth0Sub = nil
	m.ErrorOnSaveUserByAuth0Sub = nil
	m.ErrorOnUpdateAuth0Profile = nil
	m.ErrorOnLinkAuth0Account = nil
	m.ErrorOnGetThemePreferences = nil
	m.ErrorOnSaveThemePreferences = nil
	m.ErrorOnGetSchedulePreferences = nil
	m.ErrorOnSaveSchedulePreferences = nil
	m.ErrorOnSaveSecret = nil
	m.ErrorOnIsUserSuspended = nil

	// Clear all call tracking
	m.SaveDayCalls = nil
	m.GetDayCalls = nil
	m.SaveMonthCalls = nil
	m.GetMonthCalls = nil
	m.GetYearCalls = nil
	m.SaveNoteCalls = nil
	m.GetNoteCalls = nil
	m.GetNotesCalls = nil
	m.GetUserByGHIDCalls = nil
	m.GetUserBySecretCalls = nil
	m.GetUserLinkedAccountsCalls = nil
	m.GetUserByAuth0SubCalls = nil
	m.SaveUserByAuth0SubCalls = nil
	m.UpdateAuth0ProfileCalls = nil
	m.LinkAuth0AccountCalls = nil
	m.GetThemePreferencesCalls = nil
	m.SaveThemePreferencesCalls = nil
	m.GetSchedulePreferencesCalls = nil
	m.SaveSchedulePreferencesCalls = nil
	m.SaveSecretCalls = nil
	m.IsUserSuspendedCalls = nil
}

// Databaser interface implementation

func (m *MockDB) SaveDay(userID int, day int, month int, year int, state model.DayState) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.SaveDayCalls = append(m.SaveDayCalls, SaveDayCall{UserID: userID, Day: day, Month: month, Year: year, State: state})

	if m.ErrorOnSaveDay != nil {
		return m.ErrorOnSaveDay
	}

	key := fmt.Sprintf("%d:%d:%d:%d", userID, day, month, year)
	m.days[key] = state
	return nil
}

func (m *MockDB) GetDay(userID int, day int, month int, year int) (model.DayState, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	m.GetDayCalls = append(m.GetDayCalls, GetDayCall{UserID: userID, Day: day, Month: month, Year: year})

	if m.ErrorOnGetDay != nil {
		return model.DayState{}, m.ErrorOnGetDay
	}

	key := fmt.Sprintf("%d:%d:%d:%d", userID, day, month, year)
	state, exists := m.days[key]
	if !exists {
		return model.DayState{State: model.StateUntracked}, nil
	}
	return state, nil
}

func (m *MockDB) SaveMonth(userID int, month int, year int, state model.MonthState) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.SaveMonthCalls = append(m.SaveMonthCalls, SaveMonthCall{UserID: userID, Month: month, Year: year, State: state})

	if m.ErrorOnSaveMonth != nil {
		return m.ErrorOnSaveMonth
	}

	// Save each day individually
	for day, dayState := range state.Days {
		key := fmt.Sprintf("%d:%d:%d:%d", userID, day, month, year)
		m.days[key] = dayState
	}

	key := fmt.Sprintf("%d:%d:%d", userID, month, year)
	m.months[key] = state
	return nil
}

func (m *MockDB) GetMonth(userID int, month int, year int) (model.MonthState, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	m.GetMonthCalls = append(m.GetMonthCalls, GetMonthCall{UserID: userID, Month: month, Year: year})

	if m.ErrorOnGetMonth != nil {
		return model.MonthState{}, m.ErrorOnGetMonth
	}

	// Collect all days for this month
	monthState := model.MonthState{
		Days: make(map[int]model.DayState),
	}

	for day := 1; day <= 31; day++ {
		key := fmt.Sprintf("%d:%d:%d:%d", userID, day, month, year)
		if dayState, exists := m.days[key]; exists {
			monthState.Days[day] = dayState
		}
	}

	return monthState, nil
}

func (m *MockDB) GetYear(userID int, year int) (model.YearState, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	m.GetYearCalls = append(m.GetYearCalls, GetYearCall{UserID: userID, Year: year})

	if m.ErrorOnGetYear != nil {
		return model.YearState{}, m.ErrorOnGetYear
	}

	yearState := model.YearState{
		Months: make(map[int]model.MonthState),
	}

	// Get all months for this year (academic year: Oct year-1 to Sep year)
	for month := 1; month <= 12; month++ {
		monthState := model.MonthState{
			Days: make(map[int]model.DayState),
		}

		for day := 1; day <= 31; day++ {
			key := fmt.Sprintf("%d:%d:%d:%d", userID, day, month, year)
			if dayState, exists := m.days[key]; exists {
				monthState.Days[day] = dayState
			}
		}

		if len(monthState.Days) > 0 {
			yearState.Months[month] = monthState
		}
	}

	return yearState, nil
}

func (m *MockDB) SaveNote(userID int, month int, year int, note string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.SaveNoteCalls = append(m.SaveNoteCalls, SaveNoteCall{UserID: userID, Month: month, Year: year, Note: note})

	if m.ErrorOnSaveNote != nil {
		return m.ErrorOnSaveNote
	}

	key := fmt.Sprintf("%d:%d:%d", userID, month, year)
	m.notes[key] = model.Note{Note: note}
	return nil
}

func (m *MockDB) GetNote(userID int, month int, year int) (model.Note, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	m.GetNoteCalls = append(m.GetNoteCalls, GetNoteCall{UserID: userID, Month: month, Year: year})

	if m.ErrorOnGetNote != nil {
		return model.Note{}, m.ErrorOnGetNote
	}

	key := fmt.Sprintf("%d:%d:%d", userID, month, year)
	note, exists := m.notes[key]
	if !exists {
		return model.Note{}, nil
	}
	return note, nil
}

func (m *MockDB) GetNotes(userID int, year int) (map[int]model.Note, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	m.GetNotesCalls = append(m.GetNotesCalls, GetNotesCall{UserID: userID, Year: year})

	if m.ErrorOnGetNotes != nil {
		return nil, m.ErrorOnGetNotes
	}

	notes := make(map[int]model.Note)
	for month := 1; month <= 12; month++ {
		key := fmt.Sprintf("%d:%d:%d", userID, month, year)
		if note, exists := m.notes[key]; exists && note.Note != "" {
			notes[month] = note
		}
	}

	return notes, nil
}

func (m *MockDB) GetUserByGHID(ghID string) (int, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	m.GetUserByGHIDCalls = append(m.GetUserByGHIDCalls, GetUserByGHIDCall{GHID: ghID})

	if m.ErrorOnGetUserByGHID != nil {
		return 0, m.ErrorOnGetUserByGHID
	}

	userID, exists := m.ghUsers[ghID]
	if !exists {
		return 0, database.ErrNoUser
	}
	return userID, nil
}

func (m *MockDB) GetUserBySecret(secret string) (int, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	m.GetUserBySecretCalls = append(m.GetUserBySecretCalls, GetUserBySecretCall{Secret: secret})

	if m.ErrorOnGetUserBySecret != nil {
		return 0, m.ErrorOnGetUserBySecret
	}

	userID, exists := m.secrets[secret]
	if !exists {
		return 0, database.ErrNoUser
	}
	return userID, nil
}

func (m *MockDB) GetUserLinkedAccounts(userID int) ([]model.LinkedAccount, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	m.GetUserLinkedAccountsCalls = append(m.GetUserLinkedAccountsCalls, GetUserLinkedAccountsCall{UserID: userID})

	if m.ErrorOnGetUserLinkedAccounts != nil {
		return nil, m.ErrorOnGetUserLinkedAccounts
	}

	accounts := m.linkedAccounts[userID]
	if accounts == nil {
		return []model.LinkedAccount{}, nil
	}
	return accounts, nil
}

func (m *MockDB) GetUserByAuth0Sub(sub string) (int, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	m.GetUserByAuth0SubCalls = append(m.GetUserByAuth0SubCalls, GetUserByAuth0SubCall{Sub: sub})

	if m.ErrorOnGetUserByAuth0Sub != nil {
		return 0, m.ErrorOnGetUserByAuth0Sub
	}

	userID, exists := m.auth0Users[sub]
	if !exists {
		return 0, database.ErrNoUser
	}
	return userID, nil
}

func (m *MockDB) SaveUserByAuth0Sub(sub string, profile string) (int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.SaveUserByAuth0SubCalls = append(m.SaveUserByAuth0SubCalls, SaveUserByAuth0SubCall{Sub: sub, Profile: profile})

	if m.ErrorOnSaveUserByAuth0Sub != nil {
		return 0, m.ErrorOnSaveUserByAuth0Sub
	}

	userID := m.nextUserID
	m.nextUserID++
	m.auth0Users[sub] = userID
	m.auth0Profiles[sub] = profile
	return userID, nil
}

func (m *MockDB) UpdateAuth0Profile(sub string, profile string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.UpdateAuth0ProfileCalls = append(m.UpdateAuth0ProfileCalls, UpdateAuth0ProfileCall{Sub: sub, Profile: profile})

	if m.ErrorOnUpdateAuth0Profile != nil {
		return m.ErrorOnUpdateAuth0Profile
	}

	if _, exists := m.auth0Users[sub]; !exists {
		return database.ErrNoUser
	}

	m.auth0Profiles[sub] = profile
	return nil
}

func (m *MockDB) LinkAuth0Account(userID int, sub string, profile string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.LinkAuth0AccountCalls = append(m.LinkAuth0AccountCalls, LinkAuth0AccountCall{UserID: userID, Sub: sub, Profile: profile})

	if m.ErrorOnLinkAuth0Account != nil {
		return m.ErrorOnLinkAuth0Account
	}

	// Check if this sub is already linked to a different user
	if existingUserID, exists := m.auth0Users[sub]; exists && existingUserID != userID {
		return fmt.Errorf("auth0 account already associated with another user")
	}

	m.auth0Users[sub] = userID
	m.auth0Profiles[sub] = profile
	return nil
}

func (m *MockDB) GetThemePreferences(userID int) (model.ThemePreferences, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	m.GetThemePreferencesCalls = append(m.GetThemePreferencesCalls, GetThemePreferencesCall{UserID: userID})

	if m.ErrorOnGetThemePreferences != nil {
		return model.ThemePreferences{}, m.ErrorOnGetThemePreferences
	}

	prefs, exists := m.themePrefs[userID]
	if !exists {
		return model.ThemePreferences{}, nil
	}
	return prefs, nil
}

func (m *MockDB) SaveThemePreferences(userID int, prefs model.ThemePreferences) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.SaveThemePreferencesCalls = append(m.SaveThemePreferencesCalls, SaveThemePreferencesCall{UserID: userID, Prefs: prefs})

	if m.ErrorOnSaveThemePreferences != nil {
		return m.ErrorOnSaveThemePreferences
	}

	m.themePrefs[userID] = prefs
	return nil
}

func (m *MockDB) GetSchedulePreferences(userID int) (model.SchedulePreferences, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	m.GetSchedulePreferencesCalls = append(m.GetSchedulePreferencesCalls, GetSchedulePreferencesCall{UserID: userID})

	if m.ErrorOnGetSchedulePreferences != nil {
		return model.SchedulePreferences{}, m.ErrorOnGetSchedulePreferences
	}

	prefs, exists := m.schedulePrefs[userID]
	if !exists {
		return model.SchedulePreferences{}, nil
	}
	return prefs, nil
}

func (m *MockDB) SaveSchedulePreferences(userID int, prefs model.SchedulePreferences) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.SaveSchedulePreferencesCalls = append(m.SaveSchedulePreferencesCalls, SaveSchedulePreferencesCall{UserID: userID, Prefs: prefs})

	if m.ErrorOnSaveSchedulePreferences != nil {
		return m.ErrorOnSaveSchedulePreferences
	}

	m.schedulePrefs[userID] = prefs
	return nil
}

func (m *MockDB) SaveSecret(userID int, secret string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.SaveSecretCalls = append(m.SaveSecretCalls, SaveSecretCall{UserID: userID, Secret: secret})

	if m.ErrorOnSaveSecret != nil {
		return m.ErrorOnSaveSecret
	}

	m.secrets[secret] = userID
	return nil
}

func (m *MockDB) IsUserSuspended(userID int) (bool, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	m.IsUserSuspendedCalls = append(m.IsUserSuspendedCalls, IsUserSuspendedCall{UserID: userID})

	if m.ErrorOnIsUserSuspended != nil {
		return false, m.ErrorOnIsUserSuspended
	}

	suspended, exists := m.suspendedUsers[userID]
	if !exists {
		return false, nil
	}
	return suspended, nil
}

// Verify interface compliance
var _ database.Databaser = (*MockDB)(nil)
