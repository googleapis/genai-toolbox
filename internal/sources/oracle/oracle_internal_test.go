// Copyright © 2025, Oracle and/or its affiliates.
package oracle

import (
	"errors"
	"testing"
)

type fakeResult struct {
	rows int64
	err  error
}

func (f fakeResult) LastInsertId() (int64, error) {
	return 0, nil
}

func (f fakeResult) RowsAffected() (int64, error) {
	if f.err != nil {
		return 0, f.err
	}
	return f.rows, nil
}

func TestFormatExecResponseWithRowsAffected(t *testing.T) {
	got := formatExecResponse(fakeResult{rows: 7})

	if status, ok := got["status"]; !ok || status != "success" {
		t.Fatalf("expected status=success, got %#v", got["status"])
	}

	if rows, ok := got["rows_affected"]; !ok || rows != int64(7) {
		t.Fatalf("expected rows_affected=7, got %#v", got["rows_affected"])
	}
}

func TestFormatExecResponseWithoutRowsAffected(t *testing.T) {
	got := formatExecResponse(fakeResult{err: errors.New("rows affected unavailable")})

	if status, ok := got["status"]; !ok || status != "success" {
		t.Fatalf("expected status=success, got %#v", got["status"])
	}

	if _, ok := got["rows_affected"]; ok {
		t.Fatalf("expected rows_affected to be omitted when unavailable, got %#v", got["rows_affected"])
	}
}
