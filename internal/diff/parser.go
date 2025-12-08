package diff

import (
	"bufio"
	"regexp"
	"strconv"
	"strings"
)

type LineType int

const (
	LineContext LineType = iota
	LineAdded
	LineRemoved
	LineHeader
)

type Line struct {
	Type    LineType
	Content string
	OldNum  int
	NewNum  int
}

type Hunk struct {
	OldStart int
	OldCount int
	NewStart int
	NewCount int
	Lines    []Line
	Header   string
}

type FileDiff struct {
	OldPath   string
	NewPath   string
	IsNew     bool
	IsDeleted bool
	IsRenamed bool
	IsBinary  bool
	Language  string
	Hunks     []Hunk
}

type Diff struct {
	Files []FileDiff
}

var (
	diffHeaderRe = regexp.MustCompile(`^diff --git a/(.+) b/(.+)$`)
	oldFileRe    = regexp.MustCompile(`^--- (?:a/)?(.+)$`)
	newFileRe    = regexp.MustCompile(`^\+\+\+ (?:b/)?(.+)$`)
	hunkHeaderRe = regexp.MustCompile(`^@@ -(\d+)(?:,(\d+))? \+(\d+)(?:,(\d+))? @@(.*)$`)
)

var languageExtensions = map[string]string{
	".go":    "go",
	".js":    "javascript",
	".jsx":   "jsx",
	".ts":    "typescript",
	".tsx":   "tsx",
	".py":    "python",
	".rb":    "ruby",
	".rs":    "rust",
	".c":     "c",
	".cpp":   "cpp",
	".h":     "c",
	".hpp":   "cpp",
	".java":  "java",
	".kt":    "kotlin",
	".swift": "swift",
	".sh":    "bash",
	".bash":  "bash",
	".zsh":   "zsh",
	".fish":  "fish",
	".html":  "html",
	".css":   "css",
	".scss":  "scss",
	".less":  "less",
	".json":  "json",
	".yaml":  "yaml",
	".yml":   "yaml",
	".toml":  "toml",
	".xml":   "xml",
	".md":    "markdown",
	".sql":   "sql",
	".php":   "php",
	".lua":   "lua",
	".vim":   "vim",
	".el":    "emacs-lisp",
	".clj":   "clojure",
	".ex":    "elixir",
	".exs":   "elixir",
	".erl":   "erlang",
	".hs":    "haskell",
	".ml":    "ocaml",
	".scala": "scala",
	".r":     "r",
	".R":     "r",
	".pl":    "perl",
	".pm":    "perl",
	".cs":    "csharp",
	".fs":    "fsharp",
	".tf":    "terraform",
	".proto": "protobuf",
	".graphql": "graphql",
	".gql":    "graphql",
}

func detectLanguage(path string) string {
	for ext, lang := range languageExtensions {
		if strings.HasSuffix(path, ext) {
			return lang
		}
	}
	return "text"
}

