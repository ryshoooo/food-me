package foodme

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"text/template"
)

// Compile Payload
type CompilePayloadInput struct {
	UserInfo map[string]interface{} `json:"userinfo"`
}

type CompilePayload struct {
	Query    string              `json:"query"`
	Unknowns []string            `json:"unknowns"`
	Input    CompilePayloadInput `json:"input"`
}

// Compile Response
type CompileResponse struct {
	Result CompileResponseResult `json:"result"`
}

func (c *CompileResponse) IsAllowed() bool {
	for _, query := range c.Result.Queries {
		if len(query) == 0 {
			return true
		}
	}
	return false
}

func (c *CompileResponse) IsDisallowed() bool {
	return len(c.Result.Queries) == 0
}

func (c *CompileResponse) Compile(stringEscapeChar, tableName, tableAlias string) (string, error) {
	resp := make([]string, len(c.Result.Queries))

	for qidx, query := range c.Result.Queries {
		iresp := make([]string, len(query))
		allExtraTables := []string{}

		for qqidx, iq := range query {
			cnd, err := iq.Compile(stringEscapeChar, tableName, tableAlias)
			if err != nil {
				return "", fmt.Errorf("failed to compile response: %w", err)
			}
			iresp[qqidx] = fmt.Sprintf("(%s)", cnd.Value)

			for _, et := range cnd.ExtraTables {
				if !contains(allExtraTables, et) {
					allExtraTables = append(allExtraTables, et)
				}
			}
		}

		if len(allExtraTables) > 0 {
			resp[qidx] = fmt.Sprintf("(exists (select 1 from %s where (%s)))", strings.Join(allExtraTables, ", "), strings.Join(iresp, " AND "))
		} else {
			resp[qidx] = fmt.Sprintf("(%s)", strings.Join(iresp, " AND "))
		}
	}

	return strings.Join(resp, " OR "), nil
}

type CompileResponseResult struct {
	Queries [][]CompileResponseQuery `json:"queries"`
}

type CompileResponseQuery struct {
	Index   int                   `json:"index"`
	Negated bool                  `json:"negated"`
	Terms   []CompileResponseTerm `json:"terms"`
}

func (c *CompileResponseQuery) Compile(stringEscapeChart, tableName, tableAlias string) (*CompiledQuery, error) {
	if len(c.Terms) != 3 {
		return nil, fmt.Errorf("unexpected number of terms in query: %d", len(c.Terms))
	}

	ra := make([]*CompiledTerm, len(c.Terms))
	for idx, term := range c.Terms {
		ct, err := term.Compile(stringEscapeChart, tableName, tableAlias)
		if err != nil {
			return nil, fmt.Errorf("failed to compile query: %w", err)
		}
		ra[idx] = ct
	}

	_ = setIndicesForCompiledTerms(ra)
	result := make([]string, 3)
	extraTables := []string{}
	for _, compiledTerm := range ra {
		if result[compiledTerm.Index] != "" {
			return nil, fmt.Errorf("index already used: %d (value %s)", compiledTerm.Index, compiledTerm.Value)
		}
		result[compiledTerm.Index] = compiledTerm.Value
		if compiledTerm.IsTableReference {
			extraTableName := strings.Split(compiledTerm.Value, ".")[0]
			if extraTableName != tableName && extraTableName != tableAlias {
				extraTables = append(extraTables, extraTableName)
			}
		}
	}

	f := strings.Join(result, " ")
	if c.Negated {
		return &CompiledQuery{Value: fmt.Sprintf("NOT (%s)", f), ExtraTables: extraTables}, nil
	} else {
		return &CompiledQuery{Value: f, ExtraTables: extraTables}, nil
	}
}

type CompileResponseTerm struct {
	Type  string      `json:"type"`
	Value interface{} `json:"value"`
}

