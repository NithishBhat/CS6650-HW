from locust import HttpUser, task, between

class ProductUser(HttpUser):
    wait_time = between(1, 3)

    @task
    def get_product(self):
        # Testing the endpoint you just verified
        self.client.get("/products/12345")