// Copyright 2026 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package postgres

import (
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

func TestRegisterGoogleUUID(t *testing.T) {
	m := pgtype.NewMap()
	registerGoogleUUID(m)

	want := uuid.MustParse("f47ac10b-58cc-4372-a567-0e02b2c3d479")

	encodePlan := m.PlanEncode(pgtype.UUIDOID, pgtype.BinaryFormatCode, want)
	if encodePlan == nil {
		t.Fatal("PlanEncode returned nil")
	}

	encoded, err := encodePlan.Encode(want, nil)
	if err != nil {
		t.Fatalf("Encode returned error: %s", err)
	}
	if string(encoded) != string(want[:]) {
		t.Fatalf("unexpected encoded UUID: got %x, want %x", encoded, want[:])
	}

	var got uuid.UUID
	scanPlan := m.PlanScan(pgtype.UUIDOID, pgtype.BinaryFormatCode, &got)
	if scanPlan == nil {
		t.Fatal("PlanScan returned nil")
	}
	if err := scanPlan.Scan(encoded, &got); err != nil {
		t.Fatalf("Scan returned error: %s", err)
	}
	if got != want {
		t.Fatalf("unexpected scanned UUID: got %s, want %s", got, want)
	}

	typ, ok := m.TypeForOID(pgtype.UUIDOID)
	if !ok {
		t.Fatal("TypeForOID returned false")
	}

	value, err := typ.Codec.DecodeValue(m, pgtype.UUIDOID, pgtype.TextFormatCode, []byte(want.String()))
	if err != nil {
		t.Fatalf("DecodeValue returned error: %s", err)
	}
	if value != want {
		t.Fatalf("unexpected decoded UUID: got %v, want %s", value, want)
	}
}

func TestRegisterGoogleNullUUID(t *testing.T) {
	m := pgtype.NewMap()
	registerGoogleUUID(m)

	want := uuid.NullUUID{UUID: uuid.MustParse("f47ac10b-58cc-4372-a567-0e02b2c3d479"), Valid: true}

	encodePlan := m.PlanEncode(pgtype.UUIDOID, pgtype.TextFormatCode, want)
	if encodePlan == nil {
		t.Fatal("PlanEncode returned nil")
	}

	encoded, err := encodePlan.Encode(want, nil)
	if err != nil {
		t.Fatalf("Encode returned error: %s", err)
	}
	if string(encoded) != want.UUID.String() {
		t.Fatalf("unexpected encoded UUID: got %s, want %s", encoded, want.UUID)
	}

	var got uuid.NullUUID
	scanPlan := m.PlanScan(pgtype.UUIDOID, pgtype.TextFormatCode, &got)
	if scanPlan == nil {
		t.Fatal("PlanScan returned nil")
	}
	if err := scanPlan.Scan(encoded, &got); err != nil {
		t.Fatalf("Scan returned error: %s", err)
	}
	if got != want {
		t.Fatalf("unexpected scanned UUID: got %+v, want %+v", got, want)
	}

	if err := scanPlan.Scan(nil, &got); err != nil {
		t.Fatalf("Scan NULL returned error: %s", err)
	}
	if got.Valid {
		t.Fatalf("unexpected valid NullUUID after scanning NULL: %+v", got)
	}
}
