package entity

import "github.com/kataras/iris/v14/middleware/httpcost"

// Metric is the cost metric for a request.
type Metric struct {
	*httpcost.Metric

	ClientID   string `pg:"type=uuid,ref=oauth2_clients(id cascade),index" json:"client_id"`
	Path       string `pg:"type=varchar(255)" json:"path"`
	StatusCode int    `pg:"type=smallint" json:"status_code"`
}
