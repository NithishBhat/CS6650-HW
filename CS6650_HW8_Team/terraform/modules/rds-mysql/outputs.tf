output "db_endpoint" {
  description = "RDS endpoint address"
  value       = aws_db_instance.mysql.address
}

output "db_port" {
  description = "RDS port"
  value       = aws_db_instance.mysql.port
}

output "db_name" {
  description = "Initial database name"
  value       = aws_db_instance.mysql.db_name
}

output "rds_security_group_id" {
  description = "ID of the security group attached to the RDS instance"
  value       = aws_security_group.rds.id
}

