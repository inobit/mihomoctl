package interact

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/mattn/go-runewidth"
)

var (
	activeStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("6")).Bold(true)
	inactiveStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("7"))
	labelStyle    = lipgloss.NewStyle()
	hintStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("7")).Faint(true)
	errorStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("1"))
	currentStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("3"))
	dividerStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("8")).Faint(true)
	loadingStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("4"))
	// 测速分级着色：<100ms 绿 / 100-300ms 黄 / >=300ms 红 / timeout 灰
	delayFastStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("2"))
	delayMediumStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("3"))
	delaySlowStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("1"))
	timeoutStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
)

const numberTimeout = 800 * time.Millisecond

type speedTestResultMsg struct {
	delays map[string]int
	err    error
}

type selectItem struct {
	text  string
	extra string
	name  string // 原始节点名，用于测速结果匹配
}

type SpeedTestFn func(groupName string) tea.Cmd

type SelectModel struct {
	label         string
	filteredLabel string
	allLabel      string
	items         []selectItem

	allItems      []selectItem
	filteredItems []selectItem
	showAll       bool
	hasFilter     bool

	cursor   int
	width    int
	height   int
	selected int
	aborted  bool

	numBuf        string
	numSeq        int
	numOutOfRange bool

	keyHints  string
	loading   bool
	speedTest SpeedTestFn
	speedErr  string
}

func NewSelectModel(label string, texts []string, extras []string, names []string, hints string) *SelectModel {
	items := buildItems(texts, extras, names)
	return &SelectModel{
		label:    label,
		items:    items,
		cursor:   0,
		selected: -1,
		keyHints: hints,
		showAll:  true,
	}
}

func NewFilteredSelectModel(filteredLabel, allLabel string, texts []string, extras []string, allTexts []string, allExtras []string, hints string) *SelectModel {
	filtered := buildItems(texts, extras, nil)
	all := buildItems(allTexts, allExtras, nil)
	return &SelectModel{
		label:         filteredLabel,
		filteredLabel: filteredLabel,
		allLabel:      allLabel,
		items:         filtered,
		filteredItems: filtered,
		allItems:      all,
		cursor:        0,
		selected:      -1,
		keyHints:      hints,
		showAll:       false,
		hasFilter:     true,
	}
}

func buildItems(texts, extras, names []string) []selectItem {
	items := make([]selectItem, len(texts))
	for i, t := range texts {
		e := ""
		if i < len(extras) {
			e = extras[i]
		}
		n := ""
		if i < len(names) {
			n = names[i]
		}
		items[i] = selectItem{text: t, extra: e, name: n}
	}
	return items
}

// delayExtraStyle 根据延迟分级着色：timeout 灰 / <100ms 绿 / 100-300ms 黄 / >=300ms 红
// 0ms 视为 timeout（Mihomo 对失败节点有时返回 0），避免误标绿色
func delayExtraStyle(extra string) lipgloss.Style {
	if extra == "timeout" {
		return timeoutStyle
	}
	var ms int
	if _, err := fmt.Sscanf(extra, "%dms", &ms); err == nil {
		if ms == 0 {
			return timeoutStyle
		}
		switch {
		case ms < 100:
			return delayFastStyle
		case ms < 300:
			return delayMediumStyle
		default:
			return delaySlowStyle
		}
	}
	return currentStyle
}

func (m *SelectModel) Init() tea.Cmd {
	return nil
}

func (m *SelectModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.KeyMsg:
		return m.handleKey(msg)

	case numberTickMsg:
		if msg.seq >= m.numSeq {
			m.numBuf = ""
			m.numOutOfRange = false
		}
		return m, nil

	case speedTestResultMsg:
		m.loading = false
		if msg.err != nil {
			m.speedErr = msg.err.Error()
			return m, nil
		}
		m.speedErr = ""
		for i := range m.items {
			if m.items[i].name != "" {
				if delay, ok := msg.delays[m.items[i].name]; ok {
					m.items[i].extra = fmt.Sprintf("%dms", delay)
				} else {
					m.items[i].extra = "timeout"
				}
			}
		}
		return m, nil
	}

	return m, nil
}

