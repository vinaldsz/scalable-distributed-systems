"""
Comprehensive Load Testing Suite for Product API Server
========================================================

This file demonstrates load testing using both HttpUser (standard) and 
FastHttpUser (optimized) to compare performance differences.

Key Concepts:
- HttpUser: Uses standard urllib3 (blocking, creates new connection per request)
- FastHttpUser: Uses gevent+socketpool (non-blocking, connection pooling)

Real-world scenario: Product API is read-heavy (80% GETs, 20% POSTs)
Since reads use RWMutex, multiple concurrent reads should be handled well.
Writes are exclusive locks - these create contention!
"""

from locust import HttpUser, FastHttpUser, task, between, constant, events
import random
import json
import time


# ============================================================================
# SCENARIO 1: Read-Heavy Workload (Realistic E-commerce)
# ============================================================================
class ReadHeavyHttpUser(HttpUser):
    """Standard HttpUser with read-heavy workload (80% GET, 20% POST)"""
    
    wait_time = between(0.5, 1.5)  # User thinks between requests
    
    def on_start(self):
        """Called when user starts"""
        # Pre-populate some products
        for i in range(1, 11):
            self.post_product(i)
    
    @task(80)  # 80% of requests are GETs
    def get_product(self):
        """GET a random product"""
        product_id = random.randint(1, 10)
        with self.client.get(f"/products/{product_id}", catch_response=True) as response:
            if response.status_code == 200:
                response.success()
            else:
                response.failure(f"Got status {response.status_code}")
    
    @task(20)  # 20% of requests are POSTs
    def post_product(self, product_id=None):
        """POST product details"""
        if product_id is None:
            product_id = random.randint(1, 10)
        
        payload = {
            "product_id": product_id,
            "sku": f"SKU-{product_id:04d}",
            "manufacturer": f"Manufacturer-{random.randint(1, 5)}",
            "category_id": random.randint(1, 10),
            "weight": random.randint(100, 5000),
            "some_other_id": random.randint(1, 100)
        }
        
        with self.client.post(f"/products/{product_id}/details", 
                              json=payload, 
                              catch_response=True) as response:
            if response.status_code == 204:
                response.success()
            else:
                response.failure(f"Got status {response.status_code}")


class ReadHeavyFastHttpUser(FastHttpUser):
    """FastHttpUser with read-heavy workload (80% GET, 20% POST)
    
    FastHttpUser advantages:
    - Connection pooling (reuses connections)
    - Gevent-based (non-blocking)
    - Lower latency on high concurrency
    """
    
    wait_time = between(0.5, 1.5)
    
    def on_start(self):
        """Called when user starts"""
        for i in range(1, 11):
            self.post_product(i)
    
    @task(80)
    def get_product(self):
        """GET a random product"""
        product_id = random.randint(1, 10)
        with self.client.get(f"/products/{product_id}", catch_response=True) as response:
            if response.status_code == 200:
                response.success()
            else:
                response.failure(f"Got status {response.status_code}")
    
    @task(20)
    def post_product(self, product_id=None):
        """POST product details"""
        if product_id is None:
            product_id = random.randint(1, 10)
        
        payload = {
            "product_id": product_id,
            "sku": f"SKU-{product_id:04d}",
            "manufacturer": f"Manufacturer-{random.randint(1, 5)}",
            "category_id": random.randint(1, 10),
            "weight": random.randint(100, 5000),
            "some_other_id": random.randint(1, 100)
        }
        
        with self.client.post(f"/products/{product_id}/details", 
                              json=payload, 
                              catch_response=True) as response:
            if response.status_code == 204:
                response.success()
            else:
                response.failure(f"Got status {response.status_code}")


# ============================================================================
# SCENARIO 2: Write-Heavy Workload (Testing lock contention)
# ============================================================================
class WriteHeavyHttpUser(HttpUser):
    """Heavy write workload (20% GET, 80% POST) - tests mutex contention"""
    
    wait_time = between(0.5, 1.5)
    
    @task(20)
    def get_product(self):
        """GET a random product"""
        product_id = random.randint(1, 100)
        with self.client.get(f"/products/{product_id}", catch_response=True) as response:
            if response.status_code in [200, 404]:
                response.success()
            else:
                response.failure(f"Got status {response.status_code}")
    
    @task(80)
    def post_product(self):
        """POST product details"""
        product_id = random.randint(1, 100)
        
        payload = {
            "product_id": product_id,
            "sku": f"SKU-{product_id:04d}",
            "manufacturer": f"Manufacturer-{random.randint(1, 5)}",
            "category_id": random.randint(1, 10),
            "weight": random.randint(100, 5000),
            "some_other_id": random.randint(1, 100)
        }
        
        with self.client.post(f"/products/{product_id}/details", 
                              json=payload, 
                              catch_response=True) as response:
            if response.status_code == 204:
                response.success()
            else:
                response.failure(f"Got status {response.status_code}")


