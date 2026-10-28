package interact

import (
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"golang.org/x/term"
)

// 通用并发任务进度界面：每个任务一行独立 spinner（docker pull 风格），
// 完成即变 ✓/✗，全部结束后退出并保留结果清单。
// 目前 provider 拉取在用，后续任何「多条慢操作并行」的场景都可以复用。

var (
	taskOKStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("2"))
	taskFailStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("1"))
	taskPendStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("8")).Faint(true)
)

var taskSpinnerFrames = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

const defaultTaskConcurrency = 4

// Task 一个可并行的任务；Run 由调用方提供，进度界面不关心任务内容
type Task struct {
	Label string
	Run   func() error
}

// ProgressOptions 进度界面选项
type ProgressOptions struct {
	// Concurrency 并发上限，<=0 时用默认值 4
	Concurrency int
	// Title 可选标题，显示在进度列表上方
	Title string
	// Hint 可选提示，全部完成后显示在结果下方
	Hint string
}

// TaskResult 单个任务的执行结果
type TaskResult struct {
	Label   string
	Err     error
	Elapsed time.Duration
}

// ProgressResult 全部任务的执行结果，顺序与入参一致
type ProgressResult struct {
	Results []TaskResult
}

// Failed 失败任务数
func (r ProgressResult) Failed() int {
	n := 0
	for _, res := range r.Results {
		if res.Err != nil {
			n++
		}
	}
	return n
}

// Succeeded 成功任务数
func (r ProgressResult) Succeeded() int {
	return len(r.Results) - r.Failed()
}

type taskState int

const (
	taskPending taskState = iota
	taskRunning
	taskDone
	taskFailed
)

type taskOutcome struct {
	index   int
	err     error
	elapsed time.Duration
}

type taskStartMsg struct {
	index int
	at    time.Time
}

type taskResultMsg taskOutcome

type progressTickMsg time.Time

type progressItem struct {
	label     string
	state     taskState
	err       error
	startedAt time.Time
	elapsed   time.Duration
}

// ProgressModel 并发任务进度界面
type ProgressModel struct {
	tasks       []Task
	items       []progressItem
	title       string
	hint        string
	concurrency int
	ch          chan tea.Msg
	frame       int
	done        int
	failed      int
	finished    bool
}

func NewProgressModel(tasks []Task, opts ProgressOptions) *ProgressModel {
	items := make([]progressItem, len(tasks))
	for i, t := range tasks {
		items[i] = progressItem{label: t.Label}
	}
	return &ProgressModel{
		tasks:       tasks,
		items:       items,
		title:       opts.Title,
		hint:        opts.Hint,
		concurrency: opts.Concurrency,
		ch:          make(chan tea.Msg, len(tasks)*2),
	}
}

func (m *ProgressModel) Init() tea.Cmd {
	go func() {
		runTasks(m.tasks, m.concurrency,
			func(i int) { m.ch <- taskStartMsg{index: i, at: time.Now()} },
			func(o taskOutcome) { m.ch <- taskResultMsg(o) },
		)
		close(m.ch)
	}()
	return tea.Batch(m.waitForEvent(), progressTick())
}

func (m *ProgressModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case taskStartMsg:
		m.items[msg.index].state = taskRunning
		m.items[msg.index].startedAt = msg.at
		return m, m.waitForEvent()

	case taskResultMsg:
		it := &m.items[msg.index]
		it.elapsed = msg.elapsed
		if msg.err != nil {
			it.state = taskFailed
			it.err = msg.err
			m.failed++
		} else {
			it.state = taskDone
		}
		m.done++
		if m.done == len(m.items) {
			m.finished = true
			return m, tea.Quit
		}
		return m, m.waitForEvent()

	case progressTickMsg:
		if m.finished {
			return m, nil
		}
		m.frame++
		return m, progressTick()
	}

	return m, nil
}

func (m *ProgressModel) View() string {
	var b strings.Builder

	if m.title != "" {
		fmt.Fprintf(&b, "%s\n", labelStyle.Render(m.title))
	}

	for i := range m.items {
		it := &m.items[i]
		switch it.state {
		case taskPending:
			fmt.Fprintf(&b, "  %s %s\n", taskPendStyle.Render("·"), it.label)
		case taskRunning:
			spin := taskSpinnerFrames[m.frame%len(taskSpinnerFrames)]
			fmt.Fprintf(&b, "  %s %s %s\n", loadingStyle.Render(spin), it.label,
				hintStyle.Render(fmt.Sprintf("%.0fs", time.Since(it.startedAt).Seconds())))
		case taskDone:
			fmt.Fprintf(&b, "  %s %s %s\n", taskOKStyle.Render("✓"), it.label,
				hintStyle.Render(it.elapsed.Round(time.Millisecond).String()))
		case taskFailed:
			fmt.Fprintf(&b, "  %s %s\n     %s\n", taskFailStyle.Render("✗"), it.label,
				taskFailStyle.Render(it.err.Error()))
		}
	}

	if m.finished {
		succeeded := m.done - m.failed
		fmt.Fprintf(&b, "\n%s\n", hintStyle.Render(fmt.Sprintf("共 %d 个：成功 %d 个，失败 %d 个", len(m.items), succeeded, m.failed)))
		if m.hint != "" && succeeded > 0 {
			fmt.Fprintf(&b, "%s\n", hintStyle.Render(m.hint))
		}
	}

	return b.String()
}

