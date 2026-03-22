output "cart_repository_url" {
  description = "URL of the Shopping Cart ECR repository"
  value       = aws_ecr_repository.cart.repository_url
}

output "cart_repository_arn" {
  description = "ARN of the Shopping Cart ECR repository"
  value       = aws_ecr_repository.cart.arn
}
