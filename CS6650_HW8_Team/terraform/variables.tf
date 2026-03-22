# variables.tf
# store the variables that can be modified
# can be overridden by -var or .tfvars during deployment

variable "aws_region" {
  type        = string
  description = "AWS region (e.g. us-west-2)"
  default     = "us-east-1"
}

variable "project_name" {
  type        = string
  description = "Project name, used for resource naming"
  default     = "shopping-cart-hw8"
}

variable "image_tag" {
  type        = string
  description = "Docker image tag, used for ECS tasks"
  default     = "v2"
}

variable "cart_min_count" {
  type        = number
  description = "Minimum number of ECS tasks (auto scaling lower bound)"
  default     = 1
}

variable "cart_max_count" {
  type        = number
  description = "Maximum number of ECS tasks (auto scaling upper bound)"
  default     = 4
}

variable "worker_count" {
  type        = number
  description = "Number of worker goroutines in the order processor"
  default     = 1
}

variable "cpu_target_value" {
  type        = number
  description = "Target CPU utilization (%) that triggers auto scaling"
  default     = 70
}

variable "rds_db_name" {
  type        = string
  description = "RDS initial database name"
  default     = "app"
}

variable "rds_db_username" {
  type        = string
  description = "RDS master username"
  default     = "admin"
}

variable "rds_db_password" {
  type        = string
  description = "RDS master password"
  sensitive   = true
}

variable "enable_order_processor_lambda" {
  type        = bool
  description = "Whether to deploy the async order-processor Lambda (not required for cart performance tests)"
  default     = false
}
