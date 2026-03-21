// Copyright 2022 Teamgram Authors
//  All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//   http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//
// Author: teamgramio (teamgram.io@gmail.com)
//

package core

import (
	_ "embed"
	"strings"
	"sync"

	"github.com/teamgram/proto/mtproto"
)

//go:embed PhoneCountries.txt
var phoneCountriesData string

var (
	countriesOnce sync.Once
	cachedResult  *mtproto.Help_CountriesList
)

func loadCountries() *mtproto.Help_CountriesList {
	countriesOnce.Do(func() {
		var countries []*mtproto.Help_Country
		lines := strings.Split(strings.TrimSpace(phoneCountriesData), "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}
			parts := strings.SplitN(line, ";", 4)
			if len(parts) < 4 {
				continue
			}
			dialCode := parts[0]
			iso2 := parts[1]
			pattern := parts[2]
			name := parts[3]

			var patterns []string
			if pattern != "" {
				patterns = []string{pattern}
			}

			countryCode := mtproto.MakeTLHelpCountryCode(&mtproto.Help_CountryCode{
				CountryCode: dialCode,
				Patterns:    patterns,
			}).To_Help_CountryCode()

			country := mtproto.MakeTLHelpCountry(&mtproto.Help_Country{
				Hidden:       false,
				Iso2:         iso2,
				DefaultName:  name,
				CountryCodes: []*mtproto.Help_CountryCode{countryCode},
			}).To_Help_Country()

			countries = append(countries, country)
		}

		cachedResult = mtproto.MakeTLHelpCountriesList(&mtproto.Help_CountriesList{
			Countries: countries,
			Hash:      1,
		}).To_Help_CountriesList()
	})
	return cachedResult
}

// HelpGetCountriesList
// help.getCountriesList#735787a8 lang_code:string hash:int = help.CountriesList;
func (c *ConfigurationCore) HelpGetCountriesList(in *mtproto.TLHelpGetCountriesList) (*mtproto.Help_CountriesList, error) {
	result := loadCountries()
	c.Logger.Debugf("help.getCountriesList - returning %d countries", len(result.Countries))
	return result, nil
}
