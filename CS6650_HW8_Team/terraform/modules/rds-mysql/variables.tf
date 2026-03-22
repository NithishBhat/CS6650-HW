variable "project_name" {
  type        = string
  description = "Project name"
}

variable "vpc_id" {
  type        = string
  description = "VPC ID"
}

variable "subnet_ids" {
  type        = list(string)
  description = "Private subnet IDs for the RDS instance"
}

variable "ecs_security_group_id" {
  type        = string
  description = "Security group ID for ECS tasks (only this SG can reach RDS)"
}

variable "db_port" {
  type        = number
  description = "Database port"
  default     = 3306
}

variable "allocated_storage" {
  type        = number
  description = "RDS storage in GB (Free tier-friendly)"
  default     = 20
}

variable "db_name" {
  type        = string
  description = "Initial database name"
  default     = "app"
}

variable "db_username" {
  type        = string
  description = "Master username"
  default     = "admin"
}

variable "db_password" {
  type        = string
  description = "Master password"
}

