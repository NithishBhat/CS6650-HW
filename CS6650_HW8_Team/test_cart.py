import requests
import json
import time
import statistics
from datetime import datetime, timezone

BASE_URL = "http://your-load-balancer-url"  # Replace with your load balancer URL
RESULTS_FILE = "mysql_test_results.json"

results = []

def record(operation, start, resp):
    elapsed_ms = (time.time() - start) * 1000
    results.append({
        "operation": operation,
        "response_time": round(elapsed_ms, 2),
        "success": 200 <= resp.status_code < 300,
        "status_code": resp.status_code,
        "timestamp": datetime.now(timezone.utc).strftime("%Y-%m-%dT%H:%M:%SZ")
    })

# Phase 1: Create 50 carts
print("Phase 1: Creating 50 carts...")
cart_ids = []
for i in range(50):
    start = time.time()
    resp = requests.post(f"{BASE_URL}/shopping-carts", json={"customer_id": i + 1})
    record("create_cart", start, resp)
    if resp.status_code == 201:
        cart_ids.append(resp.json()["cart_id"])
    if (i + 1) % 10 == 0:
        print(f"  Created {i + 1}/50")

print(f"  Done. {len(cart_ids)} carts created.\n")

# Phase 2: Add items to 50 carts
print("Phase 2: Adding items to 50 carts...")
for i in range(50):
    cart_id = cart_ids[i % len(cart_ids)]
    start = time.time()
    resp = requests.post(f"{BASE_URL}/shopping-carts/{cart_id}/items", json={
        "product_id": i + 100,
        "quantity": (i % 5) + 1
    })
    record("add_items", start, resp)
    if (i + 1) % 10 == 0:
        print(f"  Added {i + 1}/50")

print("  Done.\n")

# Phase 3: Get 50 carts
print("Phase 3: Getting 50 carts...")
for i in range(50):
    cart_id = cart_ids[i % len(cart_ids)]
    start = time.time()
    resp = requests.get(f"{BASE_URL}/shopping-carts/{cart_id}")
    record("get_cart", start, resp)
    if (i + 1) % 10 == 0:
        print(f"  Retrieved {i + 1}/50")

print("  Done.\n")

# Save results
with open(RESULTS_FILE, "w") as f:
    json.dump(results, f, indent=2)

# Summary
success_count = sum(1 for r in results if r["success"])
all_times = [r["response_time"] for r in results]
total_time = sum(all_times) / 1000

print("=" * 50)
print("Per-operation breakdown:")
print("-" * 50)
for op in ["create_cart", "add_items", "get_cart"]:
    items = [r for r in results if r["operation"] == op]
    times = [r["response_time"] for r in items]
    success = sum(1 for r in items if r["success"])
    print(f"  {op}: count={len(items)}, success={success}, "
          f"avg={sum(times)/len(times):.2f}ms, "
          f"min={min(times):.2f}ms, max={max(times):.2f}ms")

print("-" * 50)
print(f"Overall: count={len(results)}, success={success_count}/{len(results)}, "
      f"avg={sum(all_times)/len(all_times):.2f}ms, total_time={total_time:.2f}s")

# Percentiles
sorted_times = sorted(all_times)
p50 = statistics.median(sorted_times)
p95 = sorted_times[int(len(sorted_times) * 0.95)]
p99 = sorted_times[int(len(sorted_times) * 0.99)]
success_rate = (success_count / len(results)) * 100
print(f"Success rate: {success_rate:.1f}%")

print(f"\nPercentiles:")
print(f"  P50: {p50:.2f} ms")
print(f"  P95: {p95:.2f} ms")
print(f"  P99: {p99:.2f} ms")

print(f"\nResults saved to: {RESULTS_FILE}")
