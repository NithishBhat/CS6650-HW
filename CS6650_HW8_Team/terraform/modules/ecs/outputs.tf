output "cluster_name" {
  description = "Name of the created ECS cluster"
  value       = aws_ecs_cluster.main.name
}

output "service_name" {
  description = "Name of the created ECS service"
  value       = aws_ecs_service.cart.name
}
