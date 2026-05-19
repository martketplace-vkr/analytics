package service

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/martketplace-vkr/analytics/domain"
)

type fakeRepository struct {
	ownsProduct bool
	rows        []domain.SalesReportRow
}

func (f fakeRepository) RefreshAggregates(context.Context) error { return nil }
func (f fakeRepository) GetOverview(context.Context, int64, domain.DateRange) (domain.Overview, error) {
	return domain.Overview{}, nil
}
func (f fakeRepository) GetNiches(context.Context, int64, domain.DateRange, string, int) ([]domain.NicheMetric, error) {
	return nil, nil
}
func (f fakeRepository) GetProducts(context.Context, int64, domain.DateRange) ([]domain.ProductMetric, error) {
	return []domain.ProductMetric{{ProductID: 10, CostPrice: "12.50", HasCost: true}}, nil
}
func (f fakeRepository) UpsertProductCost(context.Context, domain.ProductCost) error { return nil }
func (f fakeRepository) VendorOwnsProduct(context.Context, int64, int64) (bool, error) {
	return f.ownsProduct, nil
}
func (f fakeRepository) GetSalesReportRows(context.Context, int64, domain.DateRange, int) ([]domain.SalesReportRow, error) {
	return f.rows, nil
}

func TestUpsertProductCostRejectsForeignProduct(t *testing.T) {
	svc := New(fakeRepository{ownsProduct: false})
	_, err := svc.UpsertProductCost(context.Background(), domain.ProductCost{
		VendorID:  1,
		ProductID: 10,
		CostPrice: "12.50",
	})
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected not found, got %v", err)
	}
}

func TestExportSalesReportCSV(t *testing.T) {
	svc := New(fakeRepository{
		rows: []domain.SalesReportRow{
			{
				Day:           "2026-05-19",
				OrderID:       1,
				Status:        "success",
				ProductID:     10,
				ProductName:   "Товар",
				CategoryName:  "Категория",
				Quantity:      2,
				UnitPrice:     "100.00",
				TotalPrice:    "200.00",
				CostPrice:     "70.00",
				GrossProfit:   "60.00",
				MarginPercent: 30,
			},
		},
	})

	report, err := svc.ExportSalesReport(context.Background(), 1, "2026-05-01", "2026-05-19", "csv")
	if err != nil {
		t.Fatalf("export csv: %v", err)
	}
	if report.ContentType != "text/csv; charset=utf-8" {
		t.Fatalf("unexpected content type: %s", report.ContentType)
	}
	if !strings.Contains(string(report.Content), "Валовая прибыль") {
		t.Fatal("csv must contain headers")
	}
	if !strings.Contains(string(report.Content), "Товар") {
		t.Fatal("csv must contain row data")
	}
}

func TestExportSalesReportXLSX(t *testing.T) {
	svc := New(fakeRepository{})
	report, err := svc.ExportSalesReport(context.Background(), 1, "2026-05-01", "2026-05-19", "xlsx")
	if err != nil {
		t.Fatalf("export xlsx: %v", err)
	}
	if report.ContentType != "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet" {
		t.Fatalf("unexpected content type: %s", report.ContentType)
	}
	if len(report.Content) < 4 || string(report.Content[:2]) != "PK" {
		t.Fatal("xlsx must be a zip payload")
	}
}

func TestExportSalesReportRejectsUnsupportedFormat(t *testing.T) {
	svc := New(fakeRepository{})
	_, err := svc.ExportSalesReport(context.Background(), 1, "2026-05-01", "2026-05-19", "pdf")
	if !errors.Is(err, ErrInvalidArgument) {
		t.Fatalf("expected invalid argument, got %v", err)
	}
}
