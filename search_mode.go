package main

import "encoding/json"

var enums []string

//SearchMode is an enum describing different types of models
type SearchMode int

func (sm SearchMode) String() string {
	return enums[int(sm-1)]
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
	*sm = FromString(str)
	return nil
}

func ciota(s string) SearchMode {
	enums = append(enums, s)
	return SearchMode(len(enums))
}

//FromString creates search mode from string
func FromString(s string) SearchMode {
	for i, e := range enums {
		if s == e {
			return SearchMode(i + 1)
		}
	}
	return -1
}

//Search mode types
var (
	SearchModeAll        = ciota("ALL")
	SearchModeLaunchName = ciota("LAUNCH_NAME")
)
