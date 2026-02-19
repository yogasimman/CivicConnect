# =============================================================================
# Content Service – Docker Build (Node.js)
# =============================================================================
FROM node:20-alpine

WORKDIR /app
COPY content-service/package*.json ./
RUN npm install --omit=dev
COPY content-service/ .
COPY proto/ /app/proto/

EXPOSE 8082 50051
CMD ["node", "index.js"]
