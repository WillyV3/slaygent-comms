# Slaygent TUI Feature Development Guide

## Architecture Overview

The Slaygent TUI follows a distributed Bubble Tea architecture:

```
main.go          - Model definition, Init(), View(), helper functions
update.go        - Update() function with all state transitions
views/           - Stateless view rendering functions
  agents.go      - Agent list view
  messages.go    - Message history view
  sync.go        - Registry sync customization view
history/         - Database layer for messages
registry.go      - Agent registry operations
tmux.go          - tmux pane interaction
```

## Adding a New View Feature - Step by Step

### Step 1: Find a Bubble Tea Example

Browse `/Users/williamvansickleiii/charmtuitemplate/bubbletea-docs/bubbletea-repo/examples/` for a similar component:
- `textarea/` - For text editing
- `table/` - For data grids
- `viewport/` - For scrollable content
- `list-simple/` - For selectable lists
- `split-editors/` - For multi-panel layouts
- `progress-animated/` - For progress indicators

### Step 2: Update the Model (main.go)

Add fields for your new view state:

```go
type model struct {
    // Existing fields...

    // New view fields
    yourViewMode    string        // "your_view"
    yourViewData    YourDataType  // View-specific data
    yourViewInput   textarea.Model // If input needed
    yourViewState   string        // Any state tracking
}
```

### Step 3: Create Stateless View (views/your_view.go)

Create a new file in `views/` with stateless rendering:

```go
package views

import (
    "fmt"
    "github.com/charmbracelet/lipgloss"
)

// YourViewData contains all data needed to render your view
type YourViewData struct {
    // All data needed for rendering
    Content      string
    Width        int
    Height       int
    SelectedItem int
}

// RenderYourView renders the view without any state management
func RenderYourView(data YourViewData) string {
    if data.Width == 0 || data.Height == 0 {
        return "Loading..."
    }

    // Style definitions
    headerStyle := lipgloss.NewStyle().
        Foreground(lipgloss.Color("#87CEEB")).
        Bold(true)

    // Build view
    header := headerStyle.Render("YOUR VIEW TITLE")
    content := formatContent(data)

    return fmt.Sprintf("%s\n\n%s", header, content)
}

func formatContent(data YourViewData) string {
    // Format your content
    return data.Content
}
```

### Step 4: Add View Mode to Update() (update.go)

Add state transitions in the Update function with early returns for clarity:

```go
func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    // ... existing code ...

    // Handle your view mode with early return pattern
    if m.viewMode != "your_view" {
        // Let main Update handle other modes
        return m.handleOtherViews(msg)
    }

    // Update components first if they have focus
    if m.yourViewInput.Focused() {
        var cmd tea.Cmd
        m.yourViewInput, cmd = m.yourViewInput.Update(msg)
        return m, cmd
    }

    // Handle view-specific keys
    switch msg := msg.(type) {
    case tea.KeyMsg:
        switch msg.String() {
        case "esc":
            m.viewMode = "agents"
            return m, nil
        case "tab":
            m.yourViewInput.Focus()
            return m, nil
        }
    }

    return m, nil
}
```

### Step 5: Wire Up View Rendering (main.go View())

Add to the View() function:

```go
func (m model) View() string {
    // ... existing code ...

    switch m.viewMode {
    case "agents":
        // existing
    case "messages":
        // existing
    case "sync":
        // existing
    case "your_view":
        data := views.YourViewData{
            Content:      m.yourViewData,
            Width:        m.width,
            Height:       m.height,
            SelectedItem: m.selected,
        }
        return views.RenderYourView(data)
    default:
        // existing
    }
}
```

### Step 6: Add Navigation (update.go)

Add key binding to enter your view from agents view:

```go
// In Update(), agents view section
case "y":  // Your chosen key
    m.viewMode = "your_view"
    // Initialize any components
    m.yourViewInput = buildYourInput()
    return m, nil
```

## View Consistency Rules (Critical!)

### View Structure Pattern
Every view MUST follow this consistent pattern to avoid complexity:

