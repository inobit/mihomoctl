package interact

import (
	"fmt"
	"sort"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/inobit/mihomoctl/internal/client"
	"github.com/inobit/mihomoctl/internal/model"
	"github.com/mattn/go-runewidth"
)

const autoRefreshInterval = 5 * time.Second

type refreshTickMsg struct{}

// viewMode 连接页的两个视图：逐条连接 / 按策略聚合
type viewMode int

const (
	viewConnections viewMode = iota
	viewPolicies
)

// policyStat 按策略/节点聚合的连接统计。
//
// 计数口径：某条连接的链路 (chains) 中包含该名称即计入。
// 因此一条连接会同时计入其链路里的每一层策略与末端节点，
// 中间策略的连接数天然包含其下级策略的连接（重叠计数、关闭时按连接 ID 去重）。
type policyStat struct {
	Name         string
	Kind         string   // 组类型 (Selector/URLTest/...) 或 "节点"
	IsGroup      bool     // 是否为策略组
	Intermediate bool     // 组内成员含其他组 = 中间策略，否则为末端策略
	Count        int      // 承载该策略/节点的连接数
	Sample       []string // 代表性链路（取最短的一条）
	IDs          []string // 去重后的连接 ID，用于批量断开
}

type connectionModel struct {
	client *client.Client

	mode     viewMode
	conns    []model.ConnectionItem
	groups   map[string]model.GroupInfo // 策略组信息（名称 → 组），用于区分组/节点
	policies []policyStat

	connCursor   int
	policyCursor int

	err      error
	loading  bool
	width    int
	height   int
	quitting bool

	autoRefresh bool
}

func ConnectionList(c *client.Client) error {
	m := &connectionModel{
		client:      c,
		autoRefresh: true,
	}

	p := tea.NewProgram(m, tea.WithAltScreen())
	_, err := p.Run()
	return err
}

func (m *connectionModel) Init() tea.Cmd {
	return tea.Batch(m.fetchData(), m.autoRefreshTick())
}

func (m *connectionModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.KeyMsg:
		return m.handleKey(msg)

	case refreshTickMsg:
		if m.autoRefresh {
			return m, tea.Batch(m.fetchData(), m.autoRefreshTick())
		}
		return m, nil

	case fetchResultMsg:
		m.loading = false
		m.err = msg.err
		if msg.err == nil {
			m.conns = msg.conns
			if msg.groups != nil {
				m.groups = make(map[string]model.GroupInfo, len(msg.groups))
				for _, g := range msg.groups {
					m.groups[g.Name] = g
				}
			}
			m.rebuildPolicies()
			if m.connCursor >= len(m.conns) {
				m.connCursor = 0
			}
		}
		return m, nil

	case closeResultMsg:
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}
		return m, m.fetchData()
	}

	return m, nil
}

func (m *connectionModel) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "esc", "ctrl+c":
		m.quitting = true
		return m, tea.Quit

	case "r":
		return m, m.fetchData()

	case "a":
		m.autoRefresh = !m.autoRefresh
		if m.autoRefresh {
			return m, m.autoRefreshTick()
		}
		return m, nil

	case "tab", "v":
		if m.mode == viewConnections {
			m.mode = viewPolicies
			// 首次进入需要策略组信息判断组/节点与中间/末端
			if m.groups == nil {
				m.loading = true
				return m, m.fetchData()
			}
			m.rebuildPolicies()
		} else {
			m.mode = viewConnections
		}
		return m, nil

	case "up", "k":
		m.moveCursor(-1)
		return m, nil

	case "down", "j":
		m.moveCursor(1)
		return m, nil

	case "d":
		return m.closeSelected()

	case "D":
		if len(m.conns) > 0 {
			return m, m.closeAllConnections()
		}
		return m, nil
	}

	return m, nil
}

func (m *connectionModel) moveCursor(delta int) {
	n := m.currentLen()
	if n == 0 {
		return
	}
	cur := m.currentCursor()
	cur += delta
	if cur < 0 {
		cur = n - 1
	}
	if cur >= n {
		cur = 0
	}
	m.setCursor(cur)
}

func (m *connectionModel) currentLen() int {
	if m.mode == viewPolicies {
		return len(m.policies)
	}
	return len(m.conns)
}

func (m *connectionModel) currentCursor() int {
	if m.mode == viewPolicies {
		return m.policyCursor
	}
	return m.connCursor
}

func (m *connectionModel) setCursor(cur int) {
	if m.mode == viewPolicies {
		m.policyCursor = cur
		return
	}
	m.connCursor = cur
}

