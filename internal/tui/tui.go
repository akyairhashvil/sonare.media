package tui

import (
	"fmt"

	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"sonare.media/internal/store"
)

var baseStyle = lipgloss.NewStyle().
	BorderStyle(lipgloss.NormalBorder()).
	BorderForeground(lipgloss.Color("240"))

var detailStyle = lipgloss.NewStyle().
	Padding(1, 2).
	Border(lipgloss.RoundedBorder()).
	BorderForeground(lipgloss.Color("62"))

type model struct {
	table          table.Model
	viewport       viewport.Model
	activeTab      int // 0: Leads, 1: Analytics
	leads          []store.Lead
	analytics      []store.Analytics
	viewingDetails bool
	selectedIdx    int
	ready          bool
}

func (m model) Init() tea.Cmd { return nil }

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.viewport.Width = msg.Width
		m.viewport.Height = msg.Height - 10
		m.table.SetWidth(msg.Width - 10)
		m.ready = true

	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		
		case "esc":
			if m.viewingDetails {
				m.viewingDetails = false
				return m, nil
			}

		case "enter":
			if !m.viewingDetails {
				selectedRow := m.table.Cursor()
				if selectedRow >= 0 {
					m.viewingDetails = true
					m.selectedIdx = selectedRow
					m.updateDetailViewport()
				}
			}

		case "tab":
			if !m.viewingDetails {
				m.activeTab = (m.activeTab + 1) % 2
				m.refreshTable()
			}

		case "r": // Refresh
			if !m.viewingDetails {
				m.refreshTable()
			}
		}
	}

	if m.viewingDetails {
		m.viewport, cmd = m.viewport.Update(msg)
		return m, cmd
	}

	m.table, cmd = m.table.Update(msg)
	return m, cmd
}

func (m *model) updateDetailViewport() {
	var content string
	
	if m.activeTab == 0 {
		// Leads
		if m.selectedIdx < len(m.leads) {
			l := m.leads[m.selectedIdx]
			content = fmt.Sprintf(`
TITLE: Lead #%d
-----------------------------
NAME:     %s
BUSINESS: %s
EMAIL:    %s
SYSTEM:   %s
PALETTE:  %s
SCALE:    %d Hours / %d Stores

TIME:     %s

MESSAGE:
%s
`, 
				l.ID, l.Name, l.Business, l.Email, l.Playback, l.Palette, l.HoursEst, l.StoreCount, 
				l.CreatedAt.Format("Mon Jan 2 15:04:05 2006"), 
				l.Message)
		}
	} else {
		// Analytics
		if m.selectedIdx < len(m.analytics) {
			a := m.analytics[m.selectedIdx]
			content = fmt.Sprintf(`
TITLE: Analytics Event #%d
-----------------------------
IP ADDRESS: %s
LOCATION:   %s, %s
PATH:       %s
METHOD:     %s
USER AGENT: %s
TIME:       %s
`, 
				a.ID, a.IP, a.City, a.Country, a.Path, a.Method, a.UserAgent,
				a.CreatedAt.Format("Mon Jan 2 15:04:05 2006"))
		}
	}

	m.viewport.SetContent(detailStyle.Render(content))
}

func (m model) View() string {
	if !m.ready {
		return "Initializing..."
	}

	if m.viewingDetails {
		return fmt.Sprintf("%s\n\nPress 'esc' to go back • 'q' to quit", m.viewport.View())
	}

	tabs := []string{"Form Entries (Leads)", "Analytics"}
	var tabRow string
	for i, t := range tabs {
		style := lipgloss.NewStyle().Padding(0, 1).Foreground(lipgloss.Color("240"))
		if i == m.activeTab {
			style = style.Foreground(lipgloss.Color("205")).Bold(true)
		}
		tabRow += style.Render(t) + "  "
	}

	return baseStyle.Render(
		lipgloss.JoinVertical(lipgloss.Left,
			tabRow+"\n",
			m.table.View(),
			"\nPress 'enter' to view details • 'tab' to switch • 'r' to refresh • 'q' to quit",
		),
	) + "\n"
}

func (m *model) refreshTable() {
	columns := []table.Column{}

rows := []table.Row{}

	if m.activeTab == 0 {
		// Fetch Leads
		var err error
		m.leads, err = store.GetLeads()
		if err != nil {
			m.leads = []store.Lead{} // Handle error gracefully
		}

		columns = []table.Column{
			{Title: "ID", Width: 4},
			{Title: "Name", Width: 15},
			{Title: "Business", Width: 15},
			{Title: "Email", Width: 20},
			{Title: "Message", Width: 30},
			{Title: "Time", Width: 20},
		}

		for _, l := range m.leads {
			rows = append(rows, table.Row{
				fmt.Sprintf("%d", l.ID),
				l.Name,
				l.Business,
				l.Email,
				truncate(l.Message, 28),
				l.CreatedAt.Format("2006-01-02 15:04"),
			})
		}
	} else {
		// Fetch Analytics
		var err error
		m.analytics, err = store.GetAnalytics()
		if err != nil {
			m.analytics = []store.Analytics{}
		}

		columns = []table.Column{
			{Title: "IP", Width: 15},
			{Title: "Loc", Width: 15},
			{Title: "Path", Width: 15},
			{Title: "Method", Width: 6},
			{Title: "Time", Width: 20},
		}

		for _, a := range m.analytics {
			loc := fmt.Sprintf("%s, %s", a.City, a.Country)
			rows = append(rows, table.Row{
				a.IP,
				truncate(loc, 15),
				a.Path,
				a.Method,
				a.CreatedAt.Format("15:04:05"),
			})
		}
	}

	m.table.SetRows([]table.Row{}) // Clear to prevent panic
	m.table.SetColumns(columns)
	m.table.SetRows(rows)
}

func truncate(s string, max int) string {
	if len(s) > max {
		return s[:max-1] + "…"
	}
	return s
}

func Start() error {
	columns := []table.Column{{Title: "Loading...", Width: 10}}
	t := table.New(
		table.WithColumns(columns),
		table.WithFocused(true),
		table.WithHeight(10),
	)

	s := table.DefaultStyles()
	// Using default styles to avoid compilation issues with method chaining
	t.SetStyles(s)
	
	vp := viewport.New(0, 0)

	m := model{
		table: t, 
		viewport: vp,
		activeTab: 0,
		ready: false, // Wait for window size msg
	}
	m.refreshTable() // Load initial data

	// Hack: Simulate a ready state for non-TTY environments just in case, 
	// though Bubble Tea usually sends a window size msg immediately.
	// We'll trust the event loop.

	if _, err := tea.NewProgram(m, tea.WithAltScreen()).Run(); err != nil {
		return err
	}
	return nil
}