package model

// GroupInfo GET /group 返回的单个策略组信息
type GroupInfo struct {
	Name string   `json:"name"`
	Type string   `json:"type"`
	Now  string   `json:"now"`  // 当前选中节点
	All  []string `json:"all"`  // 该组内所有节点名
	UDP  bool     `json:"udp"`
}

// GroupResponse GET /group 返回
type GroupResponse struct {
	Proxies []GroupInfo `json:"proxies"`
}

// DelayHistory 延迟历史条目
type DelayHistory struct {
	Time  string `json:"time"`
	Delay int    `json:"delay"`
}

// ProxyInfo GET /proxies/:name 返回 (也可以是策略组详情)
type ProxyInfo struct {
	Name    string         `json:"name"`
	Type    string         `json:"type"`
	Now     string         `json:"now"`
	All     []string       `json:"all"`
	History []DelayHistory `json:"history"`
	UDP     bool           `json:"udp"`
}

// SwitchRequest PUT /proxies/:name 请求体
type SwitchRequest struct {
	Name string `json:"name"`
}

// LogLine GET /logs 返回的逐行 JSON
type LogLine struct {
	Type    string `json:"type"`
	Payload string `json:"payload"`
}

// SpeedTestResult 测速结果
type SpeedTestResult struct {
	Delays map[string]int // 节点名 → 延迟(ms)
	Err    error
}

// ConfigInfo GET /configs 返回
type ConfigInfo struct {
	Mode      string `json:"mode"`
	Port      int    `json:"port"`
	SocksPort int    `json:"socks-port"`
	MixedPort int    `json:"mixed-port"`
	AllowLan  bool   `json:"allow-lan"`
	LogLevel  string `json:"log-level"`
	IPv6      bool   `json:"ipv6"`
}

// TrafficInfo GET /traffic 返回
type TrafficInfo struct {
	Up        int64 `json:"up"`
	Down      int64 `json:"down"`
	UpTotal   int64 `json:"upTotal"`
	DownTotal int64 `json:"downTotal"`
}

// MemoryInfo GET /memory 返回
type MemoryInfo struct {
	Inuse   int64 `json:"inuse"`
	Oslimit int64 `json:"oslimit"`
}

// ConnectionItem 单条连接信息
type ConnectionItem struct {
	ID       string         `json:"id"`
	Metadata ConnectionMeta `json:"metadata"`
	Upload   int64          `json:"upload"`
	Download int64          `json:"download"`
	Start    string         `json:"start"`
	Chains   []string       `json:"chains"`
	Rule     string         `json:"rule"`
	RulePayload string      `json:"rulePayload"`
}

// ConnectionMeta 连接元数据
type ConnectionMeta struct {
	Network    string `json:"network"`
	Type       string `json:"type"`
	SourceIP   string `json:"sourceIP"`
	SourcePort string `json:"sourcePort"`
	DestinationIP   string `json:"destinationIP"`
	DestinationPort string `json:"destinationPort"`
	Host       string `json:"host"`
}

// ConnectionsResponse GET /connections 返回
type ConnectionsResponse struct {
	DownloadTotal int64            `json:"downloadTotal"`
	UploadTotal   int64            `json:"uploadTotal"`
	Memory        int64            `json:"memory"`
	Connections   []ConnectionItem `json:"connections"`
}

// RuleItem 单条规则
type RuleItem struct {
	Type    string `json:"type"`
	Payload string `json:"payload"`
	Proxy   string `json:"proxy"`
	Size    int    `json:"size"`
}

// RulesResponse GET /rules 返回
type RulesResponse struct {
	Rules []RuleItem `json:"rules"`
}
