# Kubernetes Deployment Guide for Pizza API

This guide provides step-by-step instructions for deploying the Pizza API to a local microk8s cluster with HTTPS enabled.

## Prerequisites

Before deploying, ensure you have the following installed:

- **microk8s** - Local Kubernetes cluster
  ```bash
  sudo snap install microk8s --classic
  sudo usermod -a -G microk8s $USER
  sudo chown -f -R $USER ~/.kube
  newgrp microk8s
  ```

- **kubectl** - Kubernetes CLI (included with microk8s)
  ```bash
  microk8s kubectl version
  # Or create an alias: alias kubectl='microk8s kubectl'
  ```

- **Docker** - For building the container image
  ```bash
  sudo apt-get install docker.io
  sudo usermod -aG docker $USER
  ```

## Step 1: Enable microk8s Addons

Enable the required microk8s addons:

```bash
# Enable DNS for service discovery
microk8s enable dns

# Enable Ingress for HTTPS access
microk8s enable ingress

# Verify addons are running
microk8s status
```

## Step 2: Build the Docker Image

Build the Pizza API Docker image:

```bash
# From the project root directory
docker build -t pizza-api:latest .

# Import the image into microk8s
docker save pizza-api:latest | microk8s ctr image import -

# Verify the image is available
microk8s ctr images ls | grep pizza-api
```

## Step 3: Create TLS Certificate for Local Development

Generate a self-signed TLS certificate for `pizza-api.local`:

```bash
# Generate private key and certificate
openssl req -x509 -nodes -days 365 -newkey rsa:2048 \
  -keyout tls.key \
  -out tls.crt \
  -subj "/CN=pizza-api.local/O=Pizza API Demo"

# Create Kubernetes TLS secret
microk8s kubectl create secret tls pizza-api-tls \
  --cert=tls.crt \
  --key=tls.key

# Verify secret was created
microk8s kubectl get secret pizza-api-tls

# Clean up local files (optional)
rm tls.key tls.crt
```

## Step 4: Deploy the Application

Apply all Kubernetes manifests:

```bash
# Apply all manifests in the k8s/ directory
microk8s kubectl apply -f k8s/

# Verify all resources are created
microk8s kubectl get all

# Watch pods until they're ready (may take 30-60 seconds)
microk8s kubectl get pods -w
```

Expected output:
```
NAME                         READY   STATUS    RESTARTS   AGE
pod/pizza-api-xxxxxxxxx-xxx  1/1     Running   0          30s
pod/pizza-api-xxxxxxxxx-xxx  1/1     Running   0          30s

NAME                        TYPE        CLUSTER-IP      EXTERNAL-IP   PORT(S)
service/pizza-api-service   ClusterIP   10.152.183.xx   <none>        80/TCP

NAME                        READY   UP-TO-DATE   AVAILABLE   AGE
deployment.apps/pizza-api   2/2     2            2           30s
```

## Step 5: Configure Local DNS

Add an entry to your `/etc/hosts` file to resolve `pizza-api.local` to localhost:

```bash
# Add this line to /etc/hosts (requires sudo)
echo "127.0.0.1 pizza-api.local" | sudo tee -a /etc/hosts

# Verify the entry
cat /etc/hosts | grep pizza-api
```

## Step 6: Test the Deployment

### Test Health Endpoint

```bash
# Test HTTPS health endpoint (use -k to skip certificate validation)
curl -k https://pizza-api.local/health

# Expected output: OK
```

### Test OAuth Token Endpoint

```bash
# Test OAuth token endpoint
curl -k https://pizza-api.local/api/v1/oauth/token
```

### Create Development OAuth Client

First, create a development OAuth client by running the setup script inside a pod:

```bash
# Copy the dev client script to a pod
POD_NAME=$(microk8s kubectl get pods -l app=pizza-api -o jsonpath='{.items[0].metadata.name}')
microk8s kubectl cp scripts/create_dev_client.go $POD_NAME:/tmp/create_dev_client.go

# Execute the script
microk8s kubectl exec $POD_NAME -- go run /tmp/create_dev_client.go

# Or manually via curl:
curl -k -X POST https://pizza-api.local/api/v1/clients \
  -H "Content-Type: application/json" \
  -d '{
    "client_id": "dev-client",
    "client_secret": "dev-secret-123",
    "user_id": 1
  }'
```

### Test Complete OAuth Flow

```bash
# Get OAuth token
TOKEN=$(curl -sk -X POST https://pizza-api.local/api/v1/oauth/token \
  -d "grant_type=client_credentials" \
  -d "client_id=dev-client" \
  -d "client_secret=dev-secret-123" | grep -o '"access_token":"[^"]*"' | cut -d'"' -f4)

echo "Token: $TOKEN"

# Create a pizza using the token
curl -k -X POST https://pizza-api.local/api/v1/pizzas \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Kubernetes Margherita",
    "description": "Deployed via Kubernetes",
    "ingredients": ["mozzarella", "tomato", "basil"],
    "price": 12.99
  }'

# List all pizzas
curl -k https://pizza-api.local/api/v1/public/pizzas
```

