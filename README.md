# caaspay-api-go

## Overview
CaasPay API is a Go-based backend service designed to handle payment processing, authentication, and other core functionalities. It is built with a modular architecture to ensure scalability and maintainability.

## Features
- Authentication and Authorization (JWT, OAuth, RBAC)
- Rate Limiting and Middleware Support
- Redis-based Message Broker
- Metrics Integration with Datadog
- OpenAPI Specification Support
- RPC Client Pool for Inter-Service Communication

## Setup Instructions

### Prerequisites
- Go 1.20 or later
- Redis (for message broker)
- Datadog (optional, for metrics)

### Installation
1. Clone the repository:
   ```bash
   git clone https://github.com/your-repo/caaspay-api-go.git
   cd caaspay-api-go
   ```

2. Install dependencies:
   ```bash
   go mod tidy
   ```

3. Set up environment variables:
   - Create a `.env` file in the root directory.
   - Add the following variables:
     ```
     REDIS_URL=redis://localhost:6379
     DATADOG_API_KEY=your-datadog-api-key
     ```

4. Run the application:
   ```bash
   go run main.go
   ```

### Testing
Run unit tests:
```bash
go test ./...
```

## Contributing
Contributions are welcome! Please follow the [contribution guidelines](CONTRIBUTING.md) when submitting a pull request.
