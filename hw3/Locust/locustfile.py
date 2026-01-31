from locust import HttpUser, task, between

class AlbumUser(HttpUser):
    wait_time = between(1, 3)  # Wait 1-3 seconds between requests
    
    @task(4)  # Runs 4x more often
    def get_all_albums(self):
        """GET /albums - Retrieve all albums"""
        self.client.get("/albums")
    
    @task(2)  # Runs 2x more often
    def get_album_by_id(self):
        """GET /albums/:id - Retrieve specific album"""
        self.client.get("/albums/1")
    
    @task(1)  # Runs 1x
    def create_album(self):
        """POST /albums - Create new album"""
        payload = {
            "id": "4",
            "title": "Kind of Blue",
            "artist": "Miles Davis",
            "price": 45.99
        }
        self.client.post("/albums", json=payload)