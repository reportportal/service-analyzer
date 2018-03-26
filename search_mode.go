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
