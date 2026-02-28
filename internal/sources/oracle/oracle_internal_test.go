// Copyright © 2025, Oracle and/or its affiliates.
package oracle

import (
	"errors"
	"testing"

	"github.com/google/go-cmp/cmp"
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
	want := map[string]any{
		"status":        "success",
		"rows_affected": int64(7),
	}
	got := formatExecResponse(fakeResult{rows: 7})

	if !cmp.Equal(want, got) {
		t.Fatalf("formatExecResponse() mismatch (-want +got):\n%s", cmp.Diff(want, got))
	}
}

func TestFormatExecResponseWithoutRowsAffected(t *testing.T) {
	want := map[string]any{
		"status": "success",
	}
	got := formatExecResponse(fakeResult{err: errors.New("rows affected unavailable")})

	if !cmp.Equal(want, got) {
		t.Fatalf("formatExecResponse() mismatch (-want +got):\n%s", cmp.Diff(want, got))
	}
}
