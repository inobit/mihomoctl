package client

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"

	"github.com/inobit/mihomoctl/internal/model"
)

type Client struct {
	BaseURL    string
	Secret     string
	HTTPClient *http.Client
}

func New(server, secret string) *Client {
	return &Client{
		BaseURL: strings.TrimRight(server, "/"),
		Secret:  secret,
		HTTPClient: &http.Client{
			Timeout: 15 * time.Second,
		},
	}
}

func (c *Client) doRequest(method, path string, body any) (*http.Response, error) {
	url := c.BaseURL + path

	var bodyReader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("marshal request body: %w", err)
		}
		bodyReader = bytes.NewReader(data)
	}

	req, err := http.NewRequest(method, url, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if c.Secret != "" {
		req.Header.Set("Authorization", "Bearer "+c.Secret)
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}

	if resp.StatusCode >= 400 {
		defer resp.Body.Close()
		errBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error (%d): %s", resp.StatusCode, strings.TrimSpace(string(errBody)))
	}

	return resp, nil
}

func (c *Client) GetGroups() ([]model.GroupInfo, error) {
	resp, err := c.doRequest("GET", "/group", nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result model.GroupResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}

	// 服务端基于 map 迭代，顺序随机；客户端固定按名称排序，保证每次显示一致
	sort.Slice(result.Proxies, func(i, j int) bool {
		return result.Proxies[i].Name < result.Proxies[j].Name
	})

	return result.Proxies, nil
}

func (c *Client) GetProxy(name string) (*model.ProxyInfo, error) {
	resp, err := c.doRequest("GET", "/proxies/"+name, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var info model.ProxyInfo
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	return &info, nil
}

func (c *Client) SwitchProxy(group, node string) error {
	resp, err := c.doRequest("PUT", "/proxies/"+group, model.SwitchRequest{Name: node})
	if err != nil {
		return err
	}
	resp.Body.Close()
	return nil
}

func (c *Client) GetGroupDelay(groupName, testURL string, timeout int) (map[string]int, error) {
	path := fmt.Sprintf("/group/%s/delay?url=%s&timeout=%d", url.PathEscape(groupName), url.QueryEscape(testURL), timeout)

	resp, err := c.doRequest("GET", path, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result map[string]int
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	return result, nil
}

func (c *Client) GetConfig() (*model.ConfigInfo, error) {
	resp, err := c.doRequest("GET", "/configs", nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var cfg model.ConfigInfo
	if err := json.NewDecoder(resp.Body).Decode(&cfg); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	return &cfg, nil
}

func (c *Client) PatchConfig(patch map[string]any) error {
	resp, err := c.doRequest("PATCH", "/configs", patch)
	if err != nil {
		return err
	}
	resp.Body.Close()
	return nil
}

func (c *Client) GetTraffic() (*model.TrafficInfo, error) {
	resp, err := c.doRequest("GET", "/traffic", nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var info model.TrafficInfo
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	return &info, nil
}

func (c *Client) GetMemory() (*model.MemoryInfo, error) {
	resp, err := c.doRequest("GET", "/memory", nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var info model.MemoryInfo
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	return &info, nil
}

func (c *Client) GetConnections() (*model.ConnectionsResponse, error) {
	resp, err := c.doRequest("GET", "/connections", nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result model.ConnectionsResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	return &result, nil
}

func (c *Client) CloseConnection(id string) error {
	resp, err := c.doRequest("DELETE", "/connections/"+id, nil)
	if err != nil {
		return err
	}
	resp.Body.Close()
	return nil
}

func (c *Client) CloseAllConnections() error {
	resp, err := c.doRequest("DELETE", "/connections", nil)
	if err != nil {
		return err
	}
	resp.Body.Close()
	return nil
}

func (c *Client) FlushDNS() error {
	resp, err := c.doRequest("POST", "/cache/dns/flush", nil)
	if err != nil {
		return err
	}
	resp.Body.Close()
	return nil
}

func (c *Client) GetRules() ([]model.RuleItem, error) {
	resp, err := c.doRequest("GET", "/rules", nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result model.RulesResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	return result.Rules, nil
}

func (c *Client) StreamLogs(ctx context.Context, level string, out io.Writer) error {
	path := "/logs"
	if level != "" {
		path += "?level=" + level
	}

	req, err := http.NewRequestWithContext(ctx, "GET", c.BaseURL+path, nil)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	if c.Secret != "" {
		req.Header.Set("Authorization", "Bearer "+c.Secret)
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		errBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API error (%d): %s", resp.StatusCode, strings.TrimSpace(string(errBody)))
	}

	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		var logLine model.LogLine
		line := scanner.Text()
		if err := json.Unmarshal([]byte(line), &logLine); err != nil {
			fmt.Fprintf(out, "%s\n", line)
			continue
		}
		fmt.Fprintf(out, "%s\n", formatLogLine(logLine))
	}
	return scanner.Err()
}

func formatLogLine(l model.LogLine) string {
	var levelColor string
	switch l.Type {
	case "warning":
		levelColor = "\033[33m" // yellow
	case "error":
		levelColor = "\033[31m" // red
	case "debug":
		levelColor = "\033[36m" // cyan
	default:
		levelColor = "\033[37m" // white
	}
	reset := "\033[0m"
	return fmt.Sprintf("%s%-5s%s %s", levelColor, strings.ToUpper(l.Type), reset, l.Payload)
}
