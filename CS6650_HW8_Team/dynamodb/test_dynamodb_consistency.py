import requests
import time
import json
from datetime import datetime, timezone

BASE_URL = "http://shopping-cart-hw8-alb-117374627.us-west-2.elb.amazonaws.com"
RESULTS_FILE = "dynamodb_consistency_results.json"
TRIALS = 20

results = []


def now_utc() -> str:
    return datetime.now(timezone.utc).strftime("%Y-%m-%dT%H:%M:%SZ")


def record_result(test_name, trial, cart_id, normal_ok, strong_ok, details):
    results.append({
        "test_name": test_name,
        "trial": trial,
        "cart_id": cart_id,
        "normal_read_matches_expected": normal_ok,
        "strong_read_matches_expected": strong_ok,
        "details": details,
        "timestamp": now_utc()
    })


def safe_json(resp):
    try:
        return resp.json()
    except Exception:
        return {"raw_text": resp.text}


def get_cart(cart_id: str, consistent: bool = False):
    url = f"{BASE_URL}/shopping-carts/{cart_id}"
    if consistent:
        url += "?consistent=true"
    return requests.get(url, timeout=10)


def create_cart(customer_id: int):
    return requests.post(
        f"{BASE_URL}/shopping-carts",
        json={"customer_id": customer_id},
        timeout=10
    )


def add_item(cart_id: str, product_id: int, quantity: int):
    return requests.post(
        f"{BASE_URL}/shopping-carts/{cart_id}/items",
        json={"product_id": product_id, "quantity": quantity},
        timeout=10
    )


def test_create_then_immediate_read(trials: int):
    print(f"Running create-then-immediate-read test for {trials} trials...")

    for i in range(1, trials + 1):
        create_resp = create_cart(1000 + i)
        create_body = safe_json(create_resp)

        if create_resp.status_code != 201:
            record_result(
                test_name="create_then_immediate_read",
                trial=i,
                cart_id=None,
                normal_ok=False,
                strong_ok=False,
                details={
                    "create_status": create_resp.status_code,
                    "create_body": create_body
                }
            )
            continue

        cart_id = create_body["cart_id"]

        # immediate normal read
        normal_resp = get_cart(cart_id, consistent=False)
        normal_body = safe_json(normal_resp)

        # immediate strong read
        strong_resp = get_cart(cart_id, consistent=True)
        strong_body = safe_json(strong_resp)

        expected_customer_id = 1000 + i

        normal_ok = (
            normal_resp.status_code == 200
            and normal_body.get("cart_id") == cart_id
            and normal_body.get("customer_id") == expected_customer_id
        )

        strong_ok = (
            strong_resp.status_code == 200
            and strong_body.get("cart_id") == cart_id
            and strong_body.get("customer_id") == expected_customer_id
        )

        record_result(
            test_name="create_then_immediate_read",
            trial=i,
            cart_id=cart_id,
            normal_ok=normal_ok,
            strong_ok=strong_ok,
            details={
                "create_status": create_resp.status_code,
                "normal_status": normal_resp.status_code,
                "strong_status": strong_resp.status_code,
                "normal_body": normal_body,
                "strong_body": strong_body
            }
        )

        if i % 5 == 0:
            print(f"  Completed {i}/{trials}")


def test_add_item_then_immediate_read(trials: int):
    print(f"Running add-item-then-immediate-read test for {trials} trials...")

    for i in range(1, trials + 1):
        # create a fresh cart first
        create_resp = create_cart(2000 + i)
        create_body = safe_json(create_resp)

        if create_resp.status_code != 201:
            record_result(
                test_name="add_item_then_immediate_read",
                trial=i,
                cart_id=None,
                normal_ok=False,
                strong_ok=False,
                details={
                    "create_status": create_resp.status_code,
                    "create_body": create_body
                }
            )
            continue

        cart_id = create_body["cart_id"]
        product_id = 5000 + i
        quantity = (i % 3) + 1

        add_resp = add_item(cart_id, product_id, quantity)
        add_body = safe_json(add_resp)

        if add_resp.status_code != 201:
            record_result(
                test_name="add_item_then_immediate_read",
                trial=i,
                cart_id=cart_id,
                normal_ok=False,
                strong_ok=False,
                details={
                    "add_status": add_resp.status_code,
                    "add_body": add_body
                }
            )
            continue

        # immediate normal read
        normal_resp = get_cart(cart_id, consistent=False)
        normal_body = safe_json(normal_resp)

        # immediate strong read
        strong_resp = get_cart(cart_id, consistent=True)
        strong_body = safe_json(strong_resp)

        def has_expected_item(body):
            items = body.get("items", [])
            for item in items:
                if item.get("product_id") == product_id and item.get("quantity") == quantity:
                    return True
            return False

        normal_ok = normal_resp.status_code == 200 and has_expected_item(normal_body)
        strong_ok = strong_resp.status_code == 200 and has_expected_item(strong_body)

        record_result(
            test_name="add_item_then_immediate_read",
            trial=i,
            cart_id=cart_id,
            normal_ok=normal_ok,
            strong_ok=strong_ok,
            details={
                "add_status": add_resp.status_code,
                "normal_status": normal_resp.status_code,
                "strong_status": strong_resp.status_code,
                "add_body": add_body,
                "normal_body": normal_body,
                "strong_body": strong_body
            }
        )

        if i % 5 == 0:
            print(f"  Completed {i}/{trials}")


def print_summary():
    print("\n" + "=" * 60)
    print("Consistency Test Summary")
    print("=" * 60)

    grouped = {}
    for r in results:
        grouped.setdefault(r["test_name"], []).append(r)

    for test_name, rows in grouped.items():
        total = len(rows)
        normal_pass = sum(1 for r in rows if r["normal_read_matches_expected"])
        strong_pass = sum(1 for r in rows if r["strong_read_matches_expected"])

        print(f"\n{test_name}")
        print("-" * 60)
        print(f"Trials: {total}")
        print(f"Normal read matched expected: {normal_pass}/{total}")
        print(f"Strong read matched expected: {strong_pass}/{total}")
        print(f"Normal read mismatch count: {total - normal_pass}")
        print(f"Strong read mismatch count: {total - strong_pass}")

    with open(RESULTS_FILE, "w") as f:
        json.dump(results, f, indent=2)

    print(f"\nDetailed results saved to: {RESULTS_FILE}")


if __name__ == "__main__":
    test_create_then_immediate_read(TRIALS)
    test_add_item_then_immediate_read(TRIALS)
    print_summary()