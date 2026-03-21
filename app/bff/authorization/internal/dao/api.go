// Copyright (c) 2021-present,  Teamgram Studio (https://teamgram.io).
//  All rights reserved.
//
// Author: teamgramio (teamgram.io@gmail.com)
//

package dao

import (
	"net"

	"github.com/zeromicro/go-zero/core/logx"
)

func (d *Dao) CheckApiIdAndHash(apiId int32, apiHash string) error {
	// TODO(@benqi): check api_id and api_hash
	// 400	API_ID_INVALID	API ID无效
	// 400	API_ID_PUBLISHED_FLOOD	这个API ID已发布在某个地方，您现在不能使用

	_ = apiId
	_ = apiHash

	return nil
}

func (d *Dao) GetCountryAndRegionByIp(ip string) (string, string) {
	if d.MMDB == nil {
		return "UNKNOWN", ""
	} else {
		r, err := d.MMDB.City(net.ParseIP(ip))
		if err != nil {
			logx.Errorf("getCountryAndRegionByIp - error: %v", err)
			return "UNKNOWN", ""
		}

		return r.City.Names["en"] + ", " + r.Country.Names["en"], r.Country.IsoCode
	}
}

// countryToLocale maps ISO 3166-1 country codes to GeoIP2 locale keys.
// GeoIP2 City DB supports: de, en, es, fr, ja, pt-BR, ru, zh-CN
var countryToLocale = map[string]string{
	// Chinese-speaking
	"CN": "zh-CN", "TW": "zh-CN", "HK": "zh-CN", "MO": "zh-CN", "SG": "zh-CN",
	// Japanese
	"JP": "ja",
	// German-speaking
	"DE": "de", "AT": "de", "CH": "de", "LI": "de",
	// Spanish-speaking
	"ES": "es", "MX": "es", "AR": "es", "CO": "es", "CL": "es",
	"PE": "es", "VE": "es", "EC": "es", "GT": "es", "CU": "es",
	"BO": "es", "DO": "es", "HN": "es", "PY": "es", "SV": "es",
	"NI": "es", "CR": "es", "PA": "es", "UY": "es",
	// French-speaking
	"FR": "fr", "BE": "fr", "LU": "fr",
	// Portuguese
	"BR": "pt-BR", "PT": "pt-BR",
	// Russian-speaking
	"RU": "ru", "BY": "ru", "KZ": "ru", "KG": "ru",
}

// GetCityAndLocaleByIp returns the city name in the local language and the locale code.
// Falls back to English if the local language is not available in GeoIP2.
func (d *Dao) GetCityAndLocaleByIp(ip string) (cityName string, locale string) {
	if d.MMDB == nil {
		return "", "en"
	}

	r, err := d.MMDB.City(net.ParseIP(ip))
	if err != nil {
		logx.Errorf("GetCityAndLocaleByIp - error: %v", err)
		return "", "en"
	}

	countryCode := r.Country.IsoCode
	locale = "en"
	if l, ok := countryToLocale[countryCode]; ok {
		locale = l
	}

	// Try locale-specific city name first, then fallback to English
	if name, ok := r.City.Names[locale]; ok && name != "" {
		cityName = name
	} else if name, ok := r.City.Names["en"]; ok && name != "" {
		cityName = name
		locale = "en"
	}

	return cityName, locale
}
