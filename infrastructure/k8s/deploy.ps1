# =============================================================================
# Civic Connect – Kubernetes Deployment Script
# =============================================================================
# Usage:
#   .\deploy.ps1                  → Full deploy (build images + apply manifests)
#   .\deploy.ps1 -SkipBuild       → Apply manifests only (images already built)
#   .\deploy.ps1 -Teardown        → Delete all resources
#   .\deploy.ps1 -Status          → Show pod/service status
# =============================================================================
param(
    [switch]$SkipBuild,
    [switch]$Teardown,
    [switch]$Status
)

$ErrorActionPreference = "Stop"
$k8sDir = Split-Path -Parent $MyInvocation.MyCommand.Path
$projectRoot = (Resolve-Path "$k8sDir\..\..").Path

# ── Status Check ────────────────────────────────────────────────────────────
if ($Status) {
    Write-Host "`n=== Pods ===" -ForegroundColor Cyan
    kubectl get pods -n civic-connect -o wide
    Write-Host "`n=== Services ===" -ForegroundColor Cyan
    kubectl get svc -n civic-connect
    Write-Host "`n=== Deployments ===" -ForegroundColor Cyan
    kubectl get deployments -n civic-connect
    Write-Host "`n=== StatefulSets ===" -ForegroundColor Cyan
    kubectl get statefulsets -n civic-connect

    # Show access URLs
    $minikubeIP = minikube ip 2>$null
    if ($minikubeIP) {
        Write-Host "`n=== Access URLs (minikube) ===" -ForegroundColor Green
        Write-Host "  NGINX Gateway:   http://${minikubeIP}:30080"
        Write-Host "  Admin Panel:     http://${minikubeIP}:30000"
        Write-Host "  Admin Service:   http://${minikubeIP}:30081"
        Write-Host "  Content Service: http://${minikubeIP}:30082"
        Write-Host "  Complaint Svc:   http://${minikubeIP}:30083"
        Write-Host "  Chatbot Service: http://${minikubeIP}:30084"
        Write-Host "  PostgreSQL:      ${minikubeIP}:30432"
        Write-Host "  RabbitMQ Mgmt:   http://${minikubeIP}:31672"
        Write-Host "  MinIO Console:   http://${minikubeIP}:30901"
    }
    exit 0
}

# ── Teardown ────────────────────────────────────────────────────────────────
if ($Teardown) {
    Write-Host "Tearing down civic-connect namespace..." -ForegroundColor Red
    kubectl delete namespace civic-connect --ignore-not-found
    Write-Host "Done. All resources deleted." -ForegroundColor Green
    exit 0
}

# ── Build Docker Images ────────────────────────────────────────────────────
if (-not $SkipBuild) {
    Write-Host "`n=== Building Docker Images ===" -ForegroundColor Cyan

    # Point docker CLI to minikube's Docker daemon
    Write-Host "Configuring Docker to use minikube daemon..." -ForegroundColor Yellow
    & minikube -p minikube docker-env --shell powershell | Invoke-Expression

    Write-Host "[1/5] Building admin-service..." -ForegroundColor Yellow
    docker build -t civic-connect/admin-service:latest "$projectRoot\admin-service"

    Write-Host "[2/5] Building content-service..." -ForegroundColor Yellow
    docker build -t civic-connect/content-service:latest -f "$projectRoot\content-service\Dockerfile" "$projectRoot"

    Write-Host "[3/5] Building complaint-service..." -ForegroundColor Yellow
    docker build -t civic-connect/complaint-service:latest "$projectRoot\complaint-service"

    Write-Host "[4/5] Building ai-worker..." -ForegroundColor Yellow
    docker build -t civic-connect/ai-worker:latest -f "$projectRoot\ai-worker\Dockerfile" "$projectRoot"

    Write-Host "[5/5] Building chatbot-service..." -ForegroundColor Yellow
    docker build -t civic-connect/chatbot-service:latest "$projectRoot\chatbot-service"

    Write-Host "[6/6] Building admin-panel..." -ForegroundColor Yellow
    docker build -t civic-connect/admin-panel:latest "$projectRoot\admin-panel"

    Write-Host "All images built successfully." -ForegroundColor Green
}

# ── Apply Kubernetes Manifests ──────────────────────────────────────────────
Write-Host "`n=== Applying Kubernetes Manifests ===" -ForegroundColor Cyan

# 1. Namespace
Write-Host "[1/7] Creating namespace..." -ForegroundColor Yellow
kubectl apply -f "$k8sDir\namespace.yaml"

# 2. Secrets & ConfigMaps
Write-Host "[2/7] Applying secrets and config..." -ForegroundColor Yellow
kubectl apply -f "$k8sDir\secrets.yaml"

# 3. Infrastructure (order matters: postgres first, then others)
Write-Host "[3/7] Deploying PostgreSQL..." -ForegroundColor Yellow
kubectl apply -f "$k8sDir\postgres.yaml"
Write-Host "  Waiting for PostgreSQL to be ready..."
kubectl wait --for=condition=ready pod -l app=postgres -n civic-connect --timeout=120s

Write-Host "[4/7] Deploying RabbitMQ, Redis, MinIO..." -ForegroundColor Yellow
kubectl apply -f "$k8sDir\rabbitmq.yaml"
kubectl apply -f "$k8sDir\redis.yaml"
kubectl apply -f "$k8sDir\minio.yaml"
Write-Host "  Waiting for infrastructure..."
kubectl wait --for=condition=ready pod -l app=rabbitmq -n civic-connect --timeout=120s
kubectl wait --for=condition=ready pod -l app=redis -n civic-connect --timeout=60s
kubectl wait --for=condition=ready pod -l app=minio -n civic-connect --timeout=60s

# 4. Backend Services
Write-Host "[5/7] Deploying backend services..." -ForegroundColor Yellow
kubectl apply -f "$k8sDir\services.yaml"

# 5. NGINX Gateway
Write-Host "[6/7] Deploying NGINX gateway..." -ForegroundColor Yellow
kubectl apply -f "$k8sDir\nginx.yaml"

# 6. Wait for all pods
Write-Host "[7/7] Waiting for all pods to be ready..." -ForegroundColor Yellow
kubectl wait --for=condition=ready pod --all -n civic-connect --timeout=180s

Write-Host "`n=== Deployment Complete ===" -ForegroundColor Green

# Show status
$minikubeIP = minikube ip 2>$null
if ($minikubeIP) {
    Write-Host "`n  NGINX Gateway:   http://${minikubeIP}:30080" -ForegroundColor Cyan
    Write-Host "  Admin Panel:     http://${minikubeIP}:30000" -ForegroundColor Cyan
    Write-Host "  RabbitMQ Mgmt:   http://${minikubeIP}:31672" -ForegroundColor Cyan
    Write-Host "  MinIO Console:   http://${minikubeIP}:30901" -ForegroundColor Cyan
}

Write-Host "`nRun '.\deploy.ps1 -Status' to check pod health." -ForegroundColor White
