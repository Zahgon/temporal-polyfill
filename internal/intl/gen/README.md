# ICU tables

`data/` holds what `internal/intl` needs from Intl in Node.js 24.22.1 (ICU 78.3, CLDR 48.0, tz 2026e). Every
table comes from that Node, and nothing is written by hand.

Run in the pinned Node image, offline, with TZ=UTC and the built original at `/orig/dist/index.cjs`. `OUT` defaults to
`/tp/extract/out`. `run.sh` does all of this in order.

| Script | Output |
| --- | --- |
| `grid.mjs OUT` | `icu_grid.json`: the locale x bag x zone x calendar grid every other script reads |
| `meta.mjs` | locale defaults, languages, calendars, numbering systems, digits |
| `zone_candidates.mjs FILE` | `zone_candidates.txt`: every id in the Node binary that Intl accepts |
| `tz.mjs CHUNK CHUNKS` | offsets per primary zone, bisected to the second |
| `cal.mjs` | Japanese era starts, Umm al-Qura months, Persian year starts |
| `lunar.mjs CAL FROM TO OUT` | month walk of chinese, dangi, islamic, islamic-rgsa, stopping at the first ICU error |
| `lunar_sweep.mjs`, `lunar_sweep.sh` | chinese and dangi day by day, which also finds the single days ICU gets wrong |
| `trace.mjs CHUNK CHUNKS`, `trace.mjs merge` | per context, the option bags the polyfill's amend functions build from the grid's bags (needs the built original at `/orig/dist/index.cjs`) |
| `tpl.mjs CHUNK CHUNKS` | per (context, bag) the part shape, value lists, year rules and formatRange levels |
| `names.mjs CHUNK CHUNKS` | zone names per period, for the grid's zones |
| `duration.mjs` | units, list patterns and number symbols for DurationFormat |
| `vectors.mjs DIR FACTOR SEED` | test vectors; `FACTOR 1` writes `testdata`, a larger one a stress run |
| `mkdata.py OUT DATA` | the compact `data/*.gz` |

A *context* is what formatter text depends on besides the bag: base tag, resolved calendar, numbering system and
a `-u-hc-` keyword. Data is split per language so a process only reads the language it uses, and formatRange
tables load on first use.

`INTL_VECTOR_DIR=DIR go test ./internal/intl` replays a stress run instead of `testdata`.
