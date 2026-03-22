# outputs.tf
# key output values for interaction with the system

output "ecr_cart_url" {
  description = "Shopping Cart image pushed to ECR address"
  value       = module.ecr.cart_repository_url
}

output "ecs_cluster_name" {
  description = "Name of the created ECS cluster"
  value       = module.ecs.cluster_name
}

output "ecs_service_name" {
  description = "Name of the created ECS service"
  value       = module.ecs.service_name
}

output "alb_dns_name" {
  description = "ALB DNS name — use this as the Locust host"
  value       = module.alb.alb_dns_name
}

output "order_processing_topic_arn" {
  value = aws_sns_topic.order_processing_events.arn
}

output "order_processing_queue_url" {
  value = aws_sqs_queue.order_processing_queue.id
}

output "rds_mysql_endpoint" {
  description = "RDS MySQL endpoint address (private)"
  value       = module.rds_mysql.db_endpoint
}

output "rds_mysql_port" {
  description = "RDS MySQL port"
  value       = module.rds_mysql.db_port
}

output "rds_mysql_db_name" {
  description = "RDS MySQL database name"
  value       = module.rds_mysql.db_name
}

output "dynamodb_table_name" {
  description = "DynamoDB table name for shopping carts"
  value       = aws_dynamodb_table.shopping_carts.name
}