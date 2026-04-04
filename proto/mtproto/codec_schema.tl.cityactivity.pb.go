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
		// photos vector
		x.Int(int32(CRC32_vector))
		x.Int(int32(len(m.GetPhotos())))
		for _, photo := range m.GetPhotos() {
			photo.Encode(x, layer)
		}
		x.Long(m.GetChatId())
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

	// photos vector
	c18 := dBuf.Int()
	if c18 != int32(CRC32_vector) {
		// backward compat: older data without photos field
	} else {
		l18 := dBuf.Int()
		m.Photos = make([]*Photo, l18)
		for i := int32(0); i < l18; i++ {
			m.Photos[i] = &Photo{}
			m.Photos[i].Decode(dBuf)
		}
		// chat_id (appended after photos)
		if dBuf.GetError() == nil {
			m.ChatId = dBuf.Long()
		}
	}

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
	x.UInt(uint32(m.Constructor))
	x.String(m.GetCity())
	x.Int(m.GetOffset())
	x.Int(m.GetLimit())
	x.Int(m.GetFilter())
	return nil
}

func (m *TLCityActivityGetActivities) CalcByteSize(layer int32) int {
	return 0
}

func (m *TLCityActivityGetActivities) Decode(dBuf *DecodeBuf) error {
	m.City = dBuf.String()
	m.Offset = dBuf.Int()
	m.Limit = dBuf.Int()
	m.Filter = dBuf.Int()
	fmt.Printf("[DECODE] TLCityActivityGetActivities - city: %s, offset: %d, limit: %d, filter: %d\n", m.City, m.Offset, m.Limit, m.Filter)
	return dBuf.GetError()
}

func (m *TLCityActivityGetActivities) DebugString() string {
	jsonm := &jsonpb.Marshaler{OrigName: true}
	dbgString, _ := jsonm.MarshalToString(m)
	return dbgString
}

// TLCityActivityGetActivity
func (m *TLCityActivityGetActivity) Encode(x *EncodeBuf, layer int32) error {
	x.UInt(uint32(m.Constructor))
	x.Long(m.GetId())
	return nil
}

func (m *TLCityActivityGetActivity) CalcByteSize(layer int32) int {
	return 0
}

func (m *TLCityActivityGetActivity) Decode(dBuf *DecodeBuf) error {
	m.Id = dBuf.Long()
	return dBuf.GetError()
}

func (m *TLCityActivityGetActivity) DebugString() string {
	jsonm := &jsonpb.Marshaler{OrigName: true}
	dbgString, _ := jsonm.MarshalToString(m)
	return dbgString
}

// TLCityActivityGetMyActivities
func (m *TLCityActivityGetMyActivities) Encode(x *EncodeBuf, layer int32) error {
	x.UInt(uint32(m.Constructor))
	x.Int(m.GetOffset())
	x.Int(m.GetLimit())
	return nil
}

func (m *TLCityActivityGetMyActivities) CalcByteSize(layer int32) int {
	return 0
}

func (m *TLCityActivityGetMyActivities) Decode(dBuf *DecodeBuf) error {
	m.Offset = dBuf.Int()
	m.Limit = dBuf.Int()
	return dBuf.GetError()
}

func (m *TLCityActivityGetMyActivities) DebugString() string {
	jsonm := &jsonpb.Marshaler{OrigName: true}
	dbgString, _ := jsonm.MarshalToString(m)
	return dbgString
}

// TLCityActivityCreateActivity
func (m *TLCityActivityCreateActivity) Encode(x *EncodeBuf, layer int32) error {
	x.UInt(uint32(m.Constructor))
	x.String(m.GetTitle())
	x.String(m.GetDescription())
	x.Long(m.GetPhotoId())
	x.String(m.GetCity())
	x.Long(m.GetStartTime())
	x.Long(m.GetEndTime())
	x.Int(m.GetMaxParticipants())
	// photo_ids vector
	x.Int(int32(CRC32_vector))
	x.Int(int32(len(m.GetPhotoIds())))
	for _, id := range m.GetPhotoIds() {
		x.Long(id)
	}
	m.GetIsGlobal().Encode(x, layer)
	return nil
}

func (m *TLCityActivityCreateActivity) CalcByteSize(layer int32) int {
	return 0
}

