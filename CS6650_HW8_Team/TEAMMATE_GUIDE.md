# Go App Integration Guide (for MySQL / RDS)

## Overview

The RDS MySQL instance runs in a **private subnet** and is only accessible from ECS tasks.
You **cannot** connect from your local machine — this is by design.
The Go app must initialize the database schema on startup.

## Environment Variables Available in ECS

The following env vars are injected into the `cart` container automatically via Terraform:

| Variable      | Description                  | Example                                      |
|---------------|------------------------------|----------------------------------------------|
| `DB_HOST`     | RDS endpoint address         | `shopping-cart-hw8-mysql.xxx.us-west-2.rds.amazonaws.com` |
| `DB_PORT`     | MySQL port                   | `3306`                                       |
| `DB_NAME`     | Database name                | `app`                                        |
| `DB_USER`     | MySQL username               | `admin`                                      |
| `DB_PASSWORD` | MySQL password               | *(set via terraform.tfvars)*                 |

## Schema Auto-Initialization

Since no one can connect to RDS from outside the VPC, the Go app **must** create tables on startup.

Run the SQL from `sql/schema.sql` at boot. Example pattern:

```go
import (
    "database/sql"
    "fmt"
    "os"

    _ "github.com/go-sql-driver/mysql"
)

func initDB() (*sql.DB, error) {
    dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?parseTime=true",
        os.Getenv("DB_USER"),
        os.Getenv("DB_PASSWORD"),
        os.Getenv("DB_HOST"),
        os.Getenv("DB_PORT"),
        os.Getenv("DB_NAME"),
    )

    db, err := sql.Open("mysql", dsn)
    if err != nil {
        return nil, err
    }

    // Connection pool settings
    db.SetMaxOpenConns(25)
    db.SetMaxIdleConns(5)

    // Auto-create tables (safe to run every time)
    if err := runSchema(db); err != nil {
        return nil, err
    }

    return db, nil
}

func runSchema(db *sql.DB) error {
    cartsTable := `
    CREATE TABLE IF NOT EXISTS shopping_carts (
        cart_id     BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
        customer_id BIGINT UNSIGNED NOT NULL,
        status      ENUM('active', 'checked_out') NOT NULL DEFAULT 'active',
        created_at  DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
        updated_at  DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
        PRIMARY KEY (cart_id),
        KEY idx_carts_customer_created (customer_id, created_at DESC)
    ) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;`

    itemsTable := `
    CREATE TABLE IF NOT EXISTS cart_items (
        item_id    BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
        cart_id    BIGINT UNSIGNED NOT NULL,
        product_id BIGINT UNSIGNED NOT NULL,
        quantity   INT UNSIGNED    NOT NULL DEFAULT 1,
        created_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
        updated_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
        PRIMARY KEY (item_id),
        UNIQUE KEY uk_cart_product (cart_id, product_id),
        CONSTRAINT fk_cart_items_cart FOREIGN KEY (cart_id)
            REFERENCES shopping_carts (cart_id) ON DELETE CASCADE
    ) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;`

    if _, err := db.Exec(cartsTable); err != nil {
        return err
    }
    _, err := db.Exec(itemsTable)
    return err
}
```

> Both tables use `CREATE TABLE IF NOT EXISTS`, safe to run on every startup.

## Deployment Workflow

```
terraform apply          # Creates RDS, ECS, networking, etc.
docker build & push      # Push Go app image to ECR
# ECS picks up new image → app starts → auto-creates tables → ready
```

## Do NOT

- Try to connect to RDS from your local machine (it will timeout — private subnet)
- Hardcode DB credentials in the Go app (use env vars above)
- Skip `CREATE TABLE IF NOT EXISTS` (there is no other way to init the schema)
