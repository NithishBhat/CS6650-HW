#!/bin/bash
set -e

REGION="us-east-1"
ACCOUNT_ID=$(aws sts get-caller-identity --query Account --output text)

echo "=== Step 1: Terraform init (creates ECR repos, SNS, SQS, etc.) ==="
# First apply just ECR repos so we can push images
cd terraform
terraform init
terraform apply -target=aws_ecr_repository.api -target=aws_ecr_repository.worker -auto-approve

API_ECR=$(terraform output -raw api_ecr_url)
WORKER_ECR=$(terraform output -raw worker_ecr_url)
cd ..

echo "=== Step 2: Authenticate Docker with ECR ==="
aws ecr get-login-password --region $REGION | docker login --username AWS --password-stdin $ACCOUNT_ID.dkr.ecr.$REGION.amazonaws.com

echo "=== Step 3: Build & push API image ==="
cd api
docker build --platform linux/amd64 -t hw7-api .
docker tag hw7-api:latest $API_ECR:latest
docker push $API_ECR:latest
cd ..

echo "=== Step 4: Build & push Worker image ==="
cd worker
docker build --platform linux/amd64 -t hw7-worker .
docker tag hw7-worker:latest $WORKER_ECR:latest
docker push $WORKER_ECR:latest
cd ..

echo "=== Step 5: Build Lambda binary ==="
cd lambda
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o bootstrap .
zip bootstrap.zip bootstrap
cd ..

echo "=== Step 6: Full Terraform apply ==="
cd terraform
terraform apply -auto-approve

echo ""
echo "=== DONE ==="
echo "ALB DNS: $(terraform output -raw alb_dns)"
echo "SQS URL: $(terraform output -raw sqs_queue_url)"
echo ""
echo "Test sync:  curl -X POST http://$(terraform output -raw alb_dns)/orders/sync -H 'Content-Type: application/json' -d '{\"customer_id\": 1, \"items\": [{\"product_id\": 1, \"quantity\": 1}]}'"
echo "Test async: curl -X POST http://$(terraform output -raw alb_dns)/orders/async -H 'Content-Type: application/json' -d '{\"customer_id\": 1, \"items\": [{\"product_id\": 1, \"quantity\": 1}]}'"