func (m *TLCityActivityCreateActivity) Decode(dBuf *DecodeBuf) error {
	m.Title = dBuf.String()
	m.Description = dBuf.String()
	m.PhotoId = dBuf.Long()
	m.City = dBuf.String()
	m.StartTime = dBuf.Long()
	m.EndTime = dBuf.Long()
	m.MaxParticipants = dBuf.Int()
	// photo_ids vector
	c10 := dBuf.Int()
	if c10 == int32(CRC32_vector) {
		l10 := dBuf.Int()
		m.PhotoIds = make([]int64, l10)
		for i := int32(0); i < l10; i++ {
			m.PhotoIds[i] = dBuf.Long()
		}
	}
	// is_global
	m11 := &Bool{}
	m11.Decode(dBuf)
	m.IsGlobal = m11
	return dBuf.GetError()
}

func (m *TLCityActivityCreateActivity) DebugString() string {
	jsonm := &jsonpb.Marshaler{OrigName: true}
	dbgString, _ := jsonm.MarshalToString(m)
	return dbgString
}

// TLCityActivityEditActivity
func (m *TLCityActivityEditActivity) Encode(x *EncodeBuf, layer int32) error {
	x.UInt(uint32(m.Constructor))
	x.Long(m.GetId())
	x.String(m.GetTitle())
	x.String(m.GetDescription())
	x.Long(m.GetPhotoId())
	x.Long(m.GetStartTime())
	x.Long(m.GetEndTime())
	x.Int(m.GetStatus())
	return nil
}

func (m *TLCityActivityEditActivity) CalcByteSize(layer int32) int {
	return 0
}

func (m *TLCityActivityEditActivity) Decode(dBuf *DecodeBuf) error {
	m.Id = dBuf.Long()
	m.Title = dBuf.String()
	m.Description = dBuf.String()
	m.PhotoId = dBuf.Long()
	m.StartTime = dBuf.Long()
	m.EndTime = dBuf.Long()
	m.Status = dBuf.Int()
	return dBuf.GetError()
}

func (m *TLCityActivityEditActivity) DebugString() string {
	jsonm := &jsonpb.Marshaler{OrigName: true}
	dbgString, _ := jsonm.MarshalToString(m)
	return dbgString
}

// TLCityActivityDeleteActivity
func (m *TLCityActivityDeleteActivity) Encode(x *EncodeBuf, layer int32) error {
	x.UInt(uint32(m.Constructor))
	x.Long(m.GetId())
	return nil
}

func (m *TLCityActivityDeleteActivity) CalcByteSize(layer int32) int {
	return 0
}

func (m *TLCityActivityDeleteActivity) Decode(dBuf *DecodeBuf) error {
	m.Id = dBuf.Long()
	return dBuf.GetError()
}

func (m *TLCityActivityDeleteActivity) DebugString() string {
	jsonm := &jsonpb.Marshaler{OrigName: true}
	dbgString, _ := jsonm.MarshalToString(m)
	return dbgString
}

// TLCityActivityJoinActivity
func (m *TLCityActivityJoinActivity) Encode(x *EncodeBuf, layer int32) error {
	x.UInt(uint32(m.Constructor))
	x.Long(m.GetId())
	x.String(m.GetCity())
	return nil
}

func (m *TLCityActivityJoinActivity) CalcByteSize(layer int32) int {
	return 0
}

func (m *TLCityActivityJoinActivity) Decode(dBuf *DecodeBuf) error {
	m.Id = dBuf.Long()
	m.City = dBuf.String()
	return dBuf.GetError()
}

func (m *TLCityActivityJoinActivity) DebugString() string {
	jsonm := &jsonpb.Marshaler{OrigName: true}
	dbgString, _ := jsonm.MarshalToString(m)
	return dbgString
}

// TLCityActivityLeaveActivity
func (m *TLCityActivityLeaveActivity) Encode(x *EncodeBuf, layer int32) error {
	x.UInt(uint32(m.Constructor))
	x.Long(m.GetId())
	return nil
}

func (m *TLCityActivityLeaveActivity) CalcByteSize(layer int32) int {
	return 0
}

func (m *TLCityActivityLeaveActivity) Decode(dBuf *DecodeBuf) error {
	m.Id = dBuf.Long()
	return dBuf.GetError()
}

func (m *TLCityActivityLeaveActivity) DebugString() string {
	jsonm := &jsonpb.Marshaler{OrigName: true}
	dbgString, _ := jsonm.MarshalToString(m)
	return dbgString
}
