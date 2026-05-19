package service

import (
	"bytes"
	"context"
	"encoding/csv"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/martketplace-vkr/analytics/domain"
	"github.com/martketplace-vkr/analytics/internal/repository/pg"
	"github.com/xuri/excelize/v2"
)

const (
	defaultPeriodDays = 30
	maxReportDays     = 365
	reportRowLimit    = 50000
)

type repository interface {
	RefreshAggregates(ctx context.Context) error
	GetOverview(ctx context.Context, vendorID int64, dateRange domain.DateRange) (domain.Overview, error)
	GetNiches(ctx context.Context, vendorID int64, dateRange domain.DateRange, sortKey string, limit int) ([]domain.NicheMetric, error)
	GetProducts(ctx context.Context, vendorID int64, dateRange domain.DateRange) ([]domain.ProductMetric, error)
	UpsertProductCost(ctx context.Context, cost domain.ProductCost) error
	VendorOwnsProduct(ctx context.Context, vendorID int64, productID int64) (bool, error)
	GetSalesReportRows(ctx context.Context, vendorID int64, dateRange domain.DateRange, limit int) ([]domain.SalesReportRow, error)
}

type Service struct {
	repository repository
}

func New(repository repository) *Service {
	return &Service{repository: repository}
}

func (s *Service) RefreshAggregates(ctx context.Context) error {
	return s.repository.RefreshAggregates(ctx)
}

func (s *Service) GetOverview(ctx context.Context, vendorID int64, from string, to string) (domain.Overview, error) {
	dateRange, err := parseDateRange(from, to)
	if err != nil {
		return domain.Overview{}, err
	}
	if vendorID <= 0 {
		return domain.Overview{}, fmt.Errorf("%w: vendor_id must be positive", ErrInvalidArgument)
	}
	return s.repository.GetOverview(ctx, vendorID, dateRange)
}

func (s *Service) GetNiches(ctx context.Context, vendorID int64, from string, to string, sortKey string, limit int) ([]domain.NicheMetric, error) {
	dateRange, err := parseDateRange(from, to)
	if err != nil {
		return nil, err
	}
	if vendorID <= 0 {
		return nil, fmt.Errorf("%w: vendor_id must be positive", ErrInvalidArgument)
	}
	switch sortKey {
	case "", "opportunity", "demand", "margin":
	default:
		return nil, fmt.Errorf("%w: unsupported sort", ErrInvalidArgument)
	}
	return s.repository.GetNiches(ctx, vendorID, dateRange, sortKey, limit)
}

func (s *Service) GetProducts(ctx context.Context, vendorID int64, from string, to string) ([]domain.ProductMetric, error) {
	dateRange, err := parseDateRange(from, to)
	if err != nil {
		return nil, err
	}
	if vendorID <= 0 {
		return nil, fmt.Errorf("%w: vendor_id must be positive", ErrInvalidArgument)
	}
	return s.repository.GetProducts(ctx, vendorID, dateRange)
}

func (s *Service) UpsertProductCost(ctx context.Context, cost domain.ProductCost) (domain.ProductMetric, error) {
	if cost.VendorID <= 0 {
		return domain.ProductMetric{}, fmt.Errorf("%w: vendor_id must be positive", ErrInvalidArgument)
	}
	if cost.ProductID <= 0 {
		return domain.ProductMetric{}, fmt.Errorf("%w: product_id must be positive", ErrInvalidArgument)
	}
	normalizedCost, err := normalizeCost(cost.CostPrice)
	if err != nil {
		return domain.ProductMetric{}, err
	}

	owns, err := s.repository.VendorOwnsProduct(ctx, cost.VendorID, cost.ProductID)
	if err != nil {
		return domain.ProductMetric{}, err
	}
	if !owns {
		return domain.ProductMetric{}, ErrNotFound
	}

	cost.CostPrice = normalizedCost
	if strings.TrimSpace(cost.Currency) == "" {
		cost.Currency = "RUB"
	}

	if err = s.repository.UpsertProductCost(ctx, cost); err != nil {
		return domain.ProductMetric{}, err
	}

	_ = s.repository.RefreshAggregates(ctx)

	products, err := s.repository.GetProducts(ctx, cost.VendorID, defaultDateRange())
	if err != nil {
		return domain.ProductMetric{}, err
	}
	for _, product := range products {
		if product.ProductID == cost.ProductID {
			return product, nil
		}
	}

	return domain.ProductMetric{}, ErrNotFound
}