### Access Swagger UI

Open your browser and navigate to:
```
https://pizza-api.local/swagger/index.html
```

You'll see a certificate warning (because it's self-signed). Accept the risk and continue.

## Viewing Logs

View application logs from the pods:

```bash
# View logs from all pizza-api pods
microk8s kubectl logs -l app=pizza-api

# Follow logs in real-time
microk8s kubectl logs -l app=pizza-api -f

# View logs from a specific pod
POD_NAME=$(microk8s kubectl get pods -l app=pizza-api -o jsonpath='{.items[0].metadata.name}')
microk8s kubectl logs $POD_NAME
```

## Troubleshooting

### Pods Not Starting

Check pod status and events:
```bash
microk8s kubectl describe pods -l app=pizza-api
microk8s kubectl get events --sort-by='.lastTimestamp'
```

### Image Pull Errors

Verify the image exists in microk8s:
```bash
microk8s ctr images ls | grep pizza-api
```

If missing, rebuild and import:
```bash
docker build -t pizza-api:latest .
docker save pizza-api:latest | microk8s ctr image import -
```

### Ingress Not Working

Check ingress status:
```bash
microk8s kubectl get ingress
microk8s kubectl describe ingress pizza-api-ingress
```

Verify ingress controller is running:
```bash
microk8s kubectl get pods -n ingress
```

### Certificate Issues

Recreate the TLS secret:
```bash
microk8s kubectl delete secret pizza-api-tls
openssl req -x509 -nodes -days 365 -newkey rsa:2048 \
  -keyout tls.key -out tls.crt \
  -subj "/CN=pizza-api.local/O=Pizza API Demo"
microk8s kubectl create secret tls pizza-api-tls --cert=tls.crt --key=tls.key
```

### Database Issues

The database is stored in an emptyDir volume, which means data is lost when pods are deleted. If you need persistent data:

1. Create a PersistentVolumeClaim
2. Update the deployment to use the PVC instead of emptyDir

To reset the database:
```bash
# Delete all pods (they will be recreated with fresh database)
microk8s kubectl delete pods -l app=pizza-api
```

### Connectivity Issues

Test connectivity from within the cluster:
```bash
# Create a test pod
microk8s kubectl run test-pod --rm -it --image=curlimages/curl -- sh

# From inside the pod
curl http://pizza-api-service/health
```

## Updating the Deployment

After making code changes:

```bash
# 1. Rebuild the Docker image
docker build -t pizza-api:latest .

# 2. Import into microk8s
docker save pizza-api:latest | microk8s ctr image import -

# 3. Restart the deployment
microk8s kubectl rollout restart deployment/pizza-api

# 4. Watch the rollout
microk8s kubectl rollout status deployment/pizza-api
```

## Scaling the Deployment

Scale the number of replicas:

```bash
# Scale to 3 replicas
microk8s kubectl scale deployment/pizza-api --replicas=3

# Verify
microk8s kubectl get pods -l app=pizza-api
```

## Uninstalling

Remove all Pizza API resources:

```bash
# Delete all resources
microk8s kubectl delete -f k8s/

# Delete TLS secret
microk8s kubectl delete secret pizza-api-tls

# Remove /etc/hosts entry
sudo sed -i '/pizza-api.local/d' /etc/hosts

# Verify cleanup
microk8s kubectl get all
```

## Production Considerations

For production deployments, consider:

1. **Database**: Use PostgreSQL or MySQL with a dedicated database pod/service
2. **Secrets Management**: Use a proper secrets manager (e.g., HashiCorp Vault, Sealed Secrets)
3. **TLS Certificates**: Use cert-manager with Let's Encrypt for automatic certificate management
4. **Resource Limits**: Adjust based on actual load testing
5. **Persistent Storage**: Use PersistentVolumeClaims instead of emptyDir
6. **Monitoring**: Add Prometheus metrics and Grafana dashboards
7. **High Availability**: Run in a multi-node cluster with anti-affinity rules
8. **Ingress**: Use a production ingress controller (nginx, traefik, etc.)
9. **Horizontal Pod Autoscaler**: Auto-scale based on CPU/memory usage
10. **Network Policies**: Restrict pod-to-pod communication

## Additional Resources

- [microk8s Documentation](https://microk8s.io/docs)
- [Kubernetes Documentation](https://kubernetes.io/docs/)
- [Ingress NGINX Documentation](https://kubernetes.github.io/ingress-nginx/)
