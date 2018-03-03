package main

type EsQueryRQ struct {
	Size  int      `json:"size,omitempty"`
	Query *EsQuery `json:"query,omitempty"`
}

type EsQuery struct {
	Bool *BoolCondition `json:"bool,omitempty"`
}

type BoolCondition struct {
	MustNot *Condition  `json:"must_not,omitempty"`
	Must    []Condition `json:"must,omitempty"`
	Should  []Condition `json:"should,omitempty"`
}

type Condition struct {
	Wildcard     map[string]interface{}   `json:"wildcard,omitempty"`
	Term         map[string]TermCondition `json:"term,omitempty"`
	Exists       *ExistsCondition         `json:"exists,omitempty"`
	MoreLikeThis *MoreLikeThisCondition   `json:"more_like_this,omitempty"`
}

type ExistsCondition struct {
	Field string `json:"field,omitempty"`
}
type MoreLikeThisCondition struct {
	Fields         []string `json:"fields,omitempty"`
	Like           string   `json:"like,omitempty"`
	MinDocFreq     float64  `json:"min_doc_freq,omitempty"`
	MinTermFreq    float64  `json:"min_term_freq,omitempty"`
	MinShouldMatch string   `json:"minimum_should_match,omitempty"`
}
type TermCondition struct {
	Value interface{} `json:"value,omitempty"`
	Boost *Boost      `json:"boost,omitempty"`
}
type Boost float64

func NewBoost(val float64) *Boost {
	b := Boost(val)
	return &b
}
