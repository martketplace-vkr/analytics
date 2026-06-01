package vendor

import (
	"context"

	"github.com/martketplace-vkr/analytics/domain"
)

type analyticsService interface {
	GetOverview(ctx context.Context, vendorID int64, from string, to string) (domain.Overview, error)
	GetNiches(ctx context.Context, vendorID int64, from string, to string, sortKey string, limit int) ([]domain.NicheMetric, error)
	GetProducts(ctx context.Context, vendorID int64, from string, to string) ([]domain.ProductMetric, error)
	RecordProductView(ctx context.Context, view domain.ProductView) (bool, error)
	UpsertProductCost(ctx context.Context, cost domain.ProductCost) (domain.ProductMetric, error)
	ExportSalesReport(ctx context.Context, vendorID int64, from string, to string, format string) (domain.SalesReport, error)
}
