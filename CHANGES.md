# Changelog

## 1.0.0

- Add a board view of draft, approved and started specs
- Add `c` to advance the selected spec with Claude, in the project's repo
- Keep an agent instruction as the first line of every spec's frontmatter
- Shorten the `spx help --skill` guide by a quarter

## 0.3.0

- Add spec lifecycle commands: `spx new`, `edit`, `approve`, `send-back`, `drop`, `log`, `place`, `accept`, `close`, `start`, `amend`, `sweep`
- Add `spx repo`, `status` and `check` commands, and map each project to its repo
- Serialize store writes with a store write lock
- Fix `repo ..` handling
- Update `spx help --skill` to document the full store guide
- Show children in tree mode
- Bump golang.org/x/text and goldmark to fix known vulnerabilities

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
