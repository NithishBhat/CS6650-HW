terraform {
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.0"
    }
  }
}

provider "aws" {
  region = var.region
}

# ── ECR ──────────────────────────────────────────────────────────────
resource "aws_ecr_repository" "app" {
  name         = "${var.prefix}-app"
  force_delete = true
}

# ── S3: photo storage (public read) ─────────────────────────────────
resource "aws_s3_bucket" "photos" {
  bucket_prefix = "${var.prefix}-photos-"
  force_destroy = true
}

resource "aws_s3_bucket_public_access_block" "photos" {
  bucket                  = aws_s3_bucket.photos.id
  block_public_acls       = false
  block_public_policy     = false
  ignore_public_acls      = false
  restrict_public_buckets = false
}

resource "aws_s3_bucket_policy" "photos" {
  depends_on = [aws_s3_bucket_public_access_block.photos]
  bucket     = aws_s3_bucket.photos.id
  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Sid       = "PublicRead"
      Effect    = "Allow"
      Principal = "*"
      Action    = "s3:GetObject"
      Resource  = "${aws_s3_bucket.photos.arn}/*"
    }]
  })
}

# ── DynamoDB ─────────────────────────────────────────────────────────
resource "aws_dynamodb_table" "albums" {
  name         = "${var.prefix}-albums"
  billing_mode = "PAY_PER_REQUEST"
  hash_key     = "album_id"

  attribute {
    name = "album_id"
    type = "S"
  }
}

resource "aws_dynamodb_table" "photos" {
  name         = "${var.prefix}-photos"
  billing_mode = "PAY_PER_REQUEST"
  hash_key     = "photo_id"

  attribute {
    name = "photo_id"
    type = "S"
  }
}

# ── VPC Endpoints (S3 + DynamoDB over AWS backbone) ──────────────────
data "aws_route_tables" "vpc" {
  vpc_id = var.vpc_id
}

resource "aws_vpc_endpoint" "s3" {
  vpc_id            = var.vpc_id
  service_name      = "com.amazonaws.${var.region}.s3"
  vpc_endpoint_type = "Gateway"
  route_table_ids   = data.aws_route_tables.vpc.ids
}

resource "aws_vpc_endpoint" "dynamodb" {
  vpc_id            = var.vpc_id
  service_name      = "com.amazonaws.${var.region}.dynamodb"
  vpc_endpoint_type = "Gateway"
  route_table_ids   = data.aws_route_tables.vpc.ids
}

# ── IAM: ECS execution role (pull image + logs) ─────────────────────
resource "aws_iam_role" "ecs_exec" {
  name_prefix = "${var.prefix}-exec-"
  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Effect    = "Allow"
      Principal = { Service = "ecs-tasks.amazonaws.com" }
      Action    = "sts:AssumeRole"
    }]
  })
}

resource "aws_iam_role_policy_attachment" "ecs_exec" {
  role       = aws_iam_role.ecs_exec.name
  policy_arn = "arn:aws:iam::aws:policy/service-role/AmazonECSTaskExecutionRolePolicy"
}

# ── IAM: ECS task role (DynamoDB + S3) ───────────────────────────────
resource "aws_iam_role" "ecs_task" {
  name_prefix = "${var.prefix}-task-"
  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Effect    = "Allow"
      Principal = { Service = "ecs-tasks.amazonaws.com" }
      Action    = "sts:AssumeRole"
    }]
  })
}

resource "aws_iam_role_policy" "ecs_task" {
  role = aws_iam_role.ecs_task.id
  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Effect   = "Allow"
        Action   = "dynamodb:*"
        Resource = [
          aws_dynamodb_table.albums.arn,
          aws_dynamodb_table.photos.arn,
        ]
      },
      {
        Effect = "Allow"
        Action = "s3:*"
        Resource = [
          aws_s3_bucket.photos.arn,
          "${aws_s3_bucket.photos.arn}/*",
        ]
      },
    ]
  })
}

