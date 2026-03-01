#!/usr/bin/env python3
"""
HW7 Load Testing: Cascading Failure vs Bulkhead Pattern
Demonstrates how slow dependencies can crash the entire system
"""

from locust import HttpUser, task, between, events
import json
import time
import logging
import random

# Setup logging
logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

# Metrics tracking
class Metrics:
    def __init__(self):
        self.total_requests = 0
        self.failed_requests = 0
        self.cascading_failures = 0  # Requests that timed out waiting for bulkhead
        self.max_response_time = 0
        self.min_response_time = float('inf')
        self.response_times = []

metrics = Metrics()

class SearchUser(HttpUser):
    """
    Simulates users searching for products
    Tests the search service with/without bulkhead protection
    """
    
    # Wait 100ms - 500ms between requests
    wait_time = between(0.1, 0.5)
    
    # Common search queries
    search_queries = [
        "laptop",
        "electronics",
        "book",
        "home",
        "sports",
        "clothing",
        "ultra",
        "pro",
        "max",
        "smart"
    ]
    
    @task(3)
    def search_product(self):
        """Search for a product (most common task)"""
        query = random.choice(self.search_queries)
        
        with self.client.get(
            f"/products/search?q={query}",
            catch_response=True
        ) as response:
            try:
                if response.status_code == 200:
                    data = response.json()
                    metrics.total_requests += 1
                    
                    # Track response time
                    resp_time = response.elapsed.total_seconds() * 1000
                    metrics.response_times.append(resp_time)
                    metrics.max_response_time = max(metrics.max_response_time, resp_time)
                    metrics.min_response_time = min(metrics.min_response_time, resp_time)
                    
                    # Log if bulkhead was full (no recommendations returned)
                    if not data.get("recommendations"):
                        metrics.cascading_failures += 1
                        logger.warning(f"[CASCADING] Query={query} | Time={resp_time:.0f}ms | Goroutines={data.get('active_goroutines')}")
                    
                    response.success()
                else:
                    metrics.failed_requests += 1
                    logger.error(f"[FAIL] Search {query}: HTTP {response.status_code} | Body: {response.text[:100]}")
                    response.failure(f"HTTP {response.status_code}")
            except Exception as e:
                metrics.failed_requests += 1
                logger.error(f"[ERROR] Search {query}: {e} | Response len: {len(response.text)} | Status: {response.status_code}")
                response.failure(f"Error: {e}")
    
    @task(1)
    def health_check(self):
        """Health check endpoint (should always respond quickly)"""
        with self.client.get("/health", catch_response=True, timeout=2) as response:
            if response.status_code == 200:
                response.success()
            else:
                logger.error(f"[FAIL] Health: HTTP {response.status_code} | Body: {response.text[:100]}")
                response.failure(f"Health check failed: {response.status_code}")
    
    @task(1)
    def check_metrics(self):
        """Check active goroutine count"""
        with self.client.get("/metrics", catch_response=True) as response:
            try:
                if response.status_code == 200:
                    data = response.json()
                    goroutines = data.get("active_goroutines", 0)
                    if goroutines > 50:  # Alert if goroutines spike
                        logger.warning(f"[SPIKE] Active goroutines: {goroutines}")
                    response.success()
                else:
                    logger.error(f"[FAIL] Metrics: HTTP {response.status_code} | Body: {response.text[:100]}")
                    response.failure(f"Metrics request failed: {response.status_code}")
            except Exception as e:
                logger.error(f"[ERROR] Metrics parse failed: {e} | Status: {response.status_code} | Body: {response.text[:100]}")
                response.failure(f"Metrics parse error: {e}")


@events.test_start.add_listener
def on_test_start(environment, **kwargs):
    """Called when load test starts"""
    logger.info("=" * 80)
    logger.info("HW7: Cascading Failure & Bulkhead Recovery Load Test")
    logger.info("=" * 80)
    logger.info("Testing: Product search with slow recommendation service")
    logger.info("Metrics: Latency, throughput, goroutine accumulation")
    
    # Diagnostic: Check if service is reachable before test starts
    logger.info("\nPre-test connectivity check...")
    try:
        response = environment.client.get("/health", timeout=2)
        logger.info(f"✓ Health endpoint: HTTP {response.status_code} | Body: {response.text[:100]}")
        if response.status_code != 200:
            logger.warning(f"⚠ Warning: Health check returned non-200 status!")
    except Exception as e:
        logger.error(f"✗ Cannot connect to health endpoint: {e}")
        logger.error(f"  Check: ALB URL, network connectivity, service deployment")
    logger.info("Starting load test...\n")


@events.test_stop.add_listener
def on_test_stop(environment, **kwargs):
    """Called when load test completes"""
    logger.info("=" * 80)
    logger.info("LOAD TEST SUMMARY")
    logger.info("=" * 80)
    
    if metrics.response_times:
        rates = sorted(metrics.response_times)
        p50 = rates[int(len(rates) * 0.5)]
        p95 = rates[int(len(rates) * 0.95)]
        p99 = rates[int(len(rates) * 0.99)]
        
        logger.info(f"Total Requests: {metrics.total_requests}")
        logger.info(f"Failed Requests: {metrics.failed_requests}")
        logger.info(f"Cascading Failures (no recommendations): {metrics.cascading_failures}")
        logger.info(f"Success Rate: {(metrics.total_requests - metrics.failed_requests) / metrics.total_requests * 100:.1f}%")
        logger.info(f"Response Time - Min: {metrics.min_response_time:.0f}ms, P50: {p50:.0f}ms, P95: {p95:.0f}ms, P99: {p99:.0f}ms, Max: {metrics.max_response_time:.0f}ms")
    else:
        logger.info("No successful search responses were recorded.")
        logger.info("Likely cause: connectivity/host issue (HTTP 0) or all requests failed.")
        logger.info("Check metrics/*_failures.csv and verify --host with curl /health before rerunning.")
    
    logger.info("=" * 80)
