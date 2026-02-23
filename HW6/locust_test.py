from locust import task, constant, FastHttpUser
import random

class ProductSearchUser(FastHttpUser):
    wait_time = constant(0) 

    @task(20) 
    def search_products(self):
        queries = ["Electronics", "Books", "Home", "Clothing", "Garden", "Alpha", "Beta"]
        query = random.choice(queries)
        self.client.get(f"/products/search?q={query}", name="/products/search")

    @task(1) 
    def get_product(self):
        product_id = random.randint(1, 100000)
        self.client.get(f"/products/{product_id}", name="/products/:id")