output "cart_log_group_name" {
  description = "Name of the CloudWatch log group for the Shopping Cart service"
  value       = aws_cloudwatch_log_group.cart.name
}
