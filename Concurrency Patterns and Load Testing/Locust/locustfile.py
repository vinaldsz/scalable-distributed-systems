from locust import FastHttpUser, task, between

class AlbumsUser(FastHttpUser):
    # Wait 1-2 seconds between requests (simulates real user behavior)
    wait_time = between(1, 2)
    
    # Base URL of your server
    host = "http://52.24.162.55:8080"  # Replace with YOUR actual EC2 IP
    
    @task(3)  # Weight 3 - runs 3x more than POST
    def get_albums(self):
        """GET all albums"""
        self.client.get("/albums")
    
    @task(1)  # Weight 1 - runs less frequently
    def post_album(self):
        """POST new album"""
        self.client.post("/albums", json={
            "id": "4",
            "title": "The Modern Sound of Betty Carter",
            "artist": "Betty Carter",
            "price": 49.99
        })