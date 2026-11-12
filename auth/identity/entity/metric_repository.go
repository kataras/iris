package entity

import (
	"context"

	"github.com/kataras/pg"
)

// MetricRepository handles cost metric-related database operations.
type MetricRepository struct {
	*pg.Repository[Metric]

	BillingRepository *pg.Repository[Billing]
}

// NewMetricRepository creates a new MetricRepository instance.
func NewMetricRepository(db *pg.DB) *MetricRepository {
	return &MetricRepository{
		Repository:        pg.NewRepository[Metric](db),
		BillingRepository: pg.NewRepository[Billing](db),
	}
}

// InTransaction executes a function within a database transaction.
func (r *MetricRepository) InTransaction(ctx context.Context, fn func(*MetricRepository) error) error {
	conn := r.Repository.DB()
	if conn.IsTransaction() {
		return fn(r)
	}

	return conn.InTransaction(ctx, func(db *pg.DB) error {
		return fn(NewMetricRepository(db))
	})
}

// CreateMetric stores a new Metric to the database.
func (r *MetricRepository) CreateMetric(ctx context.Context, metric Metric) error {
	return r.InsertSingle(ctx, metric, nil)
}

type Billing struct {
	MemoryUsedPretty  string  `pg:"type=varchar" json:"memory_used_pretty"`
	NetworkUsedPretty string  `pg:"type=varchar" json:"network_used_pretty"`
	CPUTimePretty     string  `pg:"type=varchar" json:"cpu_time_pretty"`
	TotalCost         float64 `pg:"type=numeric" json:"total_cost"`
}

func (r *MetricRepository) GetBilling(ctx context.Context) (Billing, error) {
	query := `SELECT
  -- Pretty memory used
  CASE
    WHEN SUM(memory_used) < 1024 THEN ROUND(SUM(memory_used)::numeric, 2) || ' bytes'
    WHEN SUM(memory_used) < 1024^2 THEN ROUND(SUM(memory_used)/1024::numeric, 2) || ' KiB'
    WHEN SUM(memory_used) < 1024^3 THEN ROUND(SUM(memory_used)/1024^2::numeric, 2) || ' MiB'
    WHEN SUM(memory_used) < 1024^4 THEN ROUND(SUM(memory_used)/1024^3::numeric, 2) || ' GiB'
    ELSE ROUND(SUM(memory_used)/1024^4::numeric, 2) || ' TiB'
  END AS memory_used_pretty,

  -- Pretty network used
  CASE
    WHEN SUM(network_used) < 1024 THEN ROUND(SUM(network_used)::numeric, 2) || ' bytes'
    WHEN SUM(network_used) < 1024^2 THEN ROUND(SUM(network_used)/1024::numeric, 2) || ' KiB'
    WHEN SUM(network_used) < 1024^3 THEN ROUND(SUM(network_used)/1024^2::numeric, 2) || ' MiB'
    WHEN SUM(network_used) < 1024^4 THEN ROUND(SUM(network_used)/1024^3::numeric, 2) || ' GiB'
    ELSE ROUND(SUM(network_used)/1024^4::numeric, 2) || ' TiB'
  END AS network_used_pretty,

  -- Pretty CPU time
  CASE
    WHEN SUM(cpu_time) < 60 THEN ROUND(SUM(cpu_time)::numeric, 2) || ' sec'
    WHEN SUM(cpu_time) < 3600 THEN ROUND(SUM(cpu_time)/60::numeric, 2) || ' min'
    WHEN SUM(cpu_time) < 86400 THEN ROUND(SUM(cpu_time)/3600::numeric, 2) || ' hr'
    ELSE ROUND(SUM(cpu_time)/86400::numeric, 2) || ' days'
  END AS cpu_time_pretty,

  -- Total cost
  ROUND(SUM(cost)::numeric, 2) AS total_cost
FROM metrics;`

	return r.BillingRepository.SelectSingle(ctx, query)
}
