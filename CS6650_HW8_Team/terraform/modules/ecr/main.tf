# ecr/main.tf
# create an Elastic Container Registry (ECR) repository
# store the Docker image for the Shopping Cart service

resource "aws_ecr_repository" "cart" {
  name                 = "${var.project_name}/cart"
  image_tag_mutability = "MUTABLE"

  image_scanning_configuration {
    scan_on_push = false
  }

  tags = { Name = "${var.project_name}-cart" }
}
