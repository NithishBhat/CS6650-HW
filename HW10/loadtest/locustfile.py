"""
load test for the kv store. set env vars to control behavior:
  WRITE_RATIO / READ_RATIO - controls the mix (e.g. 10/90)
  MODE - "leader" or "leaderless"

example:
  WRITE_RATIO=10 READ_RATIO=90 MODE=leader locust -f locustfile.py --host http://localhost:8080
"""

import os
import random
import threading
import time
import uuid

from locust import HttpUser, task, between, events

WRITE_RATIO = int(os.getenv("WRITE_RATIO", "10"))
READ_RATIO = int(os.getenv("READ_RATIO", "90"))
MODE = os.getenv("MODE", "leader")  # "leader" or "leaderless"

# only 10 keys so reads and writes hit the same key often (temporal locality)
KEYS = [f"key-{i}" for i in range(10)]

FOLLOWER_PORTS = [8081, 8082, 8083, 8084]
ALL_PORTS = [8080, 8081, 8082, 8083, 8084]

# we track the latest version we've seen per key so we can spot stale reads
version_lock = threading.Lock()
known_versions = {}
stale_read_count = 0

# track timestamps of last write and each read per key, to compute rw intervals
last_write_time = {}  # key -> timestamp of last write
rw_intervals = []     # list of (interval_ms) between a write and the next read on same key


STALE_FILE = os.getenv("STALE_FILE", "stale_reads.txt")
INTERVALS_FILE = os.getenv("INTERVALS_FILE", "rw_intervals.csv")


@events.test_stop.add_listener
def on_test_stop(environment, **kwargs):
    print(f"\n=== STALE READS DETECTED: {stale_read_count} ===\n")
    with open(STALE_FILE, "w") as f:
        f.write(str(stale_read_count))
    # dump read-write intervals
    with open(INTERVALS_FILE, "w") as f:
        f.write("interval_ms\n")
        for iv in rw_intervals:
            f.write(f"{iv:.1f}\n")


class KVUser(HttpUser):
    wait_time = between(0.01, 0.05)

    def on_start(self):
        # seed a key so we dont just get 404s at the start
        key = random.choice(KEYS)
        self.client.post("/set", json={"key": key, "value": "init"})

    @task(WRITE_RATIO)
    def write_key(self):
        key = random.choice(KEYS)
        value = str(uuid.uuid4())[:8]

        if MODE == "leader":
            # in leader mode writes only go to the leader on 8080
            with self.client.post("/set", json={"key": key, "value": value},
                                  catch_response=True, name="/set") as resp:
                if resp.status_code == 201:
                    data = resp.json()
                    ver = data.get("version", 0)
                    now = time.time()
                    with version_lock:
                        if ver > known_versions.get(key, 0):
                            known_versions[key] = ver
                        last_write_time[key] = now
                    resp.success()
                else:
                    resp.failure(f"write failed: {resp.status_code}")
        else:
            # leaderless - pick any node, it becomes the coordinator
            port = random.choice(ALL_PORTS)
            with self.client.post(f"http://localhost:{port}/set",
                                  json={"key": key, "value": value},
                                  catch_response=True, name="/set") as resp:
                if resp.status_code == 201:
                    data = resp.json()
                    ver = data.get("version", 0)
                    now = time.time()
                    with version_lock:
                        if ver > known_versions.get(key, 0):
                            known_versions[key] = ver
                        last_write_time[key] = now
                    resp.success()
                else:
                    resp.failure(f"write failed: {resp.status_code}")

    @task(READ_RATIO)
    def read_key(self):
        global stale_read_count
        key = random.choice(KEYS)

        if MODE == "leader":
            # spread reads across followers
            port = random.choice(FOLLOWER_PORTS)
            url = f"http://localhost:{port}/get/{key}"
        else:
            # leaderless - read from whoever
            port = random.choice(ALL_PORTS)
            url = f"http://localhost:{port}/get/{key}"

        with self.client.get(url, catch_response=True, name="/get/{key}") as resp:
            if resp.status_code == 200:
                data = resp.json()
                ver = data.get("version", 0)
                now = time.time()
                with version_lock:
                    max_ver = known_versions.get(key, 0)
                    if ver < max_ver:
                        stale_read_count += 1
                        resp.failure(f"stale read: got v{ver}, expected v{max_ver}")
                    else:
                        resp.success()
                    # log interval between this read and last write to same key
                    if key in last_write_time:
                        interval = (now - last_write_time[key]) * 1000
                        rw_intervals.append(interval)
            elif resp.status_code == 404:
                resp.success()  # hasnt been written yet, thats fine
            else:
                resp.failure(f"read failed: {resp.status_code}")
