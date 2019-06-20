/*
* Copyright 2019 EPAM Systems
*
* Licensed under the Apache License, Version 2.0 (the "License");
* you may not use this file except in compliance with the License.
* You may obtain a copy of the License at
*
* http://www.apache.org/licenses/LICENSE-2.0
*
* Unless required by applicable law or agreed to in writing, software
* distributed under the License is distributed on an "AS IS" BASIS,
* WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
* See the License for the specific language governing permissions and
* limitations under the License.
 */
package main

import (
	"encoding/json"
)

var enums []string

//SearchMode is an enum describing different types of models
type SearchMode int

func (sm SearchMode) String() string {
	return enums[int(sm)]
}

//MarshalJSON serializes to JSON
func (sm SearchMode) MarshalJSON() ([]byte, error) {
	return json.Marshal(sm.String())
}

//UnmarshalJSON deserializes from JSON
func (sm *SearchMode) UnmarshalJSON(data []byte) error {
	var str string
	err := json.Unmarshal(data, &str)
	if nil != err {
		return err
	}
	val := FromString(str)
	*sm = val
	return nil
}

func ciota(s string) SearchMode {
	enums = append(enums, s)
	return SearchMode(len(enums) - 1)
}

//FromString creates search mode from string
func FromString(s string) SearchMode {
	for i, e := range enums {
		if s == e {
			return SearchMode(i)
		}
	}
	return SearchModeNotFound
}

//SearchModeNotFound is a special case when type is not provided
const SearchModeNotFound = -1

//Search mode types
var (
	SearchModeAll           = ciota("ALL")
	SearchModeLaunchName    = ciota("LAUNCH_NAME")
	SearchModeCurrentLaunch = ciota("CURRENT_LAUNCH")
)
