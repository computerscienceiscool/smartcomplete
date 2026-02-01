# Repository Guidelines

Contributor guide for smartcomplete (Neovim + Automerge bridge). Keep changes small, documented, and consistent with the completion workflow described in `spec.md`.

## Project Structure & Module Organization
- `lua/vimbeam/` contains Neovim plugin Lua modules; `plugin/` holds the Vim loader script.
- `node-helper/` contains the Node.js bridge to Automerge for document sync.
- `docs/` stores specs and protocol references; keep it authoritative for interface changes.
- `TODO/` tracks work; `TODO/TODO.md` is the priority-sorted index.
- Local state like `.grok/` and generated binaries are ignored—never commit them.

## Build, Test, and Development Commands
- `cd node-helper && npm install` to install Node.js dependencies.
- Test the plugin by loading it in Neovim (via a plugin manager or manual `runtimepath` inclusion).
- Node helper smoke test: `node node-helper/index.js`.
- Add per-task helper scripts under `scripts/` with a one-line usage note when needed.

## Coding Style & Naming Conventions
- Lua follows standard Neovim plugin conventions; prefer `vim.api.*` and `vim.fn.*` for editor access.
- Node.js uses ES modules (`"type": "module"` in `package.json`); keep imports/exports idiomatic.
- Prefer small, focused edits; avoid rearranging files without a clear need. Use `git mv` for renames to preserve history.
- Keep names descriptive (e.g., `completion.lua`, `sync_bridge.js`); avoid stutter.

## Testing Guidelines
- Validate plugin behavior manually in Neovim with a running sync server; cover hotkeys and buffer sync paths.
- Keep tests deterministic; mock external dependencies (Automerge, network) where possible.
- When adding new flows, include a minimal smoke script or checklist to reproduce.

## TODO Tracking
- Maintain `TODO/` with an index at `TODO/TODO.md`; number items with zero-padded 3 digits (e.g., 005) and do not renumber.
- Subtasks in `TODO/*` use numbered checkboxes (e.g., `- [ ] 005.1 describe subtask`).
- When completing a TODO, check it off in `TODO/TODO.md` (e.g., `- [x] 005 - ...`) and keep history intact.

## Commit & Pull Request Guidelines
- Commit subjects are short, imperative, and capitalized (e.g., "Fix cursor sync"); bodies summarize the diff with a bullet section per file.
- Stage files explicitly (avoid `git add .` / `git add -A`); run relevant format/tests before committing.
- PRs include a concise summary, test commands run, linked issues, and before/after notes when behavior changes.
- A line containing only `commit` means: add and commit all changes with a compliant message.

## Agent-Specific Instructions (Codex CLI)
- Periodically check `~/.codex/AGENTS.md` for updated local conventions.
- Use `notify-send` when requesting user attention and when a task is complete.
