output "base_url" {
  value = "http://${aws_lb.app.dns_name}"
}

output "alb_dns" {
  value = aws_lb.app.dns_name
}

output "ecr_url" {
  value = aws_ecr_repository.app.repository_url
}

output "photos_bucket" {
  value = aws_s3_bucket.photos.id
}

output "log_group" {
  value = aws_cloudwatch_log_group.app.name
}
