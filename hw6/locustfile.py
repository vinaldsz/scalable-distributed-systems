"""
Locust load tests for the Product Search Service.

Test 1 — Baseline:      locust -f locustfile.py --users 5  --spawn-rate 1 --run-time 2m
Test 2 — Breaking Point: locust -f locustfile.py --users 20 --spawn-rate 2 --run-time 3m

Or open the Locust web UI (default http://localhost:8089) and configure interactively.
"""

import random
from locust import task, between
from locust.contrib.fasthttp import FastHttpUser

# ── Search terms that will produce varied but consistent workloads ─────────────
SEARCH_TERMS = [
    "Electronics",
    "Alpha",
    "Books",
    "Pro",
    "Sports",
    "Beta",
    "Home",
    "Ultra",
    "Gamma",
    "Clothing",
]


class ProductSearchUser(FastHttpUser):
    """
    Simulates a user continuously hitting the search endpoint.
    wait_time=(0.1, 0.3) keeps pressure high so CPU is stressed quickly.
    """
    wait_time = between(0.1, 0.3)  # seconds between requests

    @task(10)
    def search_common_term(self):
        """Search for a randomly chosen common term."""
        term = random.choice(SEARCH_TERMS)
        with self.client.get(
            f"/products/search?q={term}",
            name="/products/search",
            catch_response=True,
        ) as resp:
            if resp.status_code == 200:
                resp.success()
            else:
                resp.failure(f"Unexpected status {resp.status_code}")

    @task(2)
    def search_brand(self):
        """Occasionally search by a specific brand name."""
        brands = ["Alpha", "Beta", "Delta", "Echo"]
        term = random.choice(brands)
        with self.client.get(
            f"/products/search?q={term}",
            name="/products/search [brand]",
            catch_response=True,
        ) as resp:
            if resp.status_code == 200:
                resp.success()
            else:
                resp.failure(f"Unexpected status {resp.status_code}")

    @task(1)
    def health_check(self):
        """Lightweight health probe — should always be fast."""
        self.client.get("/health", name="/health")