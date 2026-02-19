# Civic Connect – Distributed Microservices Platform

Urban governance platform built with a microservices architecture.

## Architecture

| Service             | Tech Stack              | Port  | Database      |
|---------------------|------------------------|-------|---------------|
| admin-service       | Go + Gin + GORM        | 8081  | admin_db      |
| content-service     | Node.js + Express + pg | 8082  | content_db    |
| complaint-service   | Go + Gin + GORM        | 8083  | complaint_db  |
| ai-worker           | Python + pika          | -     | -             |
| chatbot-service     | Python + FastAPI       | 8084  | -             |
| admin-panel         | Vue.js + Vite          | 3000  | -             |

**Infrastructure:** PostgreSQL 16 (PostGIS), RabbitMQ, Redis, MinIO, NGINX

## Quick Start (Docker Compose)

```bash
cd infrastructure/docker
docker compose up --build -d
```

Services will be available at:
- **NGINX Gateway:** http://localhost
- **Admin API:** http://localhost/api/v1/admin/
- **Content API:** http://localhost/api/v1/content/
- **Complaints API:** http://localhost/api/v1/complaints/
- **Chatbot WS:** ws://localhost/ws/
- **RabbitMQ Mgmt:** http://localhost:15672

## Kubernetes (Minikube)

```bash
minikube start
minikube addons enable ingress
kubectl apply -f infrastructure/k8s/
```

## CI/CD

- **GitHub Actions:** `.github/workflows/ci-cd.yml`
- **Jenkins:** `Jenkinsfile`
- **Skaffold:** `skaffold dev --port-forward`
