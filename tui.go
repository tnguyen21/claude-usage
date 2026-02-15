package main

import (
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// messages

type usageFetchedMsg struct {
	usage *UsageResponse
	err   error
}

type tickMsg time.Time

// styles

var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("99"))

	labelStyle = lipgloss.NewStyle().
			Width(16).
			Foreground(lipgloss.Color("252"))

	resetStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("243")).
			MarginLeft(16)

	percentStyle = lipgloss.NewStyle().
			Width(6).
			Align(lipgloss.Right).
			Foreground(lipgloss.Color("252"))

	errorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("196"))

	staleStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("243")).
			Italic(true)

	footerStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("243"))

	borderStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("99")).
			Padding(1, 2)
)

// model

type model struct {
	usage     *UsageResponse
	err       error
	lastFetch time.Time
	stale     bool

	sessionBar progress.Model
	weeklyBar  progress.Model
	opusBar    progress.Model
	spinner    spinner.Model

	loading     bool
	width       int
	height      int
	token       string
	subType     string
	lastRefresh time.Time // debounce
	hoverBar    int       // -1 = none, 0 = session, 1 = weekly, 2 = opus
}

func newModel(token, subType string) model {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("99"))

	barWidth := 30

	return model{
		sessionBar: newBar(barWidth),
		weeklyBar:  newBar(barWidth),
		opusBar:    newBar(barWidth),
		spinner:    s,
		loading:    true,
		token:      token,
		subType:    subType,
		hoverBar:   -1,
	}
}

func newBar(width int) progress.Model {
	p := progress.New(
		progress.WithScaledGradient("#76EEC6", "#FF6347"),
		progress.WithWidth(width),
		progress.WithoutPercentage(),
	)
	return p
}

func (m model) Init() tea.Cmd {
	return tea.Batch(
		m.spinner.Tick,
		fetchCmd(m.token),
		tickCmd(),
	)
}

func fetchCmd(token string) tea.Cmd {
	return func() tea.Msg {
		usage, err := fetchUsage(token)
		return usageFetchedMsg{usage: usage, err: err}
	}
}

func tickCmd() tea.Cmd {
	return tea.Tick(5*time.Minute, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "r":
			if time.Since(m.lastRefresh) < 10*time.Second {
				return m, nil
			}
			m.loading = true
			m.lastRefresh = time.Now()
			return m, tea.Batch(m.spinner.Tick, fetchCmd(m.token))
		}

	case usageFetchedMsg:
		m.loading = false
		if msg.err != nil {
			if m.usage != nil {
				// keep stale data
				m.stale = true
				m.err = msg.err
			} else {
				m.err = msg.err
			}
			return m, nil
		}
		m.usage = msg.usage
		m.err = nil
		m.stale = false
		m.lastFetch = time.Now()

		var cmds []tea.Cmd
		if m.usage.FiveHour != nil {
			cmds = append(cmds, m.sessionBar.SetPercent(m.usage.FiveHour.Utilization/100))
		}
		if m.usage.SevenDay != nil {
			cmds = append(cmds, m.weeklyBar.SetPercent(m.usage.SevenDay.Utilization/100))
		}
		if m.usage.SevenDayOpus != nil {
			cmds = append(cmds, m.opusBar.SetPercent(m.usage.SevenDayOpus.Utilization/100))
		}
		return m, tea.Batch(cmds...)

	case tickMsg:
		m.loading = true
		return m, tea.Batch(m.spinner.Tick, fetchCmd(m.token), tickCmd())

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		barWidth := max(20, min(msg.Width-40, 40))
		m.sessionBar.Width = barWidth
		m.weeklyBar.Width = barWidth
		m.opusBar.Width = barWidth
		return m, nil

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd

	case tea.MouseMsg:
		// border(1) + padding(1) + title(1) + blank(1) = 4 lines before first bar
		// each bar section: bar line, reset line, blank line = 3 lines
		y := msg.Y
		barStart := 4 // first bar line Y
		old := m.hoverBar
		m.hoverBar = -1
		for i := 0; i < 3; i++ {
			rowTop := barStart + i*3
			rowBot := rowTop + 1 // bar line + reset line
			if y >= rowTop && y <= rowBot {
				m.hoverBar = i
				break
			}
		}
		if m.hoverBar != old {
			return m, nil
		}
		return m, nil

	case progress.FrameMsg:
		var cmds []tea.Cmd
		var cmd tea.Cmd

		pm, c := m.sessionBar.Update(msg)
		m.sessionBar = pm.(progress.Model)
		cmd = c
		cmds = append(cmds, cmd)

		pm, c = m.weeklyBar.Update(msg)
		m.weeklyBar = pm.(progress.Model)
		cmds = append(cmds, c)

		pm, c = m.opusBar.Update(msg)
		m.opusBar = pm.(progress.Model)
		cmds = append(cmds, c)

		return m, tea.Batch(cmds...)
	}

	return m, nil
}

