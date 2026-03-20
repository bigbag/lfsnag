package client

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/bigbag/lfsnag/internal/logfire"
	"github.com/bigbag/lfsnag/internal/output"
)

type Client struct {
	token      string
	baseURL    string
	printer    *output.Printer
	httpClient *http.Client
}

func New(token, baseURL string, printer *output.Printer) *Client {
	return &Client{
		token:   token,
		baseURL: baseURL,
		printer: printer,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (c *Client) Query(sql string) (json.RawMessage, error) {
	endpoint := logfire.Endpoint(c.baseURL)

	reqURL := endpoint + "?" + url.Values{"sql": {sql}}.Encode()

	headers := map[string]string{
		"Authorization": "Bearer " + c.token,
	}
	c.printer.PrintRequest("GET", reqURL, headers, nil)

	req, err := http.NewRequest("GET", reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.token)

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

func (c *Client) QueryTrace(traceID, fields string) (json.RawMessage, error) {
	return c.Query(logfire.BuildTraceQuery(traceID, fields))
}
