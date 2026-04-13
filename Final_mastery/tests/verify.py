#!/usr/bin/env python3
"""
Functional verification of all Album Store endpoints.

Usage:  python verify.py http://<server-ip>
"""

import sys
import time
import uuid
import requests

BASE = sys.argv[1].rstrip("/") if len(sys.argv) > 1 else "http://localhost"
PASS_COUNT = 0
FAIL_COUNT = 0


def check(name, ok, detail=""):
    global PASS_COUNT, FAIL_COUNT
    tag = "PASS" if ok else "FAIL"
    if ok:
        PASS_COUNT += 1
    else:
        FAIL_COUNT += 1
    suffix = f" -- {detail}" if detail and not ok else ""
    print(f"  [{tag}] {name}{suffix}")
    return ok


def main():
    album_id = str(uuid.uuid4())
    photo_bytes = b"\xff\xd8\xff\xe0" + b"\x00" * 100

    # ── S1: Health ───────────────────────────────────────────────────
    print("S1: Health Check")
    r = requests.get(f"{BASE}/health")
    check("status 200", r.status_code == 200, f"got {r.status_code}")
    check('body has status=ok', r.json().get("status") == "ok", r.text)

    # ── S2: Album Create + Read ──────────────────────────────────────
    print("S2: Album Create + Read")
    album = {
        "album_id": album_id,
        "title": "Test Album",
        "description": "Desc",
        "owner": "a@b.com",
    }
    r = requests.put(f"{BASE}/albums/{album_id}", json=album)
    check("PUT returns 201", r.status_code == 201, f"got {r.status_code}")
    check("PUT body matches", r.json().get("album_id") == album_id)

    # Idempotent update
    r = requests.put(f"{BASE}/albums/{album_id}", json=album)
    check("PUT idempotent 200", r.status_code == 200, f"got {r.status_code}")

    # GET
    r = requests.get(f"{BASE}/albums/{album_id}")
    check("GET 200", r.status_code == 200, f"got {r.status_code}")
    expected = {"album_id": album_id, "title": "Test Album", "description": "Desc", "owner": "a@b.com"}
    check("GET fields match", r.json() == expected, r.text)

    # 404
    r = requests.get(f"{BASE}/albums/{uuid.uuid4()}")
    check("GET unknown 404", r.status_code == 404, f"got {r.status_code}")

    # ── S3: Async Photo Upload ───────────────────────────────────────
    print("S3: Async Photo Upload")
    r = requests.post(
        f"{BASE}/albums/{album_id}/photos",
        files={"photo": ("test.jpg", photo_bytes, "image/jpeg")},
    )
    check("POST 202", r.status_code == 202, f"got {r.status_code}")
    data = r.json()
    photo_id = data.get("photo_id", "")
    check("has photo_id", bool(photo_id))
    check("has seq", "seq" in data)
    check("seq == 1", data.get("seq") == 1)
    check("status == processing", data.get("status") == "processing")

    # Poll for completion
    url = None
    for _ in range(30):
        r = requests.get(f"{BASE}/albums/{album_id}/photos/{photo_id}")
        if r.status_code == 200 and r.json().get("status") == "completed":
            url = r.json().get("url")
            break
        time.sleep(1)
    check("photo reached completed", url is not None, "timed out after 30s")
    if url:
        r = requests.get(url)
        check("url returns 200", r.status_code == 200, f"got {r.status_code}")

    # ── S4: Photo Delete ─────────────────────────────────────────────
    print("S4: Photo Delete")
    r = requests.delete(f"{BASE}/albums/{album_id}/photos/{photo_id}")
    check("DELETE 200 or 204", r.status_code in (200, 204), f"got {r.status_code}")

    r = requests.get(f"{BASE}/albums/{album_id}/photos/{photo_id}")
    check("GET after delete => 404", r.status_code == 404, f"got {r.status_code}")

    if url:
        time.sleep(1)
        r = requests.get(url)
        check("url no longer 200", r.status_code != 200, f"got {r.status_code}")

    # ── S5: List Albums ──────────────────────────────────────────────
    print("S5: List Albums")
    r = requests.get(f"{BASE}/albums")
    check("list 200", r.status_code == 200, f"got {r.status_code}")
    data = r.json()
    if isinstance(data, dict):
        data = data.get("albums", [])
    ids = [a.get("album_id") for a in data]
    check("album in list", album_id in ids)

    # ── S10: Per-Album Photo Sequence ────────────────────────────────
    print("S10: Per-Album Photo Sequence")
    album2 = str(uuid.uuid4())
    requests.put(
        f"{BASE}/albums/{album2}",
        json={"album_id": album2, "title": "A2", "description": "D2", "owner": "o@o.com"},
    )

    r1 = requests.post(
        f"{BASE}/albums/{album_id}/photos",
        files={"photo": ("t.jpg", photo_bytes, "image/jpeg")},
    )
    r2 = requests.post(
        f"{BASE}/albums/{album2}/photos",
        files={"photo": ("t.jpg", photo_bytes, "image/jpeg")},
    )
    # album_id already had seq=1 (deleted), counter keeps going → 2
    check("album1 seq == 2", r1.json().get("seq") == 2, f"got {r1.json().get('seq')}")
    # album2 is fresh → 1
    check("album2 seq == 1", r2.json().get("seq") == 1, f"got {r2.json().get('seq')}")

    # ── Summary ──────────────────────────────────────────────────────
    print()
    total = PASS_COUNT + FAIL_COUNT
    print(f"Results: {PASS_COUNT}/{total} passed, {FAIL_COUNT} failed")
    if FAIL_COUNT == 0:
        print("ALL PASSED")
    else:
        print("SOME TESTS FAILED")
    sys.exit(0 if FAIL_COUNT == 0 else 1)


if __name__ == "__main__":
    main()
