# =============================================================================
# AI Worker – Docker Build (Python)
# =============================================================================
FROM python:3.12-slim

WORKDIR /app

# Install proto compiler
RUN apt-get update && apt-get install -y --no-install-recommends \
    protobuf-compiler && rm -rf /var/lib/apt/lists/*

COPY ai-worker/requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt

COPY ai-worker/ .
COPY proto/ /app/proto/

# Generate gRPC stubs from both proto files
RUN python -m grpc_tools.protoc \
    -Iproto \
    --python_out=. \
    --grpc_python_out=. \
    proto/summary.proto proto/chat.proto

CMD ["python", "main.py"]
