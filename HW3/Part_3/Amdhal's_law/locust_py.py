from locust import HttpUser, task, between

class MyServerTest(HttpUser):
    wait_time = between(1, 2)

    # weight=3 for the 3:1 ratio 
    @task(3)
    def test_get_albums(self):
        self.client.get("/albums")

    # weight=1
    @task(1)
    def test_post_albums(self):
        self.client.post("/albums", json={
            "id": "4",
            "title": "A Love Supreme",
            "artist": "John Coltrane",
            "price": 29.99
        })