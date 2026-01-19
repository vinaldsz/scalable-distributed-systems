# Homework 1b: AWS EC2 Deployment and Performance Testing

This assignment focuses on deploying the Go-Gin RESTful API from Homework 1a to AWS EC2 and performing load testing to analyze performance characteristics.

## Table of Contents

- [Part I: AWS Setup and EC2 Deployment](#part-i-aws-setup-and-ec2-deployment)
- [Part II: Learning Outcomes](#part-ii-learning-outcomes)
- [Part III: Performance Testing](#part-iii-performance-testing)
- [Troubleshooting](#troubleshooting)

---

## Part I: AWS Setup and EC2 Deployment

### Step I - Setup AWS Learner Account

Follow the AWS Academy Learner Lab - Student Guide. The login link is: https://awsacademy.instructure.com/

**Important:** Be aware of your budget and always clean up resources you don't need when you're finished.

### Step II - Create EC2 Instance

1. Change the region on the top right to **Oregon (us-west-2)**
2. Search for **EC2** in the console search bar
3. Click **Launch an instance**

**Recommended Configuration:**

- **OS:** Amazon Linux 2023
- **Instance Type:** t2.micro (free tier eligible)
- **Key Pair:** Create a new RSA .pem key pair (or .ppk for PuTTY)
- **Security Group:** Create one that allows SSH traffic from your IP
- **Advanced Details → IAM Instance Profile:** LabInstanceProfile
- Click **Launch Instance**

4. Once your instance is running, select it and click **Connect**
5. Use the provided SSH command to establish a connection. Ensure:
   - Your .pem key is in the correct directory
   - Permissions are set correctly (typically `chmod 400 <key>.pem`)

### Step III - Cross Compile Your Go-Gin Code

Before deployment, you need to cross-compile your Go code for Linux. This is done on your local machine.

**Prerequisites:**

- Update your Go code to bind to `0.0.0.0:8080` instead of `localhost:8080`
- Open a terminal/bash shell in your Go project directory (where `main.go` is located)

**Cross-Compile Command:**

```bash
GOOS=linux GOARCH=amd64 go build -o <your-filename> main.go
```

**Explanation:**

- `GOOS=linux` - Sets target OS to Linux
- `GOARCH=amd64` - Sets target architecture to amd64 (use `arm64` if `amd64` fails on AWS)
- `-o <your-filename>` - Output executable name (e.g., `newmain`)

For a list of available OS and architecture options, see the [Go Build documentation](https://pkg.go.dev/cmd/go).

### Step IV - Upload Binary to EC2 and Run

#### Configure Security Group for Port 8080

1. Go to EC2 service in AWS console
2. Select your running instance
3. Scroll down to "Security groups" and click on your security group
4. Click on the "Inbound rules" tab
5. Add a new inbound rule:
   - **Type:** Custom TCP
   - **Port:** 8080
   - **Source:** Your IP (or `0.0.0.0/0` to allow from anywhere)

#### Upload and Run Your Binary

1. SSH into your EC2 instance
   `ssh -i <path to .pem file> ec2-user:<EC2_IP>`

2. Create a directory and set permissions:

```bash
mkdir <your_dir_name>
chmod -R 777 <your_dir_name>
```

3. From your local machine, use SCP to copy the binary:

```bash
scp -i <path-to-pem-file> <path-to-executable> ec2-user@<EC2_IP>:<remote-folder>
```

Example:

```bash
scp -i ~/.aws/my-key.pem ./newmain ec2-user@54.123.45.67:/home/ec2-user/myapp/
```

4. SSH back into your EC2 instance and run the executable:

```bash
cd /home/ec2-user/myapp/
./<your-filename>
```

If you get permission denied, run:

```bash
chmod -R 777 <your-filename>
```

#### Test Your Deployment

Test your server using curl from your local machine or within the EC2 instance:

```bash
# From your local machine
curl -v http://<your-public-ip>:8080/albums

# From within EC2 instance
curl localhost:8080/albums
```

---

## Usage

Once the server is running, you can interact with the API using curl, Postman, or any HTTP client.

### Example Requests

Get all albums:

```bash
curl http://localhost:8080/albums
```

Get a specific album by ID:

```bash
curl http://localhost:8080/albums/1
```

Add a new album:

```bash
curl http://localhost:8080/albums \
    --include \
    --header "Content-Type: application/json" \
    --request "POST" \
    --data '{"id": "4","title": "The Modern Sound of Betty Carter","artist": "Betty Carter","price": 49.99}'
```

## API Endpoints

| Method | Endpoint      | Description                     |
| ------ | ------------- | ------------------------------- |
| GET    | `/albums`     | Retrieve all albums             |
| GET    | `/albums/:id` | Retrieve a specific album by ID |
| POST   | `/albums`     | Create a new album              |

### Request/Response Format

Albums are represented as JSON objects:

```json
{
  "id": "1",
  "title": "Blue Train",
  "artist": "John Coltrane",
  "price": 56.99
}
```

## Part III: Performance Testing and Understanding Response Times

Now that your Go web service is running in the cloud, investigate its performance characteristics under load.

### Step 1: Create a Load Testing Script

Create a Python script to analyze response time characteristics of your EC2-hosted service.

**Requirements:**

- Duration: 30 seconds (or longer for more data)
- Sends GET requests to your EC2 instance's public IP
- Records response time for each request
- Visualizes results in histogram and scatter plot

**Script Template:**

Write a script to test the load to EC2 machine.

### Step 2: Install Required Dependencies

Before running the script, install the required Python packages:

```bash
pip install requests matplotlib numpy
```

### Step 3: Run the Load Test

1. Update the `EC2_URL` variable with your actual EC2 public IP address
2. Run the script:

```bash
python performanceTest.py
```

---

## Troubleshooting

### Permission Denied Errors

If you receive permission errors when uploading or running files:

```bash
sudo chmod -R 777 <directory-or-filename>
```

### Executable Won't Run on EC2

Try recompiling with a different architecture:

```bash
# Try arm64 if amd64 fails
GOOS=linux GOARCH=arm64 go build -o <your-filename> main.go
```

### Can't Connect to Server

- Verify your security group allows inbound traffic on port 8080
- Check that your URL and public IP address are correct
- Ensure the server is running with `./<your-filename>`

### IP Address Changes

- Keep your security group updated if your local IP changes
- Consider using a security group rule with `0.0.0.0/0` for testing (not recommended for production)
- For persistent access, research AWS Elastic IP addresses

---

## Project Structure

```
.
├── performanceTest.py       # Load testing script (Part III)
├── newmain                  # Cross-compiled binary for Linux
└── README.md                # This file
```

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.