func (m *SelectModel) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c":
		m.aborted = true
		return m, tea.Quit

	case "esc":
		m.aborted = true
		return m, tea.Quit

	case "t":
		if m.speedTest != nil && !m.loading {
			m.loading = true
			m.speedErr = ""
			return m, m.speedTest(m.label)
		}
		return m, nil

	case "tab":
		if m.hasFilter {
			m.showAll = !m.showAll
			if m.showAll {
				m.items = m.allItems
				m.label = m.allLabel
			} else {
				m.items = m.filteredItems
				m.label = m.filteredLabel
			}
			if m.cursor >= len(m.items) {
				m.cursor = 0
			}
			m.numBuf = ""
			return m, nil
		}
		return m, nil

	case "up", "k":
		if len(m.items) == 0 {
			return m, nil
		}
		if m.cursor > 0 {
			m.cursor--
		} else {
			m.cursor = len(m.items) - 1
		}
		m.numBuf = ""
		return m, nil

	case "down", "j":
		if len(m.items) == 0 {
			return m, nil
		}
		if m.cursor < len(m.items)-1 {
			m.cursor++
		} else {
			m.cursor = 0
		}
		m.numBuf = ""
		return m, nil

	case "enter":
		if m.loading {
			return m, nil
		}
		if m.numBuf != "" {
			if idx := m.parseNumBuf(); idx >= 0 {
				m.cursor = idx
				m.selected = idx
				m.numBuf = ""
				return m, tea.Quit
			}
			m.numBuf = ""
			m.numOutOfRange = false
			return m, nil
		}
		if len(m.items) > 0 {
			m.selected = m.cursor
		}
		return m, tea.Quit

	default:
		r := msg.String()
		if len(r) == 1 && r[0] >= '0' && r[0] <= '9' {
			return m.handleDigit(r)
		}
	}

	return m, nil
}

func (m *SelectModel) handleDigit(d string) (tea.Model, tea.Cmd) {
	m.numBuf += d
	m.numSeq++
	m.numOutOfRange = false

	if idx := m.parseNumBuf(); idx >= 0 {
		m.cursor = idx
	} else {
		m.numOutOfRange = true
	}

	seq := m.numSeq
	return m, tea.Tick(numberTimeout, func(t time.Time) tea.Msg {
		return numberTickMsg{seq: seq}
	})
}

func (m *SelectModel) parseNumBuf() int {
	if m.numBuf == "" {
		return -1
	}
	var n int
	fmt.Sscanf(m.numBuf, "%d", &n)
	if n < 1 || n > len(m.items) {
		return -1
	}
	return n - 1
}

type numberTickMsg struct {
	seq int
}

func (m *SelectModel) View() string {
	if m.aborted || m.selected >= 0 || len(m.items) == 0 {
		return ""
	}

	var b strings.Builder

	b.WriteString(labelStyle.Render(m.label))
	b.WriteString("\n")

	width := m.width
	if width == 0 {
		width = 80
	}
	// 使用 runewidth 保证含 emoji 的宽度计算正确，divider 按可视宽度铺满
	dividerWidth := width
	if runewidth.StringWidth(strings.Repeat("─", width)) != width {
		dividerWidth = width
	}
	b.WriteString(dividerStyle.Render(strings.Repeat("─", dividerWidth)))
	b.WriteString("\n")

	// loading 指示器
	if m.loading {
		b.WriteString(loadingStyle.Render("  ⏳ 测速中..."))
		b.WriteString("\n")
		b.WriteString(dividerStyle.Render(strings.Repeat("─", dividerWidth)))
		b.WriteString("\n")
	}

	total := len(m.items)
	visible := m.height - 4
	if m.loading {
		visible -= 2
	}
	if visible < 5 {
		visible = 5
	}
	if visible > total {
		visible = total
	}

	start := m.cursor - visible/2
	if start < 0 {
		start = 0
	}
	if start+visible > total {
		start = total - visible
	}
	if start < 0 {
		start = 0
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
		prefix := fmt.Sprintf("[%d/%d]", i+1, total)
		line := fmt.Sprintf("%s %s", prefix, m.items[i].text)

		if m.items[i].extra != "" {
			line += " " + delayExtraStyle(m.items[i].extra).Render(m.items[i].extra)
		}

		if i == m.cursor && !m.loading {
			b.WriteString(activeStyle.Render("▸ " + line))
		} else {
			b.WriteString(inactiveStyle.Render("  " + line))
		}
		b.WriteString("\n")
	}

	if end < total {
		b.WriteString(hintStyle.Render(fmt.Sprintf("  ... 下面还有 %d 项", total-end)))
		b.WriteString("\n")
	}

	b.WriteString("\n")
	b.WriteString(hintStyle.Render(m.keyHints))
	b.WriteString("\n")

	if m.speedErr != "" {
		b.WriteString(errorStyle.Render(fmt.Sprintf("测速失败: %s", m.speedErr)))
		b.WriteString("\n")
	}

	if m.numBuf != "" {
		if m.numOutOfRange {
			b.WriteString(errorStyle.Render(fmt.Sprintf("超出范围: %s (1-%d)", m.numBuf, total)))
		} else {
			b.WriteString(hintStyle.Render(fmt.Sprintf("跳转: %s", m.numBuf)))
		}
		b.WriteString("\n")
	}

	return b.String()
}

func (m *SelectModel) ShowAll() bool {
	return m.showAll
}