class WriteHeavyFastHttpUser(FastHttpUser):
    """Heavy write workload with FastHttpUser"""
    
    wait_time = between(0.5, 1.5)
    
    @task(20)
    def get_product(self):
        """GET a random product"""
        product_id = random.randint(1, 100)
        with self.client.get(f"/products/{product_id}", catch_response=True) as response:
            if response.status_code in [200, 404]:
                response.success()
            else:
                response.failure(f"Got status {response.status_code}")
    
    @task(80)
    def post_product(self):
        """POST product details"""
        product_id = random.randint(1, 100)
        
        payload = {
            "product_id": product_id,
            "sku": f"SKU-{product_id:04d}",
            "manufacturer": f"Manufacturer-{random.randint(1, 5)}",
            "category_id": random.randint(1, 10),
            "weight": random.randint(100, 5000),
            "some_other_id": random.randint(1, 100)
        }
        
        with self.client.post(f"/products/{product_id}/details", 
                              json=payload, 
                              catch_response=True) as response:
            if response.status_code == 204:
                response.success()
            else:
                response.failure(f"Got status {response.status_code}")


# ============================================================================
# SCENARIO 3: Balanced Workload (50% GET, 50% POST)
# ============================================================================
class BalancedHttpUser(HttpUser):
    """Balanced workload - equal reads and writes"""
    
    wait_time = constant(0.1)  # Minimal wait - aggressive load
    
    @task(50)
    def get_product(self):
        """GET a random product"""
        product_id = random.randint(1, 50)
        with self.client.get(f"/products/{product_id}", catch_response=True) as response:
            if response.status_code in [200, 404]:
                response.success()
            else:
                response.failure(f"Got status {response.status_code}")
    
    @task(50)
    def post_product(self):
        """POST product details"""
        product_id = random.randint(1, 50)
        
        payload = {
            "product_id": product_id,
            "sku": f"SKU-{product_id:04d}",
            "manufacturer": f"Manufacturer-{random.randint(1, 5)}",
            "category_id": random.randint(1, 10),
            "weight": random.randint(100, 5000),
            "some_other_id": random.randint(1, 100)
        }
        
        with self.client.post(f"/products/{product_id}/details", 
                              json=payload, 
                              catch_response=True) as response:
            if response.status_code == 204:
                response.success()
            else:
                response.failure(f"Got status {response.status_code}")


class BalancedFastHttpUser(FastHttpUser):
    """Balanced workload with FastHttpUser"""
    
    wait_time = constant(0.1)
    
    @task(50)
    def get_product(self):
        """GET a random product"""
        product_id = random.randint(1, 50)
        with self.client.get(f"/products/{product_id}", catch_response=True) as response:
            if response.status_code in [200, 404]:
                response.success()
            else:
                response.failure(f"Got status {response.status_code}")
    
    @task(50)
    def post_product(self):
        """POST product details"""
        product_id = random.randint(1, 50)
        
        payload = {
            "product_id": product_id,
            "sku": f"SKU-{product_id:04d}",
            "manufacturer": f"Manufacturer-{random.randint(1, 5)}",
            "category_id": random.randint(1, 10),
            "weight": random.randint(100, 5000),
            "some_other_id": random.randint(1, 100)
        }
        
        with self.client.post(f"/products/{product_id}/details", 
                              json=payload, 
                              catch_response=True) as response:
            if response.status_code == 204:
                response.success()
            else:
                response.failure(f"Got status {response.status_code}")


# ============================================================================
# Event Listeners for metrics collection
# ============================================================================
@events.test_start.add_listener
def on_test_start(environment, **kwargs):
    print("\n" + "="*70)
    print("LOAD TEST STARTED")
    print("="*70)
    print(f"Target: {environment.host}")
    print(f"Timestamp: {time.strftime('%Y-%m-%d %H:%M:%S')}")
    print("="*70 + "\n")


@events.test_stop.add_listener
def on_test_stop(environment, **kwargs):
    print("\n" + "="*70)
    print("LOAD TEST COMPLETED - Analyzing Results...")
    print("="*70)
    
    # Get stats
    stats = environment.stats
    print(f"\nTotal Requests: {stats.total.num_requests}")
    print(f"Failed Requests: {stats.total.num_failures}")
    print(f"Median Response Time: {stats.total.median_response_time}ms")
    print(f"95th Percentile: {stats.total.get_response_time_percentile(0.95)}ms")
    print(f"99th Percentile: {stats.total.get_response_time_percentile(0.99)}ms")
    print(f"Requests/sec: {stats.total.total_rps:.2f}")
    print("="*70 + "\n")
