package tui

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/kube-server/kube-server/internal/metrics"
)

type tickMsg time.Time
type metricsMsg struct {
	nodes []metrics.NodeMetrics
	pods  []metrics.PodMetrics
}

type view int

const (
	viewNodes view = iota
	viewPods
	viewLogs
)

var (
	headerStyle  = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("86"))
	titleStyle   = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("212")).Padding(0, 1)
	activeTab    = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("212")).Underline(true).Padding(0, 1)
	inactiveTab  = lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Padding(0, 1)
	borderStyle  = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("240"))
	okStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("82"))
	warnStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("214"))
	errorStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
)

type Model struct {
	collector   *metrics.Collector
	clusterName string
	nodes       []metrics.NodeMetrics
	pods        []metrics.PodMetrics
	nodeTable   table.Model
	podTable    table.Model
	currentView view
	width       int
	height      int
	err         error
}

func New(collector *metrics.Collector, clusterName string) *Model {
	return &Model{
		collector:   collector,
		clusterName: clusterName,
		nodeTable:   newNodeTable(),
		podTable:    newPodTable(),
	}
}

func newNodeTable() table.Model {
	cols := []table.Column{
		{Title: "NAME", Width: 20},
		{Title: "CPU", Width: 15},
		{Title: "CPU%", Width: 10},
		{Title: "MEMORY", Width: 15},
		{Title: "MEM%", Width: 10},
	}
	t := table.New(table.WithColumns(cols), table.WithFocused(true))
	s := table.DefaultStyles()
	s.Header = s.Header.Bold(true).Foreground(lipgloss.Color("86"))
	s.Selected = s.Selected.Foreground(lipgloss.Color("212")).Bold(true)
	t.SetStyles(s)
	return t
}

func newPodTable() table.Model {
	cols := []table.Column{
		{Title: "NAME", Width: 30},
		{Title: "NAMESPACE", Width: 15},
		{Title: "NODE", Width: 20},
		{Title: "CPU", Width: 10},
		{Title: "MEM", Width: 10},
		{Title: "STATUS", Width: 10},
		{Title: "RESTARTS", Width: 10},
	}
	t := table.New(table.WithColumns(cols), table.WithFocused(true))
	s := table.DefaultStyles()
	s.Header = s.Header.Bold(true).Foreground(lipgloss.Color("86"))
	s.Selected = s.Selected.Foreground(lipgloss.Color("212")).Bold(true)
	t.SetStyles(s)
	return t
}

func (m *Model) Init() tea.Cmd {
	return tea.Batch(tickCmd(), m.fetchMetrics())
}

func tickCmd() tea.Cmd {
	return tea.Tick(5*time.Second, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func (m *Model) fetchMetrics() tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		nodes, err := m.collector.GetNodeMetrics(ctx)
		if err != nil {
			return err
		}
		pods, err := m.collector.GetPodMetrics(ctx, "")
		if err != nil {
			return err
		}
		return metricsMsg{nodes: nodes, pods: pods}
	}
}

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "1":
			m.currentView = viewNodes
		case "2":
			m.currentView = viewPods
		case "3":
			m.currentView = viewLogs
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case metricsMsg:
		m.nodes = msg.nodes
		m.pods = msg.pods
		m.updateTables()

	case tickMsg:
		return m, tea.Batch(tickCmd(), m.fetchMetrics())

	case error:
		m.err = msg
	}

	var cmd tea.Cmd
	switch m.currentView {
	case viewNodes:
		m.nodeTable, cmd = m.nodeTable.Update(msg)
	case viewPods:
		m.podTable, cmd = m.podTable.Update(msg)
	}
	return m, cmd
}

func (m *Model) updateTables() {
	nodeRows := make([]table.Row, len(m.nodes))
	for i, n := range m.nodes {
		nodeRows[i] = table.Row{
			n.Name,
			n.CPUUsage,
			fmt.Sprintf("%.1f%%", n.CPUPercent),
			n.MemUsage,
			fmt.Sprintf("%.1f%%", n.MemPercent),
		}
	}
	m.nodeTable.SetRows(nodeRows)

	podRows := make([]table.Row, len(m.pods))
	for i, p := range m.pods {
		podRows[i] = table.Row{
			p.Name,
			p.Namespace,
			p.Node,
			p.CPUUsage,
			p.MemUsage,
			colorStatus(p.Status),
			fmt.Sprintf("%d", p.Restarts),
		}
	}
	m.podTable.SetRows(podRows)
}

func colorStatus(status string) string {
	switch status {
	case "Running":
		return okStyle.Render(status)
	case "Pending":
		return warnStyle.Render(status)
	default:
		return errorStyle.Render(status)
	}
}

func progressBar(pct float64, width int) string {
	filled := int(pct / 100 * float64(width))
	if filled > width {
		filled = width
	}
	bar := strings.Repeat("█", filled) + strings.Repeat("░", width-filled)
	color := "82"
	if pct > 80 {
		color = "196"
	} else if pct > 60 {
		color = "214"
	}
	return lipgloss.NewStyle().Foreground(lipgloss.Color(color)).Render(bar)
}

func (m *Model) View() string {
	if m.err != nil {
		return errorStyle.Render(fmt.Sprintf("Error: %v\n\nPress q to quit", m.err))
	}

	title := titleStyle.Render(fmt.Sprintf("⎈ kube-server | cluster: %s", m.clusterName))

	tabs := lipgloss.JoinHorizontal(lipgloss.Top,
		m.tabStyle(viewNodes, "1: Nodes"),
		m.tabStyle(viewPods, "2: Pods"),
		m.tabStyle(viewLogs, "3: Logs"),
	)

	header := lipgloss.JoinVertical(lipgloss.Left, title, tabs, headerStyle.Render(strings.Repeat("─", m.width)))

	var content string
	switch m.currentView {
	case viewNodes:
		content = m.renderNodes()
	case viewPods:
		content = m.renderPods()
	case viewLogs:
		content = "Logs view - coming soon"
	}

	help := lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Render("q: quit • 1-3: switch view • ↑↓: navigate")

	return lipgloss.JoinVertical(lipgloss.Left, header, content, help)
}

func (m *Model) tabStyle(v view, label string) string {
	if m.currentView == v {
		return activeTab.Render(label)
	}
	return inactiveTab.Render(label)
}

func (m *Model) renderNodes() string {
	if len(m.nodes) == 0 {
		return warnStyle.Render("No nodes found or metrics-server not available")
	}

	bars := ""
	for _, n := range m.nodes {
		bars += fmt.Sprintf("  %-20s CPU %s %.1f%%  MEM %s %.1f%%\n",
			n.Name,
			progressBar(n.CPUPercent, 20),
			n.CPUPercent,
			progressBar(n.MemPercent, 20),
			n.MemPercent,
		)
	}

	return lipgloss.JoinVertical(lipgloss.Left,
		bars,
		borderStyle.Render(m.nodeTable.View()),
	)
}

func (m *Model) renderPods() string {
	if len(m.pods) == 0 {
		return warnStyle.Render("No pods found")
	}
	return borderStyle.Render(m.podTable.View())
}

func Run(collector *metrics.Collector, clusterName string) error {
	m := New(collector, clusterName)
	p := tea.NewProgram(m, tea.WithAltScreen())
	_, err := p.Run()
	return err
}
