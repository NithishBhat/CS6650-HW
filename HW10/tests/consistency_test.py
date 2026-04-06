"""
Tests to prove our KV store is (or isn't) consistent.

Start the cluster first:
  W=5 R=1 docker compose -f docker-compose-leader.yml up --build -d
  pytest tests/consistency_test.py -v
"""

import requests
import threading
import time
import uuid

LEADER = "http://localhost:8080"
FOLLOWERS = [f"http://localhost:{p}" for p in [8081, 8082, 8083, 8084]]
ALL_NODES = [LEADER] + FOLLOWERS


# -- leader-follower tests (run with W=5 R=1) --

def test_leader_set_then_get_from_leader():
    """write to leader, read back from leader - should always match"""
    key = f"test-{uuid.uuid4()}"
    value = "hello"
    resp = requests.post(f"{LEADER}/set", json={"key": key, "value": value})
    assert resp.status_code == 201

    resp = requests.get(f"{LEADER}/get/{key}")
    assert resp.status_code == 200
    assert resp.json()["value"] == value


def test_leader_set_then_get_from_follower():
    """with W=5 the leader waits for everyone, so followers should have the data too"""
    key = f"test-{uuid.uuid4()}"
    value = "world"
    resp = requests.post(f"{LEADER}/set", json={"key": key, "value": value})
    assert resp.status_code == 201

    for follower in FOLLOWERS:
        resp = requests.get(f"{follower}/get/{key}")
        assert resp.status_code == 200
        assert resp.json()["value"] == value


def test_local_read_inconsistency_during_write():
    """
    the fun one - fire off a write but dont wait for it, then peek at a
    follower's local data. since replication is still in progress the
    follower should still have the old value.
    """
    key = f"test-{uuid.uuid4()}"
    requests.post(f"{LEADER}/set", json={"key": key, "value": "v1"})

    stale_reads = []

    def do_write():
        requests.post(f"{LEADER}/set", json={"key": key, "value": "v2"})

    # kick off write in a thread so we dont block
    t = threading.Thread(target=do_write)
    t.start()

    # poke the last follower (node4) - it gets updated last so its
    # the most likely to still have stale data
    time.sleep(0.05)
    for _ in range(5):
        resp = requests.get(f"{FOLLOWERS[-1]}/local_read/{key}")
        if resp.status_code == 200:
            val = resp.json()["value"]
            if val != "v2":
                stale_reads.append(val)
                break
        time.sleep(0.05)

    t.join()

    print(f"Stale reads caught: {stale_reads}")
    assert len(stale_reads) > 0, "should have caught at least one stale read"


# -- leaderless tests --
# swap to the leaderless cluster before running these:
#   docker compose -f docker-compose-leaderless.yml up --build -d
#   pytest tests/consistency_test.py::TestLeaderless -v

class TestLeaderless:

    def test_read_during_write_inconsistency(self):
        """same trick as above - write to node0, read node4 while its still propagating"""
        key = f"test-{uuid.uuid4()}"
        coordinator = ALL_NODES[0]
        other = ALL_NODES[4]

        requests.post(f"{coordinator}/set", json={"key": key, "value": "v1"})

        stale_reads = []

        def do_write():
            requests.post(f"{coordinator}/set", json={"key": key, "value": "v2"})

        t = threading.Thread(target=do_write)
        t.start()

        time.sleep(0.05)
        for _ in range(5):
            resp = requests.get(f"{other}/get/{key}")
            if resp.status_code == 200:
                val = resp.json()["value"]
                if val != "v2":
                    stale_reads.append(val)
                    break
            time.sleep(0.05)

        t.join()
        print(f"Stale reads caught: {stale_reads}")
        assert len(stale_reads) > 0, "should have caught at least one stale read"

    def test_read_from_coordinator_after_ack(self):
        """once the coordinator says 201 it definitely has the data locally"""
        key = f"test-{uuid.uuid4()}"
        coordinator = ALL_NODES[0]

        resp = requests.post(f"{coordinator}/set", json={"key": key, "value": "done"})
        assert resp.status_code == 201

        resp = requests.get(f"{coordinator}/get/{key}")
        assert resp.status_code == 200
        assert resp.json()["value"] == "done"

    def test_read_from_other_after_ack(self):
        """W=N so by the time we get 201 back, everyone should have the data"""
        key = f"test-{uuid.uuid4()}"
        coordinator = ALL_NODES[0]

        resp = requests.post(f"{coordinator}/set", json={"key": key, "value": "synced"})
        assert resp.status_code == 201

        for node in ALL_NODES[1:]:
            resp = requests.get(f"{node}/get/{key}")
            assert resp.status_code == 200
            assert resp.json()["value"] == "synced"
