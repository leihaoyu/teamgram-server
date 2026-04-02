/*
 * WARNING! All changes made in this file will be lost!
 * Created from 'scheme.tl' by 'mtprotoc'
 *
 * Copyright (c) 2024-present, Teamgram Authors.
 *  All rights reserved.
 *
 * Author: teamgramio (teamgram.io@gmail.com)
 */

package mtproto

import (
	"fmt"

	"github.com/gogo/protobuf/jsonpb"
)

const (
	Predicate_cityActivity            = "cityActivity"
	Predicate_cityActivity_activities = "cityActivity_activities"
)

///////////////////////////////////////////////////////////////////////////////
// CityActivity
///////////////////////////////////////////////////////////////////////////////

func MakeTLCityActivity(data2 *CityActivity) *TLCityActivity {
	if data2 == nil {
		return &TLCityActivity{Data2: &CityActivity{
			PredicateName: Predicate_cityActivity,
		}}
	} else {
		data2.PredicateName = Predicate_cityActivity
		return &TLCityActivity{Data2: data2}
	}
}

func (m *CityActivity) To_CityActivity() *TLCityActivity {
	m.PredicateName = Predicate_cityActivity
	return &TLCityActivity{Data2: m}
}

func (m *TLCityActivity) To_CityActivity() *CityActivity {
	m.Data2.PredicateName = Predicate_cityActivity
	return m.Data2
}

func (m *CityActivity) Encode(x *EncodeBuf, layer int32) error {
	switch m.PredicateName {
	case Predicate_cityActivity:
		x.UInt(0x7a160c01)
		x.Long(m.GetId())
		x.Long(m.GetUserId())
		x.String(m.GetTitle())
		x.String(m.GetDescription())
		x.Long(m.GetPhotoId())
		x.String(m.GetCity())
		x.Long(m.GetStartTime())
		x.Long(m.GetEndTime())
		x.Int(m.GetMaxParticipants())
		x.Int(m.GetStatus())
		m.GetIsGlobal().Encode(x, layer)
		x.Int(m.GetParticipantCount())
		m.GetIsJoined().Encode(x, layer)
		x.String(m.GetCreatorName())
		x.Long(m.GetCreatedAt())
	default:
		return fmt.Errorf("CityActivity: invalid predicate: %s", m.PredicateName)
	}
	return nil
}

func (m *CityActivity) CalcByteSize(layer int32) int {
	return 0
}

func (m *CityActivity) Decode(dBuf *DecodeBuf) error {
	m.PredicateName = Predicate_cityActivity
	m.Id = dBuf.Long()
	m.UserId = dBuf.Long()
	m.Title = dBuf.String()
	m.Description = dBuf.String()
	m.PhotoId = dBuf.Long()
	m.City = dBuf.String()
	m.StartTime = dBuf.Long()
	m.EndTime = dBuf.Long()
	m.MaxParticipants = dBuf.Int()
	m.Status = dBuf.Int()

	m11 := &Bool{}
	m11.Decode(dBuf)
	m.IsGlobal = m11

	m.ParticipantCount = dBuf.Int()

	m13 := &Bool{}
	m13.Decode(dBuf)
	m.IsJoined = m13

	m.CreatorName = dBuf.String()
	m.CreatedAt = dBuf.Long()

	return dBuf.GetError()
}

func (m *CityActivity) DebugString() string {
	jsonm := &jsonpb.Marshaler{OrigName: true}
	dbgString, _ := jsonm.MarshalToString(m)
	return dbgString
}

// TLCityActivity
func (m *TLCityActivity) Encode(x *EncodeBuf, layer int32) error {
	return m.Data2.Encode(x, layer)
}

func (m *TLCityActivity) CalcByteSize(layer int32) int {
	return 0
}

func (m *TLCityActivity) Decode(dBuf *DecodeBuf) error {
	m.Data2 = &CityActivity{}
	return m.Data2.Decode(dBuf)
}

func (m *TLCityActivity) DebugString() string {
	jsonm := &jsonpb.Marshaler{OrigName: true}
	dbgString, _ := jsonm.MarshalToString(m)
	return dbgString
}

///////////////////////////////////////////////////////////////////////////////
// CityActivity_Activities
///////////////////////////////////////////////////////////////////////////////

func MakeTLCityActivityActivities(data2 *CityActivity_Activities) *TLCityActivityActivities {
	if data2 == nil {
		return &TLCityActivityActivities{Data2: &CityActivity_Activities{
			PredicateName: Predicate_cityActivity_activities,
		}}
	} else {
		data2.PredicateName = Predicate_cityActivity_activities
		return &TLCityActivityActivities{Data2: data2}
	}
}

func (m *CityActivity_Activities) To_CityActivity_Activities() *TLCityActivityActivities {
	m.PredicateName = Predicate_cityActivity_activities
	return &TLCityActivityActivities{Data2: m}
}

