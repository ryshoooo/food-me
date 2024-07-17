package foodme

import (
	"encoding/json"
	"testing"

	"gotest.tools/v3/assert"
)

func TestAllowEverythingParse(t *testing.T) {
	data := `{"result": {"queries": [[]]}}`
	b := &CompileResponse{}
	err := json.Unmarshal([]byte(data), b)
	assert.NilError(t, err)
	assert.Assert(t, b.IsAllowed())
}

func TestDisallowEverythingParse(t *testing.T) {
	data := `{"result": {}}`
	b := &CompileResponse{}
	err := json.Unmarshal([]byte(data), b)
	assert.NilError(t, err)
	assert.Assert(t, b.IsDisallowed())
}

func AssertCompiledTerm(t *testing.T, term *CompiledTerm, index int, isOperator, isTableReference, isUnknown, isValue bool, value string) {
	assert.Assert(t, term.Index == index)
	assert.Assert(t, term.IsOperator == isOperator)
	assert.Assert(t, term.IsTableReference == isTableReference)
	assert.Assert(t, term.IsUnknown == isUnknown)
	assert.Assert(t, term.IsValue == isValue)
	assert.Equal(t, term.Value, value)
}

func TestCompileResponseTermBoolean(t *testing.T) {
	term := CompileResponseTerm{Type: "boolean", Value: true}
	temC, err := term.Compile("", "", "")
	assert.NilError(t, err)
	AssertCompiledTerm(t, temC, 0, false, false, false, true, "true")
}

func TestCompileResponseTermNumber(t *testing.T) {
	term := CompileResponseTerm{Type: "number", Value: 3}
	temC, err := term.Compile("", "", "")
	assert.NilError(t, err)
	AssertCompiledTerm(t, temC, 0, false, false, false, true, "3")

	term = CompileResponseTerm{Type: "number", Value: 3.14}
	temC, err = term.Compile("", "", "")
	assert.NilError(t, err)
	AssertCompiledTerm(t, temC, 0, false, false, false, true, "3.14")
}

func TestCompileResponseTermString(t *testing.T) {
	term := CompileResponseTerm{Type: "string", Value: "mystring"}
	temC, err := term.Compile("", "", "")
	assert.NilError(t, err)
	AssertCompiledTerm(t, temC, 0, false, false, false, true, "mystring")

	temC, err = term.Compile("'", "", "")
	assert.NilError(t, err)
	AssertCompiledTerm(t, temC, 0, false, false, false, true, "'mystring'")

	temC, err = term.Compile("\"", "", "")
	assert.NilError(t, err)
	AssertCompiledTerm(t, temC, 0, false, false, false, true, "\"mystring\"")
}

func TestCompileResponseTermOperators(t *testing.T) {
	term := CompileResponseTerm{Type: "var", Value: "eq"}
	temC, err := term.Compile("", "", "")
	assert.NilError(t, err)
	AssertCompiledTerm(t, temC, 0, true, false, false, false, "=")

	term = CompileResponseTerm{Type: "var", Value: "equal"}
	temC, err = term.Compile("", "", "")
	assert.NilError(t, err)
	AssertCompiledTerm(t, temC, 0, true, false, false, false, "=")

	term = CompileResponseTerm{Type: "var", Value: "neq"}
	temC, err = term.Compile("", "", "")
	assert.NilError(t, err)
	AssertCompiledTerm(t, temC, 0, true, false, false, false, "!=")

	term = CompileResponseTerm{Type: "var", Value: "lt"}
	temC, err = term.Compile("", "", "")
	assert.NilError(t, err)
	AssertCompiledTerm(t, temC, 0, true, false, false, false, "<")

	term = CompileResponseTerm{Type: "var", Value: "lte"}
	temC, err = term.Compile("", "", "")
	assert.NilError(t, err)
	AssertCompiledTerm(t, temC, 0, true, false, false, false, "<=")

	term = CompileResponseTerm{Type: "var", Value: "gt"}
	temC, err = term.Compile("", "", "")
	assert.NilError(t, err)
	AssertCompiledTerm(t, temC, 0, true, false, false, false, ">")

	term = CompileResponseTerm{Type: "var", Value: "gte"}
	temC, err = term.Compile("", "", "")
	assert.NilError(t, err)
	AssertCompiledTerm(t, temC, 0, true, false, false, false, ">=")
}

