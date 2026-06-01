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
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

type googleUUID uuid.UUID

func (u *googleUUID) ScanUUID(v pgtype.UUID) error {
	if !v.Valid {
		return fmt.Errorf("cannot scan NULL into *uuid.UUID")
	}

	*u = googleUUID(v.Bytes)
	return nil
}

func (u googleUUID) UUIDValue() (pgtype.UUID, error) {
	return pgtype.UUID{Bytes: [16]byte(u), Valid: true}, nil
}

type googleNullUUID uuid.NullUUID

func (u *googleNullUUID) ScanUUID(v pgtype.UUID) error {
	*u = googleNullUUID{UUID: uuid.UUID(v.Bytes), Valid: v.Valid}
	return nil
}

func (u googleNullUUID) UUIDValue() (pgtype.UUID, error) {
	return pgtype.UUID{Bytes: [16]byte(u.UUID), Valid: u.Valid}, nil
}

func tryWrapGoogleUUIDEncodePlan(value any) (pgtype.WrappedEncodePlanNextSetter, any, bool) {
	switch value := value.(type) {
	case uuid.UUID:
		return &wrapGoogleUUIDEncodePlan{}, googleUUID(value), true
	case uuid.NullUUID:
		return &wrapGoogleNullUUIDEncodePlan{}, googleNullUUID(value), true
	}

	return nil, nil, false
}

type wrapGoogleUUIDEncodePlan struct {
	next pgtype.EncodePlan
}

func (plan *wrapGoogleUUIDEncodePlan) SetNext(next pgtype.EncodePlan) { plan.next = next }

func (plan *wrapGoogleUUIDEncodePlan) Encode(value any, buf []byte) ([]byte, error) {
	return plan.next.Encode(googleUUID(value.(uuid.UUID)), buf)
}

type wrapGoogleNullUUIDEncodePlan struct {
	next pgtype.EncodePlan
}

func (plan *wrapGoogleNullUUIDEncodePlan) SetNext(next pgtype.EncodePlan) { plan.next = next }

func (plan *wrapGoogleNullUUIDEncodePlan) Encode(value any, buf []byte) ([]byte, error) {
	return plan.next.Encode(googleNullUUID(value.(uuid.NullUUID)), buf)
}

func tryWrapGoogleUUIDScanPlan(target any) (pgtype.WrappedScanPlanNextSetter, any, bool) {
	switch target := target.(type) {
	case *uuid.UUID:
		return &wrapGoogleUUIDScanPlan{}, (*googleUUID)(target), true
	case *uuid.NullUUID:
		return &wrapGoogleNullUUIDScanPlan{}, (*googleNullUUID)(target), true
	}

	return nil, nil, false
}

type wrapGoogleUUIDScanPlan struct {
	next pgtype.ScanPlan
}

func (plan *wrapGoogleUUIDScanPlan) SetNext(next pgtype.ScanPlan) { plan.next = next }

func (plan *wrapGoogleUUIDScanPlan) Scan(src []byte, dst any) error {
	return plan.next.Scan(src, (*googleUUID)(dst.(*uuid.UUID)))
}

type wrapGoogleNullUUIDScanPlan struct {
	next pgtype.ScanPlan
}

func (plan *wrapGoogleNullUUIDScanPlan) SetNext(next pgtype.ScanPlan) { plan.next = next }

func (plan *wrapGoogleNullUUIDScanPlan) Scan(src []byte, dst any) error {
	return plan.next.Scan(src, (*googleNullUUID)(dst.(*uuid.NullUUID)))
}

type googleUUIDCodec struct {
	pgtype.UUIDCodec
}

func (googleUUIDCodec) DecodeValue(m *pgtype.Map, oid uint32, format int16, src []byte) (any, error) {
	if src == nil {
		return nil, nil
	}

	var target uuid.UUID
	scanPlan := m.PlanScan(oid, format, &target)
	if scanPlan == nil {
		return nil, fmt.Errorf("PlanScan did not find a plan")
	}

	if err := scanPlan.Scan(src, &target); err != nil {
		return nil, err
	}

	return target, nil
}

func registerGoogleUUID(m *pgtype.Map) {
	m.TryWrapEncodePlanFuncs = append([]pgtype.TryWrapEncodePlanFunc{tryWrapGoogleUUIDEncodePlan}, m.TryWrapEncodePlanFuncs...)
	m.TryWrapScanPlanFuncs = append([]pgtype.TryWrapScanPlanFunc{tryWrapGoogleUUIDScanPlan}, m.TryWrapScanPlanFuncs...)

	m.RegisterType(&pgtype.Type{
		Name:  "uuid",
		OID:   pgtype.UUIDOID,
		Codec: googleUUIDCodec{},
	})
}
