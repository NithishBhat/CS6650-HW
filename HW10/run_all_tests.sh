#!/bin/bash
# Run all 16 load test combinations sequentially.
# Usage: bash run_all_tests.sh

set -e
cd "$(dirname "$0")"

USERS=10
RATE=5
DURATION=60s

mkdir -p results

wait_healthy() {
    echo "  Waiting for nodes..."
    for i in {1..30}; do
        if curl -sf http://localhost:8080/health > /dev/null 2>&1 && \
           curl -sf http://localhost:8084/health > /dev/null 2>&1; then
            echo "  Nodes healthy."
            return 0
        fi
        sleep 1
    done
    echo "  ERROR: nodes not healthy after 30s"
    return 1
}

run_test() {
    local CONFIG_NAME=$1   # e.g. "leader_w5r1"
    local COMPOSE_FILE=$2  # e.g. "docker-compose-leader.yml"
    local W_VAL=$3
    local R_VAL=$4
    local WRITE_RATIO=$5
    local READ_RATIO=$6
    local MODE=$7          # "leader" or "leaderless"

    local RUN_NAME="${CONFIG_NAME}_${WRITE_RATIO}write"
    echo ""
    echo "=========================================="
    echo "  Running: $RUN_NAME (W=$W_VAL R=$R_VAL, ${WRITE_RATIO}w/${READ_RATIO}r, mode=$MODE)"
    echo "=========================================="

    # Start cluster
    echo "  Starting cluster..."
    W=$W_VAL R=$R_VAL docker compose -f "$COMPOSE_FILE" up --build -d 2>&1 | tail -1
    wait_healthy

    # Run locust
    echo "  Running locust for $DURATION..."
    cd loadtest
    WRITE_RATIO=$WRITE_RATIO \
    READ_RATIO=$READ_RATIO \
    MODE=$MODE \
    STALE_FILE="../results/${RUN_NAME}_stale.txt" \
    INTERVALS_FILE="../results/${RUN_NAME}_intervals.csv" \
    python -m locust -f locustfile.py \
        --host http://localhost:8080 \
        --headless \
        -u $USERS -r $RATE -t $DURATION \
        --csv "../results/${RUN_NAME}" \
        2>&1 | tail -20
    cd ..

    # Stop cluster
    echo "  Stopping cluster..."
    docker compose -f "$COMPOSE_FILE" down 2>&1 | tail -1
    echo "  Done: $RUN_NAME"
    sleep 2
}

RATIOS=("1 99" "10 90" "50 50" "90 10")

# Config 1: Leader W=5 R=1
for ratio in "${RATIOS[@]}"; do
    w=$(echo $ratio | cut -d' ' -f1)
    r=$(echo $ratio | cut -d' ' -f2)
    run_test "leader_w5r1" "docker-compose-leader.yml" 5 1 "$w" "$r" "leader"
done

# Config 2: Leader W=1 R=5
for ratio in "${RATIOS[@]}"; do
    w=$(echo $ratio | cut -d' ' -f1)
    r=$(echo $ratio | cut -d' ' -f2)
    run_test "leader_w1r5" "docker-compose-leader.yml" 1 5 "$w" "$r" "leader"
done

# Config 3: Leader W=3 R=3
for ratio in "${RATIOS[@]}"; do
    w=$(echo $ratio | cut -d' ' -f1)
    r=$(echo $ratio | cut -d' ' -f2)
    run_test "leader_w3r3" "docker-compose-leader.yml" 3 3 "$w" "$r" "leader"
done

# Config 4: Leaderless W=5 R=1
for ratio in "${RATIOS[@]}"; do
    w=$(echo $ratio | cut -d' ' -f1)
    r=$(echo $ratio | cut -d' ' -f2)
    run_test "leaderless_w5r1" "docker-compose-leaderless.yml" 5 1 "$w" "$r" "leaderless"
done

echo ""
echo "=========================================="
echo "  ALL 16 RUNS COMPLETE"
echo "=========================================="
echo "  Results in results/"
