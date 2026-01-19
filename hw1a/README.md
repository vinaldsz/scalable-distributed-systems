# RESTful-api-Go-Gin

A demonstration of building a RESTful web service using Go and the Gin Web Framework.

## Table of Contents

- [Features](#features)
- [Prerequisites](#prerequisites)
- [Installation](#installation)
- [Usage](#usage)
- [API Endpoints](#api-endpoints)
- [Project Structure](#project-structure)
- [License](#license)

## Features

- RESTful API for managing album records
- Built with Go and Gin Web Framework
- JSON request/response handling
- GET, POST endpoints for albums
- In-memory data storage

## Prerequisites

- Go 1.16 or higher
- Git

## Installation

1. Clone the repository:

```bash
git clone https://github.com/vinaldsouza/RESTful-api-Go-Gin.git
cd RESTful-api-Go-Gin
```

2. Install dependencies:

```bash
go mod download
```

3. Run the application:

```bash
go run main.go
```

The server will start on `localhost:8080`

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

## Project Structure

```
.
├── main.go          # Main application file with endpoints
├── go.mod           # Go module definition
├── go.sum           # Go dependencies checksums
└── README.md        # This file
```

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.