func TestCompileResponseTermOperatorUnknown(t *testing.T) {
	term := CompileResponseTerm{Type: "var", Value: "data"}
	temC, err := term.Compile("", "", "")
	assert.NilError(t, err)
	AssertCompiledTerm(t, temC, 0, false, false, true, false, "data")
}

func TestCompileResponseTermOperatorsEdgeCases(t *testing.T) {
	term := CompileResponseTerm{Type: "var", Value: 3}
	_, err := term.Compile("", "", "")
	assert.Error(t, err, "unexpected type for var value: int (value: 3)")

	term = CompileResponseTerm{Type: "var", Value: "unknown"}
	_, err = term.Compile("", "", "")
	assert.Error(t, err, "unexpected value for var type: unknown")
}

func TestCompileResponseTermRefOperator(t *testing.T) {
	term := CompileResponseTerm{Type: "ref", Value: []interface{}{map[string]interface{}{"type": "var", "value": "neq"}}}
	temC, err := term.Compile("", "", "")
	assert.NilError(t, err)
	AssertCompiledTerm(t, temC, 0, true, false, false, false, "!=")
}

func TestCompileResponseTermRefOperatorEdgeCases(t *testing.T) {
	// Empty value
	term := CompileResponseTerm{Type: "ref", Value: []interface{}{}}
	_, err := term.Compile("", "", "")
	assert.Error(t, err, "unexpected number of terms in ref value: 0 (value: [])")

	// Missing type
	term = CompileResponseTerm{Type: "ref", Value: []interface{}{map[string]interface{}{}}}
	_, err = term.Compile("", "", "")
	assert.Error(t, err, "missing type for ref value term: map[] (value [map[]])")

	// Missing value
	term = CompileResponseTerm{Type: "ref", Value: []interface{}{map[string]interface{}{"type": "var"}}}
	_, err = term.Compile("", "", "")
	assert.Error(t, err, "missing value for ref value term: map[type:var] (value [map[type:var]])")

	// Wrong data type of type
	term = CompileResponseTerm{Type: "ref", Value: []interface{}{map[string]interface{}{"type": 3, "value": "neq"}}}
	_, err = term.Compile("", "", "")
	assert.Error(t, err, "unexpected type for ref value term type: int (value: [map[type:3 value:neq]])")

	// Bad inside term
	term = CompileResponseTerm{Type: "ref", Value: []interface{}{map[string]interface{}{"type": "var", "value": "unknown"}}}
	_, err = term.Compile("", "", "")
	assert.Error(t, err, "unexpected value for var type: unknown")

	// Bad interface type
	term = CompileResponseTerm{Type: "ref", Value: []string{"a"}}
	_, err = term.Compile("", "", "")
	assert.Error(t, err, "unexpected type for ref value: []string (value: [a])")

	// Bad interface type 2
	term = CompileResponseTerm{Type: "ref", Value: []interface{}{"ahoj"}}
	_, err = term.Compile("", "", "")
	assert.Error(t, err, "unexpected type for ref value term: string (value: [ahoj])")

	// Bad operator data
	term = CompileResponseTerm{Type: "ref", Value: []interface{}{map[string]interface{}{"type": "var", "value": "eq"}, map[string]interface{}{"type": "var", "value": "eq"}}}
	_, err = term.Compile("", "", "")
	assert.Error(t, err, "unexpected number of terms in operator ref value: 2 (value [map[type:var value:eq] map[type:var value:eq]])")
}

