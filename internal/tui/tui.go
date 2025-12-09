package tui

import (
	"fmt"
	"strings"

	"github.com/alecthomas/chroma/v2"
	"github.com/alecthomas/chroma/v2/lexers"
	"github.com/alecthomas/chroma/v2/styles"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/jm/hnk/internal/diff"
	"github.com/jm/hnk/internal/grouper"
)

type theme struct {
	added     lipgloss.Style
	removed   lipgloss.Style
	title     lipgloss.Style
	desc      lipgloss.Style
	file      lipgloss.Style
	lineNum   lipgloss.Style
	hunk      lipgloss.Style
	context   lipgloss.Style
	addedBg   lipgloss.Color
	removedBg lipgloss.Color
	syntax    *chroma.Style
}

var darkTheme = theme{
	added:     lipgloss.NewStyle().Background(lipgloss.Color("22")),
	removed:   lipgloss.NewStyle().Background(lipgloss.Color("52")),
	title:     lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("cyan")),
	desc:      lipgloss.NewStyle().Faint(true),
	file:      lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("blue")),
	lineNum:   lipgloss.NewStyle().Faint(true),
	hunk:      lipgloss.NewStyle().Foreground(lipgloss.Color("magenta")),
	context:   lipgloss.NewStyle(),
	addedBg:   lipgloss.Color("22"),
	removedBg: lipgloss.Color("52"),
}

var lightTheme = theme{
	added:     lipgloss.NewStyle().Background(lipgloss.Color("194")),
	removed:   lipgloss.NewStyle().Background(lipgloss.Color("224")),
	title:     lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("blue")),
	desc:      lipgloss.NewStyle().Foreground(lipgloss.Color("240")),
	file:      lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("magenta")),
	lineNum:   lipgloss.NewStyle().Foreground(lipgloss.Color("240")),
	hunk:      lipgloss.NewStyle().Foreground(lipgloss.Color("magenta")),
	context:   lipgloss.NewStyle(),
	addedBg:   lipgloss.Color("194"),
	removedBg: lipgloss.Color("224"),
}

type Model struct {
	groups       []grouper.SemanticGroup
	groupIndex   int
	scrollOffset int
	width        int
	height       int
	theme        theme
	lineNums     bool
	lines        []string
}

type Options struct {
	LightMode   bool
	LineNumbers bool
	StyleName   string
}

func New(groups []grouper.SemanticGroup, opts Options) Model {
	th := darkTheme
	syntaxStyle := "monokai"
	if opts.LightMode {
		th = lightTheme
		syntaxStyle = "github"
	}
	if opts.StyleName != "" {
		syntaxStyle = opts.StyleName
	}
	th.syntax = styles.Get(syntaxStyle)
	if th.syntax == nil {
		th.syntax = styles.Fallback
	}

	m := Model{
		groups:   groups,
		theme:    th,
		lineNums: opts.LineNumbers,
		width:    80,
		height:   24,
	}
	m.rebuildLines()
	return m
}

func (m Model) Init() tea.Cmd {
	return tea.WindowSize()
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c", "esc":
			return m, tea.Quit
		case "left", "h":
			if m.groupIndex > 0 {
				m.groupIndex--
				m.scrollOffset = 0
				m.rebuildLines()
			}
		case "right", "l":
			if m.groupIndex < len(m.groups)-1 {
				m.groupIndex++
				m.scrollOffset = 0
				m.rebuildLines()
			}
		case "up", "k":
			if m.scrollOffset > 0 {
				m.scrollOffset--
			}
		case "down", "j":
			maxScroll := len(m.lines) - m.contentHeight()
			if maxScroll < 0 {
				maxScroll = 0
			}
			if m.scrollOffset < maxScroll {
				m.scrollOffset++
			}
		case " ", "pgdown":
			maxScroll := len(m.lines) - m.contentHeight()
			if maxScroll < 0 {
				maxScroll = 0
			}
			m.scrollOffset += m.contentHeight()
			if m.scrollOffset > maxScroll {
				m.scrollOffset = maxScroll
			}
		case "pgup":
			m.scrollOffset -= m.contentHeight()
			if m.scrollOffset < 0 {
				m.scrollOffset = 0
			}
		case "g":
			m.scrollOffset = 0
		case "G":
			maxScroll := len(m.lines) - m.contentHeight()
			if maxScroll < 0 {
				maxScroll = 0
			}
			m.scrollOffset = maxScroll
		}
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	}
	return m, nil
}

func (m *Model) contentHeight() int {
	if m.height < 2 {
		return 1
	}
	return m.height - 1
}

func (m *Model) rebuildLines() {
	if len(m.groups) == 0 {
		m.lines = []string{"No changes to display"}
		return
	}

	group := m.groups[m.groupIndex]
	var lines []string

	lines = append(lines, m.theme.title.Render(group.Title))
	lines = append(lines, m.theme.desc.Render(group.Description))
	lines = append(lines, "")

	for _, gh := range group.Hunks {
		lines = append(lines, m.fileHeader(gh.File))
		lines = append(lines, m.hunkLines(gh.File, gh.Hunk)...)
		lines = append(lines, "")
	}

	m.lines = lines
}