func Parse(input string) (*Diff, error) {
	diff := &Diff{}
	scanner := bufio.NewScanner(strings.NewReader(input))

	var currentFile *FileDiff
	var currentHunk *Hunk
	oldLineNum, newLineNum := 0, 0

	for scanner.Scan() {
		line := scanner.Text()

		if matches := diffHeaderRe.FindStringSubmatch(line); matches != nil {
			if currentFile != nil {
				if currentHunk != nil {
					currentFile.Hunks = append(currentFile.Hunks, *currentHunk)
				}
				diff.Files = append(diff.Files, *currentFile)
			}
			currentFile = &FileDiff{
				OldPath:  matches[1],
				NewPath:  matches[2],
				Language: detectLanguage(matches[2]),
			}
			currentHunk = nil
			continue
		}

		if currentFile == nil {
			continue
		}

		if strings.HasPrefix(line, "new file mode") {
			currentFile.IsNew = true
			continue
		}
		if strings.HasPrefix(line, "deleted file mode") {
			currentFile.IsDeleted = true
			continue
		}
		if strings.HasPrefix(line, "rename from") || strings.HasPrefix(line, "rename to") {
			currentFile.IsRenamed = true
			continue
		}
		if strings.HasPrefix(line, "Binary files") {
			currentFile.IsBinary = true
			continue
		}
		if strings.HasPrefix(line, "index ") || strings.HasPrefix(line, "similarity index") {
			continue
		}

		if matches := oldFileRe.FindStringSubmatch(line); matches != nil {
			if matches[1] != "/dev/null" {
				currentFile.OldPath = matches[1]
			}
			continue
		}
		if matches := newFileRe.FindStringSubmatch(line); matches != nil {
			if matches[1] != "/dev/null" {
				currentFile.NewPath = matches[1]
			}
			continue
		}

		if matches := hunkHeaderRe.FindStringSubmatch(line); matches != nil {
			if currentHunk != nil {
				currentFile.Hunks = append(currentFile.Hunks, *currentHunk)
			}

			oldStart, _ := strconv.Atoi(matches[1])
			oldCount := 1
			if matches[2] != "" {
				oldCount, _ = strconv.Atoi(matches[2])
			}
			newStart, _ := strconv.Atoi(matches[3])
			newCount := 1
			if matches[4] != "" {
				newCount, _ = strconv.Atoi(matches[4])
			}

			currentHunk = &Hunk{
				OldStart: oldStart,
				OldCount: oldCount,
				NewStart: newStart,
				NewCount: newCount,
				Header:   strings.TrimSpace(matches[5]),
			}
			oldLineNum = oldStart
			newLineNum = newStart
			continue
		}

		if currentHunk != nil {
			var lineType LineType
			content := line

			switch {
			case strings.HasPrefix(line, "+"):
				lineType = LineAdded
				content = strings.TrimPrefix(line, "+")
				currentHunk.Lines = append(currentHunk.Lines, Line{
					Type:    lineType,
					Content: content,
					NewNum:  newLineNum,
				})
				newLineNum++
			case strings.HasPrefix(line, "-"):
				lineType = LineRemoved
				content = strings.TrimPrefix(line, "-")
				currentHunk.Lines = append(currentHunk.Lines, Line{
					Type:    lineType,
					Content: content,
					OldNum:  oldLineNum,
				})
				oldLineNum++
			case strings.HasPrefix(line, " ") || line == "":
				lineType = LineContext
				if strings.HasPrefix(line, " ") {
					content = strings.TrimPrefix(line, " ")
				}
				currentHunk.Lines = append(currentHunk.Lines, Line{
					Type:    lineType,
					Content: content,
					OldNum:  oldLineNum,
					NewNum:  newLineNum,
				})
				oldLineNum++
				newLineNum++
			}
		}
	}

	if currentFile != nil {
		if currentHunk != nil {
			currentFile.Hunks = append(currentFile.Hunks, *currentHunk)
		}
		diff.Files = append(diff.Files, *currentFile)
	}

	return diff, scanner.Err()
}

func (d *Diff) RawString() string {
	var sb strings.Builder
	for _, f := range d.Files {
		sb.WriteString("diff --git a/" + f.OldPath + " b/" + f.NewPath + "\n")
		for _, h := range f.Hunks {
			sb.WriteString("@@ -" + strconv.Itoa(h.OldStart) + "," + strconv.Itoa(h.OldCount) +
				" +" + strconv.Itoa(h.NewStart) + "," + strconv.Itoa(h.NewCount) + " @@")
			if h.Header != "" {
				sb.WriteString(" " + h.Header)
			}
			sb.WriteString("\n")
			for _, l := range h.Lines {
				switch l.Type {
				case LineAdded:
					sb.WriteString("+" + l.Content + "\n")
				case LineRemoved:
					sb.WriteString("-" + l.Content + "\n")
				case LineContext:
					sb.WriteString(" " + l.Content + "\n")
				}
			}
		}
	}
	return sb.String()
}
