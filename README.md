# Goten

> Composable authentication for Go — inspired by [better-auth](https://better-auth.com), [Limen](https://limenauth.dev), and Go-better-auth.

**Status**: 🚧 v0.1.0-dev (under active construction, see [GitHub Issues](https://github.com/dnahilman/goten/issues))

## Planned Features (MVP)

- ✅ Email/password sign-up & sign-in (Issue #4)
- ✅ Session management (opaque token `g10_`, cookie + Bearer) (Issue #3)
- ✅ Plugin system with capability interfaces (Issue #5)
- ✅ Username plugin (login via username) (Issue #7)
- ✅ CLI tool for migrations (`goten migrate up/down/status/generate`) (Issue #6)
- ✅ GORM adapter for Postgres (Issue #2)

## Architecture

```
goten/
├── (core)              # Auth, session, crypto, adapter interface
├── adapters/gorm       # Separate module — GORM adapter
├── plugins/username    # Separate module — username plugin
├── cmd/goten           # Separate module — CLI tool
├── examples/basic      # Separate module — runnable example
└── test                # Separate module — all tests centralized here
```

Each adapter, plugin, and CLI is a **separate Go module** — install only what you use.

## Implementation Roadmap

Work tracked via GitHub Issues:

| Issue | Title | Milestone |
|-------|-------|-----------|
| #1 | Foundation (multi-module setup, crypto, models, adapter interface) | A |
| #2 | GORM Adapter + core migrations | B |
| #3 | Session & Cookie Management | C |
| #4 | HTTP Layer & Handlers | D |
| #5 | Plugin System (interfaces + hooks + lifecycle) | E |
| #6 | CLI Tool (migrate up/down/status/generate) | F |
| #7 | Username Plugin | G |
| #8 | Polish (example, README, CI, CSRF, Makefile) | H |

## License

MIT — see [LICENSE](LICENSE) when added in Issue #8.