func (m *connectionModel) closeSelected() (tea.Model, tea.Cmd) {
	if m.mode == viewPolicies {
		if m.policyCursor >= len(m.policies) {
			return m, nil
		}
		st := m.policies[m.policyCursor]
		if len(st.IDs) == 0 {
			return m, nil
		}
		return m, m.closePolicyConnections(st.IDs)
	}
	if m.connCursor < len(m.conns) {
		return m, m.closeConnection(m.conns[m.connCursor].ID)
	}
	return m, nil
}

// fetchData 拉取连接；处于策略聚合视图时同时刷新策略组信息
func (m *connectionModel) fetchData() tea.Cmd {
	wantGroups := m.mode == viewPolicies
	return func() tea.Msg {
		result, err := m.client.GetConnections()
		if err != nil {
			return fetchResultMsg{err: err}
		}
		msg := fetchResultMsg{conns: result.Connections}
		if wantGroups {
			groups, gerr := m.client.GetGroups()
			if gerr != nil {
				// 组信息只影响组/节点标注，保留连接列表继续展示
				msg.err = fmt.Errorf("读取策略组失败: %w", gerr)
			} else {
				msg.groups = groups
			}
		}
		return msg
	}
}

func (m *connectionModel) autoRefreshTick() tea.Cmd {
	return tea.Tick(autoRefreshInterval, func(t time.Time) tea.Msg {
		return refreshTickMsg{}
	})
}

func (m *connectionModel) closeConnection(id string) tea.Cmd {
	return func() tea.Msg {
		err := m.client.CloseConnection(id)
		return closeResultMsg{err: err, id: id}
	}
}

// closePolicyConnections 逐条关闭经过某策略的连接，按 ID 去重避免重复请求
func (m *connectionModel) closePolicyConnections(ids []string) tea.Cmd {
	c := m.client
	return func() tea.Msg {
		seen := make(map[string]struct{}, len(ids))
		failed := 0
		var firstErr error
		for _, id := range ids {
			if _, ok := seen[id]; ok {
				continue
			}
			seen[id] = struct{}{}
			if err := c.CloseConnection(id); err != nil {
				failed++
				if firstErr == nil {
					firstErr = err
				}
			}
		}
		if firstErr != nil {
			return closeResultMsg{err: fmt.Errorf("关闭 %d/%d 条连接失败: %w", failed, len(seen), firstErr)}
		}
		return closeResultMsg{}
	}
}

func (m *connectionModel) closeAllConnections() tea.Cmd {
	return func() tea.Msg {
		err := m.client.CloseAllConnections()
		return closeResultMsg{err: err}
	}
}

type fetchResultMsg struct {
	conns  []model.ConnectionItem
	groups []model.GroupInfo
	err    error
}

type closeResultMsg struct {
	err error
	id  string
}

// rebuildPolicies 依据当前连接与策略组信息重建聚合视图数据
func (m *connectionModel) rebuildPolicies() {
	if m.groups == nil {
		m.policies = nil
		return
	}
	m.policies = buildPolicyStats(m.conns, m.groups)
	if m.policyCursor >= len(m.policies) {
		m.policyCursor = 0
	}
}

// buildPolicyStats 聚合连接：每个 chains 元素都计入「经过」计数。
// 组再区分中间/末端（成员是否含其他组）与有活动的下级数量。
func buildPolicyStats(conns []model.ConnectionItem, groups map[string]model.GroupInfo) []policyStat {
	byName := make(map[string]*policyStat, len(conns))
	order := make([]string, 0, len(conns))

	for _, c := range conns {
		seen := make(map[string]struct{}, len(c.Chains))
		for _, name := range c.Chains {
			if name == "" {
				continue
			}
			if _, dup := seen[name]; dup {
				continue
			}
			seen[name] = struct{}{}

			st, ok := byName[name]
			if !ok {
				st = &policyStat{Name: name, Kind: "节点"}
				if g, isGroup := groups[name]; isGroup {
					st.IsGroup = true
					st.Kind = g.Type
				}
				byName[name] = st
				order = append(order, name)
			}
			st.Count++
			st.IDs = append(st.IDs, c.ID)
			if len(st.Sample) == 0 || len(c.Chains) < len(st.Sample) {
				st.Sample = c.Chains
			}
		}
	}

	stats := make([]policyStat, 0, len(order))
	for _, name := range order {
		st := byName[name]
		if st.IsGroup {
			for _, member := range groups[name].All {
				if _, isGroup := groups[member]; isGroup {
					st.Intermediate = true
					break
				}
			}
		}
		stats = append(stats, *st)
	}

	sort.Slice(stats, func(i, j int) bool {
		if stats[i].Count != stats[j].Count {
			return stats[i].Count > stats[j].Count
		}
		return stats[i].Name < stats[j].Name
	})
	return stats
}

