package main

var enums []string

type SearchMode int

func (e SearchMode) String() string {
	return enums[int(e)]
}

func ciota(s string) SearchMode {
	enums = append(enums, s)
	return SearchMode(len(enums) - 1)
}

func FromString(s string) SearchMode {
	for i, e := range enums {
		if s == e {
			return SearchMode(i)
		}
	}
	return -1
}

var (
	SearchModeAll        = ciota("ALL")
	SearchModeLaunchName = ciota("LAUNCH_NAME")
)
