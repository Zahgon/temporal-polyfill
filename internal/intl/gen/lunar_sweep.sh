#!/bin/sh
# Sweep every recorded day of the table-driven calendars with $JOBS processes. Usage: lunar_sweep.sh SCRIPT_DIR OUT_DIR
dir=$1; out=$2; JOBS=${JOBS:-10}
for spec in "chinese -11707483 23860219" "dangi -11707483 24001937" "islamic -1815100 1471800" "islamic-rgsa -1815100 1471800"; do
  set -- $spec
  cal=$1; lo=$2; hi=$3; step=$(( (hi - lo) / JOBS + 1 )); i=0; a=$lo
  while [ $a -lt $hi ]; do
    b=$((a + step)); [ $b -gt $hi ] && b=$hi
    node "$dir/lunar_sweep.mjs" "$cal" "$a" "$b" "$out/sweep_${cal}_$i.json" &
    a=$b; i=$((i + 1))
  done
  wait
done