func (m *Model) fileHeader(f *diff.FileDiff) string {
	var label string
	switch {
	case f.IsNew:
		label = fmt.Sprintf("+ %s (new)", f.NewPath)
	case f.IsDeleted:
		label = fmt.Sprintf("- %s (deleted)", f.OldPath)
	case f.IsRenamed:
		label = fmt.Sprintf("%s → %s", f.OldPath, f.NewPath)
	default:
		label = f.NewPath
	}
	return m.theme.file.Render(label)
}

func (m *Model) hunkLines(f *diff.FileDiff, h *diff.Hunk) []string {
	var lines []string

	header := fmt.Sprintf("@@ -%d,%d +%d,%d @@", h.OldStart, h.OldCount, h.NewStart, h.NewCount)
	if h.Header != "" {
		header += " " + h.Header
	}
	lines = append(lines, m.theme.hunk.Render(header))

	for _, line := range h.Lines {
		lines = append(lines, m.renderLine(f.Language, &line))
	}

	return lines
}

func (m *Model) renderLine(language string, line *diff.Line) string {
	var lineNumStr string

	if m.lineNums {
		switch line.Type {
		case diff.LineAdded:
			lineNumStr = fmt.Sprintf("   %4d ", line.NewNum)
		case diff.LineRemoved:
			lineNumStr = fmt.Sprintf("%4d    ", line.OldNum)
		case diff.LineContext:
			lineNumStr = fmt.Sprintf("%4d %4d ", line.OldNum, line.NewNum)
		}
	}

	numPart := m.theme.lineNum.Render(lineNumStr)

	switch line.Type {
	case diff.LineAdded:
		highlighted := m.highlightLine(language, line.Content, m.theme.addedBg)
		return numPart + m.theme.added.Render("+") + highlighted
	case diff.LineRemoved:
		highlighted := m.highlightLine(language, line.Content, m.theme.removedBg)
		return numPart + m.theme.removed.Render("-") + highlighted
	case diff.LineContext:
		highlighted := m.highlightLine(language, line.Content, "")
		return numPart + " " + highlighted
	}
	return ""
}

func (m *Model) highlightLine(language, content string, bg lipgloss.Color) string {
	if content == "" {
		if bg != "" {
			return lipgloss.NewStyle().Background(bg).Render(" ")
		}
		return ""
	}

	lexer := lexers.Get(language)
	if lexer == nil {
		lexer = lexers.Fallback
	}
	lexer = chroma.Coalesce(lexer)

	iterator, err := lexer.Tokenise(nil, content)
	if err != nil {
		if bg != "" {
			return lipgloss.NewStyle().Background(bg).Render(content)
		}
		return content
	}

	var result strings.Builder
	for _, token := range iterator.Tokens() {
		style := lipgloss.NewStyle()
		if bg != "" {
			style = style.Background(bg)
		}

		entry := m.theme.syntax.Get(token.Type)
		if entry.Colour.IsSet() {
			style = style.Foreground(lipgloss.Color(entry.Colour.String()))
		}
		if entry.Bold == chroma.Yes {
			style = style.Bold(true)
		}
		if entry.Italic == chroma.Yes {
			style = style.Italic(true)
		}

		result.WriteString(style.Render(token.Value))
	}
	return result.String()
}

func (m Model) View() string {
	if len(m.groups) == 0 {
		return "No changes to display"
	}

	var b strings.Builder

	contentHeight := m.contentHeight()
	endIdx := m.scrollOffset + contentHeight
	if endIdx > len(m.lines) {
		endIdx = len(m.lines)
	}

	visibleLines := m.lines[m.scrollOffset:endIdx]
	for _, line := range visibleLines {
		b.WriteString(line)
		b.WriteString("\n")
	}

	for i := len(visibleLines); i < contentHeight; i++ {
		b.WriteString("\n")
	}

	statusStyle := lipgloss.NewStyle().
		Background(lipgloss.Color("240")).
		Foreground(lipgloss.Color("252")).
		Padding(0, 1)

	progress := ""
	if len(m.lines) > contentHeight {
		pct := 0
		maxScroll := len(m.lines) - contentHeight
		if maxScroll > 0 {
			pct = (m.scrollOffset * 100) / maxScroll
		}
		progress = fmt.Sprintf(" %d%%", pct)
	}

	status := fmt.Sprintf("Group %d/%d%s │ ←/→: groups │ j/k: scroll │ space: page │ q: quit",
		m.groupIndex+1, len(m.groups), progress)
	b.WriteString(statusStyle.Render(status))

	return b.String()
}

func Run(groups []grouper.SemanticGroup, opts Options) error {
	p := tea.NewProgram(
		New(groups, opts),
		tea.WithAltScreen(),
	)
	_, err := p.Run()
	return err
}