func (c *CompileResponseTerm) Compile(stringEscapeChar, tableName, tableAlias string) (*CompiledTerm, error) {
	switch c.Type {
	case "boolean", "number":
		return &CompiledTerm{Value: fmt.Sprintf("%v", c.Value), IsValue: true}, nil
	case "string":
		return &CompiledTerm{Value: fmt.Sprintf("%s%v%s", stringEscapeChar, c.Value, stringEscapeChar), IsValue: true}, nil
	case "ref":
		switch vt := c.Value.(type) {
		case []interface{}:
			if len(vt) < 1 {
				return nil, fmt.Errorf("unexpected number of terms in ref value: %d (value: %v)", len(vt), c.Value)
			}

			vtc := make([]*CompiledTerm, len(vt))
			for idx, term := range vt {
				switch termt := term.(type) {
				case map[string]interface{}:
					if _, ok := termt["type"]; !ok {
						return nil, fmt.Errorf("missing type for ref value term: %v (value %v)", termt, c.Value)
					}
					if _, ok := termt["value"]; !ok {
						return nil, fmt.Errorf("missing value for ref value term: %v (value %v)", termt, c.Value)
					}

					var termtType string
					switch termt["type"].(type) {
					case string:
						termtType = termt["type"].(string)
					default:
						return nil, fmt.Errorf("unexpected type for ref value term type: %T (value: %v)", termt["type"], c.Value)
					}

					parsedTerm := &CompileResponseTerm{Type: termtType, Value: termt["value"]}
					termCompiled, err := parsedTerm.Compile("", tableName, tableAlias)
					if err != nil {
						return nil, err
					}

					vtc[idx] = termCompiled
				default:
					return nil, fmt.Errorf("unexpected type for ref value term: %T (value: %v)", termt, c.Value)
				}
			}

			if vtc[0].IsOperator && len(vtc) != 1 {
				return nil, fmt.Errorf("unexpected number of terms in operator ref value: %d (value %v)", len(vtc), c.Value)
			} else if vtc[0].IsOperator {
				return &CompiledTerm{IsOperator: true, Value: vtc[0].Value}, nil
			}

			if vtc[0].IsUnknown && len(vtc) < 3 {
				return nil, fmt.Errorf("unexpected number of terms in unknown ref value: %d (value: %v)", len(vtc), c.Value)
			} else if vtc[0].IsUnknown && vtc[1].Value != "tables" {
				return nil, fmt.Errorf("unexpected value for unknown ref value: %s (value: %v)", vtc[1].Value, c.Value)
			} else if vtc[0].IsUnknown {
				var tb string
				if vtc[2].Value == tableName && tableAlias != "" {
					tb = tableAlias
				} else {
					tb = vtc[2].Value
				}

				cvtc := make([]string, len(vtc[3:]))
				for idx, vtc_ := range vtc[3:] {
					cvtc[idx] = vtc_.Value
				}
				col := strings.Join(cvtc, ".")

				return &CompiledTerm{IsTableReference: true, Value: fmt.Sprintf("%s.%s", tb, col)}, nil
			}

			return nil, fmt.Errorf("failed to parse ref value: %s (value: %v)", vt, c.Value)
		default:
			return nil, fmt.Errorf("unexpected type for ref value: %T (value: %v)", vt, c.Value)
		}

	case "var":
		switch vt := c.Value.(type) {
		case string:
			switch vt {
			case "eq", "equal":
				return &CompiledTerm{IsOperator: true, Value: "="}, nil
			case "neq":
				return &CompiledTerm{IsOperator: true, Value: "!="}, nil
			case "lt":
				return &CompiledTerm{IsOperator: true, Value: "<"}, nil
			case "lte":
				return &CompiledTerm{IsOperator: true, Value: "<="}, nil
			case "gt":
				return &CompiledTerm{IsOperator: true, Value: ">"}, nil
			case "gte":
				return &CompiledTerm{IsOperator: true, Value: ">="}, nil
			case "data":
				return &CompiledTerm{IsUnknown: true, Value: "data"}, nil
			default:
				return nil, fmt.Errorf("unexpected value for var type: %s", c.Value)
			}
		default:
			return nil, fmt.Errorf("unexpected type for var value: %T (value: %v)", vt, c.Value)
		}
	}

	return nil, fmt.Errorf("unexpected type for term: %s (value: %s)", c.Type, c.Value)
}

type CompiledTerm struct {
	Value            string
	IsTableReference bool
	IsOperator       bool
	IsValue          bool
	IsUnknown        bool
	Index            int
}

type CompiledQuery struct {
	Value       string
	ExtraTables []string
}

// Template context
type TemplateContext struct {
	TableName string
}

type OPASQL struct {
	Address             string
	SelectQueryTemplate string
	CreateQuery         string
	UpdateQuery         string
	DeleteQuery         string
	StringEscapeChar    string
	httpClient          IHttpClient

	allowedCreate bool
	allowedUpdate bool
	allowedDelete bool
}

func NewOPASQL(
	address, selectQueryTemplate, createQuery, updateQuery, deleteQuery, stringEscapeChar string,
	httpClient IHttpClient,
) *OPASQL {
	return &OPASQL{
		Address:             address,
		SelectQueryTemplate: selectQueryTemplate,
		CreateQuery:         createQuery,
		UpdateQuery:         updateQuery,
		DeleteQuery:         deleteQuery,
		httpClient:          httpClient,
		StringEscapeChar:    stringEscapeChar,
	}
}

