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

// I'd like to expose that method for ease of use, but since Go doesn't have union types,
// we need to accept interface{} here when in reality there is a finite set of types that are valid.
// To maintain a semblance of type safety, we hide methods taking & returning interface{} and expose type safe methods,
// at the cost of repeating ourselves.
func (e Expression) and(subexpr interface{}) Expression {
	if e.body == nil {
		e.body = subexpr
	} else {
		e.body = andNode{And: [2]interface{}{e.body, subexpr}}
	}

	return e
}

func (e Expression) or(subexpr interface{}) Expression {
	if e.body == nil {
		e.body = subexpr
	} else {
		e.body = orNode{Or: [2]interface{}{e.body, subexpr}}
	}

	return e
}

// AndTag ANDs a tag condition with the query.
func (e Expression) AndTag(tag string) Expression {
	return e.and(tagNode{Tag: tag})
}

// AndKeyValue ANDs a key/value condition with the query.
func (e Expression) AndKeyValue(key string, value string) Expression {
	return e.and(keyValueNode{Key: key, Value: value})
}

// AndHasKey ANDs a "has key" condition with the query.
func (e Expression) AndHasKey(key string) Expression {
	return e.and(hasKeyNode{Key: key})
}

// OrTag ORs a tag condition with the query.
func (e Expression) OrTag(tag string) Expression {
	return e.or(tagNode{Tag: tag})
}

// OrKeyValue ORs a key/value condition with the query.
func (e Expression) OrKeyValue(key string, value string) Expression {
	return e.or(keyValueNode{Key: key, Value: value})
}

// OrHasKey ORs a "has key" condition with the query.
func (e Expression) OrHasKey(key string) Expression {
	return e.or(hasKeyNode{Key: key})
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