func (m model) View() string {
	var b strings.Builder

	// title row
	title := titleStyle.Render("claude-usage")
	right := ""
	if m.loading {
		right = m.spinner.View()
	} else if m.stale {
		right = staleStyle.Render("stale")
	}
	titleRow := title
	if right != "" {
		titleRow += "  " + right
	}
	b.WriteString(titleRow + "\n\n")

	// error only (no data yet)
	if m.err != nil && m.usage == nil {
		b.WriteString(errorStyle.Render("  " + m.err.Error()) + "\n")
		return borderStyle.Render(b.String())
	}

	if m.usage != nil {
		barIdx := 0
		if m.usage.FiveHour != nil {
			b.WriteString(renderBar("Session (5h)", m.sessionBar, m.usage.FiveHour, m.hoverBar == barIdx))
			barIdx++
		}
		if m.usage.SevenDay != nil {
			b.WriteString(renderBar("Weekly (7d)", m.weeklyBar, m.usage.SevenDay, m.hoverBar == barIdx))
			barIdx++
		}
		if m.usage.SevenDayOpus != nil {
			b.WriteString(renderBar("Opus (7d)", m.opusBar, m.usage.SevenDayOpus, m.hoverBar == barIdx))
		}
	}

	// stale error
	if m.stale && m.err != nil {
		b.WriteString(staleStyle.Render("  " + m.err.Error()) + "\n\n")
	}

	// footer
	footer := ""
	if m.subType != "" {
		footer += strings.ToUpper(m.subType[:1]) + m.subType[1:]
	}
	if !m.lastFetch.IsZero() {
		if footer != "" {
			footer += "  •  "
		}
		footer += "updated " + m.lastFetch.Format("15:04")
	}
	if footer != "" {
		b.WriteString(footerStyle.Render(footer))
	}

	return borderStyle.Render(b.String())
}

var hoverPercentStyle = lipgloss.NewStyle().
	Width(8).
	Align(lipgloss.Right).
	Bold(true).
	Foreground(lipgloss.Color("255"))

func renderBar(label string, bar progress.Model, bucket *UsageBucket, hover bool) string {
	pct := bucket.Utilization
	var pctStr string
	if hover {
		pctStr = hoverPercentStyle.Render(fmt.Sprintf("%.2f%%", pct))
	} else {
		pctStr = percentStyle.Render(fmt.Sprintf("%.0f%%", pct))
	}
	line := labelStyle.Render(label) + bar.View() + " " + pctStr + "\n"

	resetLine := ""
	if bucket.ResetsAt != nil {
		resetLine = resetStyle.Render(formatReset(*bucket.ResetsAt)) + "\n"
	} else {
		resetLine = resetStyle.Render("—") + "\n"
	}

	return line + resetLine + "\n"
}

func formatReset(iso string) string {
	t, err := time.Parse(time.RFC3339, iso)
	if err != nil {
		return iso
	}

	until := time.Until(t)
	if until <= 0 {
		return "resetting..."
	}

	if until < time.Hour {
		return fmt.Sprintf("resets in %dm", int(math.Ceil(until.Minutes())))
	}
	if until < 24*time.Hour {
		h := int(until.Hours())
		m := int(until.Minutes()) % 60
		return fmt.Sprintf("resets in %dh %dm", h, m)
	}
	return "resets " + t.Local().Format("Mon Jan 2")
}
