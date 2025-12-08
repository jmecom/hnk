package render

import (
	"fmt"
	"io"
	"strings"

	"github.com/alecthomas/chroma/v2"
	"github.com/alecthomas/chroma/v2/formatters"
	"github.com/alecthomas/chroma/v2/lexers"
	"github.com/alecthomas/chroma/v2/styles"
	"github.com/jm/hnk/internal/diff"
	"github.com/jm/hnk/internal/grouper"
)

const (
	colorReset   = "\033[0m"
	colorBold    = "\033[1m"
	colorDim     = "\033[2m"
	colorMagenta = "\033[35m"
)

type theme struct {
	added      string
	removed    string
	title      string
	desc       string
	file       string
	lineNum    string
	chromaStyle string
}

var darkTheme = theme{
	added:       "\033[48;5;22m",
	removed:     "\033[48;5;52m",
	title:       "\033[1m\033[36m",
	desc:        "\033[2m",
	file:        "\033[1m\033[34m",
	lineNum:     "\033[2m",
	chromaStyle: "monokai",
}

var lightTheme = theme{
	added:       "\033[48;5;194m",
	removed:     "\033[48;5;224m",
	title:       "\033[1m\033[34m",
	desc:        "\033[90m",
	file:        "\033[1m\033[35m",
	lineNum:     "\033[90m",
	chromaStyle: "github",
}

type Renderer struct {
	out         io.Writer
	useColor    bool
	style       *chroma.Style
	lineNums    bool
	compactMode bool
	theme       theme
}

type Option func(*Renderer)

func WithColor(enabled bool) Option {
	return func(r *Renderer) {
		r.useColor = enabled
	}
}

func WithLineNumbers(enabled bool) Option {
	return func(r *Renderer) {
		r.lineNums = enabled
	}
}

func WithCompact(enabled bool) Option {
	return func(r *Renderer) {
		r.compactMode = enabled
	}
}

func WithStyle(styleName string) Option {
	return func(r *Renderer) {
		if styleName == "" {
			return
		}
		if s := styles.Get(styleName); s != nil {
			r.style = s
		}
	}
}

func WithLight(enabled bool) Option {
	return func(r *Renderer) {
		if enabled {
			r.theme = lightTheme
			r.style = styles.Get(lightTheme.chromaStyle)
		}
	}
}

func New(out io.Writer, opts ...Option) *Renderer {
	r := &Renderer{
		out:      out,
		useColor: true,
		style:    styles.Get(darkTheme.chromaStyle),
		lineNums: true,
		theme:    darkTheme,
	}
	for _, opt := range opts {
		opt(r)
	}
	return r
}

func (r *Renderer) RenderGroups(groups []grouper.SemanticGroup) error {
	for i, group := range groups {
		if i > 0 {
			r.writeDivider()
		}
		if err := r.renderGroup(&group); err != nil {
			return err
		}
	}
	return nil
}

func (r *Renderer) renderGroup(group *grouper.SemanticGroup) error {
	r.writeGroupHeader(group.Title, group.Description)

	for _, gh := range group.Hunks {
		r.writeFileHeader(gh.File)
		r.renderHunk(gh.File, gh.Hunk)
	}

	return nil
}

func (r *Renderer) writeGroupHeader(title, description string) {
	if r.useColor {
		fmt.Fprintf(r.out, "\n%s%s%s\n", r.theme.title, title, colorReset)
		fmt.Fprintf(r.out, "%s%s%s\n\n", r.theme.desc, description, colorReset)
	} else {
		fmt.Fprintf(r.out, "\n%s\n", title)
		fmt.Fprintf(r.out, "%s\n\n", description)
	}
}

func (r *Renderer) writeFileHeader(f *diff.FileDiff) {
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

	if r.useColor {
		fmt.Fprintf(r.out, "%s%s%s\n", r.theme.file, label, colorReset)
	} else {
		fmt.Fprintf(r.out, "%s\n", label)
	}
}

func (r *Renderer) renderHunk(f *diff.FileDiff, h *diff.Hunk) {
	if r.useColor {
		header := fmt.Sprintf("@@ -%d,%d +%d,%d @@", h.OldStart, h.OldCount, h.NewStart, h.NewCount)
		if h.Header != "" {
			header += " " + h.Header
		}
		fmt.Fprintf(r.out, "%s%s%s\n", colorMagenta, header, colorReset)
	} else {
		fmt.Fprintf(r.out, "@@ -%d,%d +%d,%d @@", h.OldStart, h.OldCount, h.NewStart, h.NewCount)
		if h.Header != "" {
			fmt.Fprintf(r.out, " %s", h.Header)
		}
		fmt.Fprintln(r.out)
	}

	for _, line := range h.Lines {
		r.renderLine(f.Language, &line)
	}
	fmt.Fprintln(r.out)
}

