package admin

import (
	"context"

	"github.com/martketplace-vkr/analytics/domain"
)

type analyticsService interface {
	ListTariffs(ctx context.Context) ([]domain.Tariff, error)
	CreateTariff(ctx context.Context, tariff domain.Tariff) (domain.Tariff, error)
	UpdateTariff(ctx context.Context, tariff domain.Tariff) (domain.Tariff, error)
	SetDefaultTariff(ctx context.Context, tariffID int64) (domain.Tariff, error)
	AssignVendorTariff(ctx context.Context, vendorID int64, tariffID int64) (domain.VendorTariffAssignment, error)
	GetVendorTariff(ctx context.Context, vendorID int64) (domain.Tariff, error)
}
