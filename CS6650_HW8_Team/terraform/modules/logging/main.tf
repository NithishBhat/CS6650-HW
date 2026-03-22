# logging/main.tf
# create a CloudWatch log group for the Shopping Cart service
# collect the container logs for the Shopping Cart service

resource "aws_cloudwatch_log_group" "cart" {
  name               = "/ecs/${var.project_name}-cart"
  retention_in_days  = var.log_retention_days

  tags = { Name = "${var.project_name}-cart" }
}
