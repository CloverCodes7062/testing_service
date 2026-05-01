package db

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

var Pool *pgxpool.Pool

func InitDB(ctx context.Context, databaseURL string) error {
	config, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return err
	}
	config.MaxConns = 10

	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return err
	}

	if err := pool.Ping(ctx); err != nil {
		return err
	}

	Pool = pool
	return nil
}

func PoolStats() map[string]int64 {
	if Pool == nil {
		return map[string]int64{"total": 0, "idle": 0, "inUse": 0}
	}
	stat := Pool.Stat()
	return map[string]int64{
		"total":  int64(stat.TotalConns()),
		"idle":   int64(stat.IdleConns()),
		"inUse":  int64(stat.AcquiredConns()),
	}
}