func (s *Service) ExportSalesReport(ctx context.Context, vendorID int64, from string, to string, format string) (domain.SalesReport, error) {
	dateRange, err := parseDateRange(from, to)
	if err != nil {
		return domain.SalesReport{}, err
	}
	if vendorID <= 0 {
		return domain.SalesReport{}, fmt.Errorf("%w: vendor_id must be positive", ErrInvalidArgument)
	}
	if dateRange.To.Sub(dateRange.From) > maxReportDays*24*time.Hour {
		return domain.SalesReport{}, fmt.Errorf("%w: report period must be 365 days or less", ErrInvalidArgument)
	}

	rows, err := s.repository.GetSalesReportRows(ctx, vendorID, dateRange, reportRowLimit)
	if errors.Is(err, pg.ErrReportTooLarge) {
		return domain.SalesReport{}, ErrReportTooLarge
	}
	if err != nil {
		return domain.SalesReport{}, err
	}

	format = strings.ToLower(strings.TrimSpace(format))
	if format == "" {
		format = "csv"
	}

	filename := fmt.Sprintf("sales-report-%s-%s.%s", dateRange.From.Format("2006-01-02"), dateRange.To.Format("2006-01-02"), format)
	switch format {
	case "csv":
		content, err := buildCSVReport(rows)
		if err != nil {
			return domain.SalesReport{}, err
		}
		return domain.SalesReport{Content: content, Filename: filename, ContentType: "text/csv; charset=utf-8"}, nil
	case "xlsx":
		content, err := buildXLSXReport(rows)
		if err != nil {
			return domain.SalesReport{}, err
		}
		return domain.SalesReport{Content: content, Filename: filename, ContentType: "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"}, nil
	default:
		return domain.SalesReport{}, fmt.Errorf("%w: unsupported report format", ErrInvalidArgument)
	}
}

func parseDateRange(from string, to string) (domain.DateRange, error) {
	now := time.Now().UTC()
	toDate := truncateDate(now)
	fromDate := toDate.AddDate(0, 0, -defaultPeriodDays+1)

	if strings.TrimSpace(to) != "" {
		parsed, err := parseDate(to)
		if err != nil {
			return domain.DateRange{}, err
		}
		toDate = parsed
	}

	if strings.TrimSpace(from) != "" {
		parsed, err := parseDate(from)
		if err != nil {
			return domain.DateRange{}, err
		}
		fromDate = parsed
	}

	if fromDate.After(toDate) {
		return domain.DateRange{}, fmt.Errorf("%w: from must be before to", ErrInvalidArgument)
	}

	return domain.DateRange{From: fromDate, To: toDate}, nil
}

func defaultDateRange() domain.DateRange {
	to := truncateDate(time.Now().UTC())
	return domain.DateRange{From: to.AddDate(0, 0, -defaultPeriodDays+1), To: to}
}

func parseDate(value string) (time.Time, error) {
	parsed, err := time.Parse("2006-01-02", strings.TrimSpace(value))
	if err != nil {
		return time.Time{}, fmt.Errorf("%w: date must use YYYY-MM-DD", ErrInvalidArgument)
	}
	return parsed, nil
}

func truncateDate(value time.Time) time.Time {
	return time.Date(value.Year(), value.Month(), value.Day(), 0, 0, 0, 0, time.UTC)
}

func normalizeCost(value string) (string, error) {
	normalized := strings.ReplaceAll(strings.TrimSpace(value), " ", "")
	normalized = strings.ReplaceAll(normalized, ",", ".")
	parsed, err := strconv.ParseFloat(normalized, 64)
	if err != nil || parsed < 0 {
		return "", fmt.Errorf("%w: cost_price must be a non-negative decimal", ErrInvalidArgument)
	}
	return strconv.FormatFloat(parsed, 'f', 2, 64), nil
}

func buildCSVReport(rows []domain.SalesReportRow) ([]byte, error) {
	buffer := &bytes.Buffer{}
	buffer.WriteString("\xEF\xBB\xBF")
	writer := csv.NewWriter(buffer)

	if err := writer.Write(reportHeaders()); err != nil {
		return nil, err
	}

	for _, row := range rows {
		if err := writer.Write(reportRecord(row)); err != nil {
			return nil, err
		}
	}

	writer.Flush()
	if err := writer.Error(); err != nil {
		return nil, err
	}

	return buffer.Bytes(), nil
}

func buildXLSXReport(rows []domain.SalesReportRow) ([]byte, error) {
	file := excelize.NewFile()
	const sheet = "Продажи"
	index, err := file.NewSheet(sheet)
	if err != nil {
		return nil, err
	}
	file.DeleteSheet("Sheet1")
	file.SetActiveSheet(index)

	for idx, header := range reportHeaders() {
		cell, _ := excelize.CoordinatesToCellName(idx+1, 1)
		if err := file.SetCellValue(sheet, cell, header); err != nil {
			return nil, err
		}
	}

	for rowIndex, row := range rows {
		for colIndex, value := range reportRecord(row) {
			cell, _ := excelize.CoordinatesToCellName(colIndex+1, rowIndex+2)
			if err := file.SetCellValue(sheet, cell, value); err != nil {
				return nil, err
			}
		}
	}

	buffer, err := file.WriteToBuffer()
	if err != nil {
		return nil, err
	}

	return buffer.Bytes(), nil
}

func reportHeaders() []string {
	return []string{
		"Дата",
		"ID заказа",
		"Статус",
		"ID товара",
		"Товар",
		"Категория",
		"Количество",
		"Цена за единицу",
		"Сумма",
		"Себестоимость",
		"Валовая прибыль",
		"Маржа, %",
	}
}

func reportRecord(row domain.SalesReportRow) []string {
	return []string{
		row.Day,
		strconv.FormatInt(row.OrderID, 10),
		row.Status,
		strconv.FormatInt(row.ProductID, 10),
		row.ProductName,
		row.CategoryName,
		strconv.FormatInt(row.Quantity, 10),
		row.UnitPrice,
		row.TotalPrice,
		row.CostPrice,
		row.GrossProfit,
		strconv.FormatFloat(row.MarginPercent, 'f', 2, 64),
	}
}
