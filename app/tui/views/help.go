package views

import (
	"embed"
	"fmt"

	"github.com/charmbracelet/bubbles/viewport"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"
)

//go:embed help-docs/*.md
var helpFS embed.FS

var (
	helpStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))

	// Simple tab styling
	activeTabStyle = lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("87")).
		Background(lipgloss.Color("235")).
		Padding(0, 1)

	inactiveTabStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("245")).
		Padding(0, 1)

	tabBarStyle = lipgloss.NewStyle().
		BorderBottom(true).
		BorderBottomForeground(lipgloss.Color("87")).
		MarginBottom(1)
)

type HelpTab struct {
	Name string
	File string
	Content string
}

type HelpModel struct {
	viewport    viewport.Model
	tabs        []HelpTab
	activeTab   int
	width       int
	height      int
}

func NewHelpModel(width, height int) (*HelpModel, error) {
	// Define help tabs in order
	tabs := []HelpTab{
		{Name: "Overview", File: "help-docs/overview.md"},
		{Name: "Registering", File: "help-docs/registering.md"},
		{Name: "Syncing", File: "help-docs/Syncing.md"},
		{Name: "Inter-Agent Messaging", File: "help-docs/messaging.md"},
		{Name: "Messages", File: "help-docs/stored-convos.md"},
		{Name: "About", File: "help-docs/about.md"},
	}

	// Configure glamour renderer
	const glamourGutter = 2
	glamourRenderWidth := width - 8 - glamourGutter // Account for viewport borders and padding

	renderer, err := glamour.NewTermRenderer(
		glamour.WithAutoStyle(),
		glamour.WithWordWrap(glamourRenderWidth),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create glamour renderer: %w", err)
	}

	// Load content for all tabs
	for i := range tabs {
		content, err := helpFS.ReadFile(tabs[i].File)
		if err != nil {
			return nil, fmt.Errorf("failed to read %s: %w", tabs[i].File, err)
		}

		// Render markdown content
		str, err := renderer.Render(string(content))
		if err != nil {
			return nil, fmt.Errorf("failed to render markdown for %s: %w", tabs[i].File, err)
		}

		tabs[i].Content = str
	}

	// Create viewport with responsive dimensions
	vp := viewport.New(width-4, height-10) // Account for borders, tabs, footer, and title
	vp.Style = lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("87")).
		PaddingRight(2)

	// Set initial content to first tab
	if len(tabs) > 0 {
		vp.SetContent(tabs[0].Content)
	}

	return &HelpModel{
		viewport:  vp,
		tabs:      tabs,
		activeTab: 0,
		width:     width,
		height:    height,
	}, nil
}

func (m *HelpModel) Update(width, height int) {
	// Update dimensions if changed
	if width != m.width || height != m.height {
		m.width = width
		m.height = height
		m.viewport.Width = width - 4
		m.viewport.Height = height - 10 // Account for tabs, footer, and title
	}
}

func (m *HelpModel) UpdateViewport(msg interface{}) {
	var cmd interface{}
	m.viewport, cmd = m.viewport.Update(msg)
	_ = cmd // Ignore command for now since we're just handling navigation
}

func (m *HelpModel) NextTab() {
	if len(m.tabs) > 0 {
		m.activeTab = (m.activeTab + 1) % len(m.tabs)
		m.viewport.SetContent(m.tabs[m.activeTab].Content)
		m.viewport.GotoTop() // Reset scroll position
	}
}

func (m *HelpModel) PrevTab() {
	if len(m.tabs) > 0 {
		m.activeTab = (m.activeTab - 1 + len(m.tabs)) % len(m.tabs)
		m.viewport.SetContent(m.tabs[m.activeTab].Content)
		m.viewport.GotoTop() // Reset scroll position
	}
}

func (m *HelpModel) renderTabs() string {
	var tabs []string
	for i, tab := range m.tabs {
		if i == m.activeTab {
			tabs = append(tabs, activeTabStyle.Render(tab.Name))
		} else {
			tabs = append(tabs, inactiveTabStyle.Render(tab.Name))
		}
	}
	return tabBarStyle.Render(lipgloss.JoinHorizontal(lipgloss.Top, tabs...))
}

func (m *HelpModel) View() string {
	// Simple ASCII title
	title := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#87CEEB")).
		Bold(true).
		Align(lipgloss.Center).
		Render("─── HELP ───")

	return title + "\n" + m.renderTabs() + "\n" + m.viewport.View() + m.helpFooter()
}

func (m *HelpModel) helpFooter() string {
	return helpStyle.Render("\n  ↑/↓: Navigate • ←/→: Switch tabs • q/Esc: Back to agents view\n")
}