func (o *OPASQL) SetCreateAllowed(userInfo map[string]interface{}) error {
	allowed, err := o.getDDLAllowed("create", userInfo)
	if err != nil {
		return err
	}
	o.allowedCreate = allowed
	return nil
}

func (o *OPASQL) SetUpdateAllowed(userInfo map[string]interface{}) error {
	allowed, err := o.getDDLAllowed("update", userInfo)
	if err != nil {
		return err
	}
	o.allowedUpdate = allowed
	return nil
}

func (o *OPASQL) SetDeleteAllowed(userInfo map[string]interface{}) error {
	allowed, err := o.getDDLAllowed("delete", userInfo)
	if err != nil {
		return err
	}
	o.allowedDelete = allowed
	return nil
}

func (o *OPASQL) getDDLAllowed(operation string, userInfo map[string]interface{}) (bool, error) {
	payload, err := o.BuildPayload(operation, "", userInfo)
	if err != nil {
		return false, err
	}

	resp, err := o.Query(payload)
	if err != nil {
		return false, err
	}

	return resp.IsAllowed(), nil
}

func (o *OPASQL) Query(payload *CompilePayload) (*CompileResponse, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal json payload: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, fmt.Sprintf("%s/v1/compile", o.Address), bytes.NewBuffer(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := o.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code from OPA: %d", resp.StatusCode)
	}

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	compileResp := &CompileResponse{}
	err = json.Unmarshal(respBody, compileResp)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal response body: %w", err)
	}

	return compileResp, nil
}

func (o *OPASQL) BuildPayload(operation, tableName string, userInfo map[string]interface{}) (*CompilePayload, error) {
	var query string
	switch operation {
	case "create":
		query = o.CreateQuery
	case "update":
		query = o.UpdateQuery
	case "delete":
		query = o.DeleteQuery
	case "select":
		ctx := &TemplateContext{TableName: tableName}

		qtmpl, _ := template.New("query").Parse(o.SelectQueryTemplate)
		var qrs bytes.Buffer
		err := qtmpl.Execute(&qrs, ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to execute SELECT query template: %w", err)
		}

		query = qrs.String()
	default:
		return nil, fmt.Errorf("unexpected operation: %s", operation)
	}

	return &CompilePayload{Query: query, Unknowns: []string{"data.tables"}, Input: CompilePayloadInput{UserInfo: userInfo}}, nil
}

func (o *OPASQL) SelectFilters(tableName, tableAlias string, userInfo map[string]interface{}) (*SelectFilters, error) {
	payload, err := o.BuildPayload("select", tableName, userInfo)
	if err != nil {
		return nil, fmt.Errorf("failed to build payload: %w", err)
	}

	resp, err := o.Query(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to query OPA: %w", err)
	}

	if resp.IsAllowed() {
		return &SelectFilters{WhereFilters: []string{}, JoinFilters: []*JoinFilter{}}, nil
	}

	if resp.IsDisallowed() {
		return nil, fmt.Errorf("permission denied to access table %s", tableName)
	}

	wfs, err := resp.Compile(o.StringEscapeChar, tableName, tableAlias)
	if err != nil {
		return nil, fmt.Errorf("failed to compile response: %w", err)
	}

	return &SelectFilters{WhereFilters: []string{wfs}, JoinFilters: []*JoinFilter{}}, nil
}

func (o *OPASQL) CreateAllowed() bool {
	return o.allowedCreate
}

func (o *OPASQL) UpdateAllowed() bool {
	return o.allowedUpdate
}

func (o *OPASQL) DeleteAllowed() bool {
	return o.allowedDelete
}

func setIndicesForCompiledTerms(compiledTerms []*CompiledTerm) error {
	if len(compiledTerms) != 3 {
		return fmt.Errorf("unexpected number of terms in query: %d", len(compiledTerms))
	}

	existsValueTerm := false
	for _, ct := range compiledTerms {
		if ct.IsValue {
			existsValueTerm = true
			break
		}
	}

	hasSetTableReference := false
	for _, ct := range compiledTerms {
		if ct.IsOperator {
			ct.Index = 1
		}
		if ct.IsValue {
			ct.Index = 2
		}
		if ct.IsTableReference && existsValueTerm {
			ct.Index = 0
		} else if ct.IsTableReference && !hasSetTableReference {
			ct.Index = 2
			hasSetTableReference = true
		} else if ct.IsTableReference && hasSetTableReference {
			ct.Index = 0
		}
	}

	return nil
}