func (m *TLCityActivityActivities) To_CityActivity_Activities() *CityActivity_Activities {
	m.Data2.PredicateName = Predicate_cityActivity_activities
	return m.Data2
}

func (m *CityActivity_Activities) Encode(x *EncodeBuf, layer int32) error {
	switch m.PredicateName {
	case Predicate_cityActivity_activities:
		x.UInt(0x7a160c02)
		x.Int(int32(CRC32_vector))
		x.Int(int32(len(m.GetActivities())))
		for _, v := range m.GetActivities() {
			v.Encode(x, layer)
		}
		x.Int(m.GetCount())
	default:
		return fmt.Errorf("CityActivity_Activities: invalid predicate: %s", m.PredicateName)
	}
	return nil
}

func (m *CityActivity_Activities) CalcByteSize(layer int32) int {
	return 0
}

func (m *CityActivity_Activities) Decode(dBuf *DecodeBuf) error {
	m.PredicateName = Predicate_cityActivity_activities

	c0 := dBuf.Int()
	if c0 != int32(CRC32_vector) {
		return fmt.Errorf("expected vector, got: %d", c0)
	}
	l0 := dBuf.Int()
	m.Activities = make([]*CityActivity, l0)
	for i := int32(0); i < l0; i++ {
		dBuf.UInt() // read constructor id
		m.Activities[i] = &CityActivity{}
		m.Activities[i].Decode(dBuf)
	}

	m.Count = dBuf.Int()

	return dBuf.GetError()
}

func (m *CityActivity_Activities) DebugString() string {
	jsonm := &jsonpb.Marshaler{OrigName: true}
	dbgString, _ := jsonm.MarshalToString(m)
	return dbgString
}

// TLCityActivityActivities
func (m *TLCityActivityActivities) Encode(x *EncodeBuf, layer int32) error {
	return m.Data2.Encode(x, layer)
}

func (m *TLCityActivityActivities) CalcByteSize(layer int32) int {
	return 0
}

func (m *TLCityActivityActivities) Decode(dBuf *DecodeBuf) error {
	m.Data2 = &CityActivity_Activities{}
	return m.Data2.Decode(dBuf)
}

func (m *TLCityActivityActivities) DebugString() string {
	jsonm := &jsonpb.Marshaler{OrigName: true}
	dbgString, _ := jsonm.MarshalToString(m)
	return dbgString
}

///////////////////////////////////////////////////////////////////////////////
// CityActivity RPC request types
///////////////////////////////////////////////////////////////////////////////

// TLCityActivityGetActivities
func (m *TLCityActivityGetActivities) Encode(x *EncodeBuf, layer int32) error {
	switch uint32(m.Constructor) {
	case 0x7a160c03:
		x.UInt(0x7a160c03)
		x.String(m.GetCity())
		x.Int(m.GetOffset())
		x.Int(m.GetLimit())
	}
	return nil
}

func (m *TLCityActivityGetActivities) CalcByteSize(layer int32) int {
	return 0
}

func (m *TLCityActivityGetActivities) Decode(dBuf *DecodeBuf) error {
	switch uint32(m.Constructor) {
	case 0x7a160c03:
		m.City = dBuf.String()
		m.Offset = dBuf.Int()
		m.Limit = dBuf.Int()
		return dBuf.GetError()
	}
	return dBuf.GetError()
}

func (m *TLCityActivityGetActivities) DebugString() string {
	jsonm := &jsonpb.Marshaler{OrigName: true}
	dbgString, _ := jsonm.MarshalToString(m)
	return dbgString
}

// TLCityActivityGetActivity
func (m *TLCityActivityGetActivity) Encode(x *EncodeBuf, layer int32) error {
	switch uint32(m.Constructor) {
	case 0x7a160c04:
		x.UInt(0x7a160c04)
		x.Long(m.GetId())
	}
	return nil
}

func (m *TLCityActivityGetActivity) CalcByteSize(layer int32) int {
	return 0
}

func (m *TLCityActivityGetActivity) Decode(dBuf *DecodeBuf) error {
	switch uint32(m.Constructor) {
	case 0x7a160c04:
		m.Id = dBuf.Long()
		return dBuf.GetError()
	}
	return dBuf.GetError()
}

func (m *TLCityActivityGetActivity) DebugString() string {
	jsonm := &jsonpb.Marshaler{OrigName: true}
	dbgString, _ := jsonm.MarshalToString(m)
	return dbgString
}

// TLCityActivityCreateActivity
func (m *TLCityActivityCreateActivity) Encode(x *EncodeBuf, layer int32) error {
	switch uint32(m.Constructor) {
	case 0x7a160c05:
		x.UInt(0x7a160c05)
		x.String(m.GetTitle())
		x.String(m.GetDescription())
		x.Long(m.GetPhotoId())
		x.String(m.GetCity())
		x.Long(m.GetStartTime())
		x.Long(m.GetEndTime())
		x.Int(m.GetMaxParticipants())
	}
	return nil
}

