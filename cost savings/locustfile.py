from locust import between, task
from locust.contrib.fasthttp import FastHttpUser
import random


class SyncOrderUser(FastHttpUser):
    wait_time = between(0.1, 0.5)

    @task(10)
    def create_sync_order(self):
        payload = {
            "customer_id": random.randint(1, 100000),
            "items": [
                {
                    "product_id": "sku-101",
                    "name": "Wireless Mouse",
                    "quantity": 1,
                    "price": 29.99,
                }
            ],
        }

        with self.client.post(
            "/orders/sync",
            json=payload,
            name="POST /orders/sync",
            catch_response=True,
            timeout=15,
        ) as response:
            if response.status_code == 200:
                response.success()
            else:
                response.failure(f"Unexpected status: {response.status_code}")

    @task(1)
    def health(self):
        self.client.get("/health", name="GET /health")


class AsyncOrderUser(FastHttpUser):
    wait_time = between(0.1, 0.5)

    @task(10)
    def create_async_order(self):
        payload = {
            "customer_id": random.randint(1, 100000),
            "items": [
                {
                    "product_id": "sku-101",
                    "name": "Wireless Mouse",
                    "quantity": 1,
                    "price": 29.99,
                }
            ],
        }

        with self.client.post(
            "/orders/async",
            json=payload,
            name="POST /orders/async",
            catch_response=True,
            timeout=5,
        ) as response:
            if response.status_code == 202:
                response.success()
            else:
                response.failure(f"Unexpected status: {response.status_code}")

    @task(1)
    def health(self):
        self.client.get("/health", name="GET /health")
