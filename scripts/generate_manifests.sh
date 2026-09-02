#!/bin/bash
# generate_manifests.sh - Generate K8s manifests from preflight config

set -euo pipefail

CONFIG_FILE=${1:-/tmp/preflight-config.json}

if [[ ! -f "$CONFIG_FILE" ]]; then
    echo "Error: Config file not found: $CONFIG_FILE"
    exit 1
fi

# Read config
APP_NAME=$(jq -r '.app_name' "$CONFIG_FILE")
NAMESPACE=$(jq -r '.namespace' "$CONFIG_FILE")
DOMAIN=$(jq -r '.domain' "$CONFIG_FILE")
IMAGE_NAME=$(jq -r '.image_name' "$CONFIG_FILE")
TLS_SECRET=$(jq -r '.cluster.tls_secret' "$CONFIG_FILE")
IMAGE_PULL_SECRET=$(jq -r '.cluster.image_pull_secret' "$CONFIG_FILE")
DEPLOY_STRATEGY=$(jq -r '.deploy_strategy' "$CONFIG_FILE")

# Create k8s directory structure
mkdir -p k8s/app

# Generate deployment.yaml
cat > k8s/app/deployment.yaml << EOF
apiVersion: apps/v1
kind: Deployment
metadata:
  name: \${APP_NAME}
  namespace: \${K8S_NAMESPACE}
  labels:
    app: \${APP_NAME}
spec:
  replicas: 1
  selector:
    matchLabels:
      app: \${APP_NAME}
  template:
    metadata:
      labels:
        app: \${APP_NAME}
    spec:
      imagePullSecrets:
        - name: \${IMAGE_PULL_SECRET}
      containers:
        - name: \${APP_NAME}
          image: \${IMAGE_DIGEST}
          ports:
            - containerPort: 8080
          env:
            - name: PORT
              value: "8080"
          resources:
            limits:
              cpu: "500m"
              memory: "256Mi"
            requests:
              cpu: "100m"
              memory: "64Mi"
EOF

# Generate service.yaml
cat > k8s/app/service.yaml << EOF
apiVersion: v1
kind: Service
metadata:
  name: \${APP_NAME}
  namespace: \${K8S_NAMESPACE}
  labels:
    app: \${APP_NAME}
spec:
  ports:
    - port: 8080
      targetPort: 8080
      protocol: TCP
      name: http
  selector:
    app: \${APP_NAME}
EOF

# Generate ingress-http.yaml
cat > k8s/app/ingress-http.yaml << EOF
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: \${APP_NAME}-http
  namespace: \${K8S_NAMESPACE}
  annotations:
    traefik.ingress.kubernetes.io/router.entrypoints: web
    traefik.ingress.kubernetes.io/router.middlewares: \${K8S_NAMESPACE}-redirect-https@kubernetescrd
spec:
  rules:
    - host: \${DOMAIN}
      http:
        paths:
          - path: /
            pathType: Prefix
            backend:
              service:
                name: \${APP_NAME}
                port:
                  number: 8080
EOF

# Generate ingress-https.yaml
cat > k8s/app/ingress-https.yaml << EOF
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: \${APP_NAME}
  namespace: \${K8S_NAMESPACE}
  annotations:
    traefik.ingress.kubernetes.io/router.entrypoints: websecure
    traefik.ingress.kubernetes.io/router.tls: "true"
spec:
  rules:
    - host: \${DOMAIN}
      http:
        paths:
          - path: /
            pathType: Prefix
            backend:
              service:
                name: \${APP_NAME}
                port:
                  number: 8080
  tls:
    - hosts:
        - \${DOMAIN}
      secretName: \${TLS_SECRET}
EOF

echo "Generated manifests in k8s/app/"
