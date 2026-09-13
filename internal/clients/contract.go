package clients

import "context"

type Client interface {
	CheckService(ctx context.Context, companyID string, serviceCode string) (CheckResult, error)
}

type CheckResult struct {
	Allowed bool
	Reason  string
}
