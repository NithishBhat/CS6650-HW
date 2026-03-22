# MySQL 8.0 RDS module for Free tier (db.t3.micro)

resource "aws_security_group" "rds" {
  name        = "${var.project_name}-rds-mysql"
  description = "Allow MySQL access from ECS tasks only"
  vpc_id      = var.vpc_id

  ingress {
    description      = "MySQL 3306 from ECS tasks"
    from_port        = var.db_port
    to_port          = var.db_port
    protocol         = "tcp"
    security_groups  = [var.ecs_security_group_id]
  }

  egress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }

  tags = { Name = "${var.project_name}-rds-mysql" }
}

resource "aws_db_subnet_group" "this" {
  name       = "${var.project_name}-rds-subnet-group"
  subnet_ids = var.subnet_ids

  tags = { Name = "${var.project_name}-rds-subnet-group" }
}

resource "aws_db_instance" "mysql" {
  identifier = "${var.project_name}-mysql"

  engine               = "mysql"
  engine_version      = "8.0.45"
  instance_class       = "db.t3.micro"
  allocated_storage    = var.allocated_storage
  storage_type         = "gp2"

  db_name = var.db_name
  username = var.db_username
  password = var.db_password
  port     = var.db_port

  db_subnet_group_name   = aws_db_subnet_group.this.name
  vpc_security_group_ids = [aws_security_group.rds.id]

  publicly_accessible = false
  multi_az            = false

  # For assignments: skip final snapshot and disable deletion protection.
  skip_final_snapshot   = true
  deletion_protection   = false
  backup_retention_period = 0

  apply_immediately = true

  tags = { Name = "${var.project_name}-mysql" }
}

