---
title: Show child specs under their parent in a tree mode
type: feature
priority: 3
depends-on: []
parent:
approved: "Scott Schlesier, 2026-10-12: Tree mode grouped by parent:, reviewed against the code. Cold read: not run"
status: in-review
---

`t` toggles a tree mode in the list that shows each spec indented under the spec named by its `parent:` field, so an epic and its children read as one group.

Context: the store format gives children a `parent:` field (an epic's slug, or `<project>/<slug>` across projects), but spx ignores it: `store.Spec` has no parent, and the list is flat, sorted by status, priority, project and slug (`store.Sort`), so a parent and its children land far apart. The detail pane already shows `depends-on` and what a spec blocks (`ui/deps.go`); this is about grouping in the list, not about dependencies. `t` is unused today. "Stories" in the request is read as specs, the store's only unit of work.

Out of scope:

- Grouping by `depends-on`. The detail pane covers it (see Design).
- Collapsing or expanding a branch, or showing child counts or progress on a parent.
- A CLI flag or a change to what `-d`, `-a` and `-s` print.
- Remembering the mode between runs.
- Any write to the store. spx stays read-only in the UI.

## Acceptance criteria

- [ ] `t` in the list toggles tree mode; the list starts with it off, and with it off the list is exactly as today. `t` does nothing while the query input or project popup has the keys.
- [ ] In tree mode a spec whose `parent:` resolves to a spec in the visible rows appears directly under that parent, indented two cells per level, with a branch glyph (`├─` for a non-last child, `└─` for the last) and a `│ ` guide in the column of each ancestor that has more children below (a blank column for a last one). Children keep the order they have in the flat list. Nesting works to any depth.
- [ ] A spec with no `parent:`, or whose parent is not in the visible rows (another status or project filtered out, hidden by the query, dropped, in `done/`, or missing), stays at the top level without indent, so nothing disappears.
- [ ] `parent:` is a bare slug, naming a spec in the child's own project, or `<project>/<slug>`, naming one in that project. (`depends-on` resolves within the project only; the cross-project form is new here, as the store format defines it for `parent`.)
- [ ] Each spec is shown once. A parent cycle, or a spec naming itself, shows its members at the top level.
- [ ] The project scope, the `d`/`a`/`s`/`x` status filter and the `/` query apply as in the flat list; tree mode only re-orders and indents the rows they leave. Under a query, a child whose parent doesn't match is a top-level row.
- [ ] Moving the cursor, `enter`, the detail pane, `e`, `y`/`Y` and `]`/`[` act on the selected spec as in the flat list, whatever its indent.
- [ ] Toggling `t` and reloading the store keep the same spec selected, by project and slug.
- [ ] Rows stay aligned: the indent and glyph sit at the start of the title column, and the other columns are unchanged. A row too long for the pane is cut with `…` as today.
- [ ] The footer help lists `t tree` in the list view, and while tree mode is on the footer's lead segment shows `tree`.

## Verification

- `go vet ./... && go test -race ./...` pass.
- New `store` test: `parent:` is read as a slug, as `<project>/<slug>`, and missing or empty gives no parent.
- New `ui` tests with the existing model helpers: toggle on and off (flat list identical to before), child under parent with glyphs and indent, grandchild, orphan parent in each of the six ways listed above, cross-project parent, cycle and self-parent, scope/filter/query applied, cursor and selection kept across toggle and reload, row alignment, `t` ignored while typing or picking.
- Manual: `go run .` against a store with an epic and children (a scratch `AGENT_SPECS_DIR` is enough); press `t`, move through the group, `enter` on a child, `/` a child's title, `d` then `a`, `t` again. Expected as in the criteria; `git -C ~/src/specs status` stays clean.

## Design

Decisions (defaults; overrule any):

- **"Parent" is the `parent:` field,** not `depends-on`. The store defines `parent` as the epic a child belongs to, which is the grouping the request describes. `depends-on` is many-to-many, so a spec can have several blockers and would need to appear under each or pick one; the detail pane already shows both directions.
- **`t` toggles a mode on the same list,** drawn from the same rows. The tree is a re-ordering of `Model.visible()` in `ui/ui.go`: roots in flat order, each followed by its descendants in flat order. `m.specs` holds the tree order, so cursor, detail, editor, copy and reload logic are reused; the indent and glyph are added where `Rows` builds the title column, from a depth passed in by the list view. `Rows` as the CLI uses it is unchanged.
- **An unresolved parent means a top-level row.** A tree that hides specs whose parent is filtered out would be worse than a flat list.
- **Always fully expanded.** No collapse state to hold or lose on reload.
- **`store.Spec` gains `Parent string`,** the field's text as written (empty when missing), read in `store.read` next to `depends-on`. `store/` stays read-only. The UI resolves it against `m.all`, which includes dropped specs, so the rule for "parent not visible" is one check against the visible rows.
- **Epic parents live in `draft/`,** since epics are never approved. A child in `approved/` or `started/` with a `draft` parent, and a `-d` filter that shows the parent but not the child, are both normal and covered by the orphan rule.

No flags (data migration, public API, config).

## Boundaries

Stop and ask if: tree mode would need a change to what is listed, to `store.Sort`, or to the CLI's output.
Don't touch: any write path, the flat list's behaviour with `t` off.

## Log

- 2026-10-12: Drafted from the request for a tree mode with dependent stories under their parents. Read "stories" as specs and "parents" as the `parent:` field; grouping by `depends-on` left out and noted in Design.
- 2026-10-12: Review: corrected the parent-resolution criterion (depends-on is same-project only) and fixed the tree-mode indicator to the footer lead.
- 2026-10-12: Approved: Scott Schlesier, 2026-10-12: Tree mode grouped by parent:, reviewed against the code. Cold read: not run
- 2026-10-12: Started on branch show-children-in-tree-mode
- 2026-10-12: Assumption: the narrow-layout footer test moved from width 90 to 99, since the added `t tree` hint pushes `e edit · y/Y copy` out of the footer at 90.
- 2026-10-12: Review started
- 2026-10-12: Dismissed: the `seen` guard in `walk` is unreachable after cycle removal and kept as a defensive guard; the duplicate project/slug guard is an equivalent mutant (the store holds no duplicates).
- 2026-10-12: Needs fixes (round 1): 1. Draw `│ ` guides under non-last ancestors (Scott Schlesier chose this over keeping bare indent; criterion 2 changed to say so). 2. Run the manual step in a real terminal.
- 2026-10-12: Round 1 fixes: 1.1 guides added and tested; 1.2 manual run done in tmux against a scratch store (tree, grandchild guides, cross-project parent, `/` on a child, `d` then `a`, `t` off): rendering and footer as the criteria say; `git -C ~/src/specs status` clean.
