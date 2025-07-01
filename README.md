# Deployment Controller

A Go-based microservice that receives application deployment configuration changes from GitHub Actions pipelines and stores them in PostgreSQL with versioning support. It also manages Docker registry credentials for secure image pulling.

## 🚀 Features

- **REST API** for receiving deployment configurations
- **PostgreSQL** with connection pooling (100 connections)
- **Versioning/Revision** tracking for deployments
- **Docker registry credential** management
- **Batch deployment** processing
- **JSON structured logging**
- **Health checks** and graceful shutdown
- **Optional Bearer token authentication**
- **CORS support**

## 📋 Requirements

- Go 1.23+
- PostgreSQL 12+
- Docker (optional)

## 🛠️ Installation & Setup

### 1. Clone the Repository

```bash
git clone <repository-url>
cd deployment-controller
```

### 2. Install Dependencies

```bash
make deps
# or
go mod download
```

### 3. Setup Database

```bash
# Create PostgreSQL database
createdb deployment_controller

# Run schema setup
make db-setup DB_URL=postgres://user:pass@localhost:5432/deployment_controller
```

### 4. Configure Application

Copy and modify the configuration:

```bash
cp config.yaml.example config.yaml
# Edit config.yaml with your database settings
```

### 5. Run the Application

```bash
# Development mode
make dev

# Build and run
make build && make run

# Using Docker Compose
docker-compose up -d
```

## 🔧 Configuration

The application uses YAML configuration (`config.yaml`):

```yaml
database:
  host: localhost
  port: 5432
  user: postgres
  password: password
  name: deployment_controller
  max_conns: 100

server:
  port: 8080
  log_level: info

security:
  bearer_token: "your-secret-token"  # Optional
  encryption_key: "32-character-encryption-key"
```

## 📡 API Endpoints

### Health Check
```
GET /healthz
```

### Deployment Management

#### Push Deployment Changes
```
POST /api/v1/push
Content-Type: application/json

[
  {
    "domain": "app4.poridhi.com",
    "app_name": "order-service",
    "docker_image": "registry.poridhi.com/order-service:latest",
    "port": 3003,
    "env": [
      "NODE_ENV=development",
      "API_URL=https://api.app4.poridhi.com"
    ]
  }
]
```

#### Get All Latest Deployments
```
GET /api/v1/deployments
```

#### Get Specific Deployment
```
GET /api/v1/deployments/{id}
```

#### Update Deployment Status
```
PATCH /api/v1/deployments/{id}/status
Content-Type: application/json

{
  "status": "deployed"  // pending, deploying, deployed, failed, rolled_back
}
```

#### Get Deployment Statistics
```
GET /api/v1/stats
```

### Registry Credential Management

#### Store Registry Credentials
```
POST /api/v1/registry
Content-Type: application/json

{
  "registry": "registry.mycloud.com",
  "username": "docker-user",
  "password": "docker-password"
}
```

#### Get Registry Credentials
```
GET /api/v1/registry?registry=registry.mycloud.com
```

## 🔐 Authentication

Optional Bearer token authentication can be enabled by setting `security.bearer_token` in config:

```
Authorization: Bearer your-secret-token
```

## 📊 Database Schema

### Deployments Table
```sql
CREATE TABLE deployments (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    request_id TEXT NOT NULL,
    domain TEXT NOT NULL,
    app_name TEXT NOT NULL,
    docker_image TEXT NOT NULL,
    port INTEGER NOT NULL,
    env TEXT[] DEFAULT '{}',
    version INTEGER NOT NULL DEFAULT 1,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL,
    deployed_at TIMESTAMP WITH TIME ZONE,
    status TEXT DEFAULT 'pending',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);
```

### Registry Credentials Table
```sql
CREATE TABLE docker_credentials (
    registry TEXT PRIMARY KEY,
    username TEXT NOT NULL,
    password TEXT NOT NULL,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);
```

## 🔄 Versioning System

The service automatically tracks deployment versions:

- Each new deployment for the same `domain` + `app_name` gets a new version number
- Version numbers are automatically incremented
- You can view deployment history through the database
- The `latest_deployments` view shows the most recent version of each app

## 🐳 Docker Usage

### Build Docker Image
```bash
make docker-build
```

### Run with Docker Compose
```bash
# Start all services
docker-compose up -d

# Start with pgAdmin for database management
docker-compose --profile tools up -d

# View logs
docker-compose logs -f deployment-controller
```

### Environment Variables for Docker
```bash
# Override config file location
export CONFIG_PATH=/path/to/config.yaml
```

## 🛠️ Development

### Available Make Commands
```bash
make help           # Show all available commands
make build          # Build the application
make dev            # Run in development mode
make test           # Run tests
make fmt            # Format code
make lint           # Run linter
make watch          # Run with hot reload (requires air)
make clean          # Clean build artifacts
make release        # Build for multiple platforms
```

### Project Structure
```
deployment-controller/
├── cmd/server/           # Application entry point
├── internal/
│   ├── config/          # Configuration management
│   ├── database/        # Database operations
│   ├── handlers/        # HTTP handlers
│   └── models/          # Data models
├── db/                  # Database schema
├── config.yaml          # Configuration file
├── docker-compose.yml   # Docker setup
├── Dockerfile          # Container image
├── Makefile            # Build automation
└── README.md           # This file
```

## 📝 Example Usage

### 1. Start the Service
```bash
docker-compose up -d
```

### 2. Store Registry Credentials
```bash
curl -X POST http://localhost:8080/api/v1/registry \
  -H "Content-Type: application/json" \
  -d '{
    "registry": "registry.poridhi.com",
    "username": "myuser",
    "password": "mypassword"
  }'
```

### 3. Push Deployment Changes
```bash
curl -X POST http://localhost:8080/api/v1/push \
  -H "Content-Type: application/json" \
  -d '[
    {
      "domain": "app1.poridhi.com",
      "app_name": "analytics-dashboard",
      "docker_image": "registry.poridhi.com/analytics-dashboard:v2.0.0",
      "port": 3000,
      "env": [
        "NODE_ENV=production",
        "API_URL=https://api.app1.poridhi.com"
      ]
    }
  ]'
```

### 4. Check Deployment Status
```bash
curl http://localhost:8080/api/v1/deployments
```

### 5. Get Registry Credentials (for agents)
```bash
curl "http://localhost:8080/api/v1/registry?registry=registry.poridhi.com"
```

## 🔍 Monitoring & Logging

The service provides:

- **JSON structured logging** to stdout
- **Health check endpoint** at `/healthz`
- **Request/response logging** with latency tracking
- **Database connection monitoring**

Example log entry:
```json
{
  "time": "2024-01-15T10:30:00Z",
  "level": "INFO",
  "msg": "HTTP Request",
  "method": "POST",
  "path": "/api/v1/push",
  "status": 201,
  "latency": "45.123ms",
  "ip": "192.168.1.100"
}
```

## 🚨 Error Handling

The API returns consistent error responses:

```json
{
  "success": false,
  "error": "Error description",
  "data": null
}
```

Common HTTP status codes:
- `200` - Success
- `201` - Created
- `400` - Bad Request
- `401` - Unauthorized
- `404` - Not Found
- `500` - Internal Server Error

## 🤝 Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests if applicable
5. Run `make fmt` and `make lint`
6. Submit a pull request

## 📄 License

[Your License Here]

## 🆘 Support

For issues and questions:
- Create an issue in the repository
- Check the logs using `docker-compose logs -f`
- Verify database connectivity with `make db-setup`