package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/pgbkrs/pgbackup/internal/resolve"
)

// SelectionResult holds the result of the interactive selector.
type SelectionResult struct {
	Schemas   []string // schemas where all tables were selected
	Objects   []string // individual "schema.table" objects selected
	Cancelled bool
}

type item struct {
	schema   string
	name     string // empty = schema-level row
	selected bool
}

type selectorModel struct {
	items  []item
	cursor int
	height int
	result *SelectionResult
}

var (
	cursorStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("212"))
	selectedStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("10"))
	headerStyle   = lipgloss.NewStyle().Bold(true)
	helpStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
)

func newSelectorModel(manifest *resolve.Manifest) selectorModel {
	schemaOrder := []string{}
	schemaSeen := map[string]bool{}
	tablesBySchema := map[string][]string{}
	tableSeen := map[string]bool{}

	for _, entry := range manifest.RestoreOrder {
		if entry.Kind != "table" {
			continue
		}
		if !schemaSeen[entry.Schema] {
			schemaSeen[entry.Schema] = true
			schemaOrder = append(schemaOrder, entry.Schema)
		}
		key := entry.Schema + "." + entry.Name
		if !tableSeen[key] {
			tableSeen[key] = true
			tablesBySchema[entry.Schema] = append(tablesBySchema[entry.Schema], entry.Name)
		}
	}

	var items []item
	for _, schema := range schemaOrder {
		items = append(items, item{schema: schema})
		for _, table := range tablesBySchema[schema] {
			items = append(items, item{schema: schema, name: table})
		}
	}
	return selectorModel{items: items, height: 20}
}

func (m selectorModel) Init() tea.Cmd { return nil }

func (m selectorModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q", "esc":
			m.result = &SelectionResult{Cancelled: true}
			return m, tea.Quit
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(m.items)-1 {
				m.cursor++
			}
		case " ":
			m.toggle(m.cursor)
		case "a":
			for i := range m.items {
				m.items[i].selected = true
			}
		case "d":
			for i := range m.items {
				m.items[i].selected = false
			}
		case "enter":
			m.result = m.buildResult()
			return m, tea.Quit
		}
	case tea.WindowSizeMsg:
		if msg.Height > 6 {
			m.height = msg.Height - 6
		}
	}
	return m, nil
}

func (m *selectorModel) toggle(idx int) {
	it := &m.items[idx]
	if it.name == "" {
		// Schema row: toggle all tables in this schema
		newVal := !it.selected
		it.selected = newVal
		for i := range m.items {
			if m.items[i].schema == it.schema && m.items[i].name != "" {
				m.items[i].selected = newVal
			}
		}
	} else {
		it.selected = !it.selected
	}
}

func (m selectorModel) buildResult() *SelectionResult {
	r := &SelectionResult{}

	totalBySchema := map[string]int{}
	selectedBySchema := map[string][]string{}
	for i := range m.items {
		it := &m.items[i]
		if it.name == "" {
			continue
		}
		totalBySchema[it.schema]++
		if it.selected {
			selectedBySchema[it.schema] = append(selectedBySchema[it.schema], it.name)
		}
	}

	for schema, tables := range selectedBySchema {
		if len(tables) == totalBySchema[schema] {
			r.Schemas = append(r.Schemas, schema)
		} else {
			for _, t := range tables {
				r.Objects = append(r.Objects, schema+"."+t)
			}
		}
	}
	return r
}

func (m selectorModel) View() string {
	if len(m.items) == 0 {
		return "No tables found in backup.\n\nPress q to quit."
	}

	var sb strings.Builder
	sb.WriteString(headerStyle.Render("Select schemas/tables to restore") + "\n")
	sb.WriteString(helpStyle.Render("↑/↓ navigate  space toggle  a select all  d deselect all  enter confirm  q cancel") + "\n\n")

	start := 0
	if m.cursor >= m.height {
		start = m.cursor - m.height + 1
	}
	end := start + m.height
	if end > len(m.items) {
		end = len(m.items)
	}

	for i := start; i < end; i++ {
		it := m.items[i]
		checkbox := "[ ]"
		if it.selected {
			checkbox = "[x]"
		}

		var line string
		if it.name == "" {
			label := fmt.Sprintf("%s %s/", checkbox, it.schema)
			if i == m.cursor {
				line = cursorStyle.Render("> ") + headerStyle.Render(label)
			} else {
				line = "  " + headerStyle.Render(label)
			}
		} else {
			label := fmt.Sprintf("%s   %s", checkbox, it.name)
			if it.selected {
				label = selectedStyle.Render(label)
			}
			if i == m.cursor {
				line = cursorStyle.Render("> ") + "  " + label
			} else {
				line = "      " + label
			}
		}
		sb.WriteString(line + "\n")
	}

	selCount := 0
	for _, it := range m.items {
		if it.selected && it.name != "" {
			selCount++
		}
	}
	sb.WriteString(fmt.Sprintf("\n%d table(s) selected", selCount))
	return sb.String()
}

// RunSelector launches the interactive TUI and returns the user's selection.
func RunSelector(manifest *resolve.Manifest) (*SelectionResult, error) {
	m := newSelectorModel(manifest)
	if len(m.items) == 0 {
		return &SelectionResult{}, nil
	}
	p := tea.NewProgram(m, tea.WithAltScreen())
	final, err := p.Run()
	if err != nil {
		return nil, err
	}
	fm := final.(selectorModel)
	if fm.result == nil {
		return &SelectionResult{Cancelled: true}, nil
	}
	return fm.result, nil
}