func (m *ProgressModel) waitForEvent() tea.Cmd {
	return func() tea.Msg {
		msg, ok := <-m.ch
		if !ok {
			return nil
		}
		return msg
	}
}

func progressTick() tea.Cmd {
	return tea.Tick(100*time.Millisecond, func(t time.Time) tea.Msg {
		return progressTickMsg(t)
	})
}

// RunTasks 并发执行全部任务：终端下显示每项独立 spinner，非 TTY 降级为逐行输出
func RunTasks(tasks []Task, opts ProgressOptions) ProgressResult {
	if len(tasks) == 0 {
		return ProgressResult{}
	}

	var outcomes []taskOutcome
	if !term.IsTerminal(int(os.Stdout.Fd())) {
		outcomes = runTasksPlain(tasks, opts)
	} else {
		m := NewProgressModel(tasks, opts)
		final, err := tea.NewProgram(m).Run()
		if err != nil {
			fmt.Fprintf(os.Stderr, "进度界面异常: %v\n", err)
			return ProgressResult{Results: labelsOf(tasks, nil)}
		}
		outcomes = final.(*ProgressModel).outcomes()
	}

	results := make([]TaskResult, len(tasks))
	for i, t := range tasks {
		results[i] = TaskResult{Label: t.Label}
	}
	for _, o := range outcomes {
		results[o.index].Err = o.err
		results[o.index].Elapsed = o.elapsed
	}
	return ProgressResult{Results: results}
}

func (m *ProgressModel) outcomes() []taskOutcome {
	out := make([]taskOutcome, 0, len(m.items))
	for i, it := range m.items {
		out = append(out, taskOutcome{index: i, err: it.err, elapsed: it.elapsed})
	}
	return out
}

// labelsOf 进度界面无法启动时，构造一份「全部失败」的结果占位
func labelsOf(tasks []Task, _ []taskOutcome) []TaskResult {
	results := make([]TaskResult, len(tasks))
	for i, t := range tasks {
		results[i] = TaskResult{Label: t.Label, Err: fmt.Errorf("未执行")}
	}
	return results
}

// runTasksPlain 非 TTY（重定向/管道）下的降级输出
func runTasksPlain(tasks []Task, opts ProgressOptions) []taskOutcome {
	var mu sync.Mutex
	outcomes := make([]taskOutcome, 0, len(tasks))

	runTasks(tasks, opts.Concurrency, nil, func(o taskOutcome) {
		mu.Lock()
		defer mu.Unlock()

		outcomes = append(outcomes, o)
		if o.err != nil {
			fmt.Fprintf(os.Stderr, "✗ %s: %v\n", tasks[o.index].Label, o.err)
			return
		}
		fmt.Printf("✓ %s %s\n", tasks[o.index].Label, o.elapsed.Round(time.Millisecond).String())
	})

	succeeded := 0
	for _, o := range outcomes {
		if o.err == nil {
			succeeded++
		}
	}
	fmt.Printf("共 %d 个：成功 %d 个，失败 %d 个\n", len(outcomes), succeeded, len(outcomes)-succeeded)

	if opts.Hint != "" && succeeded > 0 {
		fmt.Printf("%s\n", opts.Hint)
	}

	return outcomes
}

// runTasks 以固定并发跑完全部任务，onStart 在任务真正开始时回调
func runTasks(tasks []Task, concurrency int, onStart func(index int), onDone func(taskOutcome)) {
	if concurrency <= 0 {
		concurrency = defaultTaskConcurrency
	}

	sem := make(chan struct{}, concurrency)
	var wg sync.WaitGroup

	for i := range tasks {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()

			sem <- struct{}{}
			defer func() { <-sem }()

			if onStart != nil {
				onStart(i)
			}
			start := time.Now()
			err := tasks[i].Run()
			onDone(taskOutcome{index: i, err: err, elapsed: time.Since(start)})
		}(i)
	}

	wg.Wait()
}
