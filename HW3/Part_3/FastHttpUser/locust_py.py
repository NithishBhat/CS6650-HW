from locust import task, between
# We import FastHttpUser instead of HttpUser
from locust.contrib.fasthttp import FastHttpUser 

class MyServerTest(FastHttpUser):
    wait_time = between(1, 2)

    @task(3)
    def test_get_albums(self):
        self.client.get("/albums")

    @task(1)
    def test_post_albums(self):
        self.client.post("/albums", json={
            "id": "4",
            "title": "A Love Supreme",
            "artist": "John Coltrane",
            "price": 29.99
        })