func TestCompileResponseTermRefUnknown(t *testing.T) {
	term := CompileResponseTerm{Type: "ref", Value: []interface{}{
		map[string]interface{}{"type": "var", "value": "data"},
		map[string]interface{}{"type": "string", "value": "tables"},
		map[string]interface{}{"type": "string", "value": "tablename"},
		map[string]interface{}{"type": "string", "value": "secondtablename"},
	}}

	temC, err := term.Compile("'", "tablename", "ts")
	assert.NilError(t, err)
	AssertCompiledTerm(t, temC, 0, false, true, false, false, "ts.secondtablename")

	temC, err = term.Compile("'", "tablename", "")
	assert.NilError(t, err)
	AssertCompiledTerm(t, temC, 0, false, true, false, false, "tablename.secondtablename")
}

func TestCompileResponseTermRefUnknownEdgeCases(t *testing.T) {
	term := CompileResponseTerm{Type: "ref", Value: []interface{}{
		map[string]interface{}{"type": "var", "value": "data"},
	}}

	_, err := term.Compile("", "", "")
	assert.Error(t, err, "unexpected number of terms in unknown ref value: 1 (value: [map[type:var value:data]])")

	term = CompileResponseTerm{Type: "ref", Value: []interface{}{
		map[string]interface{}{"type": "var", "value": "data"},
		map[string]interface{}{"type": "string", "value": "blah"},
		map[string]interface{}{"type": "string", "value": "bleh"},
	}}

	_, err = term.Compile("", "", "")
	assert.Error(t, err, "unexpected value for unknown ref value: blah (value: [map[type:var value:data] map[type:string value:blah] map[type:string value:bleh]])")
}

func TestCompileResponseTermEdges(t *testing.T) {
	term := CompileResponseTerm{Type: "ref", Value: []interface{}{map[string]interface{}{"type": "string", "value": "blah"}}}
	_, err := term.Compile("", "", "")
	assert.Error(t, err, "failed to parse ref value: [map[type:string value:blah]] (value: [map[type:string value:blah]])")

	term = CompileResponseTerm{Type: "unknown", Value: "blah"}
	_, err = term.Compile("", "", "")
	assert.Error(t, err, "unexpected type for term: unknown (value: blah)")
}

func TestCompileResponseQuery(t *testing.T) {
	rq := CompileResponseQuery{Index: 0, Negated: false, Terms: []CompileResponseTerm{
		{Type: "boolean", Value: true},
		{Type: "ref", Value: []interface{}{map[string]interface{}{"type": "var", "value": "eq"}}},
		{Type: "ref", Value: []interface{}{
			map[string]interface{}{"type": "var", "value": "data"},
			map[string]interface{}{"type": "string", "value": "tables"},
			map[string]interface{}{"type": "string", "value": "tablename"},
			map[string]interface{}{"type": "string", "value": "columnname"},
		}},
	}}

	result, err := rq.Compile("'", "tablename", "")
	assert.NilError(t, err)
	assert.Equal(t, result, "tablename.columnname = true")

	result, err = rq.Compile("'", "tablename", "t")
	assert.NilError(t, err)
	assert.Equal(t, result, "t.columnname = true")

	rq.Negated = true
	result, err = rq.Compile("'", "tablename", "t")
	assert.NilError(t, err)
	assert.Equal(t, result, "NOT (t.columnname = true)")
}

func TestCompileResponseQueryEdgeCases(t *testing.T) {
	rq := CompileResponseQuery{Index: 0, Negated: false, Terms: []CompileResponseTerm{}}
	_, err := rq.Compile("", "", "")
	assert.Error(t, err, "unexpected number of terms in query: 0")

	rq = CompileResponseQuery{Index: 0, Negated: false, Terms: []CompileResponseTerm{{Type: "blah", Value: "bleh"}, {Type: "blah", Value: "bleh"}, {Type: "blah", Value: "bleh"}}}
	_, err = rq.Compile("", "", "")
	assert.Error(t, err, "failed to compile query: unexpected type for term: blah (value: bleh)")

	rq = CompileResponseQuery{Index: 0, Negated: false, Terms: []CompileResponseTerm{
		{Type: "boolean", Value: true},
		{Type: "ref", Value: []interface{}{map[string]interface{}{"type": "var", "value": "eq"}}},
		{Type: "boolean", Value: true},
	}}
	_, err = rq.Compile("", "", "")
	assert.Error(t, err, "index already used: 2 (value true)")
}

