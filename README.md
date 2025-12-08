# hnk

> **Note:** This code is 100% vibe coded.

A semantic git diff viewer that groups related changes (hunks) and explains what they do.

## What it does

Instead of showing raw diffs, `hnk` uses Claude to analyze changes and present them as logical groups with explanations.

## Install

```bash
go install github.com/jm/hnk/cmd/hnk@latest
```

Or build from source:

```bash
git clone https://github.com/jm/hnk
cd hnk
go install ./cmd/hnk
```

Requires the [Claude CLI](https://github.com/anthropics/claude-code) to be installed and authenticated.

## Usage

```bash
hnk                     # unstaged changes
hnk -s                  # staged changes
hnk HEAD~1              # show a specific commit
hnk main                # compare against a branch
hnk --from HEAD~5 --to HEAD   # range
```

### Flags

```
--staged, -s       staged changes only
--ref, -r          compare against ref
--from / --to      range comparison
--model, -m        claude model (haiku, sonnet, opus)
--light, -l        force light mode
--dark             force dark mode
--no-color         disable colors
--no-line-numbers  hide line numbers
--raw              plain output
--style            syntax theme (monokai, dracula, github, etc)
```

## Config

Optional `~/.hnk` file:

```json
{
  "theme": "auto",
  "model": "sonnet",
  "style": "monokai",
  "line_numbers": true
}
```

Theme can be `auto` (detects macOS appearance), `light`, or `dark`.

## Features

- Semantic grouping of related changes
- Syntax highlighting (50+ languages)
- Auto-detects macOS light/dark mode
- Line numbers with old/new file positions
