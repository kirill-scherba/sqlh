// Copyright 2026 Kirill Scherba <kirill@scherba.ru>. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package sqlh

import (
	"database/sql"
	"errors"
	"fmt"
	"testing"
)

// TestIsLockError verifies the lock-detection helper recognises the messages
// produced by the supported SQLite drivers (and ignores unrelated errors).
// It also confirms wrapped errors are detected, which the previous exact
// string match did not handle.
func TestIsLockError(t *testing.T) {
	tests := []struct {
		name   string
		err    error
		expect bool
	}{
		{"nil", nil, false},
		{"unrelated", errors.New("syntax error near 'select'"), false},
		{"locked plain", errors.New("database is locked"), true},
		{"locked wrapped", fmt.Errorf("exec failed: %w", errors.New("database is locked")), true},
		{"table locked", errors.New("database table is locked: users"), true},
		{"sqlite busy code", errors.New("SQLITE_BUSY: database is busy"), true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := isLockError(tc.err)
			if got != tc.expect {
				t.Errorf("isLockError(%v) = %v, want %v", tc.err, got, tc.expect)
			}
		})
	}
}

// TestExecRetries_succeedsAfterTransientLock verifies the retry loop will
// keep calling the inner function while it returns lock errors and stop as
// soon as it succeeds.
func TestExecRetries_succeedsAfterTransientLock(t *testing.T) {
	var attempts int
	result, err := execRetries(func() (sql.Result, error) {
		attempts++
		if attempts < 3 {
			return nil, errors.New("database is locked")
		}
		return nil, nil
	})
	if err != nil {
		t.Fatalf("expected success after retries, got err=%v", err)
	}
	if result != nil {
		t.Errorf("expected nil result, got %v", result)
	}
	if attempts != 3 {
		t.Errorf("expected 3 attempts, got %d", attempts)
	}
}

// TestExecRetries_doesNotRetryNonLockError verifies non-lock errors fail
// fast and the loop exits after the first call.
func TestExecRetries_doesNotRetryNonLockError(t *testing.T) {
	want := errors.New("syntax error")
	var attempts int
	_, err := execRetries(func() (sql.Result, error) {
		attempts++
		return nil, want
	})
	if !errors.Is(err, want) {
		t.Errorf("expected wrapped %v, got %v", want, err)
	}
	if attempts != 1 {
		t.Errorf("expected 1 attempt, got %d", attempts)
	}
}
