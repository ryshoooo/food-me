package foodme

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

type HTTPPermissionAgent struct {
	DDLEndpoint    string
	SelectEndpoint string

	client        IHttpClient
	createAllowed bool
	updateAllowed bool
	deleteAllowed bool
}

type DDLPayload struct {
	UserInfo  map[string]interface{} `json:"userInfo"`
	Operation string                 `json:"operation"`
}

type DDLResponse struct {
	Allowed bool `json:"allowed"`
}

type SelectPayload struct {
	UserInfo   map[string]interface{} `json:"userInfo"`
	TableName  string                 `json:"tableName"`
	TableAlias string                 `json:"tableAlias"`
}

type SelectResponse struct {
	Allowed bool           `json:"allowed"`
	Filters *SelectFilters `json:"filters"`
}

func NewHTTPPermissionAgent(ddlEndpoint, selectEndpoint string, httpClient IHttpClient) IPermissionAgent {
	return &HTTPPermissionAgent{DDLEndpoint: ddlEndpoint, SelectEndpoint: selectEndpoint, client: httpClient}
}

func (h *HTTPPermissionAgent) ddlQuery(operation string, userInfo map[string]interface{}) (*DDLResponse, error) {
	payload := &DDLPayload{UserInfo: userInfo, Operation: operation}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal json payload: %w", err)
	}

	respBody, err := h.query(http.MethodPost, h.DDLEndpoint, body)
	if err != nil {
		return nil, fmt.Errorf("failed to query ddl: %w", err)
	}

	ddlResp := &DDLResponse{}
	if err := json.Unmarshal(respBody, ddlResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response body: %w", err)
	}

	return ddlResp, nil
}

func (h *HTTPPermissionAgent) query(method, endpoint string, body []byte) ([]byte, error) {
	req, err := http.NewRequest(method, endpoint, bytes.NewBuffer(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create http request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	resp, err := h.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}

	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	return respBody, nil
}

func (h *HTTPPermissionAgent) SetCreateAllowed(userInfo map[string]interface{}) error {
	ddlResp, err := h.ddlQuery("create", userInfo)
	if err != nil {
		return fmt.Errorf("failed to query ddl: %w", err)
	}

	h.createAllowed = ddlResp.Allowed
	return nil
}

func (h *HTTPPermissionAgent) SetUpdateAllowed(userInfo map[string]interface{}) error {
	ddlResp, err := h.ddlQuery("update", userInfo)
	if err != nil {
		return fmt.Errorf("failed to query ddl: %w", err)
	}

	h.updateAllowed = ddlResp.Allowed
	return nil
}

func (h *HTTPPermissionAgent) SetDeleteAllowed(userInfo map[string]interface{}) error {
	ddlResp, err := h.ddlQuery("delete", userInfo)
	if err != nil {
		return fmt.Errorf("failed to query ddl: %w", err)
	}

	h.deleteAllowed = ddlResp.Allowed
	return nil
}

func (h *HTTPPermissionAgent) SelectFilters(tableName, tableAlias string, userInfo map[string]interface{}) (*SelectFilters, error) {
	payload := &SelectPayload{UserInfo: userInfo, TableName: tableName, TableAlias: tableAlias}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal json payload: %w", err)
	}

	respBody, err := h.query(http.MethodPost, h.SelectEndpoint, body)
	if err != nil {
		return nil, fmt.Errorf("failed to query select filters: %w", err)
	}

	selectResp := &SelectResponse{}
	if err := json.Unmarshal(respBody, selectResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response body: %w", err)
	}

	if !selectResp.Allowed {
		return nil, fmt.Errorf("permission denied to access table %s", tableName)
	}

	if selectResp.Filters == nil {
		return &SelectFilters{WhereFilters: []string{}, JoinFilters: []*JoinFilter{}}, nil
	}

	return selectResp.Filters, nil

}

func (h *HTTPPermissionAgent) CreateAllowed() bool {
	return h.createAllowed
}

func (h *HTTPPermissionAgent) UpdateAllowed() bool {
	return h.updateAllowed
}

func (h *HTTPPermissionAgent) DeleteAllowed() bool {
	return h.deleteAllowed
}
