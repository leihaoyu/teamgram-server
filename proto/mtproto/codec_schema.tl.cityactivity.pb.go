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

const (
	Predicate_cityActivity            = "cityActivity"
	Predicate_cityActivity_activities = "cityActivity_activities"
)

// MakeTLCityActivity creates a TLCityActivity wrapper
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

// To_CityActivity converts base type to TL wrapper
func (m *CityActivity) To_CityActivity() *TLCityActivity {
	m.PredicateName = Predicate_cityActivity
	return &TLCityActivity{
		Data2: m,
	}
}

// To_CityActivity converts TL wrapper back to base type
func (m *TLCityActivity) To_CityActivity() *CityActivity {
	m.Data2.PredicateName = Predicate_cityActivity
	return m.Data2
}

// MakeTLCityActivityActivities creates a TLCityActivityActivities wrapper
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

// To_CityActivity_Activities converts base type to TL wrapper
func (m *CityActivity_Activities) To_CityActivity_Activities() *TLCityActivityActivities {
	m.PredicateName = Predicate_cityActivity_activities
	return &TLCityActivityActivities{
		Data2: m,
	}
}

// To_CityActivity_Activities converts TL wrapper back to base type
func (m *TLCityActivityActivities) To_CityActivity_Activities() *CityActivity_Activities {
	m.Data2.PredicateName = Predicate_cityActivity_activities
	return m.Data2
}
