# ShopVerse - Full-Stack E-Commerce Application

A production-ready 3-tier e-commerce web application built with React, Go (Fiber), and MySQL, deployed on AWS EKS .



<img width="1280" height="720" alt="Shopverse project - Thumbnail" src="https://github.com/user-attachments/assets/fd0a3efe-aeb3-48dc-8912-714517580010" />

```

## Architecture:

```
                         +------------------+
                         |   AWS ALB        |
                         | (Ingress Controller) |
                         +--------+---------+
                                  |
                    +-------------+-------------+
                    |                           |
              /api/* routes               /* routes
                    |                           |
           +--------v---------+     +-----------v----------+
           | Backend Service  |     | Frontend Service     |
           | (Go + Fiber)     |     | (React + Nginx)      |
           | Port 8080        |     | Port 80              |
           | NodePort: 30081  |     | NodePort: 30080      |
           | 2 replicas       |     | 2 replicas           |
           +--------+---------+     +----------------------+
                    |
           +--------v---------+
           | MySQL StatefulSet|
           | Port 3306        |
           | 5Gi PVC (gp2)   |
           +------------------+
```

## Tech Stack

| Layer    | Technology                     |
|----------|--------------------------------|
| Frontend | React 18, TailwindCSS, Vite    |
| Backend  | Go 1.21, Fiber, GORM, JWT      |
| Database | MySQL 8.0 (StatefulSet)        |
| Infra    | AWS EKS, ECR, ALB, Terraform   |
| CI/CD    | GitHub Actions, Trivy    |


## API Endpoints

| Method | Endpoint            | Auth     | Description             |
|--------|---------------------|----------|-------------------------|
| POST   | /api/auth/register  | No       | Register new user       |
| POST   | /api/auth/login     | No       | Login, returns JWT      |
| GET    | /api/products       | No       | List products           |
| GET    | /api/products/:id   | No       | Get single product      |
| POST   | /api/products       | JWT      | Create product (admin)  |
| GET    | /api/cart           | JWT      | Get user's cart         |
| POST   | /api/cart           | JWT      | Add item to cart        |
| PUT    | /api/cart/:id       | JWT      | Update cart item qty    |
| DELETE | /api/cart/:id       | JWT      | Remove cart item        |
| GET    | /api/orders         | JWT      | Get user's orders       |
| POST   | /api/orders         | JWT      | Place order from cart   |
| GET    | /health             | No       | Health check            |

---

## Local Development

### Prerequisites
- Docker & Docker Compose
- Node.js 18+ (for frontend dev)
- Go 1.21+ (for backend dev)

### Quick Start with Docker Compose

```bash
# Clone the repo
git clone <repo-url> && cd shopverse

# Start all services
docker-compose up --build

# Access the app
# Frontend: http://localhost:3000
# Backend:  http://localhost:8080
```

### Run Frontend Individually (Hot Reload)

```bash
cd frontend
npm install
npm run dev
# Runs on http://localhost:3000 with proxy to backend
```

### Run Backend Individually

```bash
cd backend
go mod tidy
DB_HOST=localhost DB_USER=shopverse DB_PASSWORD=shopverse123 DB_NAME=shopverse go run ./cmd/main.go
```

---

## AWS Deployment (Step-by-Step from Local)

### Prerequisites

Install the following tools on your local machine:

| Tool       | Version  | Download |
|------------|----------|----------|
| AWS CLI v2 | Latest   | https://aws.amazon.com/cli/ |
| kubectl    | Latest   | https://kubernetes.io/docs/tasks/tools/ |
| Docker     | Latest   | https://docs.docker.com/get-docker/ |

---

### Step 1: Configure AWS CLI

```bash
aws configure
# AWS Access Key ID: <your-access-key>
# AWS Secret Access Key: <your-secret-key>
# Default region name: us-east-1
# Default output format: json

# Verify your identity
aws sts get-caller-identity
```

---

### Step 2: Create AWS Infrastructure using Terraform

Terraform modules will create: VPC, EKS Cluster, Node Group, IAM Roles, Jump Server (EC2).

See [terraform/README.md](terraform/README.md) for detailed Terraform instructions.

```bash
cd terraform

# Create S3 bucket for Terraform state (one-time setup)
aws s3api create-bucket \
  --bucket shopverse-terraform-state \
  --region us-east-1

aws s3api put-bucket-versioning \
  --bucket shopverse-terraform-state \
  --versioning-configuration Status=Enabled

# Copy and edit variables
cp terraform.tfvars.example terraform.tfvars
# Edit terraform.tfvars with your values (cluster name, region, instance types, etc.)

# Initialize Terraform
terraform init

# Preview what will be created
terraform plan

# Create the infrastructure (~15-20 minutes)
terraform apply
# Type 'yes' when prompted
```

After apply completes, note the outputs:
```bash
terraform output
```

---

### Step 3: Connect to the EKS Cluster

```bash
# Update your local kubeconfig (use cluster name from terraform output)
aws eks update-kubeconfig --name shopverse-cluster --region us-east-1

# Verify connection - you should see your worker nodes
kubectl get nodes
kubectl cluster-info
```

---

### Step 4: Create ECR Repositories

Create 3 ECR repositories for frontend, backend:

```bash
# Get your AWS Account ID
ACCOUNT_ID=$(aws sts get-caller-identity --query Account --output text)
REGION=us-east-1

# Create repositories
aws ecr create-repository --repository-name shopverse-frontend --region $REGION
aws ecr create-repository --repository-name shopverse-backend --region $REGION


# Verify repositories were created
aws ecr describe-repositories --region $REGION --query 'repositories[].repositoryName'
```

---

### Step 5: Build Docker Images

```bash
# Navigate to project root
cd ..

# Build frontend image
docker build -t shopverse-frontend:v1 ./frontend

# Build backend image
docker build -t shopverse-backend:v1 ./backend

# Verify images were built
docker images | grep shopverse
```

---

### Step 6: Tag Docker Images

Tag the images with the ECR repository URI:

```bash
ACCOUNT_ID=$(aws sts get-caller-identity --query Account --output text)
REGION=us-east-1
ECR_URI=${ACCOUNT_ID}.dkr.ecr.${REGION}.amazonaws.com

# Tag frontend image
docker tag shopverse-frontend:v1 ${ECR_URI}/shopverse-frontend:v1

# Tag backend image
docker tag shopverse-backend:v1 ${ECR_URI}/shopverse-backend:v1

# Verify tags
docker images | grep ${ACCOUNT_ID}
```

---

### Step 7: Push Docker Images to ECR

```bash
ACCOUNT_ID=$(aws sts get-caller-identity --query Account --output text)
REGION=us-east-1
ECR_URI=${ACCOUNT_ID}.dkr.ecr.${REGION}.amazonaws.com

# Login to ECR
aws ecr get-login-password --region $REGION | \
  docker login --username AWS --password-stdin ${ECR_URI}

# Push frontend image
docker push ${ECR_URI}/shopverse-frontend:v1

# Push backend image
docker push ${ECR_URI}/shopverse-backend:v1

# Verify images in ECR
aws ecr list-images --repository-name shopverse-frontend --region $REGION
aws ecr list-images --repository-name shopverse-backend --region $REGION
```

### Step 8: Install EKS Add-ons

```bash
# Install EBS CSI Driver (required for MySQL PVC)

#install manually:
eksctl utils associate-iam-oidc-provider --cluster shopverse-cluster --region us-east-1 --approve

eksctl create iamserviceaccount \
  --name ebs-csi-controller-sa \
  --namespace kube-system \
  --cluster shopverse-cluster \
  --region us-east-1 \
  --attach-policy-arn arn:aws:iam::aws:policy/service-role/AmazonEBSCSIDriverPolicy \
  --approve

aws eks create-addon --cluster-name shopverse-cluster --addon-name aws-ebs-csi-driver --region us-east-1

# Install AWS Load Balancer Controller (required for ALB Ingress)
ALB_ROLE_ARN=$(cd terraform && terraform output -raw alb_controller_role_arn)

helm repo add eks https://aws.github.io/eks-charts
helm repo update
helm install aws-load-balancer-controller eks/aws-load-balancer-controller \
  -n kube-system \
  --set clusterName=shopverse-cluster \
  --set serviceAccount.create=true \
  --set serviceAccount.name=aws-load-balancer-controller \
  --set serviceAccount.annotations."eks\.amazonaws\.com/role-arn"=$ALB_ROLE_ARN
```

---



---

### Step 9: Verify Deployment

```bash
# Check all pods are running (should see 5 pods: 2 frontend, 2 backend, 1 mysql)
kubectl get pods -n shopverse

# Check services (frontend NodePort:30080, backend NodePort:30081)
kubectl get svc -n shopverse

# Check persistent volume claims (MySQL storage)
kubectl get pvc -n shopverse

# Check all resources at once
kubectl get all -n shopverse

# Check pod logs if needed
kubectl logs -n shopverse -l component=backend --tail=50
kubectl logs -n shopverse -l component=frontend --tail=50
kubectl logs -n shopverse shopverse-mysql-0 --tail=50
```

---

### Step 12: Access the Application

**Get Node External IPs:**
```bash
kubectl get nodes -o wide
# Note the EXTERNAL-IP column
```

**Access via NodePort:**
```
Frontend:  http://<NODE_EXTERNAL_IP>:30080
Backend:   http://<NODE_EXTERNAL_IP>:30081
Health:    http://<NODE_EXTERNAL_IP>:30081/health
```

**Access via ALB Ingress (if configured):**
```bash
kubectl get ingress -n shopverse
# Use the ADDRESS field as the URL
```

> **Note:** Make sure the EKS node security group allows inbound traffic on ports **30080** and **30081**. You can update this in AWS Console > EC2 > Security Groups > find the node security group > add inbound rules for Custom TCP ports 30080 and 30081 from `0.0.0.0/0`.

---

---

## Querying the Database

### Understanding the Database

ShopVerse uses MySQL 8.0 running as a Kubernetes StatefulSet. The database contains these tables:

| Table | Description |
|-------|-------------|
| `users` | Registered users (name, email, hashed password) |
| `products` | Product catalog - 28 products across 6 categories |
| `orders` | Customer orders (total amount, status, timestamps) |
| `order_items` | Individual items within each order (product, quantity, price) |
| `cart_items` | Current shopping cart contents per user |

### Step 1: Get the Database Password

The MySQL password is stored as a Kubernetes secret (base64 encoded):

```bash
# Decode the database password from the Kubernetes secret
DB_PASSWORD=$(kubectl get secret -n shopverse shopverse-secret \
  -o jsonpath='{.data.DB_PASSWORD}' | base64 -d)

# Verify you got the password (optional)
echo $DB_PASSWORD
```

**How this works:**
- `kubectl get secret` fetches the Kubernetes secret object
- `-o jsonpath='{.data.DB_PASSWORD}'` extracts just the password field
- `| base64 -d` decodes it from base64 (Kubernetes stores secrets in base64)

### Step 2: Connect to MySQL Shell

```bash
# Open an interactive MySQL shell inside the MySQL pod
kubectl exec -it -n shopverse shopverse-mysql-0 -- mysql -u shopverse -p"$DB_PASSWORD" shopverse
```

**How this works:**
- `kubectl exec -it` runs an interactive command inside a pod
- `-n shopverse` specifies the namespace
- `shopverse-mysql-0` is the MySQL pod name (StatefulSet pod naming: `<name>-0`)
- `-- mysql -u shopverse -p"$DB_PASSWORD" shopverse` runs the MySQL client
  - `-u shopverse` = database username
  - `-p"$DB_PASSWORD"` = password (no space between `-p` and the password)
  - `shopverse` (at end) = database name to connect to

### Step 3: Run Queries Inside MySQL Shell

Once inside the MySQL shell (you'll see `mysql>` prompt):

#### View all tables
```sql
SHOW TABLES;
```
This shows all 5 tables: `users`, `products`, `orders`, `order_items`, `cart_items`.

#### View registered users
```sql
SELECT id, name, email, created_at FROM users;
```
Shows all users who registered through the app. Passwords are hashed with bcrypt so they are not shown here.

#### View all products
```sql
SELECT id, name, category, price, original_price, rating, badge FROM products;
```
Lists all 28 seeded products with their category, pricing, rating, and badge info.

#### View products grouped by category
```sql
SELECT category, COUNT(*) AS total_products,
       ROUND(AVG(price), 2) AS avg_price,
       ROUND(MIN(price), 2) AS min_price,
       ROUND(MAX(price), 2) AS max_price
FROM products
GROUP BY category
ORDER BY total_products DESC;
```
Shows product count and price stats per category (Electronics, Clothing, Accessories, Food & Drinks, Sports, Home & Living).

#### View all orders with customer info
```sql
SELECT
    o.id AS order_id,
    u.name AS customer_name,
    u.email AS customer_email,
    o.total_amount,
    o.status,
    o.created_at AS order_date
FROM orders o
JOIN users u ON o.user_id = u.id
ORDER BY o.created_at DESC;
```
**How this works:**
- `JOIN users u ON o.user_id = u.id` links each order to the user who placed it
- `ORDER BY o.created_at DESC` shows newest orders first
- `o.status` shows the order status (e.g., pending, completed)

#### View order items with product details
```sql
SELECT
    oi.order_id,
    p.name AS product_name,
    p.category,
    oi.quantity,
    oi.price AS unit_price,
    (oi.quantity * oi.price) AS subtotal
FROM order_items oi
JOIN products p ON oi.product_id = p.id
ORDER BY oi.order_id, p.name;
```
**How this works:**
- `order_items` stores what was purchased in each order
- `JOIN products p ON oi.product_id = p.id` links item to its product details
- `(oi.quantity * oi.price)` calculates the subtotal for each line item

#### View complete order breakdown (orders + items together)
```sql
SELECT
    o.id AS order_id,
    u.name AS customer,
    p.name AS product,
    oi.quantity,
    oi.price AS unit_price,
    (oi.quantity * oi.price) AS subtotal,
    o.total_amount AS order_total,
    o.status,
    o.created_at
FROM orders o
JOIN users u ON o.user_id = u.id
JOIN order_items oi ON oi.order_id = o.id
JOIN products p ON oi.product_id = p.id
ORDER BY o.id, p.name;
```
This is the most complete view - joins 4 tables to show who ordered what, quantities, prices, and order status.

#### View current cart items
```sql
SELECT
    ci.id AS cart_item_id,
    u.name AS customer,
    p.name AS product,
    p.category,
    ci.quantity,
    p.price AS unit_price,
    (ci.quantity * p.price) AS subtotal
FROM cart_items ci
JOIN users u ON ci.user_id = u.id
JOIN products p ON ci.product_id = p.id
ORDER BY u.name;
```
Shows items currently in users' shopping carts (items that haven't been ordered yet).

#### Dashboard summary
```sql
SELECT
    (SELECT COUNT(*) FROM users) AS total_users,
    (SELECT COUNT(*) FROM products) AS total_products,
    (SELECT COUNT(*) FROM orders) AS total_orders,
    (SELECT COALESCE(SUM(total_amount), 0) FROM orders) AS total_revenue,
    (SELECT COUNT(*) FROM cart_items) AS items_in_carts;
```
A quick overview of the entire application's data - total users, products, orders, revenue, and active cart items.

#### Exit MySQL shell
```sql
EXIT;
```

---

## CI/CD Pipeline (GitHub Actions)

### Configure GitHub Secrets

Go to your GitHub repo > Settings > Secrets and variables > Actions, and add:

| Secret                  | Description                                          |
|-------------------------|------------------------------------------------------|
| `AWS_ACCESS_KEY_ID`     | IAM user access key                                  |
| `AWS_SECRET_ACCESS_KEY` | IAM user secret key                                  |
| `AWS_REGION`            | e.g., `us-east-1`                                    |
| `ECR_REGISTRY`          | e.g., `123456789.dkr.ecr.us-east-1.amazonaws.com`    |
| `EKS_CLUSTER_NAME`      | e.g., `shopverse-cluster`                            |
| `TF_STATE_BUCKET`       | Root@1234                                            |
| `MYSQL_ROOT_PASSWORD`   | MySQL root password                                  |
| `MYSQL_PASSWORD`        | App@1234                                             |
| `JWT_SECRET`            | shopverse-secret-key-2024                            |

### Pipeline Stages

Push to `main` branch triggers the 4-stage pipeline:

1. **Test** - Go tests + frontend linting
2. **Security Scan** - Trivy vulnerability scanning on Docker images
3. **Build & Push** - Build images, tag with SHA, push to ECR
4. **Deploy** - Provision infra with Terraform if needed, deploy Helm chart

---


---

## Project Structure

```
shopverse/
├── frontend/                  # React + TailwindCSS (Vite)
│   ├── src/
│   │   ├── components/        # Navbar, ProductCard, CartSidebar
│   │   ├── pages/             # Auth, Home, Products, Cart, Orders, Wishlist
│   │   ├── App.jsx            # Routes, context, API client
│   │   └── main.jsx           # Entry point
│   ├── Dockerfile             # Multi-stage: Node -> Nginx
│   └── nginx.conf             # React Router + API proxy
├── backend/                   # Go + Fiber REST API
│   ├── cmd/main.go            # Entry point, routes
│   ├── internal/
│   │   ├── handlers/          # Auth, Products, Cart, Orders
│   │   ├── models/            # GORM models
│   │   ├── database/          # DB connection + seed data (28 products)
│   │   └── middleware/        # JWT auth middleware
│   └── Dockerfile             # Multi-stage: Go -> Distroless
├── k8s/shopverse/            # Helm chart
│                # K8s manifests (10 YAML files)
│   │   ├── secret.yaml        # DB passwords, JWT secret
│   │   ├── configmap.yaml     # DB host, port, name config
│   │   ├── mysql-pvc.yaml     # 5Gi persistent volume claim
│   │   ├── mysql-statefulset.yaml  # MySQL 8.0 pod
│   │   ├── mysql-service.yaml      # MySQL ClusterIP service
│   │   ├── backend-deployment.yaml # Go API (2 replicas)
│   │   ├── backend-service.yaml    # NodePort 30081
│   │   ├── frontend-deployment.yaml # React+Nginx (2 replicas)
│   │   ├── frontend-service.yaml    # NodePort 30080
│   │   └── ingress.yaml            # ALB ingress
├
├── .github/workflows/         # CI/CD pipeline
│   └── deploy.yml             # 4-stage: test -> scan -> build -> deploy
├── docker-compose.yml         # Local development
└── README.md
```

## Troubleshooting

### Pods stuck in Pending
```bash
kubectl describe pod <pod-name> -n shopverse
kubectl get pvc -n shopverse
# Common cause: EBS CSI driver not installed (PVC can't bind)
```

### Frontend can't reach backend (502/504)
```bash
kubectl get svc -n shopverse
kubectl logs -n shopverse -l component=backend
# Verify backend pods are running and healthy
```

### MySQL connection refused
```bash
kubectl get pods -n shopverse -l component=mysql
kubectl logs -n shopverse shopverse-mysql-0
# Check if MySQL is still initializing
```

### Can't access NodePort from browser
```bash
# Check node security group allows ports 30080 and 30081
# AWS Console > EC2 > Security Groups > Node security group > Inbound rules
# Add: Custom TCP, Port 30080, Source 0.0.0.0/0
# Add: Custom TCP, Port 30081, Source 0.0.0.0/0
```

### Images not updating after push
```bash
# Use a new tag instead of reusing the same one
helm upgrade shopverse ./helm/shopverse \
  --set frontend.image=<ECR>/shopverse-frontend:v2 \
  --set backend.image=<ECR>/shopverse-backend:v2 \
  --reuse-values -n shopverse
```




**pipeline**

```
name: ShopVerse CI/CD

on:
  push:
    branches: [main]

env:
  AWS_REGION: ${{ secrets.AWS_REGION }}
  ECR_REGISTRY: ${{ secrets.ECR_REGISTRY }}
  EKS_CLUSTER_NAME: ${{ secrets.EKS_CLUSTER_NAME }}
  HELM_CHART_NAME: shopverse
  TF_STATE_BUCKET: ${{ secrets.TF_STATE_BUCKET }}

jobs:
  # ──────────────────────────────────────────────
  # Stage 1: Test
  # ──────────────────────────────────────────────
  test:
    name: Test
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - name: Set up Go
        uses: actions/setup-go@v5
        with:
          go-version: "1.24"

      - name: Update go.sum
        working-directory: ./backend
        run: go mod tidy

      - name: Run Go tests
        working-directory: ./backend
        run: go test ./...

      - name: Set up Node.js
        uses: actions/setup-node@v4
        with:
          node-version: "18"

      - name: Install frontend dependencies
        working-directory: ./frontend
        run: npm install

      - name: Run frontend lint
        working-directory: ./frontend
        run: npx eslint . --ext js,jsx --report-unused-disable-directives --max-warnings 0 --config .eslintrc.cjs

  # ──────────────────────────────────────────────
  # Stage 2: Security Scan
  # ──────────────────────────────────────────────
  security-scan:
    name: Security Scan
    runs-on: ubuntu-latest
    needs: test
    steps:
      - uses: actions/checkout@v4

      - name: Build frontend image
        run: docker build -t shopverse-frontend:scan ./frontend

      - name: Build backend image
        run: docker build -t shopverse-backend:scan ./backend

      - name: Install Trivy
        run: |
          sudo apt-get install -y wget apt-transport-https gnupg
          wget -qO - https://aquasecurity.github.io/trivy-repo/deb/public.key | gpg --dearmor | sudo tee /usr/share/keyrings/trivy.gpg > /dev/null
          echo "deb [signed-by=/usr/share/keyrings/trivy.gpg] https://aquasecurity.github.io/trivy-repo/deb generic main" | sudo tee /etc/apt/sources.list.d/trivy.list
          sudo apt-get update -q
          sudo apt-get install -y trivy

      - name: Run Trivy scan on frontend
        run: |
          trivy image --exit-code 1 --severity CRITICAL --ignore-unfixed --format table shopverse-frontend:scan

      - name: Run Trivy scan on backend
        run: |
          trivy image --exit-code 1 --severity CRITICAL --ignore-unfixed --format table shopverse-backend:scan

  # ──────────────────────────────────────────────
  # Stage 3: Build, Tag & Push Images 
  # ──────────────────────────────────────────────
  build-and-push:
    name: Build, Tag & Push
    runs-on: ubuntu-latest
    needs: security-scan
    steps:
      - uses: actions/checkout@v4

      - name: Configure AWS credentials
        uses: aws-actions/configure-aws-credentials@v4
        with:
          aws-access-key-id: ${{ secrets.AWS_ACCESS_KEY_ID }}
          aws-secret-access-key: ${{ secrets.AWS_SECRET_ACCESS_KEY }}
          aws-region: ${{ env.AWS_REGION }}

      - name: Login to Amazon ECR
        id: login-ecr
        uses: aws-actions/amazon-ecr-login@v2

      - name: Ensure ECR repositories exist
        run: |
          for repo in shopverse-frontend shopverse-backend shopverse-helmchart/shopverse; do
            aws ecr describe-repositories --repository-names $repo --region ${{ env.AWS_REGION }} 2>/dev/null || \
            aws ecr create-repository --repository-name $repo --region ${{ env.AWS_REGION }}
          done

      - name: Build and tag frontend image
        run: |
          docker build -t ${{ env.ECR_REGISTRY }}/shopverse-frontend:${{ github.sha }} \
                        -t ${{ env.ECR_REGISTRY }}/shopverse-frontend:latest \
                        ./frontend

      - name: Build and tag backend image
        run: |
          docker build -t ${{ env.ECR_REGISTRY }}/shopverse-backend:${{ github.sha }} \
                        -t ${{ env.ECR_REGISTRY }}/shopverse-backend:latest \
                        ./backend

      - name: Push frontend image
        run: |
          docker push ${{ env.ECR_REGISTRY }}/shopverse-frontend:${{ github.sha }}
          docker push ${{ env.ECR_REGISTRY }}/shopverse-frontend:latest

      - name: Push backend image
        run: |
          docker push ${{ env.ECR_REGISTRY }}/shopverse-backend:${{ github.sha }}
          docker push ${{ env.ECR_REGISTRY }}/shopverse-backend:latest

      - name: Update Helm values.yaml with new image tags
        run: |
          FRONTEND_IMG="${{ env.ECR_REGISTRY }}/shopverse-frontend:${{ github.sha }}"
          BACKEND_IMG="${{ env.ECR_REGISTRY }}/shopverse-backend:${{ github.sha }}"
          sed -i "s|image:.*# frontend-image|image: ${FRONTEND_IMG}  # frontend-image|" helm/shopverse/values.yaml
          sed -i "s|image:.*# backend-image|image: ${BACKEND_IMG}  # backend-image|" helm/shopverse/values.yaml
          echo "Updated values.yaml:"
          cat helm/shopverse/values.yaml

      - name: Install Helm
        uses: azure/setup-helm@v3

      - name: Login Helm to ECR
        run: |
          aws ecr get-login-password --region ${{ env.AWS_REGION }} | \
            helm registry login --username AWS --password-stdin ${{ env.ECR_REGISTRY }}

      - name: Package Helm chart
        run: |
          helm package ./helm/shopverse \
            --version 1.0.0-${{ github.sha }} \
            --app-version ${{ github.sha }}

      - name: Push Helm chart to ECR
        run: |
          helm push shopverse-1.0.0-${{ github.sha }}.tgz \
            oci://${{ env.ECR_REGISTRY }}/shopverse-helmchart

  # ──────────────────────────────────────────────
  # Stage 4: Provision Infra (if needed) + Deploy
  # ──────────────────────────────────────────────
  deploy:
    name: Provision Infra & Deploy
    runs-on: ubuntu-latest
    needs: build-and-push
    steps:
      - uses: actions/checkout@v4

      - name: Configure AWS credentials
        uses: aws-actions/configure-aws-credentials@v4
        with:
          aws-access-key-id: ${{ secrets.AWS_ACCESS_KEY_ID }}
          aws-secret-access-key: ${{ secrets.AWS_SECRET_ACCESS_KEY }}
          aws-region: ${{ env.AWS_REGION }}

      - name: Check if EKS cluster exists
        id: check-cluster
        run: |
          if aws eks describe-cluster --name ${{ env.EKS_CLUSTER_NAME }} --region ${{ env.AWS_REGION }} > /dev/null 2>&1; then
            echo "cluster_exists=true" >> "$GITHUB_OUTPUT"
            echo "EKS cluster '${{ env.EKS_CLUSTER_NAME }}' found."
          else
            echo "cluster_exists=false" >> "$GITHUB_OUTPUT"
            echo "EKS cluster '${{ env.EKS_CLUSTER_NAME }}' NOT found. Will provision with Terraform."
          fi

      - name: Setup Terraform
        if: steps.check-cluster.outputs.cluster_exists == 'false'
        uses: hashicorp/setup-terraform@v3
        with:
          terraform_version: "1.7.0"

      - name: Terraform Init
        if: steps.check-cluster.outputs.cluster_exists == 'false'
        working-directory: ./terraform
        run: |
          terraform init \
            -backend-config="bucket=${{ env.TF_STATE_BUCKET }}" \
            -backend-config="key=eks/terraform.tfstate" \
            -backend-config="region=${{ env.AWS_REGION }}"

      - name: Terraform Plan
        if: steps.check-cluster.outputs.cluster_exists == 'false'
        working-directory: ./terraform
        run: |
          terraform plan \
            -var="aws_region=${{ env.AWS_REGION }}" \
            -var="cluster_name=${{ env.EKS_CLUSTER_NAME }}" \
            -var="project_name=shopverse" \
            -var="create_jump_server=false" \
            -out=tfplan

      - name: Terraform Apply
        if: steps.check-cluster.outputs.cluster_exists == 'false'
        working-directory: ./terraform
        run: terraform apply -auto-approve tfplan

      - name: Wait for EKS cluster to be active
        if: steps.check-cluster.outputs.cluster_exists == 'false'
        run: |
          echo "Waiting for EKS cluster to become ACTIVE..."
          aws eks wait cluster-active --name ${{ env.EKS_CLUSTER_NAME }} --region ${{ env.AWS_REGION }}
          echo "Waiting for node group to become ACTIVE..."
          aws eks wait nodegroup-active \
            --cluster-name ${{ env.EKS_CLUSTER_NAME }} \
            --nodegroup-name ${{ env.EKS_CLUSTER_NAME }}-nodes \
            --region ${{ env.AWS_REGION }}
          echo "Cluster and nodes are ready."

      - name: Install AWS Load Balancer Controller
        if: steps.check-cluster.outputs.cluster_exists == 'false'
        run: |
          aws eks update-kubeconfig --name ${{ env.EKS_CLUSTER_NAME }} --region ${{ env.AWS_REGION }}
          helm repo add eks https://aws.github.io/eks-charts
          helm repo update
          ALB_ROLE_ARN=$(cd terraform && terraform output -raw alb_controller_role_arn)
          helm install aws-load-balancer-controller eks/aws-load-balancer-controller \
            -n kube-system \
            --set clusterName=${{ env.EKS_CLUSTER_NAME }} \
            --set serviceAccount.create=true \
            --set serviceAccount.name=aws-load-balancer-controller \
            --set serviceAccount.annotations."eks\.amazonaws\.com/role-arn"=$ALB_ROLE_ARN

      - name: Install kubectl
        uses: azure/setup-kubectl@v3

      - name: Install Helm
        uses: azure/setup-helm@v3

      - name: Update kubeconfig
        run: aws eks update-kubeconfig --name ${{ env.EKS_CLUSTER_NAME }} --region ${{ env.AWS_REGION }}

      - name: Login to ECR for Helm
        run: |
          aws ecr get-login-password --region ${{ env.AWS_REGION }} | \
            helm registry login --username AWS --password-stdin ${{ env.ECR_REGISTRY }}

      - name: Deploy Helm chart
        run: |
          helm upgrade --install shopverse \
            oci://${{ env.ECR_REGISTRY }}/shopverse-helmchart/shopverse \
            --version 1.0.0-${{ github.sha }} \
            --set frontend.image=${{ env.ECR_REGISTRY }}/shopverse-frontend:${{ github.sha }} \
            --set backend.image=${{ env.ECR_REGISTRY }}/shopverse-backend:${{ github.sha }} \
            --set mysql.rootPassword=${{ secrets.MYSQL_ROOT_PASSWORD }} \
            --set mysql.password=${{ secrets.MYSQL_PASSWORD }} \
            --set jwtSecret=${{ secrets.JWT_SECRET }} \
            --namespace shopverse \
            --create-namespace \
            --wait --timeout 300s \
            --cleanup-on-fail

      - name: Verify deployment
        run: |
          echo ""
          echo "--- Nodes ---"
          kubectl get nodes -o wide
          echo ""
          echo "--- Pods ---"
          kubectl get pods -n shopverse
          echo ""
          echo "--- StatefulSets ---"
          kubectl get sts -n shopverse
          echo ""
          echo "--- Services ---"
          kubectl get svc -n shopverse
          echo ""
          echo "--- PV & PVC ---"
          kubectl get pv,pvc -n shopverse
          echo ""
          echo "--- Secrets ---"
          kubectl get secrets -n shopverse
          echo ""
          echo "--- Ingress ---"
          kubectl get ingress -n shopverse 2>/dev/null || echo "No ingress configured"



```

---

## License

MIT
