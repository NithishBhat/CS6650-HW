variable "project_name" {
  type        = string
  description = "Project name"
}

variable "aws_region" {
  type        = string
  description = "AWS region"
}

variable "image_tag" {
  type        = string
  description = "Docker image tag"
}

variable "ecr_cart_url" {
  type        = string
  description = "URL of the Shopping Cart ECR repository"
}

variable "log_group_name" {
  type        = string
  description = "Name of the CloudWatch log group"
}

variable "subnet_ids" {
  type        = list(string)
  description = "List of private subnet IDs for ECS tasks"
}

variable "ecs_security_group_id" {
  type        = string
  description = "ID of the ECS task security group"
}

variable "desired_count" {
  type        = number
  description = "Initial desired number of tasks (auto scaling will take over)"
  default     = 2
}

variable "min_count" {
  type        = number
  description = "Minimum number of tasks for auto scaling"
  default     = 2
}

variable "max_count" {
  type        = number
  description = "Maximum number of tasks for auto scaling"
  default     = 4
}

variable "target_group_arn" {
  type        = string
  description = "ARN of the ALB target group to attach the ECS service to"
}

variable "cpu_target_value" {
  type        = number
  description = "Target CPU utilization (%) for auto scaling"
  default     = 70
}

variable "worker_count" {
  type        = number
  description = "Number of worker goroutines in the processor"
  default     = 1
}

variable "order_processing_topic_arn" {
  type        = string
  description = "ARN of the order processing topic"
}

variable "order_processing_queue_url" {
  type        = string
  description = "URL of the order processing SQS queue"
}

variable "db_host" {
  type        = string
  description = "RDS MySQL endpoint address"
}

variable "db_port" {
  type        = number
  description = "RDS MySQL port"
  default     = 3306
}

variable "db_name" {
  type        = string
  description = "RDS MySQL database name"
}

variable "db_username" {
  type        = string
  description = "RDS MySQL username"
}

variable "db_password" {
  type        = string
  description = "RDS MySQL password"
  sensitive   = true
}

variable "dynamodb_table_name" {
  type        = string
  description = "DynamoDB table name for shopping carts"
}