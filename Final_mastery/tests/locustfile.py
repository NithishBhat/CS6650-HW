"""
Load tests for Album Store API.

Quick verify (1 user):
  locust -f locustfile.py --headless -u 1 -r 1 --run-time 60s -H http://<IP>

Load test:
  locust -f locustfile.py --headless -u 50 -r 10 --run-time 120s -H http://<IP>
"""

import uuid
import time
from locust import HttpUser, task, between


class AlbumStoreUser(HttpUser):
    wait_time = between(0.1, 0.5)

    def on_start(self):
        self.album_id = str(uuid.uuid4())
        self.client.put(
            f"/albums/{self.album_id}",
            json={
                "album_id": self.album_id,
                "title": "Setup Album",
                "description": "Created on start",
                "owner": "test@northeastern.edu",
            },
            name="PUT /albums/:id [setup]",
        )
        self.completed_photos = []
        self.small_photo = b"\xff\xd8\xff\xe0" + b"\x00" * 100
        self.large_photo = b"\xff\xd8\xff\xe0" + b"\x00" * (5 * 1024 * 1024)

    # ── health ───────────────────────────────────────────────────────
    @task(3)
    def health(self):
        self.client.get("/health")

    # ── album CRUD ───────────────────────────────────────────────────
    @task(5)
    def create_album(self):
        aid = str(uuid.uuid4())
        self.client.put(
            f"/albums/{aid}",
            json={
                "album_id": aid,
                "title": f"Album {aid[:8]}",
                "description": "Test",
                "owner": "test@northeastern.edu",
            },
            name="PUT /albums/:id",
        )

    @task(5)
    def get_album(self):
        self.client.get(f"/albums/{self.album_id}", name="GET /albums/:id")

    @task(2)
    def list_albums(self):
        self.client.get("/albums")

    # ── photo upload (small) ─────────────────────────────────────────
    @task(3)
    def upload_photo_small(self):
        r = self.client.post(
            f"/albums/{self.album_id}/photos",
            files={"photo": ("test.jpg", self.small_photo, "image/jpeg")},
            name="POST /albums/:id/photos",
        )
        if r.status_code == 202:
            pid = r.json().get("photo_id")
            if pid:
                self._poll(pid)

    # ── photo upload (large ~5 MB) ───────────────────────────────────
    @task(1)
    def upload_photo_large(self):
        r = self.client.post(
            f"/albums/{self.album_id}/photos",
            files={"photo": ("large.jpg", self.large_photo, "image/jpeg")},
            name="POST /albums/:id/photos [large]",
        )
        if r.status_code == 202:
            pid = r.json().get("photo_id")
            if pid:
                self._poll(pid)

    # ── delete ───────────────────────────────────────────────────────
    @task(2)
    def delete_photo(self):
        if not self.completed_photos:
            return
        pid = self.completed_photos.pop(0)
        self.client.delete(
            f"/albums/{self.album_id}/photos/{pid}",
            name="DELETE /albums/:id/photos/:id",
        )

    # ── helpers ──────────────────────────────────────────────────────
    def _poll(self, photo_id):
        for _ in range(30):
            r = self.client.get(
                f"/albums/{self.album_id}/photos/{photo_id}",
                name="GET /albums/:id/photos/:id [poll]",
            )
            if r.status_code == 200 and r.json().get("status") == "completed":
                self.completed_photos.append(photo_id)
                return
            time.sleep(1)
