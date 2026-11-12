package identity

import (
	stdContext "context"
	"time"

	"github.com/kataras/iris/v14/auth/identity/entity"
	"github.com/kataras/iris/v14/auth/oauth2"

	"github.com/kataras/iris/v14/context"
	"github.com/kataras/iris/v14/middleware/httpcost"

	"github.com/kataras/pg"
)

func resolveClientID(ctx *context.Context) string {
	// retrieve from client verifier middleware (for server apps).
	if client, ok := oauth2.GetClient(ctx); ok {
		return client.ID // we are sure client exists.
	}

	// retrieve from token (for either server or client apps that sends the token through the header).
	if claims, err := oauth2.GetClientClaims(ctx); err == nil {
		return claims.ClientID // we are not sure if client exists.
	}

	// retrieve from url parameter, from within oauth2/* endpoints.
	if clientID := ctx.Form().Get("client_id"); clientID != "" { // post, put, patch data or url query.
		return clientID // we are not sure if client exists.
	}

	return ""
}

// NewCostHandler returns a new cost handler that calculates the cost of a request
// and stores it in the provided MetricStore.
func NewCostHandler(opts httpcost.Options, db *pg.DB) context.Handler {
	if opts.CostPerCPU == 0 && opts.CostPerMB == 0 && opts.CostPerSecond == 0 {
		return nil
	}

	r := entity.NewRegistry(db)
	computer := httpcost.NewComputer(opts)
	return func(ctx *context.Context) {
		proc := computer.Start()
		// defer func() {
		// 	if err := recover(); err != nil {
		// 	}
		// No, if we have a panic don't calculate the cost, it is on us.
		// }()
		ctx.Next()
		clientID := resolveClientID(ctx) // we should resolve it after, so client verifiers and all that happen upfront.
		if clientID == "" {              // do nothing, do not calculate any cost, it's not an app call.
			ctx.Next()
			return
		}
		path := ctx.Path()
		statusCode := ctx.GetStatusCode()

		costMetric := proc.Stop(ctx.ResponseWriter().Written())
		go func() {
			cancelCtx, cancelFn := stdContext.WithTimeout(stdContext.Background(), 15*time.Second)
			defer cancelFn()

			metric := entity.Metric{
				Metric:     costMetric,
				ClientID:   clientID,
				Path:       path,
				StatusCode: statusCode,
			}

			r.Metrics().CreateMetric(cancelCtx, metric) // ignore error for now.
			// TODO: we should catch it, and filter out if it's an error from client not exist (if we allow on status codes,
			// we could filter out bad requests or 404s),
			// through Patrol.

			//
			// To pretty print the total cost and total size of a client:
			/* pg_pretty_size(memory_used) OR:
			SELECT
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
			FROM metrics
			WHERE client_id = '762ef5af-fd36-479a-aeea-dcc42251c47e';
			*/
		}()
	}
}