func TestCompileResponse(t *testing.T) {
	cr := CompileResponse{Result: CompileResponseResult{Queries: [][]CompileResponseQuery{
		{
			{Index: 0, Negated: false, Terms: []CompileResponseTerm{
				{Type: "string", Value: "val1"},
				{Type: "ref", Value: []interface{}{map[string]interface{}{"type": "var", "value": "eq"}}},
				{Type: "ref", Value: []interface{}{
					map[string]interface{}{"type": "var", "value": "data"},
					map[string]interface{}{"type": "string", "value": "tables"},
					map[string]interface{}{"type": "string", "value": "tablename"},
					map[string]interface{}{"type": "string", "value": "columnname1"},
				}},
			}},
			{Index: 1, Negated: true, Terms: []CompileResponseTerm{
				{Type: "string", Value: "val2"},
				{Type: "ref", Value: []interface{}{map[string]interface{}{"type": "var", "value": "eq"}}},
				{Type: "ref", Value: []interface{}{
					map[string]interface{}{"type": "var", "value": "data"},
					map[string]interface{}{"type": "string", "value": "tables"},
					map[string]interface{}{"type": "string", "value": "tablename"},
					map[string]interface{}{"type": "string", "value": "columnname2"},
				}},
			}},
		},
		{
			{Index: 0, Negated: false, Terms: []CompileResponseTerm{
				{Type: "number", Value: 12},
				{Type: "ref", Value: []interface{}{map[string]interface{}{"type": "var", "value": "gte"}}},
				{Type: "ref", Value: []interface{}{
					map[string]interface{}{"type": "var", "value": "data"},
					map[string]interface{}{"type": "string", "value": "tables"},
					map[string]interface{}{"type": "string", "value": "tablename"},
					map[string]interface{}{"type": "string", "value": "columnname3"},
				}},
			}},
		},
	}}}
	res, err := cr.Compile("'", "tablename", "t")
	assert.NilError(t, err)
	assert.Equal(t, res, "((t.columnname1 = 'val1') AND (NOT (t.columnname2 = 'val2'))) OR ((t.columnname3 >= 12))")
}

func TestCompileResponseEdge(t *testing.T) {
	cr := CompileResponse{Result: CompileResponseResult{Queries: [][]CompileResponseQuery{
		{{Index: 0, Negated: false, Terms: []CompileResponseTerm{{Type: "string", Value: "val1"}}}},
	}}}
	_, err := cr.Compile("", "", "")
	assert.Error(t, err, "failed to compile response: unexpected number of terms in query: 1")
}

func TestOPASQLBuildPayload(t *testing.T) {
	opa := NewOPASQL("opa-server", "data.{{ .TableName }}.allow == true", "'", map[string]interface{}{"preferred_username": "test"}, nil)
	payload, err := opa.BuildPayload("tablename")
	assert.NilError(t, err)
	assert.DeepEqual(t, payload, &CompilePayload{
		Query:    "data.tablename.allow == true",
		Unknowns: []string{"data.tables"},
		Input:    CompilePayloadInput{UserInfo: map[string]interface{}{"preferred_username": "test"}},
	})
}

func TestOPASQLBuildPayloadFailures(t *testing.T) {
	opa := NewOPASQL("opa-server", "data.{{ eq .TableName }}.allow == true", "'", map[string]interface{}{"preferred_username": "test"}, nil)
	_, err := opa.BuildPayload("tablename")
	assert.Error(t, err, "failed to execute query template: template: query:1:8: executing \"query\" at <eq .TableName>: error calling eq: missing argument for comparison")
}
