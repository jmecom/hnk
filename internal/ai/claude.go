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

func (c *ClaudeCLI) AnalyzeDiff(ctx context.Context, diffText string) (*SemanticAnalysis, error) {
	prompt := buildAnalysisPrompt(diffText)

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

func buildAnalysisPrompt(diffText string) string {
	return fmt.Sprintf(`Analyze this git diff and group related changes semantically. Return ONLY valid JSON with no additional text.

For each logical group of changes:
1. Provide a short title (like a commit message subject line)
2. Write a brief description explaining what the changes accomplish
3. List which files and hunks belong to this group

The JSON format must be:
{
  "groups": [
    {
      "title": "Short descriptive title",
      "description": "Brief explanation of what these changes do and why",
      "file_indices": [0, 1],
      "hunk_indices": [[0, 1], [0]]
    }
  ]
}

file_indices is a list of 0-based file indices.
hunk_indices is a list of lists - for each file in file_indices, which hunks (0-based) belong to this group.

Group changes that:
- Implement a single feature or fix
- Refactor related code together
- Update configuration or dependencies together
- Modify tests for the same functionality

DIFF:
%s

Return ONLY the JSON, no markdown fences, no explanation.`, diffText)
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
