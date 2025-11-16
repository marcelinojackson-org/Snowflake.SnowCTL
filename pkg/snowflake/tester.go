package snowflake

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/snowflakedb/gosnowflake"
	_ "github.com/snowflakedb/gosnowflake"

	"github.com/Snowflake-Labs/Snowflake.SnowCTL/pkg/config"
)

// TestConnection attempts to connect to Snowflake using the provided connection info and secret.
// It returns the server's CURRENT_TIMESTAMP upon success.
func TestConnection(ctx context.Context, info *config.Context, secret string) (string, error) {
	cfg := &gosnowflake.Config{
		Account:   info.Account,
		User:      info.User,
		Role:      info.Role,
		Warehouse: info.Warehouse,
		Database:  info.Database,
		Schema:    info.Schema,
	}

	switch info.AuthMethod {
	case "pat":
		cfg.Password = secret
	default:
		cfg.Password = secret
	}

	dsn, err := gosnowflake.DSN(cfg)
	if err != nil {
		return "", fmt.Errorf("build DSN: %w", err)
	}

	db, err := sql.Open("snowflake", dsn)
	if err != nil {
		return "", fmt.Errorf("open connection: %w", err)
	}
	defer db.Close()

	pingCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	if err := db.PingContext(pingCtx); err != nil {
		return "", fmt.Errorf("ping snowflake: %w", err)
	}

	var serverTime string
	if err := db.QueryRowContext(pingCtx, "select current_timestamp()").Scan(&serverTime); err != nil {
		return "", fmt.Errorf("query server time: %w", err)
	}

	return serverTime, nil
}

// RunQuery executes the provided SQL and returns rows as maps.
func RunQuery(ctx context.Context, info *config.Context, secret, stmt string) ([]map[string]any, error) {
	cfg := &gosnowflake.Config{
		Account:   info.Account,
		User:      info.User,
		Role:      info.Role,
		Warehouse: info.Warehouse,
		Database:  info.Database,
		Schema:    info.Schema,
	}

	switch info.AuthMethod {
	case "pat":
		cfg.Password = secret
	default:
		cfg.Password = secret
	}

	dsn, err := gosnowflake.DSN(cfg)
	if err != nil {
		return nil, fmt.Errorf("build DSN: %w", err)
	}

	db, err := sql.Open("snowflake", dsn)
	if err != nil {
		return nil, fmt.Errorf("open connection: %w", err)
	}
	defer db.Close()

	queryCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	rows, err := db.QueryContext(queryCtx, stmt)
	if err != nil {
		return nil, fmt.Errorf("execute query: %w", err)
	}
	defer rows.Close()

	cols, err := rows.Columns()
	if err != nil {
		return nil, fmt.Errorf("fetch columns: %w", err)
	}

	var results []map[string]any
	for rows.Next() {
		values := make([]interface{}, len(cols))
		scans := make([]interface{}, len(cols))
		for i := range values {
			scans[i] = &values[i]
		}
		if err := rows.Scan(scans...); err != nil {
			return nil, fmt.Errorf("scan row: %w", err)
		}
		row := make(map[string]any, len(cols))
		for i, col := range cols {
			val := values[i]
			if b, ok := val.([]byte); ok {
				row[col] = string(b)
			} else {
				row[col] = val
			}
		}
		results = append(results, row)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows error: %w", err)
	}
	return results, nil
}
