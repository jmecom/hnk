package main

import (
	"context"
	"fmt"
	"os"

	"github.com/jm/hnk/internal/ai"
	"github.com/jm/hnk/internal/cache"
	"github.com/jm/hnk/internal/config"
	"github.com/jm/hnk/internal/diff"
	"github.com/jm/hnk/internal/git"
	"github.com/jm/hnk/internal/grouper"
	"github.com/jm/hnk/internal/render"
	"github.com/jm/hnk/internal/tui"
	"github.com/urfave/cli/v3"
)

func main() {
	cfg := config.Load()

	app := &cli.Command{
		Name:      "hnk",
		Usage:     "Semantic git diff viewer - groups related hunks with explanations",
		ArgsUsage: "[commit] [-- paths...]",
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
				Value:   cfg.Model,
			},
			&cli.BoolFlag{
				Name:  "no-color",
				Usage: "Disable colored output",
			},
			&cli.BoolFlag{
				Name:    "light",
				Aliases: []string{"l"},
				Usage:   "Force light mode colors",
			},
			&cli.BoolFlag{
				Name:  "dark",
				Usage: "Force dark mode colors",
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
				Name:  "style",
				Usage: "Syntax highlighting style (monokai, dracula, github, etc.)",
			},
			&cli.BoolFlag{
				Name:    "tui",
				Aliases: []string{"i"},
				Usage:   "Interactive TUI mode with keyboard navigation",
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			return run(ctx, cmd, cfg)
		},
	}

	if err := app.Run(context.Background(), os.Args); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, cmd *cli.Command, cfg *config.Config) error {
	repo := git.NewRepository("")
	if !repo.IsRepo() {
		return fmt.Errorf("not a git repository")
	}

	var diffText string
	var err error

	args := cmd.Args().Slice()
	var commit string
	var paths []string

	if len(args) > 0 {
		if repo.IsValidRef(ctx, args[0]) {
			commit = args[0]
			paths = args[1:]
		} else {
			paths = args
		}
	}

	fromRef := cmd.String("from")
	toRef := cmd.String("to")
	ref := cmd.String("ref")

	switch {
	case commit != "":
		diffText, err = repo.GetCommitDiff(ctx, commit, paths...)
	case fromRef != "" && toRef != "":
		diffText, err = repo.GetDiffBetweenRefs(ctx, fromRef, toRef, paths...)
	case ref != "":
		if !repo.IsValidRef(ctx, ref) {
			return fmt.Errorf("invalid ref: %s", ref)
		}
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
	c := cache.New(cfg.CacheSizeBytes())
	grp := grouper.New(claudeAI, c)

	groups, err := grp.GroupDiff(ctx, parsed)
	if err != nil {
		return fmt.Errorf("failed to group changes: %w", err)
	}

	lightMode := resolveTheme(cfg.Theme, cmd.Bool("light"), cmd.Bool("dark"))

	lineNums := true
	if cfg.LineNumbers != nil {
		lineNums = *cfg.LineNumbers
	}
	if cmd.Bool("no-line-numbers") {
		lineNums = false
	}

	style := cfg.Style
	if cmd.String("style") != "" {
		style = cmd.String("style")
	}

	r := render.New(
		os.Stdout,
		render.WithColor(!cmd.Bool("no-color")),
		render.WithLight(lightMode),
		render.WithLineNumbers(lineNums),
		render.WithStyle(style),
	)

	if cmd.Bool("tui") {
		return tui.Run(groups, tui.Options{
			LightMode:   lightMode,
			LineNumbers: lineNums,
			StyleName:   style,
		})
	}

	if cmd.Bool("raw") {
		return r.RenderRaw(groups)
	}
	return r.RenderGroups(groups)
}

func resolveTheme(cfgTheme string, forceLight, forceDark bool) bool {
	if forceLight {
		return true
	}
	if forceDark {
		return false
	}

	switch cfgTheme {
	case "light":
		return true
	case "dark":
		return false
	default:
		return render.DetectLightMode()
	}
}