# ── CloudWatch Logs ──────────────────────────────────────────────────
resource "aws_cloudwatch_log_group" "app" {
  name              = "/ecs/${var.prefix}"
  retention_in_days = 7
}

# ── Security Groups ─────────────────────────────────────────────────
resource "aws_security_group" "alb" {
  name_prefix = "${var.prefix}-alb-"
  vpc_id      = var.vpc_id

  ingress {
    from_port   = 80
    to_port     = 80
    protocol    = "tcp"
    cidr_blocks = ["0.0.0.0/0"]
  }
  egress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }
}

resource "aws_security_group" "ecs" {
  name_prefix = "${var.prefix}-ecs-"
  vpc_id      = var.vpc_id

  ingress {
    from_port       = 8080
    to_port         = 8080
    protocol        = "tcp"
    security_groups = [aws_security_group.alb.id]
  }
  egress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }
}

# ── ALB ──────────────────────────────────────────────────────────────
resource "aws_lb" "app" {
  name_prefix        = "album-"
  load_balancer_type = "application"
  security_groups    = [aws_security_group.alb.id]
  subnets            = var.subnet_ids
}

resource "aws_lb_target_group" "app" {
  name_prefix = "album-"
  port        = 8080
  protocol    = "HTTP"
  vpc_id      = var.vpc_id
  target_type = "ip"

  health_check {
    path                = "/health"
    interval            = 10
    timeout             = 5
    healthy_threshold   = 2
    unhealthy_threshold = 3
  }

  deregistration_delay = 10
}

resource "aws_lb_listener" "app" {
  load_balancer_arn = aws_lb.app.arn
  port              = 80
  protocol          = "HTTP"

  default_action {
    type             = "forward"
    target_group_arn = aws_lb_target_group.app.arn
  }
}

# ── ECS ──────────────────────────────────────────────────────────────
resource "aws_ecs_cluster" "app" {
  name = "${var.prefix}-cluster"
}

resource "aws_ecs_task_definition" "app" {
  family                   = "${var.prefix}-task"
  requires_compatibilities = ["FARGATE"]
  network_mode             = "awsvpc"
  cpu                      = var.cpu
  memory                   = var.memory
  execution_role_arn       = aws_iam_role.ecs_exec.arn
  task_role_arn            = aws_iam_role.ecs_task.arn

  container_definitions = jsonencode([{
    name      = "app"
    image     = "${aws_ecr_repository.app.repository_url}:latest"
    essential = true

    portMappings = [{
      containerPort = 8080
      protocol      = "tcp"
    }]

    environment = [
      { name = "PORT", value = "8080" },
      { name = "AWS_REGION", value = var.region },
      { name = "S3_BUCKET", value = aws_s3_bucket.photos.id },
      { name = "ALBUM_TABLE", value = aws_dynamodb_table.albums.name },
      { name = "PHOTO_TABLE", value = aws_dynamodb_table.photos.name },
      { name = "WORKERS", value = "50" },
    ]

    logConfiguration = {
      logDriver = "awslogs"
      options = {
        "awslogs-group"         = aws_cloudwatch_log_group.app.name
        "awslogs-region"        = var.region
        "awslogs-stream-prefix" = "app"
      }
    }
  }])
}

resource "aws_ecs_service" "app" {
  name            = "${var.prefix}-service"
  cluster         = aws_ecs_cluster.app.id
  task_definition = aws_ecs_task_definition.app.arn
  desired_count   = var.desired_count
  launch_type     = "FARGATE"

  network_configuration {
    subnets          = var.subnet_ids
    security_groups  = [aws_security_group.ecs.id]
    assign_public_ip = true
  }

  load_balancer {
    target_group_arn = aws_lb_target_group.app.arn
    container_name   = "app"
    container_port   = 8080
  }

  depends_on = [aws_lb_listener.app]
}
