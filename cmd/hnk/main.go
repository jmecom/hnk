package main

import (
	"context"
	"fmt"
	"os"

	"github.com/jm/hnk/internal/ai"
	"github.com/jm/hnk/internal/diff"
	"github.com/jm/hnk/internal/git"
	"github.com/jm/hnk/internal/grouper"
	"github.com/jm/hnk/internal/render"
	"github.com/urfave/cli/v3"
)

func main() {
	app := &cli.Command{
		Name:  "hnk",
		Usage: "Semantic git diff viewer - groups related hunks with explanations",
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:    "staged",
				Aliases: []string{"s"},
				Usage:   "Show staged changes only",
			},
			&cli.StringFlag{
				Name:    "ref",
				Aliases: []string{"r"},
				Usage:   "Compare against a specific ref (branch, tag, commit)",
			},
			&cli.StringFlag{
				Name:  "from",
				Usage: "Start ref for range comparison (use with --to)",
			},
			&cli.StringFlag{
				Name:  "to",
				Usage: "End ref for range comparison (use with --from)",
			},
			&cli.StringFlag{
				Name:    "model",
				Aliases: []string{"m"},
				Usage:   "Claude model to use (haiku, sonnet, opus)",
				Value:   "sonnet",
			},
			&cli.BoolFlag{
				Name:  "no-color",
				Usage: "Disable colored output",
			},
			&cli.BoolFlag{
				Name:  "no-line-numbers",
				Usage: "Hide line numbers",
			},
			&cli.BoolFlag{
				Name:  "raw",
				Usage: "Output raw grouped diff without styling",
			},
			&cli.StringFlag{
				Name:    "style",
				Usage:   "Syntax highlighting style (monokai, dracula, github, etc.)",
				Value:   "monokai",
			},
		},
		Action: run,
	}

	if err := app.Run(context.Background(), os.Args); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, cmd *cli.Command) error {
	repo := git.NewRepository("")
	if !repo.IsRepo() {
		return fmt.Errorf("not a git repository")
	}

	var diffText string
	var err error

	paths := cmd.Args().Slice()

	fromRef := cmd.String("from")
	toRef := cmd.String("to")
	ref := cmd.String("ref")

	switch {
	case fromRef != "" && toRef != "":
		diffText, err = repo.GetDiffBetweenRefs(ctx, fromRef, toRef, paths...)
	case ref != "":
		diffText, err = repo.GetDiffAgainstRef(ctx, ref, paths...)
	default:
		diffText, err = repo.GetDiff(ctx, cmd.Bool("staged"), paths...)
	}

	if err != nil {
		return fmt.Errorf("failed to get diff: %w", err)
	}

	if diffText == "" {
		fmt.Println("No changes to display")
		return nil
	}

	parsed, err := diff.Parse(diffText)
	if err != nil {
		return fmt.Errorf("failed to parse diff: %w", err)
	}

	if len(parsed.Files) == 0 {
		fmt.Println("No changes to display")
		return nil
	}

	claudeAI := ai.NewClaudeCLI(cmd.String("model"))
	grp := grouper.New(claudeAI)

	groups, err := grp.GroupDiff(ctx, parsed)
	if err != nil {
		return fmt.Errorf("failed to group changes: %w", err)
	}

	r := render.New(
		os.Stdout,
		render.WithColor(!cmd.Bool("no-color")),
		render.WithLineNumbers(!cmd.Bool("no-line-numbers")),
		render.WithStyle(cmd.String("style")),
	)

	if cmd.Bool("raw") {
		return r.RenderRaw(groups)
	}
	return r.RenderGroups(groups)
}