func (r *Renderer) renderLine(language string, line *diff.Line) {
	var prefix string
	var lineNumStr string

	if r.lineNums {
		switch line.Type {
		case diff.LineAdded:
			lineNumStr = fmt.Sprintf("   %4d ", line.NewNum)
		case diff.LineRemoved:
			lineNumStr = fmt.Sprintf("%4d    ", line.OldNum)
		case diff.LineContext:
			lineNumStr = fmt.Sprintf("%4d %4d ", line.OldNum, line.NewNum)
		}
	}

	switch line.Type {
	case diff.LineAdded:
		prefix = "+"
		if r.useColor {
			highlighted := r.highlightWithBg(language, line.Content, r.theme.added)
			fmt.Fprintf(r.out, "%s%s%s%s%s%s%s\n",
				r.theme.lineNum, lineNumStr, colorReset,
				r.theme.added, prefix, highlighted, colorReset)
		} else {
			fmt.Fprintf(r.out, "%s%s%s\n", lineNumStr, prefix, line.Content)
		}
	case diff.LineRemoved:
		prefix = "-"
		if r.useColor {
			highlighted := r.highlightWithBg(language, line.Content, r.theme.removed)
			fmt.Fprintf(r.out, "%s%s%s%s%s%s%s\n",
				r.theme.lineNum, lineNumStr, colorReset,
				r.theme.removed, prefix, highlighted, colorReset)
		} else {
			fmt.Fprintf(r.out, "%s%s%s\n", lineNumStr, prefix, line.Content)
		}
	case diff.LineContext:
		prefix = " "
		if r.useColor {
			highlighted := r.highlightContent(language, line.Content)
			fmt.Fprintf(r.out, "%s%s%s%s%s\n",
				r.theme.lineNum, lineNumStr, colorReset,
				prefix, highlighted)
		} else {
			fmt.Fprintf(r.out, "%s%s%s\n", lineNumStr, prefix, line.Content)
		}
	}
}

func (r *Renderer) highlightWithBg(language, content, bg string) string {
	if !r.useColor || content == "" {
		return content
	}

	highlighted := r.highlightContent(language, content)
	highlighted = strings.ReplaceAll(highlighted, colorReset, colorReset+bg)
	return highlighted
}

func (r *Renderer) highlightContent(language, content string) string {
	if !r.useColor || content == "" {
		return content
	}

	lexer := lexers.Get(language)
	if lexer == nil {
		lexer = lexers.Fallback
	}
	lexer = chroma.Coalesce(lexer)

	iterator, err := lexer.Tokenise(nil, content)
	if err != nil {
		return content
	}

	var buf strings.Builder
	formatter := formatters.Get("terminal256")
	if formatter == nil {
		formatter = formatters.Fallback
	}

	err = formatter.Format(&buf, r.style, iterator)
	if err != nil {
		return content
	}

	result := buf.String()
	result = strings.TrimSuffix(result, "\n")
	return result
}

func (r *Renderer) writeDivider() {
	if r.useColor {
		fmt.Fprintf(r.out, "\n%s%s%s\n", r.theme.lineNum, strings.Repeat("─", 80), colorReset)
	} else {
		fmt.Fprintf(r.out, "\n%s\n", strings.Repeat("-", 80))
	}
}

func (r *Renderer) RenderRaw(groups []grouper.SemanticGroup) error {
	for i, group := range groups {
		if i > 0 {
			fmt.Fprintln(r.out, "---")
			fmt.Fprintln(r.out)
		}
		fmt.Fprintf(r.out, "# %s\n\n", group.Title)
		fmt.Fprintf(r.out, "%s\n\n", group.Description)
		for _, gh := range group.Hunks {
			fmt.Fprintf(r.out, "diff --git a/%s b/%s\n", gh.File.OldPath, gh.File.NewPath)
			fmt.Fprintf(r.out, "@@ -%d,%d +%d,%d @@",
				gh.Hunk.OldStart, gh.Hunk.OldCount,
				gh.Hunk.NewStart, gh.Hunk.NewCount)
			if gh.Hunk.Header != "" {
				fmt.Fprintf(r.out, " %s", gh.Hunk.Header)
			}
			fmt.Fprintln(r.out)
			for _, line := range gh.Hunk.Lines {
				switch line.Type {
				case diff.LineAdded:
					fmt.Fprintf(r.out, "+%s\n", line.Content)
				case diff.LineRemoved:
					fmt.Fprintf(r.out, "-%s\n", line.Content)
				case diff.LineContext:
					fmt.Fprintf(r.out, " %s\n", line.Content)
				}
			}
		}
	}
	return nil
}
