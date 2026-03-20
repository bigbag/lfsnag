package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/bigbag/lfsnag/internal/logfire"
	"github.com/bigbag/lfsnag/internal/output"
)

type Client struct {
	token      string
	project    string
	baseURL    string
	printer    *output.Printer
	httpClient *http.Client
}

func New(token, project, baseURL string, printer *output.Printer) *Client {
	return &Client{
		token:   token,
		project: project,
		baseURL: baseURL,
		printer: printer,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (c *Client) QueryTrace(traceID string) (json.RawMessage, error) {
	sql := logfire.BuildTraceQuery(traceID)
	url := logfire.Endpoint(c.baseURL, c.project)

	reqBody := logfire.QueryRequest{SQL: sql}
	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	headers := map[string]string{
		"Authorization": "Bearer " + c.token,
		"Content-Type":  "application/json",
	}
	c.printer.PrintRequest("POST", url, headers, body)

	req, err := http.NewRequest("POST", url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	c.printer.PrintResponse(resp)

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned %s: %s", resp.Status, string(respBody))
	}

	return json.RawMessage(respBody), nil
}
