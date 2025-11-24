#!/usr/bin/env bash
set -euo pipefail

usage() {
    cat <<EOF
Usage: $0 [MOUNTPOINT]

Runs automated FUSE smoke tests against the provided MOUNTPOINT.
No default mountpoint is used; you must pass the mountpoint path.

Example:
  $0 /mnt/mimic
EOF
}

# require a single positional mountpoint argument
if [ "${1:-}" = "-h" ] || [ "${1:-}" = "--help" ]; then
    usage
    exit 0
fi

if [ $# -ne 1 ]; then
    echo "Error: missing [MOUNTPOINT]"
    usage
    exit 2
fi

MOUNT="$1"
SLEEP_AFTER_OP=0.2

timestamp() { date +"%Y-%m-%d %H:%M:%S"; }

run_step() {
    local desc="$1"; shift
    local out
    out=$(mktemp) || out="/tmp/step.$$.tmp"
    printf "%s [STEP] %s ... " "$(timestamp)" "$desc"
    if "$@" >"$out" 2>&1; then
        echo "OK"
        rm -f "$out"
        return 0
    else
        echo "FAIL"
        echo "----- Captured output for failed step: $desc -----"
        sed -n '1,200p' "$out" || true
        echo "----- end output -----"
        rm -f "$out"
        return 1
    fi
}

# ensure background jobs are killed on exit
cleanup() {
    # kill background jobs
    jobs -p | xargs -r kill 2>/dev/null || true

    # only attempt removal if MOUNT is set and not root
    if [ -n "${MOUNT:-}" ] && [ "$MOUNT" != "/" ]; then
        echo "$(timestamp) Cleaning up test artifacts in $MOUNT"
        rm -rf "$MOUNT"/test_basic.txt \
               "$MOUNT"/test_basic.renamed \
               "$MOUNT"/test_big.bin \
               "$MOUNT"/test_big_stream.txt \
               "$MOUNT"/test_dir 2>/dev/null || true
    fi
}
trap cleanup EXIT

echo "$(timestamp) Starting automated FUSE smoke tests for mount: $MOUNT"

run_step "List mountpoint" ls -la "$MOUNT" || true

run_step "Create empty file" touch "$MOUNT"/test_basic.txt
sleep $SLEEP_AFTER_OP

run_step "Write small content" bash -c "echo 'hello world' > '$MOUNT'/test_basic.txt"
sleep $SLEEP_AFTER_OP

run_step "Read back and verify content" bash -c "grep -q 'hello world' '$MOUNT'/test_basic.txt"
run_step "Show file content" cat "$MOUNT"/test_basic.txt

run_step "Append line" bash -c "echo 'append-line' >> '$MOUNT'/test_basic.txt"
sleep $SLEEP_AFTER_OP
run_step "Verify appended line" bash -c "grep -q 'append-line' '$MOUNT'/test_basic.txt"

run_step "Create directory" mkdir "$MOUNT"/test_dir
sleep $SLEEP_AFTER_OP
run_step "Verify directory exists" bash -c "ls -ld '$MOUNT'/test_dir | grep -q '^d'"

run_step "Move file into directory" mv "$MOUNT"/test_basic.txt "$MOUNT"/test_dir/
sleep $SLEEP_AFTER_OP
run_step "Verify moved file exists" bash -c "ls -l '$MOUNT'/test_dir | grep -q test_basic.txt"

run_step "Rename file" mv "$MOUNT"/test_dir/test_basic.txt "$MOUNT"/test_basic.renamed
sleep $SLEEP_AFTER_OP
run_step "Verify renamed exists" bash -c "ls -l '$MOUNT' | grep -q test_basic.renamed"

run_step "Large write (10MB)" dd if=/dev/zero of="$MOUNT"/test_big.bin bs=1M count=10 status=none
sleep $SLEEP_AFTER_OP
run_step "Verify large file size" bash -c "size=\$(stat -c '%s' '$MOUNT'/test_big.bin 2>/dev/null || echo 0); [ \"\$size\" -ge $((10*1024*1024)) ]"

echo "$(timestamp) Starting concurrent append/read test"

( for i in {1..20}; do echo "x$i" >> "$MOUNT"/test_big_stream.txt; sleep 0.05; done ) &
APPENDER_PID=$!

sleep 0.1

tail -n +1 -f "$MOUNT"/test_big_stream.txt 2>/dev/null &
TAILPID=$!
sleep 0.5
kill "$TAILPID" 2>/dev/null || true
wait "$TAILPID" 2>/dev/null || true

wait "$APPENDER_PID" 2>/dev/null || true

echo "$(timestamp) All steps completed"