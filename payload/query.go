package payload

const defaultQueryFrom = 0
const defaultQuerySize = 20

type tagNode struct {
	Tag string `json:"tag"`
}

type keyValueNode struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type hasKeyNode struct {
	Key string `json:"key"`
}

type parentNode struct {
	Parent string `json:"parent"`
}

type andNode struct {
	And [2]interface{} `json:"and"`
}

type orNode struct {
	Or [2]interface{} `json:"or"`
}

type notNode struct {
	Not interface{} `json:"not"`
}

// Expression allows to build a structured query.
// TODO: Implement expression deserialization:
// https://medium.com/@haya14busa/sum-union-variant-type-in-go-and-static-check-tool-of-switch-case-handling-3bfc61618b1e
type Expression struct {
	body interface{}
}

// NewExpression returns an empty structured expression.
func NewExpression() Expression {
	return Expression{body: nil}
}

func ParseExpression(rawData map[string]interface{}) (Expression, error) {
	body, err := loadExpressionBody(rawData)
	return Expression{body: body}, err
}

// AndTag ANDs a tag condition with the query.
func (e Expression) AndTag(tag string) Expression {
	node := tagNode{Tag: tag}
	if e.body == nil {
		e.body = node
	} else {
		e.body = andNode{And: [2]interface{}{e.body, node}}
	}
	return e
}

// AndKeyValue ANDs a key/value condition with the query.
func (e Expression) AndKeyValue(key string, value string) Expression {
	node := keyValueNode{Key: key, Value: value}
	if e.body == nil {
		e.body = node
	} else {
		e.body = andNode{And: [2]interface{}{e.body, node}}
	}
	return e
}

// AndHasKey ANDs a "has key" condition with the query.
func (e Expression) AndHasKey(key string) Expression {
	node := hasKeyNode{Key: key}
	if e.body == nil {
		e.body = node
	} else {
		e.body = andNode{And: [2]interface{}{e.body, node}}
	}
	return e
}

// AndParent ANDs a parent condition with the query.
func (e Expression) AndParent(parentID string) Expression {
	node := parentNode{Parent: parentID}
	if e.body == nil {
		e.body = node
	} else {
		e.body = andNode{And: [2]interface{}{e.body, node}}
	}
	return e
}

// Query is the expected menmos query request.
type Query struct {
	Expression interface{} `json:"expression,omitempty"`
	From       uint32      `json:"from,omitempty"`
	Size       uint32      `json:"size,omitempty"`
	SignURLs   bool        `json:"sign_urls,omitempty"`
	Facets     bool        `json:"facets,omitempty"`
}

func newQuery(expression interface{}) *Query {
	return &Query{
		Expression: expression,
		From:       defaultQueryFrom,
		Size:       defaultQuerySize,
		SignURLs:   true,
		Facets:     false,
	}
}

// NewUnstructuredQuery returns a query with the specified expression and default parameters.
func NewUnstructuredQuery(expression string) *Query {
	return newQuery(expression)
}

// NewStructuredQuery returns a query with the specified structured query.
func NewStructuredQuery(expression Expression) *Query {
	return newQuery(expression.body)
}

// WithFrom sets the value of the `from` query field.
func (q *Query) WithFrom(from uint32) *Query {
	q.From = from
	return q
}

// WithSize sets the value of the `size` query field.
func (q *Query) WithSize(size uint32) *Query {
	q.Size = size
	return q
}

// WithSignURLs sets the value of the `sign_urls` query field.
func (q *Query) WithSignURLs(signURLs bool) *Query {
	q.SignURLs = signURLs
	return q
}

// WithFacets sets the value of the `facets` query field.
func (q *Query) WithFacets(facets bool) *Query {
	q.Facets = facets
	return q
}
