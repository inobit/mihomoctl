package interact

import (
	"fmt"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/inobit/mihomoctl/internal/client"
	"github.com/inobit/mihomoctl/internal/model"
)

func Run(c *client.Client) error {
	groups, err := c.GetGroups()
	if err != nil {
		return fmt.Errorf("获取策略组失败: %w", err)
	}
	if len(groups) == 0 {
		return fmt.Errorf("未找到可切换的策略组")
	}

	return runGroupSelect(c, groups)
}

func runGroupSelect(c *client.Client, allGroups []model.GroupInfo) error {
	filtered := filterGroups(allGroups, false)

	if len(filtered) == 0 && len(allGroups) > 0 {
		// 过滤后为空，直接显示全部
		filtered = allGroups
	}

	if len(filtered) == 0 {
		return fmt.Errorf("未找到策略组")
	}

	filteredTexts, filteredExtras := formatGroups(filtered)
	allTexts, allExtras := formatGroups(allGroups)

	filteredLabel := fmt.Sprintf("选择策略组 (已过滤 %d 组)", len(filtered))
	allLabel := fmt.Sprintf("选择策略组 (全部 %d 组)", len(allGroups))

	if len(filtered) == len(allGroups) {
		filteredLabel = fmt.Sprintf("选择策略组 (共 %d 组)", len(allGroups))
		allLabel = filteredLabel
	}

	hints := "↑↓/jk 导航  •  数字跳转  •  Enter 确认  •  Tab 切换全部/过滤  •  Esc 退出"

	m := NewFilteredSelectModel(filteredLabel, allLabel, filteredTexts, filteredExtras, allTexts, allExtras, hints)

	p := tea.NewProgram(m)
	finalModel, err := p.Run()
	if err != nil {
		return err
	}

	sm := finalModel.(*SelectModel)

	if sm.aborted {
		os.Exit(0)
	}

	if sm.selected < 0 {
		return fmt.Errorf("未选择")
	}

	var selectedGroup model.GroupInfo
	if sm.ShowAll() {
		selectedGroup = allGroups[sm.selected]
	} else {
		selectedGroup = filtered[sm.selected]
	}

	return runNodeSelect(c, selectedGroup, allGroups)
}

func runNodeSelect(c *client.Client, group model.GroupInfo, allGroups []model.GroupInfo) error {
	info, err := c.GetProxy(group.Name)
	if err != nil {
		return fmt.Errorf("获取节点列表失败: %w", err)
	}

	if len(info.All) == 0 {
		return fmt.Errorf("策略组 %s 内无可用节点", group.Name)
	}

	// 构建延迟映射
	delayMap := make(map[string]int)
	if len(info.History) > 0 {
		latest := info.History[len(info.History)-1]
		delayMap[info.Now] = latest.Delay
	}

	nodeTotal := len(info.All)
	texts := make([]string, nodeTotal)
	extras := make([]string, nodeTotal)
	for i, node := range info.All {
		isCurrent := node == info.Now
		marker := ""
		if isCurrent {
			marker = "✓ "
		} else {
			marker = "  "
		}

		if delay, ok := delayMap[node]; ok && delay > 0 {
			texts[i] = fmt.Sprintf("%s%s", marker, node)
			extras[i] = fmt.Sprintf("%dms", delay)
		} else {
			texts[i] = fmt.Sprintf("%s%s", marker, node)
		}
	}

	label := fmt.Sprintf("选择节点 [%s] (共 %d 个)", group.Name, nodeTotal)
	hints := "↑↓/jk 导航  •  数字跳转  •  t 测速  •  Enter 确认  •  Esc 返回"

	m := NewSelectModel(label, texts, extras, info.All, hints)

	groupName := group.Name
	m.speedTest = func(_ string) tea.Cmd {
		return func() tea.Msg {
			delays, err := c.GetGroupDelay(groupName, "https://www.gstatic.com/generate_204", 5000)
			return speedTestResultMsg{delays: delays, err: err}
		}
	}

	p := tea.NewProgram(m)
	finalModel, err := p.Run()
	if err != nil {
		return err
	}

	sm := finalModel.(*SelectModel)

	if sm.aborted {
		return runGroupSelect(c, allGroups)
	}

	if sm.selected < 0 {
		return fmt.Errorf("未选择")
	}

	node := info.All[sm.selected]
	if node == info.Now {
		fmt.Printf("✓ 已选中 %s [未变更]\n", node)
		return nil
	}

	if err := c.SwitchProxy(group.Name, node); err != nil {
		return fmt.Errorf("切换节点失败: %w", err)
	}

	fmt.Printf("✓ 已切换 %s → %s\n", group.Name, node)
	return nil
}

func filterGroups(groups []model.GroupInfo, showAll bool) []model.GroupInfo {
	if showAll {
		return groups
	}
	var result []model.GroupInfo
	for _, g := range groups {
		if isSelectableGroup(g.Type) {
			result = append(result, g)
		}
	}
	return result
}

func formatGroups(groups []model.GroupInfo) (texts []string, extras []string) {
	for _, g := range groups {
		texts = append(texts, fmt.Sprintf("%s  →  %s", g.Name, g.Now))
		extras = append(extras, fmt.Sprintf("[%s]", g.Type))
	}
	return
}

// isSelectableGroup 判断是否需要手动选择：只有 Selector 类型的策略组才默认显示
// URLTest、Fallback、LoadBalance、Relay 等是自动选择的，不需要手动干预
func isSelectableGroup(typ string) bool {
	return strings.EqualFold(typ, "Selector")
}

func ModeSelect(c *client.Client) error {
	texts := []string{
		"rule   — 规则模式",
		"global — 全局模式",
		"direct — 直连模式",
	}
	names := []string{"rule", "global", "direct"}

	label := "选择代理模式"
	hints := "↑↓/jk 导航  •  Enter 确认  •  Esc 退出"

	m := NewSelectModel(label, texts, nil, names, hints)

	p := tea.NewProgram(m)
	finalModel, err := p.Run()
	if err != nil {
		return err
	}

	sm := finalModel.(*SelectModel)
	if sm.aborted || sm.selected < 0 {
		return nil
	}

	mode := names[sm.selected]
	if err := c.PatchConfig(map[string]any{"mode": mode}); err != nil {
		return fmt.Errorf("切换模式失败: %w", err)
	}

	fmt.Printf("✓ 已切换到 %s 模式\n", mode)
	return nil
}
