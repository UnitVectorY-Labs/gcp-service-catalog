# UVY Design adoption

- Release: [v0.1.0](https://github.com/UnitVectorY-Labs/uvy-design/tree/12f8e3d964785d78628346516351fe4dac583cef)
- Source commit: `12f8e3d964785d78628346516351fe4dac583cef`
- Shared files are copied verbatim; application rules remain in `assets/style.css`.
- No new framework or HTMX is introduced. Assets are copied into generated `html/uvy-design/`, including both licenses.

## Shared file checksums

| File | SHA-256 |
| --- | --- |
| `assets/uvy-design/LICENSE` | `00320f7ebea7d48c92ddabd067541e3a8d3c033deafb50ada6bf7a3a56f640dc` |
| `assets/uvy-design/TABLER-ICONS-LICENSE` | `b740a1d46122672da62833e97f7e7c8a13fa85cbc7445b584b297cc00dde93db` |
| `assets/uvy-design/uvy-elements.css` | `b2f92292dd9317bb770be1a4c745c9a00b81b6cef39cbcac54d2fe192135b092` |
| `assets/uvy-design/uvy-theme.js` | `b105c0ddec8c19b6a380a33ffa5f9f558ca49baa477921a1dee9596d77bc61ed` |
| `assets/uvy-design/uvy-tokens.css` | `0e97baaece886e0015c4238eaf7dbed9ea68f330c0f50729311c43b4a5a98533` |

## Adapter and invariants

The adapter uses the standard type/spacing scale with compact mobile table cell padding. It preserves the existing horizontal navigation and home/services/APIs/domain/service/detail page hierarchy, table columns, data files and local filter functions. The only responsive changes wrap the same navigation and long values within fixed-width tables; no fields are hidden or moved behind tabs. Existing non-preferred API coloring is themed and supplemented by a short textual version label.

The top-right binary toggle uses the shared host-only cookie controller. First visit follows OS preference; explicit light/dark survives navigation and reload. The project source and MIT license footer stays in normal document flow.

Official Tabler moon, sun, search and brand-github SVGs are inlined into `templates/icons.html` from pinned icon commit `ed177a085b652fe04ec4f37ced867c94621e7840`. Their paths are unchanged; decorative accessibility attributes are added. Source files: `https://github.com/tabler/tabler-icons/tree/ed177a085b652fe04ec4f37ced867c94621e7840/icons/outline`. Full MIT notice is vendored.

## Verification and update procedure

Generation uses the committed `services.json` (2,806 services) and `directory.json` (542 API records), unchanged. Run `WEBSITE=https://gcp-service-catalog.unitvectorylabs.com go run . -generate`, `go test ./...`, `go vet ./...` and `go build .`. Browser checks cover each route type, exact before/after filtering parity, theme persistence, 320/390px fit, no-JS browse and local-only runtime requests. Screenshot matrix and full method are linked from the PR. The original site lacks dark mode, so before screenshots under both OS preferences show its unchanged light presentation.

To update, review a tagged UVY Design release, copy shared files verbatim, refresh this provenance/checksum table, and retest the adapter and screenshot matrix. Never fetch mutable shared assets at runtime.
