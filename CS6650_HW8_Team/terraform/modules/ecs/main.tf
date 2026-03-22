# ecs/main.tf
# create an Elastic Container Service (ECS) cluster
# define the ECS task definition and service

data "aws_iam_role" "lab" {
  name = "LabRole"
}

# ecs cluster
resource "aws_ecs_cluster" "main" {
  name = var.project_name

  setting {
    name  = "containerInsights"
    value = "disabled"
  }

  tags = { Name = var.project_name }
}

# ecs task definition: CPU 256, Memory 512MB
resource "aws_ecs_task_definition" "cart" {
  family                   = "${var.project_name}-cart"
  network_mode             = "awsvpc"
  requires_compatibilities = ["FARGATE"]
  cpu                      = "256"
  memory                   = "512"

  execution_role_arn = data.aws_iam_role.lab.arn
  task_role_arn      = data.aws_iam_role.lab.arn

  container_definitions = jsonencode([
    {
      name  = "cart"
      image = "${var.ecr_cart_url}:${var.image_tag}"
      portMappings = [
        { containerPort = 8080, hostPort = 8080, protocol = "tcp" }
      ]
      logConfiguration = {
        logDriver = "awslogs"
        options = {
          "awslogs-group"         = var.log_group_name
          "awslogs-region"        = var.aws_region
          "awslogs-stream-prefix" = "ecs"
        }
      }
      environment = [
        {
          name  = "APP_MODE"
          value = "receiver"
        },
        {
          name  = "AWS_REGION"
          value = var.aws_region
        },
        {
          name  = "DYNAMODB_TABLE"
          value = var.dynamodb_table_name
        },
        {
          name  = "ORDER_TOPIC_ARN"
          value = var.order_processing_topic_arn
        },
        {
          name  = "DB_HOST"
          value = var.db_host
        },
        {
          name  = "DB_PORT"
          value = tostring(var.db_port)
        },
        {
          name  = "DB_NAME"
          value = var.db_name
        },
        {
          name  = "DB_USER"
          value = var.db_username
        },
        {
          name  = "DB_PASSWORD"
          value = var.db_password
        }
      ]
    }
  ])

  tags = { Name = "${var.project_name}-cart" }
}


# processor task definition: no http
resource "aws_ecs_task_definition" "processor" {
  family                   = "${var.project_name}-processor"
  network_mode             = "awsvpc"
  requires_compatibilities = ["FARGATE"]
  cpu                      = "256"
  memory                   = "512"

  execution_role_arn = data.aws_iam_role.lab.arn
  task_role_arn      = data.aws_iam_role.lab.arn

  container_definitions = jsonencode([
    {
      name  = "processor"
      image = "${var.ecr_cart_url}:${var.image_tag}"

      logConfiguration = {
        logDriver = "awslogs"
        options = {
          "awslogs-group"         = var.log_group_name
          "awslogs-region"        = var.aws_region
          "awslogs-stream-prefix" = "ecs-processor"
        }
      }

      environment = [
        {
          name  = "APP_MODE"
          value = "processor"
        },
        {
          name  = "AWS_REGION"
          value = var.aws_region
        },
        {
          name  = "ORDER_QUEUE_URL"
          value = var.order_processing_queue_url
        },
        {
          name  = "WORKER_COUNT"
          value = tostring(var.worker_count)
        }
      ]
    }
  ])

  tags = { Name = "${var.project_name}-processor" }
}

# ecs service (runs in private subnets, no public IP)
resource "aws_ecs_service" "cart" {
  name            = "${var.project_name}-cart"
  cluster         = aws_ecs_cluster.main.id
  task_definition = aws_ecs_task_definition.cart.arn
  desired_count   = var.desired_count
  launch_type     = "FARGATE"

  network_configuration {
    subnets          = var.subnet_ids
    security_groups  = [var.ecs_security_group_id]
    assign_public_ip = false
  }

  load_balancer {
    target_group_arn = var.target_group_arn
    container_name   = "cart"
    container_port   = 8080
  }

  # Prevent Terraform from resetting desired_count after auto scaling changes it
  lifecycle {
    ignore_changes = [desired_count]
  }
}

# ecs service (processor no need to add ALB)
resource "aws_ecs_service" "processor" {
  name            = "${var.project_name}-processor"
  cluster         = aws_ecs_cluster.main.id
  task_definition = aws_ecs_task_definition.processor.arn
  desired_count   = 1
  launch_type     = "FARGATE"

  network_configuration {
    subnets          = var.subnet_ids
    security_groups  = [var.ecs_security_group_id]
    assign_public_ip = false
  }
}

# Auto Scaling: register the ECS service as a scalable target
resource "aws_appautoscaling_target" "ecs" {
  max_capacity       = var.max_count
  min_capacity       = var.min_count
  resource_id        = "service/${aws_ecs_cluster.main.name}/${aws_ecs_service.cart.name}"
  scalable_dimension = "ecs:service:DesiredCount"
  service_namespace  = "ecs"
}

# Auto Scaling policy: scale based on average CPU utilization
resource "aws_appautoscaling_policy" "cpu" {
  name               = "${var.project_name}-cpu-tracking"
  policy_type        = "TargetTrackingScaling"
  resource_id        = aws_appautoscaling_target.ecs.resource_id
  scalable_dimension = aws_appautoscaling_target.ecs.scalable_dimension
  service_namespace  = aws_appautoscaling_target.ecs.service_namespace

  target_tracking_scaling_policy_configuration {
    predefined_metric_specification {
      predefined_metric_type = "ECSServiceAverageCPUUtilization"
    }
    target_value       = var.cpu_target_value
    scale_out_cooldown = 300
    scale_in_cooldown  = 300
  }
}
