package services

import (
	"testing"
	"time"
)

func TestLoginCleanupPreservesProtection(t *testing.T) {
	now := time.Now()
	old := now.Add(-48 * time.Hour)
	active := now.Add(time.Hour)
	expired := now.Add(-time.Hour)
	s := AuthService{loginStates: map[int64]*LoginState{
		1: {LastSeen: old}, 2: {LastSeen: old, Attempts: 5, LockedUntil: &active}, 3: {LastSeen: old, Attempts: 2}, 4: {LastSeen: now}, 5: {LastSeen: old, Attempts: 5, LockedUntil: &expired},
	}}
	s.cleanupLoginStates(now)
	if len(s.loginStates) != 3 || s.loginStates[2] == nil || s.loginStates[3] == nil || s.loginStates[4] == nil {
		t.Fatal("cache cleanup changed live protection")
	}
	if s.loginStates[2].Attempts != 5 || s.loginStates[3].Attempts != 2 {
		t.Fatal("failed-attempt counters lost")
	}
}