func (m *TLCityActivityCreateActivity) CalcByteSize(layer int32) int {
	return 0
}

func (m *TLCityActivityCreateActivity) Decode(dBuf *DecodeBuf) error {
	switch uint32(m.Constructor) {
	case 0x7a160c05:
		m.Title = dBuf.String()
		m.Description = dBuf.String()
		m.PhotoId = dBuf.Long()
		m.City = dBuf.String()
		m.StartTime = dBuf.Long()
		m.EndTime = dBuf.Long()
		m.MaxParticipants = dBuf.Int()
		return dBuf.GetError()
	}
	return dBuf.GetError()
}

func (m *TLCityActivityCreateActivity) DebugString() string {
	jsonm := &jsonpb.Marshaler{OrigName: true}
	dbgString, _ := jsonm.MarshalToString(m)
	return dbgString
}

// TLCityActivityEditActivity
func (m *TLCityActivityEditActivity) Encode(x *EncodeBuf, layer int32) error {
	switch uint32(m.Constructor) {
	case 0x7a160c06:
		x.UInt(0x7a160c06)
		x.Long(m.GetId())
		x.String(m.GetTitle())
		x.String(m.GetDescription())
		x.Long(m.GetPhotoId())
		x.Long(m.GetStartTime())
		x.Long(m.GetEndTime())
		x.Int(m.GetStatus())
	}
	return nil
}

func (m *TLCityActivityEditActivity) CalcByteSize(layer int32) int {
	return 0
}

func (m *TLCityActivityEditActivity) Decode(dBuf *DecodeBuf) error {
	switch uint32(m.Constructor) {
	case 0x7a160c06:
		m.Id = dBuf.Long()
		m.Title = dBuf.String()
		m.Description = dBuf.String()
		m.PhotoId = dBuf.Long()
		m.StartTime = dBuf.Long()
		m.EndTime = dBuf.Long()
		m.Status = dBuf.Int()
		return dBuf.GetError()
	}
	return dBuf.GetError()
}

func (m *TLCityActivityEditActivity) DebugString() string {
	jsonm := &jsonpb.Marshaler{OrigName: true}
	dbgString, _ := jsonm.MarshalToString(m)
	return dbgString
}

// TLCityActivityDeleteActivity
func (m *TLCityActivityDeleteActivity) Encode(x *EncodeBuf, layer int32) error {
	switch uint32(m.Constructor) {
	case 0x7a160c07:
		x.UInt(0x7a160c07)
		x.Long(m.GetId())
	}
	return nil
}

func (m *TLCityActivityDeleteActivity) CalcByteSize(layer int32) int {
	return 0
}

func (m *TLCityActivityDeleteActivity) Decode(dBuf *DecodeBuf) error {
	switch uint32(m.Constructor) {
	case 0x7a160c07:
		m.Id = dBuf.Long()
		return dBuf.GetError()
	}
	return dBuf.GetError()
}

func (m *TLCityActivityDeleteActivity) DebugString() string {
	jsonm := &jsonpb.Marshaler{OrigName: true}
	dbgString, _ := jsonm.MarshalToString(m)
	return dbgString
}

// TLCityActivityJoinActivity
func (m *TLCityActivityJoinActivity) Encode(x *EncodeBuf, layer int32) error {
	switch uint32(m.Constructor) {
	case 0x7a160c08:
		x.UInt(0x7a160c08)
		x.Long(m.GetId())
		x.String(m.GetCity())
	}
	return nil
}

func (m *TLCityActivityJoinActivity) CalcByteSize(layer int32) int {
	return 0
}

func (m *TLCityActivityJoinActivity) Decode(dBuf *DecodeBuf) error {
	switch uint32(m.Constructor) {
	case 0x7a160c08:
		m.Id = dBuf.Long()
		m.City = dBuf.String()
		return dBuf.GetError()
	}
	return dBuf.GetError()
}

func (m *TLCityActivityJoinActivity) DebugString() string {
	jsonm := &jsonpb.Marshaler{OrigName: true}
	dbgString, _ := jsonm.MarshalToString(m)
	return dbgString
}

// TLCityActivityLeaveActivity
func (m *TLCityActivityLeaveActivity) Encode(x *EncodeBuf, layer int32) error {
	switch uint32(m.Constructor) {
	case 0x7a160c09:
		x.UInt(0x7a160c09)
		x.Long(m.GetId())
	}
	return nil
}

func (m *TLCityActivityLeaveActivity) CalcByteSize(layer int32) int {
	return 0
}

func (m *TLCityActivityLeaveActivity) Decode(dBuf *DecodeBuf) error {
	switch uint32(m.Constructor) {
	case 0x7a160c09:
		m.Id = dBuf.Long()
		return dBuf.GetError()
	}
	return dBuf.GetError()
}

func (m *TLCityActivityLeaveActivity) DebugString() string {
	jsonm := &jsonpb.Marshaler{OrigName: true}
	dbgString, _ := jsonm.MarshalToString(m)
	return dbgString
}
