#!/bin/bash
set -e

OUTFILE="${1:-test_archive.zip}"
TMPDIR=$(mktemp -d)

cleanup() { rm -rf "$TMPDIR"; }
trap cleanup EXIT

echo "Creating 200 test files in $TMPDIR..."

for i in $(seq 1 199); do
    FILE="$TMPDIR/file_$(printf '%03d' $i).txt"
    # ~2.5 MB each → ~500 MB total (base64 expands ~33%, so 1.9 MB /dev/urandom → ~2.5 MB)
    head -c 1900000 /dev/urandom | base64 > "$FILE"
done

# One file contains the EICAR test string buried in surrounding content
EICAR_FILE="$TMPDIR/file_$(printf '%03d' $((RANDOM % 199 + 1)))_infected.txt"
printf 'X5O!P%%@AP[4\\PZX54(P^)7CC)7}$EICAR-STANDARD-ANTIVIRUS-TEST-FILE!$H+H*' >> "$EICAR_FILE"

echo "Zipping into $OUTFILE..."
(cd "$TMPDIR" && zip -q "$OLDPWD/$OUTFILE" *)

echo "Done: $OUTFILE ($(du -sh "$OUTFILE" | cut -f1))"