func (m *connectionModel) View() string {
	if m.quitting {
		return ""
	}

	width := m.width
	if width == 0 {
		width = 80
	}
	height := m.height
	if height == 0 {
		height = 24
	}

	var b strings.Builder

	// 顶部留白
	b.WriteString("\n")

	autoStatus := "开"
	if !m.autoRefresh {
		autoStatus = "关"
	}

	if m.mode == viewPolicies {
		b.WriteString(fmt.Sprintf("按策略聚合 (%d 条连接 / %d 个策略)   自动刷新: %s (%ds)",
			len(m.conns), len(m.policies), autoStatus, autoRefreshInterval/time.Second))
	} else {
		b.WriteString(fmt.Sprintf("活跃连接 (%d)   自动刷新: %s (%ds)",
			len(m.conns), autoStatus, autoRefreshInterval/time.Second))
	}
	b.WriteString("\n")
	b.WriteString(dividerStyle.Render(strings.Repeat("─", width)))
	b.WriteString("\n")

	if m.loading {
		b.WriteString(loadingStyle.Render("  加载中..."))
		b.WriteString("\n")
		return b.String()
	}

	if m.err != nil {
		b.WriteString(errorStyle.Render(fmt.Sprintf("  错误: %v", m.err)))
		b.WriteString("\n")
	}

	if m.mode == viewPolicies {
		m.writePolicies(&b, width, height)
		b.WriteString(hintStyle.Render("r 刷新  •  a 自动刷新  •  d 断开该策略全部连接  •  D 关闭全部  •  Tab 返回连接列表  •  ↑↓/jk 导航  •  Esc 退出"))
		return b.String()
	}

	m.writeConnections(&b, width, height)
	b.WriteString(hintStyle.Render("r 刷新  •  a 自动刷新  •  d 关闭选中  •  D 关闭全部  •  Tab 按策略聚合  •  ↑↓/jk 导航  •  Esc 退出"))
	return b.String()
}

// 连接列表固定行：空行 + 标题 + 分隔线 + 表头 + 底部提示
const connFixedLines = 5

func (m *connectionModel) writeConnections(b *strings.Builder, width, height int) {
	if len(m.conns) == 0 {
		b.WriteString(hintStyle.Render("  无活跃连接"))
		b.WriteString("\n")
		return
	}

	srcW, dstW, chainW := connColumnWidths(m.conns, width)

	b.WriteString(hintStyle.Render("  " + padDisplay("ID", 6) + "  " +
		padDisplay("来源", srcW) + "  " + padDisplay("目标", dstW) + "  " + truncateDisplay("链路", chainW)))
	b.WriteString("\n")

	rows := make([]string, len(m.conns))
	for i, c := range m.conns {
		id := c.ID
		if len(id) > 6 {
			id = id[:6]
		}
		src := truncateDisplay(fmt.Sprintf("%s:%s", c.Metadata.SourceIP, c.Metadata.SourcePort), srcW)
		dst := truncateDisplay(connDestination(c), dstW)
		chains := truncateDisplay(joinConnChains(c.Chains), chainW)

		rows[i] = padDisplay(id, 6) + "  " + padDisplay(src, srcW) + "  " +
			padDisplay(dst, dstW) + "  " + chains
	}

	renderPagedRows(b, rows, m.connCursor, height, connFixedLines)
}

// connColumnWidths 按终端宽度分配列宽。
// 链路列优先保证完整：宽度不够时先压缩「目标」再压缩「来源」，最后才截断链路。
func connColumnWidths(conns []model.ConnectionItem, width int) (srcW, dstW, chainW int) {
	srcW, dstW, chainW = 6, 6, 8
	for _, c := range conns {
		if w := runewidth.StringWidth(fmt.Sprintf("%s:%s", c.Metadata.SourceIP, c.Metadata.SourcePort)); w > srcW {
			srcW = w
		}
		if w := runewidth.StringWidth(connDestination(c)); w > dstW {
			dstW = w
		}
		if w := runewidth.StringWidth(joinConnChains(c.Chains)); w > chainW {
			chainW = w
		}
	}

	// 行宽 = 2(选中前缀) + 6(ID) + 2 + srcW + 2 + dstW + 2 + chainW
	budget := width - 14
	if budget < 30 {
		budget = 30
	}
	for srcW+dstW+chainW > budget {
		switch {
		case dstW > 8:
			dstW--
		case srcW > 8:
			srcW--
		case chainW > 6:
			chainW--
		default:
			return srcW, dstW, chainW
		}
	}
	return srcW, dstW, chainW
}

func connDestination(c model.ConnectionItem) string {
	if c.Metadata.Host != "" {
		return c.Metadata.Host
	}
	return fmt.Sprintf("%s:%s", c.Metadata.DestinationIP, c.Metadata.DestinationPort)
}

