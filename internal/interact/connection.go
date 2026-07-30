package interact

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/inobit/mihomoctl/internal/client"
	"github.com/inobit/mihomoctl/internal/model"
)

const autoRefreshInterval = 5 * time.Second

type refreshTickMsg struct{}

type connectionModel struct {
	client   *client.Client
	conns    []model.ConnectionItem
	err      error
	loading  bool
	cursor   int
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
	return tea.Batch(m.fetchConnections(), m.autoRefreshTick())
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
			return m, tea.Batch(m.fetchConnections(), m.autoRefreshTick())
		}
		return m, nil

	case fetchResultMsg:
		m.loading = false
		m.err = msg.err
		if msg.err == nil {
			m.conns = msg.conns
			if m.cursor >= len(m.conns) && len(m.conns) > 0 {
				m.cursor = 0
			}
		}
		return m, nil

	case closeResultMsg:
		if msg.err != nil {
			m.err = msg.err
			return m, nil
		}
		return m, m.fetchConnections()
	}

	return m, nil
}

func (m *connectionModel) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "esc", "ctrl+c":
		m.quitting = true
		return m, tea.Quit

	case "r":
		return m, m.fetchConnections()

	case "a":
		m.autoRefresh = !m.autoRefresh
		if m.autoRefresh {
			return m, m.autoRefreshTick()
		}
		return m, nil

	case "up", "k":
		if len(m.conns) > 0 {
			if m.cursor > 0 {
				m.cursor--
			} else {
				m.cursor = len(m.conns) - 1
			}
		}
		return m, nil

	case "down", "j":
		if len(m.conns) > 0 {
			if m.cursor < len(m.conns)-1 {
				m.cursor++
			} else {
				m.cursor = 0
			}
		}
		return m, nil

	case "d":
		if len(m.conns) > 0 && m.cursor < len(m.conns) {
			conn := m.conns[m.cursor]
			return m, m.closeConnection(conn.ID)
		}
		return m, nil

	case "D":
		if len(m.conns) > 0 {
			return m, m.closeAllConnections()
		}
		return m, nil
	}

	return m, nil
}

func (m *connectionModel) fetchConnections() tea.Cmd {
	return func() tea.Msg {
		result, err := m.client.GetConnections()
		if err != nil {
			return fetchResultMsg{err: err}
		}
		return fetchResultMsg{conns: result.Connections}
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

func (m *connectionModel) closeAllConnections() tea.Cmd {
	return func() tea.Msg {
		err := m.client.CloseAllConnections()
		return closeResultMsg{err: err}
	}
}

type fetchResultMsg struct {
	conns []model.ConnectionItem
	err   error
}

type closeResultMsg struct {
	err error
	id  string
}

func (m *connectionModel) View() string {
	if m.quitting {
		return ""
	}

	var b strings.Builder

	// 顶部留白
	b.WriteString("\n")

	// 标题栏
	title := fmt.Sprintf("活跃连接 (%d)", len(m.conns))
	autoStatus := "开"
	if !m.autoRefresh {
		autoStatus = "关"
	}
	title += fmt.Sprintf("   自动刷新: %s (%ds)", autoStatus, autoRefreshInterval/time.Second)

	b.WriteString(title)
	b.WriteString("\n")

	width := m.width
	if width == 0 {
		width = 80
	}
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

	if len(m.conns) == 0 {
		b.WriteString(hintStyle.Render("  无活跃连接"))
		b.WriteString("\n")
	} else {
		// 表头
		header := fmt.Sprintf("  %-6s  %-22s  %-26s  %s", "ID", "来源", "目标", "链路")
		b.WriteString(hintStyle.Render(header))
		b.WriteString("\n")

		// 固定行: 空行 + label + divider + header + hints = 5
		// 可选行: 上面还有 + 下面还有 = 最多 2
		// 先按最坏情况(上下都有提示)算 visible，再根据实际 start/end 修正
		maxVisible := m.height - 7
		if maxVisible < 5 {
			maxVisible = 5
		}
		if maxVisible > len(m.conns) {
			maxVisible = len(m.conns)
		}

		visible := maxVisible
		start := m.cursor - visible/2
		if start < 0 {
			start = 0
		}
		if start+visible > len(m.conns) {
			start = len(m.conns) - visible
		}
		if start < 0 {
			start = 0
		}
		end := start + visible

		hasAbove := start > 0
		hasBelow := end < len(m.conns)

		// 修正: 如果实际可选行少于 2，visible 可以增大
		actualFixed := 5
		actualOptional := 0
		if hasAbove {
			actualOptional++
		}
		if hasBelow {
			actualOptional++
		}
		adjustedVisible := m.height - actualFixed - actualOptional
		if adjustedVisible < 5 {
			adjustedVisible = 5
		}
		if adjustedVisible > len(m.conns) {
			adjustedVisible = len(m.conns)
		}

		// 只有增大时才需要重新计算（缩小不会改变 hasAbove/hasBelow）
		if adjustedVisible > visible {
			visible = adjustedVisible
			start = m.cursor - visible/2
			if start < 0 {
				start = 0
			}
			if start+visible > len(m.conns) {
				start = len(m.conns) - visible
			}
			if start < 0 {
				start = 0
			}
			end = start + visible
		}

		if start > 0 {
			b.WriteString(hintStyle.Render(fmt.Sprintf("  ... 上面还有 %d 项", start)))
			b.WriteString("\n")
		}

		for i := start; i < end; i++ {
			c := m.conns[i]
			id := c.ID
			if len(id) > 6 {
				id = id[:6]
			}
			src := fmt.Sprintf("%s:%s", c.Metadata.SourceIP, c.Metadata.SourcePort)
			if len(src) > 22 {
				src = src[:19] + "..."
			}
			dst := c.Metadata.Host
			if dst == "" {
				dst = fmt.Sprintf("%s:%s", c.Metadata.DestinationIP, c.Metadata.DestinationPort)
			}
			if len(dst) > 26 {
				dst = dst[:23] + "..."
			}
			chains := joinConnChains(c.Chains)
			if len(chains) > 24 {
				chains = chains[:21] + "..."
			}

			line := fmt.Sprintf("%-6s  %-22s  %-26s  %s", id, src, dst, chains)

			if i == m.cursor {
				b.WriteString(activeStyle.Render("▸ " + line))
			} else {
				b.WriteString(inactiveStyle.Render("  " + line))
			}
			b.WriteString("\n")
		}

		if end < len(m.conns) {
			b.WriteString(hintStyle.Render(fmt.Sprintf("  ... 下面还有 %d 项", len(m.conns)-end)))
			b.WriteString("\n")
		}
	}

	b.WriteString(hintStyle.Render("r 刷新  •  a 自动刷新  •  d 关闭选中  •  D 关闭全部  •  ↑↓/jk 导航  •  Esc 退出"))

	return b.String()
}

func joinConnChains(chains []string) string {
	if len(chains) == 0 {
		return "-"
	}
	s := ""
	for i, c := range chains {
		if i > 0 {
			s += "→"
		}
		s += c
	}
	return s
}
