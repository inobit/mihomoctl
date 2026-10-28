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

// SetTimeout 调整后续请求的超时，用于 provider 拉取这类同步慢操作
func (c *Client) SetTimeout(d time.Duration) {
	c.HTTPClient.Timeout = d
}

func (c *Client) doRequest(method, path string, body any) (*http.Response, error) {
	reqURL := c.BaseURL + path

	var bodyReader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("marshal request body: %w", err)
		}
		bodyReader = bytes.NewReader(data)
	}

	req, err := http.NewRequest(method, reqURL, bodyReader)
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
		errBody, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
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
	resp, err := c.doRequest("GET", "/proxies/"+url.PathEscape(name), nil)
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
	resp, err := c.doRequest("PUT", "/proxies/"+url.PathEscape(group), model.SwitchRequest{Name: node})
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

// ReloadConfig PUT /configs 重载配置。
// path 为空时不指定文件，由 mihomo 按启动时记录的配置文件重载；
// force 为 true 时带 ?force=true（除代理/规则外还重建全部入站监听），否则不传该参数（mihomo 默认 false）。
func (c *Client) ReloadConfig(path string, force bool) error {
	p := "/configs"
	if force {
		p += "?force=true"
	}
	resp, err := c.doRequest("PUT", p, model.ConfigReloadRequest{Path: path})
	if err != nil {
		return err
	}
	resp.Body.Close()
	return nil
}

// PatchConfig 局部更新运行配置，未提交的字段保持当前值
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

	// /traffic 为持续推送的流式端点（每秒一条 NDJSON），单次 Decode 会阻塞至超时
	// 仅读取首条即返回，实现一次性快照语义
	scanner := bufio.NewScanner(resp.Body)
	if scanner.Scan() {
		var info model.TrafficInfo
		if err := json.Unmarshal(scanner.Bytes(), &info); err != nil {
			return nil, fmt.Errorf("decode response: %w", err)
		}
		return &info, nil
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("read traffic: %w", err)
	}
	return nil, fmt.Errorf("empty traffic response")
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

// GetProxyProviders GET /providers/proxies 列出 proxy providers
func (c *Client) GetProxyProviders() (map[string]model.ProviderInfo, error) {
	return c.getProviders("/providers/proxies")
}

// GetRuleProviders GET /providers/rules 列出 rule providers
func (c *Client) GetRuleProviders() (map[string]model.ProviderInfo, error) {
	return c.getProviders("/providers/rules")
}

func (c *Client) getProviders(path string) (map[string]model.ProviderInfo, error) {
	resp, err := c.doRequest("GET", path, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var out model.ProvidersResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	return out.Providers, nil
}

// UpdateProxyProvider PUT /providers/proxies/{name} 强制重新拉取 proxy provider（忽略 interval）
func (c *Client) UpdateProxyProvider(name string) error {
	return c.updateProvider("proxies", name)
}

// UpdateRuleProvider PUT /providers/rules/{name} 强制重新拉取 rule provider（忽略 interval）
func (c *Client) UpdateRuleProvider(name string) error {
	return c.updateProvider("rules", name)
}

func (c *Client) updateProvider(kind, name string) error {
	resp, err := c.doRequest("PUT", "/providers/"+kind+"/"+url.PathEscape(name), nil)
	if err != nil {
		return err
	}
	resp.Body.Close()
	return nil
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
