package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

type SemanticGroup struct {
	Title       string   `json:"title"`
	Description string   `json:"description"`
	FileIndices []int    `json:"file_indices"`
	HunkIndices [][]int  `json:"hunk_indices"`
}

type SemanticAnalysis struct {
	Groups []SemanticGroup `json:"groups"`
}

type ClaudeCLI struct {
	Model   string
	Timeout time.Duration
}

func NewClaudeCLI(model string) *ClaudeCLI {
	if model == "" {
		model = "sonnet"
	}
	return &ClaudeCLI{
		Model:   model,
		Timeout: 120 * time.Second,
	}
}

func (c *ClaudeCLI) AnalyzeDiff(ctx context.Context, catalog *DiffCatalog, rawDiff string) (*SemanticAnalysis, error) {
	prompt := buildAnalysisPrompt(catalog, rawDiff)

	ctx, cancel := context.WithTimeout(ctx, c.Timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "claude", "--model", c.Model, "--print")
	cmd.Stdin = strings.NewReader(prompt)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("claude: %w\n%s", err, stderr.String())
	}

	response := stdout.String()
	return parseAnalysisResponse(response)
}

type DiffCatalog struct {
	Files      []FileCatalog
	TotalHunks int
}

type FileCatalog struct {
	Index    int
	Path     string
	IsNew    bool
	IsDelete bool
	Hunks    []HunkCatalog
}

type HunkCatalog struct {
	Index   int
	Start   int
	End     int
	Header  string
	Adds    int
	Removes int
}

func buildAnalysisPrompt(catalog *DiffCatalog, rawDiff string) string {
	var sb strings.Builder

	sb.WriteString("# Diff Catalog\n\n")
	for _, f := range catalog.Files {
		status := ""
		if f.IsNew {
			status = " (new file)"
		} else if f.IsDelete {
			status = " (deleted)"
		}
		sb.WriteString(fmt.Sprintf("File[%d]: %s%s\n", f.Index, f.Path, status))
		for _, h := range f.Hunks {
			header := ""
			if h.Header != "" {
				header = fmt.Sprintf(" // %s", h.Header)
			}
			sb.WriteString(fmt.Sprintf("  Hunk[%d]: lines %d-%d (+%d/-%d)%s\n",
				h.Index, h.Start, h.End, h.Adds, h.Removes, header))
		}
	}

	maxGroups := min(len(catalog.Files)+1, 4)

	return fmt.Sprintf(`%s
# Diff Content

%s

# Instructions

Group these hunks into logical changes. Return ONLY valid JSON.

RULES:
- Create AT MOST %d groups (fewer is better, 1-2 is ideal)
- Hunks from the same file should be in the same group unless they do completely different things
- Each hunk must appear in EXACTLY ONE group (no duplicates)
- You MUST specify explicit hunk_indices for every file - never omit them
- Title should be imperative mood, <60 chars

JSON format:
{
  "groups": [
    {
      "title": "Add user authentication",
      "description": "One sentence explaining what and why",
      "file_indices": [0, 1],
      "hunk_indices": [[0, 1], [0]]
    }
  ]
}

file_indices: which files (by index)
hunk_indices: REQUIRED - for each file in file_indices, list its hunk indices

Return ONLY JSON, no markdown fences.`, sb.String(), rawDiff, maxGroups)
}

func BuildCatalog(files []FileInfo) *DiffCatalog {
	catalog := &DiffCatalog{}
	for i, f := range files {
		fc := FileCatalog{
			Index:    i,
			Path:     f.Path,
			IsNew:    f.IsNew,
			IsDelete: f.IsDeleted,
		}
		for j, h := range f.Hunks {
			hc := HunkCatalog{
				Index:   j,
				Start:   h.Start,
				End:     h.Start + h.Count,
				Header:  h.Header,
				Adds:    h.Adds,
				Removes: h.Removes,
			}
			fc.Hunks = append(fc.Hunks, hc)
			catalog.TotalHunks++
		}
		catalog.Files = append(catalog.Files, fc)
	}
	return catalog
}

type FileInfo struct {
	Path      string
	IsNew     bool
	IsDeleted bool
	Hunks     []HunkInfo
}

type HunkInfo struct {
	Start   int
	Count   int
	Header  string
	Adds    int
	Removes int
}

func parseAnalysisResponse(response string) (*SemanticAnalysis, error) {
	response = strings.TrimSpace(response)
	response = strings.TrimPrefix(response, "```json")
	response = strings.TrimPrefix(response, "```")
	response = strings.TrimSuffix(response, "```")
	response = strings.TrimSpace(response)

	var analysis SemanticAnalysis
	if err := json.Unmarshal([]byte(response), &analysis); err != nil {
		return nil, fmt.Errorf("failed to parse AI response: %w\nresponse was: %s", err, response)
	}

	return &analysis, nil
}

func (c *ClaudeCLI) GenerateDescription(ctx context.Context, diffText string) (string, error) {
	prompt := fmt.Sprintf(`Describe this code change in 1-2 sentences. Be specific about what changed and why it matters.

DIFF:
%s

Return only the description, no formatting.`, diffText)

	ctx, cancel := context.WithTimeout(ctx, c.Timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "claude", "--model", c.Model, "--print")
	cmd.Stdin = strings.NewReader(prompt)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("claude: %w\n%s", err, stderr.String())
	}

	return strings.TrimSpace(stdout.String()), nil
}
