#!/bin/sh
# Regenerates every table inside the pinned Node image, with this module mounted at /tp/gold:
#   sh gold/internal/intl/gen/run.sh
# then, on the host: python3 gold/internal/intl/gen/mkdata.py extract/out gold/internal/intl/data
set -eu
here=$(cd "$(dirname "$0")" && pwd)
OUT=${OUT:-/tp/extract/out}; export OUT
JOBS=${JOBS:-10}
cd "$here"
mkdir -p "$OUT"
node grid.mjs "$OUT/icu_grid.json"
node meta.mjs
node zone_candidates.mjs zone_candidates.txt
chunks() { i=0; while [ $i -lt "$JOBS" ]; do node "$@" $i "$JOBS" & i=$((i + 1)); done; wait; }
chunks tz.mjs
node cal.mjs
for c in chinese dangi; do node lunar.mjs $c -300000 300000 "$OUT/lunar_$c.json" & done
for c in islamic islamic-rgsa; do node lunar.mjs $c -3000 6000 "$OUT/lunar_$c.json" & done
wait
JOBS=$JOBS sh lunar_sweep.sh "$here" "$OUT"
chunks names.mjs
chunks trace.mjs
node trace.mjs merge
chunks tpl.mjs
node duration.mjs