```go
// Data struct - contains ALL data needed for rendering
type ViewNameViewData struct {
    // Required fields
    Width  int
    Height int
    // View-specific fields...
}

// Styling constants at package level (not inline!)
var (
    viewTitleStyle = lipgloss.NewStyle().
        Foreground(lipgloss.Color("#87CEEB")).
        Bold(true)

    viewControlsStyle = lipgloss.NewStyle().
        Foreground(lipgloss.Color("#888888"))

    focusedBorderColor   = lipgloss.Color("#87CEEB")
    unfocusedBorderColor = lipgloss.Color("#006666")
)

// Main render function - simple and focused
func RenderViewNameView(data ViewNameViewData) string {
    // 1. Simple calculations at top (width/height)
    leftWidth := data.Width / 3
    if leftWidth < 25 { leftWidth = 25 }
    rightWidth := data.Width - leftWidth - 6
    panelHeight := data.Height - 8

    // 2. Build content using helper functions
    title := viewTitleStyle.Render("VIEW TITLE")
    controls := viewControlsStyle.Render("↑/↓: navigate • ESC: back")
    leftPanel := renderLeftPanel(data, leftWidth, panelHeight)
    rightPanel := renderRightPanel(data, rightWidth, panelHeight)

    // 3. Assemble and return
    content := lipgloss.JoinHorizontal(lipgloss.Top, leftPanel, " ", rightPanel)
    return "\n" + title + "\n\n" + content + "\n\n" + controls
}

// Helper functions for complex components
func renderLeftPanel(data ViewNameViewData, width, height int) string {
    // Focused single responsibility
    borderColor := unfocusedBorderColor
    if data.Focus == "left" {
        borderColor = focusedBorderColor
    }

    return panelStyle.
        Width(width).
        Height(height).
        BorderForeground(borderColor).
        Render(data.LeftContent)
}
```

### ❌ Complexity Anti-Patterns
**DO NOT DO THIS** (from messages.go before refactor):

```go
// DON'T: Multiple width calculations scattered throughout
availableWidth := data.Width - 6
leftPanelWidth := availableWidth / 3
if leftPanelWidth < 25 {
    leftPanelWidth = 25  // Minimum width for readability
}
// ... 50 lines later ...
rightPanelWidth := availableWidth - leftPanelWidth - 2

// DON'T: Inline styling with complex logic
leftPanel := lipgloss.NewStyle().
    Width(leftPanelWidth).
    Height(panelHeight).
    Border(lipgloss.NormalBorder()).
    BorderForeground(getComplexBorderColor(focus, state)).
    Render(convList)

// DON'T: Responsive control text variants
fullControls := "Navigation: ↑/↓: navigate • ←/→: switch panels • d: delete conversation • ESC: return to agents"
mediumControls := "↑/↓: navigate • ←/→: panels • d: delete • ESC: back"
compactControls := "↑/↓ • ←/→ • d:del • ESC"
if lipgloss.Width(style.Render(fullControls)) <= width-4 {
    return style.Render(fullControls)
}
// etc...

// DON'T: 150+ line render functions with mixed concerns
```

### ✅ Simple Patterns
**DO THIS INSTEAD**:

```go
// DO: Calculate dimensions once at top
leftWidth := data.Width / 3
if leftWidth < 25 { leftWidth = 25 }
rightWidth := data.Width - leftWidth - 6
panelHeight := data.Height - 8

// DO: Use helper functions for complex components
leftPanel := renderLeftPanel(data, leftWidth, panelHeight)
rightPanel := renderRightPanel(data, rightWidth, panelHeight)

// DO: Simple static controls (no responsive variants)
controls := controlsStyle.Render("↑/↓: navigate • ESC: back")

// DO: Package-level styling constants
var focusedBorderColor = lipgloss.Color("#87CEEB")
```

### Color Palette (Mandatory)
- **Primary**: `#87CEEB` (Baby blue) - Headers, highlights, focused borders
- **Secondary**: `#888888` - Controls, muted text
- **Unfocused**: `#006666` - Inactive borders
- **Success**: `#00FF00` - Success messages
- **Warning/Error**: `#FF6B6B` - Errors, delete confirmations

### Messages View Refactor Example
**Before** (Complex, 150+ lines):
- Width calculations in 5+ places
- 4 different responsive control variants
- Inline styling throughout
- Mixed concerns (rendering + calculations + styling)
- Complex responsive logic

**After** (Simple, ~30 lines):
- Width calculated once at top
- Single static control text
- Styling constants at package level
- Clear helper functions for panels
- Linear, easy-to-follow logic

## Pattern Guidelines

### 0. No Fallbacks Rule - Fail Fast and Obvious
- **NEVER write fallback behavior** - if something is wrong, fail immediately
- **NO silent defaults** - missing data should panic or show clear error
- **NO recovery attempts** - let errors bubble up visibly
- Example of what NOT to do:
  ```go
  // BAD - silent fallback
  if data.Width == 0 {
      data.Width = 80  // NO! This hides problems
  }

  // GOOD - fail fast
  if data.Width == 0 {
      panic("terminal width not initialized")
  }
  ```

### 1. Stateless Views
- Views/ functions should be pure rendering functions
- Pass all needed data via structs
- No direct model access in views/
- Keep view functions under 100 lines for readability

### 2. Component Initialization
- Initialize Bubble Tea components (textarea, table, etc.) when entering view
- Store them in the model
- Update them in Update()

