resource "aws_dynamodb_table" "shopping_carts" {
  name         = "${var.project_name}-dynamodb"
  billing_mode = "PAY_PER_REQUEST"
  hash_key     = "pk"
  range_key    = "sk"

  attribute {
    name = "pk"
    type = "S"
  }

  attribute {
    name = "sk"
    type = "S"
  }

  tags = {
    Name = "${var.project_name}-dynamodb"
  }
}