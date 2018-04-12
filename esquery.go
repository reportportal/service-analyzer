package main

//EsQueryRQ is a query model
type EsQueryRQ struct {
	Size  int      `json:"size,omitempty"`
	Query *EsQuery `json:"query,omitempty"`
}

//EsQuery is a query model
type EsQuery struct {
	Bool *BoolCondition `json:"bool,omitempty"`
}

//BoolCondition is a bool condition model
type BoolCondition struct {
	MustNot *Condition  `json:"must_not,omitempty"`
	Must    []Condition `json:"must,omitempty"`
	Should  []Condition `json:"should,omitempty"`
}

//Condition is a condition model
type Condition struct {
	Wildcard     map[string]interface{}   `json:"wildcard,omitempty"`
	Term         map[string]TermCondition `json:"term,omitempty"`
	Range        map[string]interface{}   `json:"range,omitempty"`
	Exists       *ExistsCondition         `json:"exists,omitempty"`
	MoreLikeThis *MoreLikeThisCondition   `json:"more_like_this,omitempty"`
}

//ExistsCondition is a exists condition model
type ExistsCondition struct {
	Field string `json:"field,omitempty"`
}

//MoreLikeThisCondition is a more/like/this model
type MoreLikeThisCondition struct {
	Fields         []string `json:"fields,omitempty"`
	Like           string   `json:"like,omitempty"`
	MinDocFreq     float64  `json:"min_doc_freq,omitempty"`
	MinTermFreq    float64  `json:"min_term_freq,omitempty"`
	MinShouldMatch string   `json:"minimum_should_match,omitempty"`
}

//TermCondition is a term condition model
type TermCondition struct {
	Value interface{} `json:"value,omitempty"`
	Boost *Boost      `json:"boost,omitempty"`
}

//RangeCondition is a term condition model
type RangeCondition struct {
	Value map[string]interface{} `json:"value,omitempty"`
}

//Boost is a term boost model
type Boost float64

//NewBoost is a boost model
func NewBoost(val float64) *Boost {
	b := Boost(val)
	return &b
}
