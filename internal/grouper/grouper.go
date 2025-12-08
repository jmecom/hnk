package grouper

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/jm/hnk/internal/ai"
	"github.com/jm/hnk/internal/diff"
	"github.com/jm/hnk/internal/spinner"
)

type GroupedHunk struct {
	File *diff.FileDiff
	Hunk *diff.Hunk
}

type SemanticGroup struct {
	Title       string
	Description string
	Hunks       []GroupedHunk
}

type Grouper struct {
	ai         *ai.ClaudeCLI
	spinnerOut io.Writer
}

func New(ai *ai.ClaudeCLI) *Grouper {
	return &Grouper{ai: ai, spinnerOut: os.Stderr}
}

func (g *Grouper) SetSpinnerOutput(w io.Writer) {
	g.spinnerOut = w
}

func (g *Grouper) GroupDiff(ctx context.Context, d *diff.Diff) ([]SemanticGroup, error) {
	if len(d.Files) == 0 {
		return nil, nil
	}

	totalHunks := 0
	for _, f := range d.Files {
		totalHunks += len(f.Hunks)
	}
	if totalHunks == 0 {
		return nil, nil
	}

	if totalHunks == 1 && len(d.Files) == 1 {
		return g.singleHunkGroup(ctx, d)
	}

	spin := spinner.New(g.spinnerOut, "Analyzing changes...")
	spin.Start()
	analysis, err := g.ai.AnalyzeDiff(ctx, d.RawString())
	spin.Stop()

	if err != nil {
		return g.fallbackGrouping(d), nil
	}

	return g.buildGroups(d, analysis), nil
}

func (g *Grouper) singleHunkGroup(ctx context.Context, d *diff.Diff) ([]SemanticGroup, error) {
	file := d.Files[0]
	hunk := file.Hunks[0]

	spin := spinner.New(g.spinnerOut, "Analyzing changes...")
	spin.Start()
	desc, err := g.ai.GenerateDescription(ctx, d.RawString())
	spin.Stop()

	if err != nil {
		desc = fmt.Sprintf("Changes to %s", file.NewPath)
	}

	title := generateTitle(&file, &hunk)

	return []SemanticGroup{{
		Title:       title,
		Description: desc,
		Hunks:       []GroupedHunk{{File: &file, Hunk: &hunk}},
	}}, nil
}

func generateTitle(f *diff.FileDiff, h *diff.Hunk) string {
	if f.IsNew {
		return fmt.Sprintf("Add %s", f.NewPath)
	}
	if f.IsDeleted {
		return fmt.Sprintf("Remove %s", f.OldPath)
	}
	if f.IsRenamed {
		return fmt.Sprintf("Rename %s to %s", f.OldPath, f.NewPath)
	}
	if h.Header != "" {
		return fmt.Sprintf("Update %s in %s", h.Header, f.NewPath)
	}
	return fmt.Sprintf("Modify %s", f.NewPath)
}

func (g *Grouper) fallbackGrouping(d *diff.Diff) []SemanticGroup {
	var groups []SemanticGroup

	for i := range d.Files {
		file := &d.Files[i]
		for j := range file.Hunks {
			hunk := &file.Hunks[j]
			title := generateTitle(file, hunk)
			groups = append(groups, SemanticGroup{
				Title:       title,
				Description: fmt.Sprintf("Changes to %s around line %d", file.NewPath, hunk.NewStart),
				Hunks:       []GroupedHunk{{File: file, Hunk: hunk}},
			})
		}
	}

	return groups
}

func (g *Grouper) buildGroups(d *diff.Diff, analysis *ai.SemanticAnalysis) []SemanticGroup {
	var groups []SemanticGroup
	usedHunks := make(map[string]bool)

	for _, ag := range analysis.Groups {
		group := SemanticGroup{
			Title:       ag.Title,
			Description: ag.Description,
		}

		for i, fileIdx := range ag.FileIndices {
			if fileIdx < 0 || fileIdx >= len(d.Files) {
				continue
			}
			file := &d.Files[fileIdx]

			var hunkIndices []int
			if i < len(ag.HunkIndices) {
				hunkIndices = ag.HunkIndices[i]
			} else {
				for j := range file.Hunks {
					hunkIndices = append(hunkIndices, j)
				}
			}

			for _, hunkIdx := range hunkIndices {
				if hunkIdx < 0 || hunkIdx >= len(file.Hunks) {
					continue
				}
				key := fmt.Sprintf("%d-%d", fileIdx, hunkIdx)
				if usedHunks[key] {
					continue
				}
				usedHunks[key] = true
				group.Hunks = append(group.Hunks, GroupedHunk{
					File: file,
					Hunk: &file.Hunks[hunkIdx],
				})
			}
		}

		if len(group.Hunks) > 0 {
			groups = append(groups, group)
		}
	}

	for fileIdx, file := range d.Files {
		for hunkIdx := range file.Hunks {
			key := fmt.Sprintf("%d-%d", fileIdx, hunkIdx)
			if usedHunks[key] {
				continue
			}
			usedHunks[key] = true
			f := &d.Files[fileIdx]
			h := &f.Hunks[hunkIdx]
			groups = append(groups, SemanticGroup{
				Title:       generateTitle(f, h),
				Description: fmt.Sprintf("Additional changes to %s", f.NewPath),
				Hunks:       []GroupedHunk{{File: f, Hunk: h}},
			})
		}
	}

	return groups
}
