#!/bin/bash
set -e

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
REGION="us-west-2"

echo "==> Resolving Go dependencies..."
cd "$SCRIPT_DIR/server"
go mod tidy

echo "==> Initializing Terraform..."
cd "$SCRIPT_DIR/terraform"
terraform init -input=false

echo "==> Creating ECR repository..."
terraform apply -target=aws_ecr_repository.app -auto-approve

ECR_URL=$(terraform output -raw ecr_url)
echo "    ECR: $ECR_URL"

echo "==> Building Docker image..."
cd "$SCRIPT_DIR/server"
docker build -t "$ECR_URL:latest" .

echo "==> Pushing to ECR..."
aws ecr get-login-password --region $REGION | docker login --username AWS --password-stdin "$ECR_URL"
docker push "$ECR_URL:latest"

echo "==> Applying full infrastructure..."
cd "$SCRIPT_DIR/terraform"
terraform apply -auto-approve

echo ""
echo "============================================"
echo "Deployment complete!"
echo "Base URL: $(terraform output -raw base_url)"
echo ""
echo "Wait ~60s for Fargate task to start, then:"
echo "  curl $(terraform output -raw base_url)/health"
echo ""
echo "View logs:"
echo "  aws logs tail $(terraform output -raw log_group) --follow --region $REGION"
echo "============================================"
