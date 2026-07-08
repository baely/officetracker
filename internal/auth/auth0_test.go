package auth

import (
	"errors"
	"testing"

	"github.com/baely/officetracker/internal/database"
	"github.com/baely/officetracker/internal/database/dbtest"
)

func TestParseAuth0Subject(t *testing.T) {
	cases := []struct {
		sub          string
		wantProvider string
		wantID       string
		wantErr      bool
	}{
		{"github|12345", "github", "12345", false},
		{"google-oauth2|abc", "google-oauth2", "abc", false},
		{"auth0|xyz", "auth0", "xyz", false},
		{"nopipe", "", "", true},
		{"a|b|c", "", "", true}, // more than two parts
		{"", "", "", true},
	}
	for _, c := range cases {
		provider, id, err := parseAuth0Subject(c.sub)
		if c.wantErr {
			if err == nil {
				t.Errorf("parseAuth0Subject(%q) expected error", c.sub)
			}
			continue
		}
		if err != nil {
			t.Errorf("parseAuth0Subject(%q) error: %v", c.sub, err)
			continue
		}
		if provider != c.wantProvider || id != c.wantID {
			t.Errorf("parseAuth0Subject(%q) = (%q,%q), want (%q,%q)", c.sub, provider, id, c.wantProvider, c.wantID)
		}
	}
}

// Tier 1: an existing Auth0 user is returned directly and their profile is
// refreshed.
func TestSubjectToUserIDExistingAuth0(t *testing.T) {
	db := dbtest.New()
	db.GetUserByAuth0SubFn = func(sub string) (int, error) {
		if sub == "github|1" {
			return 42, nil
		}
		return 0, database.ErrNoUser
	}
	uid, err := subjectToUserID(db, Profile{Sub: "github|1"})
	if err != nil || uid != 42 {
		t.Fatalf("subjectToUserID = (%d, %v), want (42, nil)", uid, err)
	}
	if len(db.UpdatedAuth0) != 1 {
		t.Errorf("expected profile refresh, got %d UpdateAuth0Profile calls", len(db.UpdatedAuth0))
	}
}

// Tier 2: a GitHub identity not yet in auth0_users but present in gh_users is
// migrated by linking the Auth0 identity.
func TestSubjectToUserIDGithubMigration(t *testing.T) {
	db := dbtest.New()
	db.GetUserByAuth0SubFn = func(string) (int, error) { return 0, database.ErrNoUser }
	db.GetUserByGHIDFn = func(ghID string) (int, error) {
		if ghID == "999" {
			return 7, nil
		}
		return 0, database.ErrNoUser
	}
	uid, err := subjectToUserID(db, Profile{Sub: "github|999"})
	if err != nil || uid != 7 {
		t.Fatalf("subjectToUserID = (%d, %v), want (7, nil)", uid, err)
	}
	if len(db.LinkedAuth0) != 1 || db.LinkedAuth0[0].UserID != 7 {
		t.Errorf("expected github user to be linked to auth0, got %v", db.LinkedAuth0)
	}
}

// Tier 3: a brand-new identity is created.
func TestSubjectToUserIDNewSignup(t *testing.T) {
	db := dbtest.New()
	db.GetUserByAuth0SubFn = func(string) (int, error) { return 0, database.ErrNoUser }
	db.GetUserByGHIDFn = func(string) (int, error) { return 0, database.ErrNoUser }
	db.SaveUserByAuth0SubFn = func(sub, profile string) (int, error) { return 100, nil }

	uid, err := subjectToUserID(db, Profile{Sub: "google-oauth2|new"})
	if err != nil || uid != 100 {
		t.Fatalf("subjectToUserID = (%d, %v), want (100, nil)", uid, err)
	}
}

// A non-ErrNoUser error from the first lookup is surfaced, not swallowed.
func TestSubjectToUserIDLookupError(t *testing.T) {
	db := dbtest.New()
	db.GetUserByAuth0SubFn = func(string) (int, error) { return 0, errors.New("db down") }
	if _, err := subjectToUserID(db, Profile{Sub: "github|1"}); err == nil {
		t.Fatal("expected the db error to propagate")
	}
}

// A malformed subject (no provider separator) fails once it reaches the parse
// step in tier 2.
func TestSubjectToUserIDBadSubject(t *testing.T) {
	db := dbtest.New()
	db.GetUserByAuth0SubFn = func(string) (int, error) { return 0, database.ErrNoUser }
	if _, err := subjectToUserID(db, Profile{Sub: "malformed-no-pipe"}); err == nil {
		t.Fatal("expected error for malformed subject")
	}
}
