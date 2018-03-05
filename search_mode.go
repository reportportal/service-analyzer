package main

var enums []string

//SearchMode is an enum describing different types of models
type SearchMode int

func (e SearchMode) String() string {
	return enums[int(e)]
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
	return -1
}

//Search mode types
var (
	SearchModeAll        = ciota("ALL")
	SearchModeLaunchName = ciota("LAUNCH_NAME")
)
