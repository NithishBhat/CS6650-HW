from locust import HttpUser, task, between
import random
import json


class SyncUser(HttpUser):
    """Phase 1: Test synchronous endpoint.
    
    Normal test:  locust -f locustfile.py --users 5 --spawn-rate 1 --run-time 30s --host http://YOUR-ALB SyncUser
    Flash sale:   locust -f locustfile.py --users 20 --spawn-rate 10 --run-time 60s --host http://YOUR-ALB SyncUser
    """
    wait_time = between(0.1, 0.5)

    @task
    def place_sync_order(self):
        payload = {
            "customer_id": random.randint(1, 1000),
            "items": [
                {"product_id": random.randint(1, 100000), "quantity": random.randint(1, 5)}
                for _ in range(random.randint(1, 3))
            ]
        }
        self.client.post("/orders/sync", json=payload, name="/orders/sync")


class AsyncUser(HttpUser):
    """Phase 3+: Test asynchronous endpoint.
    
    Flash sale:   locust -f locustfile.py --users 20 --spawn-rate 10 --run-time 60s --host http://YOUR-ALB AsyncUser
    """
    wait_time = between(0.1, 0.5)

    @task
    def place_async_order(self):
        payload = {
            "customer_id": random.randint(1, 1000),
            "items": [
                {"product_id": random.randint(1, 100000), "quantity": random.randint(1, 5)}
                for _ in range(random.randint(1, 3))
            ]
        }
        self.client.post("/orders/async", json=payload, name="/orders/async")
