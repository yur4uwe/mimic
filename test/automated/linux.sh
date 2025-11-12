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
    jobs -p | xargs -r kill 2>/dev/null || true
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
run_step "Verify appended line" bash -c "tail -n +1 '$MOUNT'/test_basic.txt"

run_step "Rename file" mv "$MOUNT"/test_basic.txt "$MOUNT"/test_basic.renamed
sleep $SLEEP_AFTER_OP
run_step "Verify renamed exists" bash -c "ls -l '$MOUNT' | grep -q test_basic.renamed"

run_step "Large write (10MB)" dd if=/dev/zero of="$MOUNT"/test_big.bin bs=1M count=10 status=none
sleep $SLEEP_AFTER_OP
run_step "Verify large file size" stat -c '%n %s' "$MOUNT"/test_big.bin

echo "$(timestamp) Starting concurrent append/read test"
# start background appender
run_step "Start background appender" bash -c "( for i in {1..20}; do echo \"x\$i\" >> '$MOUNT'/test_big_stream.txt; sleep 0.05; done ) &"

# give writer a moment to start
sleep 0.1

# tail a little and then stop
tail -n +1 -f "$MOUNT"/test_big_stream.txt 2>/dev/null &
TAILPID=$!
sleep 0.5
kill "$TAILPID" 2>/dev/null || true
wait "$TAILPID" 2>/dev/null || true

run_step "Verify stream file exists" test -f "$MOUNT"/test_big_stream.txt

run_step "Remove test files" rm -f "$MOUNT"/test_basic.renamed "$MOUNT"/test_big.bin "$MOUNT"/test_big_stream.txt || true
sleep $SLEEP_AFTER_OP

echo "$(timestamp) All steps completed"