func (m *connectionModel) writePolicies(b *strings.Builder, width, height int) {
	if m.groups == nil && len(m.conns) > 0 {
		b.WriteString(hintStyle.Render("  正在读取策略组信息..."))
		b.WriteString("\n")
		return
	}
	if len(m.policies) == 0 {
		b.WriteString(hintStyle.Render("  无经过策略的活跃连接"))
		b.WriteString("\n")
		return
	}

	nameW, chainW := policyColumnWidths(m.policies, width)
	const markW, countW = 8, 6

	b.WriteString(hintStyle.Render(
		"  " + padDisplay("类型", markW) + "  " + padDisplay("策略/节点", nameW) + "  " +
			padLeftDisplay("连接数", countW) + "  " + truncateDisplay("策略链路", chainW)))
	b.WriteString("\n")

	rows := make([]string, len(m.policies))
	for i, st := range m.policies {
		mark := "节点"
		if st.IsGroup {
			if st.Intermediate {
				mark = "组·中间"
			} else {
				mark = "组·末端"
			}
		}
		rows[i] = padDisplay(mark, markW) + "  " + padDisplay(truncateDisplay(st.Name, nameW), nameW) + "  " +
			padLeftDisplay(fmt.Sprintf("%d", st.Count), countW) + "  " +
			truncateDisplay(joinConnChains(st.Sample), chainW)
	}

	renderPagedRows(b, rows, m.policyCursor, height, connFixedLines)
}

func policyColumnWidths(policies []policyStat, width int) (nameW, chainW int) {
	nameW, chainW = 10, 12
	for _, st := range policies {
		if w := runewidth.StringWidth(st.Name); w > nameW {
			nameW = w
		}
		if w := runewidth.StringWidth(joinConnChains(st.Sample)); w > chainW {
			chainW = w
		}
	}

	// 行宽 = 2 + 8(类型) + 2 + nameW + 2 + 6(连接数) + 2 + chainW
	budget := width - 22
	if budget < 24 {
		budget = 24
	}
	for nameW+chainW > budget {
		switch {
		case nameW > 10:
			nameW--
		case chainW > 8:
			chainW--
		default:
			return nameW, chainW
		}
	}
	return nameW, chainW
}

// renderPagedRows 渲染行列表：负责选中高亮与上下翻页提示。
// fixed 为调用方已输出的固定行数（含表头与底部提示），height 为终端高度。
func renderPagedRows(b *strings.Builder, rows []string, cursor, height, fixed int) {
	total := len(rows)
	if total == 0 {
		return
	}

	visible := height - fixed
	if visible < 3 {
		visible = 3
	}
	if visible > total {
		visible = total
	}

	start := 0
	for {
		start = cursor - visible/2
		if start > total-visible {
			start = total - visible
		}
		if start < 0 {
			start = 0
		}
		extra := 0
		if start > 0 {
			extra++
		}
		if start+visible < total {
			extra++
		}
		if visible <= 3 || fixed+extra+visible <= height {
			break
		}
		visible--
	}
	end := start + visible
	if end > total {
		end = total
	}

	if start > 0 {
		b.WriteString(hintStyle.Render(fmt.Sprintf("  ... 上面还有 %d 项", start)))
		b.WriteString("\n")
	}

	for i := start; i < end; i++ {
		if i == cursor {
			b.WriteString(activeStyle.Render("▸ " + rows[i]))
		} else {
			b.WriteString(inactiveStyle.Render("  " + rows[i]))
		}
		b.WriteString("\n")
	}

	if end < total {
		b.WriteString(hintStyle.Render(fmt.Sprintf("  ... 下面还有 %d 项", total-end)))
		b.WriteString("\n")
	}
}

func joinConnChains(chains []string) string {
	if len(chains) == 0 {
		return "-"
	}
	return strings.Join(chains, "→")
}

// padDisplay 右侧补空格到指定显示宽度（CJK 按 2 列计）
func padDisplay(s string, w int) string {
	d := runewidth.StringWidth(s)
	if d >= w {
		return s
	}
	return s + strings.Repeat(" ", w-d)
}

// padLeftDisplay 左侧补空格到指定显示宽度，用于数字右对齐
func padLeftDisplay(s string, w int) string {
	d := runewidth.StringWidth(s)
	if d >= w {
		return s
	}
	return strings.Repeat(" ", w-d) + s
}

// truncateDisplay 按显示宽度截断，超长时末尾加 "…"
func truncateDisplay(s string, w int) string {
	if w <= 0 {
		return ""
	}
	if runewidth.StringWidth(s) <= w {
		return s
	}
	if w == 1 {
		return "…"
	}
	target := w - 1
	cur := 0
	var b strings.Builder
	for _, r := range s {
		rw := runewidth.RuneWidth(r)
		if cur+rw > target {
			break
		}
		b.WriteRune(r)
		cur += rw
	}
	return b.String() + "…"
}
