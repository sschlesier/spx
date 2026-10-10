# Changelog

## 0.2.0

- Stop showing spec ids: drop the id column and id filtering, and no longer read `id:` from spec files
- Match `depends-on` and Blocks by slug instead of id

## 0.1.0

- Browse specs in a split view: a list beside the selected spec's rendered markdown
- Reload when the store changes
- Filter specs by status
- Fuzzy filter the list with `/`
- Switch between projects, picking one by typing in the popup
- Print specs by status from the command line (`-d`, `-a`, `-s`)
- Open the selected draft in an editor with `e`; copy its slug or path with `y`/`Y`
- Show dependencies in the detail pane
- Release via GitHub Releases and the Homebrew tap
