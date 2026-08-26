<p align="center"><a href="README.md">简体中文</a> · <b>English</b></p>

# KnowTree · 知树

A single-user **knowledge point management desktop app**: tree-structured knowledge points from preschool to PhD, learning-order connections, learning status tracking, annotations, plus LLM-assisted explanation / exercises / grading.

- **Form**: desktop client (Wails v2 native window + WebView2), single file, no install, double-click to run
- **Frontend** Vue 3 + Element Plus (custom canvas with Vue Flow) · **Backend** Go (Gin + GORM + modernc SQLite), all embedded in one executable
- **Data** local SQLite single file (WAL), JSON import/export as fallback, fully self-contained

## Features

- **Knowledge tree canvas**: left-to-right mind-map hierarchy layout, one-click auto layout (overlap-free); multi-grade zones with a top color bar that resizes dynamically to content
- **Node operations**: create / ＋child / ＋sibling / edit (rename, change grade, change parent) / delete; positions freely draggable with undo/redo
- **Learning-order edges**: top→bottom Bézier curves, easy to create/delete, unified edge style
- **Learning status**: not started / learning / partially learned / mastered / partially forgotten / forgotten, synced between node badges and statistics
- **Detail drawer**: Markdown + math (KaTeX) auto-saved content, resources, annotations, AI-generated exercises with answering & grading
- **AI assistance**: OpenAI-compatible API with 27 built-in provider presets, streaming explanation + exercise grading (no output length limit)
- **Help drawer**: the "Help" button in the bottom-left opens a 70%-wide documentation drawer from the left
- **Statistics**: total knowledge points / mastery ratio / connections / annotations / exercise distribution

## Getting Started

### Build

```powershell
# Windows
.\scripts\build-desktop.ps1     # output: bin/knowtree-desktop-v<version>.exe (no console, with icon, versioned filename)
```

```bash
# macOS / Linux (run on the corresponding OS)
bash scripts/build-desktop.sh   # output: bin/knowtree-desktop-v<version>-{macos,linux}-{arch}
```

Double-click to run. Data lives in the `data\` directory next to the executable; logs in `data\knowtree-desktop.log`.

### Run from source (development)

```bash
# Desktop client (must use the desktop,production build tags)
go run -tags desktop,production ./cmd/knowtree-desktop -data .\data

# Frontend hot-reload debugging: pin the client port, start vite dev in another terminal
cd frontend && pnpm dev                                  # http://localhost:6006
go run -tags desktop,production ./cmd/knowtree-desktop -addr 127.0.0.1:6010
# open http://localhost:6006 in a browser (API proxied to 6010)
```

## Project Structure

```
├─ cmd/knowtree-desktop/  # Desktop client entry (Wails native window)
├─ internal/
│  ├─ api/               # REST handlers + DTO (nodes/edges/settings/search/version)
│  ├─ config/            # Runtime config
│  ├─ db/                # SQLite open (WAL) + goose migrations
│  │  └─ migrations/     # SQL migration files (embed)
│  ├─ llm/               # OpenAI-compatible LLM client
│  └─ models/            # GORM models
├─ web/                  # go:embed frontend build output (filled by build scripts)
├─ frontend/             # Vue 3 + Vite + Element Plus
├─ build/                # App icons (appicon.png / appicon.ico / appicon.icns)
├─ scripts/              # build-desktop.ps1 / build-desktop.sh
└─ docs/                 # Requirement documents, etc.
```

## Release

Pushing a `v*` tag on the **`main`** branch triggers GitHub Actions to build the desktop client for all three platforms and publish a Release (branch guard: only tags on `main` trigger; accidental tags on feature branches are silently skipped):

```bash
git push origin main
git tag v0.1.0
git push origin v0.1.0
```

Artifacts are uniformly named `knowtree-desktop-<version>-<platform>`:

- Windows: `knowtree-desktop-v0.1.0-windows-amd64.exe`
- macOS: `knowtree-desktop-v0.1.0-macos-amd64.app` (Intel) / `knowtree-desktop-v0.1.0-macos-arm64.app` (Apple Silicon)
- Linux: `knowtree-desktop-v0.1.0-linux-amd64`

A `checksums.txt` (SHA256) is included with the release. Upgrading = replace the executable; the `data\` directory is kept as-is.
