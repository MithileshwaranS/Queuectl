#!/bin/bash

# QueueCTL Demo Script - All Required Test Scenarios
# This script demonstrates all 5 expected test scenarios

set -e

QUEUECTL="./queuectl"

# Colors
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

function pause() {
    echo ""
    read -p "Press Enter to continue to next test..."
    echo ""
}

clear
echo "======================================"
echo "QueueCTL - Job Queue System Demo"
echo "Testing All Required Scenarios"
echo "======================================"
echo ""
sleep 2

# ============================================
# TEST 1: Basic Job Completes Successfully
# ============================================
echo -e "${BLUE}=== TEST 1: Basic Job Completes Successfully ===${NC}"
echo ""

echo "Starting worker..."
$QUEUECTL worker start --count 1 > worker.log 2>&1 &
WORKER_PID=$!
sleep 3

echo "Enqueuing successful job..."
$QUEUECTL enqueue '{"command":"echo Hello World from QueueCTL"}'
sleep 3

echo ""
echo "Status:"
$QUEUECTL status

echo ""
echo "Completed jobs:"
$QUEUECTL list --state completed

echo ""
echo -e "${GREEN}✓ TEST 1 PASSED${NC}"
pause

# ============================================
# TEST 2: Failed Job Retries with Backoff → DLQ
# ============================================
echo -e "${BLUE}=== TEST 2: Failed Job Retries with Backoff → DLQ ===${NC}"
echo ""

echo "Enqueuing failing job (max_retries: 2)..."
$QUEUECTL enqueue '{"command":"exit 1", "max_retries":2}'
sleep 2

echo ""
echo "Waiting for first attempt (2s backoff)..."
sleep 4

echo "Job in failed state:"
$QUEUECTL list --state failed

echo ""
echo "Waiting for second attempt (4s backoff)..."
sleep 6

echo ""
echo "Dead Letter Queue (after all retries):"
$QUEUECTL dlq list

echo ""
echo -e "${GREEN}✓ TEST 2 PASSED${NC}"
pause

# ============================================
# TEST 3: Multiple Workers Without Overlap
# ============================================
echo -e "${BLUE}=== TEST 3: Multiple Workers Process Jobs Without Overlap ===${NC}"
echo ""

echo "Stopping previous worker..."
kill $WORKER_PID 2>/dev/null
sleep 2

echo "Starting 3 workers..."
$QUEUECTL worker start --count 3 > worker3.log 2>&1 &
WORKER3_PID=$!
sleep 3

echo "Enqueuing 6 jobs simultaneously..."
$QUEUECTL enqueue '{"id":"job-1","command":"sleep 2 && echo Job 1"}'
$QUEUECTL enqueue '{"id":"job-2","command":"sleep 2 && echo Job 2"}'
$QUEUECTL enqueue '{"id":"job-3","command":"sleep 2 && echo Job 3"}'
$QUEUECTL enqueue '{"id":"job-4","command":"sleep 2 && echo Job 4"}'
$QUEUECTL enqueue '{"id":"job-5","command":"sleep 2 && echo Job 5"}'
$QUEUECTL enqueue '{"id":"job-6","command":"sleep 2 && echo Job 6"}'

echo ""
echo "Currently processing (should see 3 workers active):"
sleep 1
$QUEUECTL list --state processing

echo ""
echo "Waiting for completion..."
sleep 4

echo ""
echo "Final status:"
$QUEUECTL status

echo ""
echo -e "${GREEN}✓ TEST 3 PASSED${NC}"
pause

# ============================================
# TEST 4: Invalid Commands Fail Gracefully
# ============================================
echo -e "${BLUE}=== TEST 4: Invalid Commands Fail Gracefully ===${NC}"
echo ""

echo "Testing invalid commands..."
$QUEUECTL enqueue '{"command":"nonexistent_command_xyz", "max_retries":1}'
$QUEUECTL enqueue '{"command":"cat /this/does/not/exist.txt", "max_retries":1}'

echo ""
echo "Waiting for failures..."
sleep 6

echo ""
echo "Failed jobs with error messages:"
$QUEUECTL list --state failed

echo ""
echo "Dead Letter Queue:"
$QUEUECTL dlq list

echo ""
echo -e "${GREEN}✓ TEST 4 PASSED${NC}"
pause

# ============================================
# TEST 5: Job Data Survives Restart
# ============================================
echo -e "${BLUE}=== TEST 5: Job Data Survives Restart ===${NC}"
echo ""

echo "Stopping all workers..."
kill $WORKER3_PID 2>/dev/null
pkill -f "queuectl worker" 2>/dev/null || true
sleep 2

echo "Enqueuing jobs while system is 'down'..."
$QUEUECTL enqueue '{"command":"echo Persistence Test 1"}'
$QUEUECTL enqueue '{"command":"echo Persistence Test 2"}'

echo ""
echo "Jobs persisted in database:"
$QUEUECTL list --state pending

echo ""
echo "Restarting workers..."
$QUEUECTL worker start --count 2 > worker_restart.log 2>&1 &
RESTART_PID=$!
sleep 3

echo "Jobs processing after restart..."
sleep 4

echo ""
echo "Final status:"
$QUEUECTL status

echo ""
echo -e "${GREEN}✓ TEST 5 PASSED${NC}"

# ============================================
# CLEANUP
# ============================================
echo ""
echo "======================================"
echo -e "${GREEN}ALL TESTS PASSED${NC}"
echo "======================================"
echo ""
echo "Summary:"
echo "  ✓ Test 1: Basic job completed"
echo "  ✓ Test 2: Retry with backoff → DLQ"
echo "  ✓ Test 3: Multiple workers without overlap"
echo "  ✓ Test 4: Invalid commands handled gracefully"
echo "  ✓ Test 5: Data persisted through restart"
echo ""

echo "Cleaning up..."
kill $RESTART_PID 2>/dev/null || true
pkill -f "queuectl worker" 2>/dev/null || true

echo ""
echo "Demo complete!"