### 3. Key Handling Priority
```go
// In Update() - use early returns to flatten control flow
func handleYourView(m model, msg tea.Msg) (tea.Model, tea.Cmd) {
    // Component updates take priority
    if m.yourComponent.Focused() {
        var cmd tea.Cmd
        m.yourComponent, cmd = m.yourComponent.Update(msg)
        return m, cmd
    }

    // Handle keyboard input
    keyMsg, isKey := msg.(tea.KeyMsg)
    if !isKey {
        return m, nil
    }

    switch keyMsg.String() {
    case "tab":
        m.yourComponent.Focus()
        return m, nil
    case "esc":
        m.viewMode = "agents"
        return m, nil
    }

    return m, nil
}
```

### 4. Progress/Async Operations
- Reuse existing progress bar from agents view
- Switch viewMode to "agents" to show progress
- Use tea.Cmd for async operations

Example:
```go
func (m model) runYourAsyncOperation() tea.Cmd {
    return func() tea.Msg {
        // Do async work
        result := performOperation()
        return yourCompleteMsg{result: result}
    }
}
```

### 5. Color System
Use consistent colors from existing views:
- Baby blue (#87CEEB) - Headers/borders
- Green (#00FF00) - Success/Active
- Red (#FF6B6B) - Errors/Warnings
- ANSI 256 colors - Dynamic content

## Common Pitfalls to Avoid

1. **Don't write fallbacks or silent defaults**
   - FAIL FAST - panic or return error immediately
   - No "if err != nil { use default }" patterns
   - Make failures obvious and unignorable

2. **Don't put business logic in views/**
   - Views should only render, not decide
   - Keep rendering logic simple and obvious

3. **Don't forget terminal dimensions**
   - Always check width/height in WindowSizeMsg
   - Pass dimensions to views
   - Panic if dimensions are zero (no fallbacks!)

4. **Don't create new files unless necessary**
   - Start in existing structure
   - Only create new files for complex features
   - Prefer extending existing files when possible

4. **Don't break the Update flow**
   - Use early returns to flatten control flow
   - Avoid deeply nested conditionals
   - Return proper tea.Cmd

5. **Don't ignore existing patterns**
   - Copy from similar views
   - Use existing helper functions
   - Follow the established naming conventions

6. **Don't over-abstract**
   - Write code for the current need, not future possibilities
   - Avoid creating helper functions without clear benefit
   - Repetition is acceptable if it improves clarity

## Testing Your New View

1. **Manual Testing Checklist:**
   - [ ] View renders at different terminal sizes
   - [ ] ESC key returns to agents view
   - [ ] All inputs respond correctly
   - [ ] Progress shows for async operations
   - [ ] No panic on edge cases

2. **Integration Points:**
   - [ ] Navigation from agents view works
   - [ ] Return to agents view works
   - [ ] Terminal resize handled
   - [ ] Components update properly

## Example: Adding a Settings View

Here's a complete minimal example without unnecessary complexity:

1. **main.go** - Add to model:
```go
settingsData map[string]string  // Just the data, no extra state
```

2. **views/settings.go**:
```go
package views

import (
    "fmt"
    "github.com/charmbracelet/lipgloss"
)

type SettingsData struct {
    Settings map[string]string
    Width    int
    Height   int
}

func RenderSettingsView(data SettingsData) string {
    headerStyle := lipgloss.NewStyle().
        Foreground(lipgloss.Color("#87CEEB")).
        Bold(true)

    header := headerStyle.Render("Settings")

    content := ""
    for key, value := range data.Settings {
        content += fmt.Sprintf("%s: %s\n", key, value)
    }

    return fmt.Sprintf("%s\n\n%s", header, content)
}
```

3. **update.go** - Add navigation:
```go
// In agents view section
case "s":
    m.viewMode = "settings"
    return m, nil

// In settings view section
if m.viewMode == "settings" && msg.String() == "esc" {
    m.viewMode = "agents"
    return m, nil
}
```

4. **main.go View()** - Add rendering:
```go
case "settings":
    return views.RenderSettingsView(views.SettingsData{
        Settings: m.settingsData,
        Width: m.width,
        Height: m.height,
    })
```

## Remember the Constitution

From BUBBLE_TEA_TMUX_TUI_GUIDE_2025.md:
- Start simple, iterate
- Single model, distributed files
- Stateless views
- Clear state transitions
- Reuse existing patterns

## Quick Reference Paths

- Examples: `/Users/williamvansickleiii/charmtuitemplate/bubbletea-docs/bubbletea-repo/examples/`
- Current views: `/Users/williamvansickleiii/charmtuitemplate/slaygent-comms/app/tui/views/`
- Model definition: `main.go:22-50`
- Update logic: `update.go`
- View rendering: `main.go View()`