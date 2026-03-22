variable "project_name" {
  type        = string
  description = "Project name"
}

variable "log_retention_days" {
  type        = number
  description = "Number of days to retain logs"
  default     = 7
}
