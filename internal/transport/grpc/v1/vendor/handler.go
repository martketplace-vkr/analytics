package vendor

import (
	"context"
	"errors"

	"github.com/martketplace-vkr/analytics/domain"
	analyticsservice "github.com/martketplace-vkr/analytics/internal/service"
	vendorpb "github.com/martketplace-vkr/analytics/pkg/api/grpc/v1/vendor"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type Handler struct {
	service analyticsService
	vendorpb.UnimplementedAnalyticsVendorServiceServer
}

func New(service analyticsService) *Handler {
	return &Handler{service: service}
}

func (h *Handler) GetOverview(ctx context.Context, req *vendorpb.GetOverviewRequest) (*vendorpb.GetOverviewResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request is required")
	}

	overview, err := h.service.GetOverview(ctx, req.GetVendorId(), req.GetFrom(), req.GetTo())
	if err != nil {
		return nil, mapError(err)
	}

	return &vendorpb.GetOverviewResponse{
		Kpi:           kpiToProto(overview.KPI),
		Trend:         trendToProto(overview.Trend),
		Tariff:        tariffToProto(overview.Tariff),
		ProductTrends: productTrendsToProto(overview.ProductTrends),
	}, nil
}

func (h *Handler) GetNiches(ctx context.Context, req *vendorpb.GetNichesRequest) (*vendorpb.GetNichesResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request is required")
	}

	niches, err := h.service.GetNiches(ctx, req.GetVendorId(), req.GetFrom(), req.GetTo(), req.GetSort(), int(req.GetLimit()))
	if err != nil {
		return nil, mapError(err)
	}

	return &vendorpb.GetNichesResponse{Niches: nichesToProto(niches)}, nil
}

func (h *Handler) GetProducts(ctx context.Context, req *vendorpb.GetProductsRequest) (*vendorpb.GetProductsResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request is required")
	}

	products, err := h.service.GetProducts(ctx, req.GetVendorId(), req.GetFrom(), req.GetTo())
	if err != nil {
		return nil, mapError(err)
	}

	return &vendorpb.GetProductsResponse{Products: productsToProto(products)}, nil
}

func (h *Handler) RecordProductView(ctx context.Context, req *vendorpb.RecordProductViewRequest) (*vendorpb.RecordProductViewResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request is required")
	}

	recorded, err := h.service.RecordProductView(ctx, domain.ProductView{
		ProductID: req.GetProductId(),
		VisitorID: req.GetVisitorId(),
	})
	if err != nil {
		return nil, mapError(err)
	}

	return &vendorpb.RecordProductViewResponse{Recorded: recorded}, nil
}

func (h *Handler) UpsertProductCost(ctx context.Context, req *vendorpb.UpsertProductCostRequest) (*vendorpb.UpsertProductCostResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request is required")
	}

	product, err := h.service.UpsertProductCost(ctx, domain.ProductCost{
		VendorID:  req.GetVendorId(),
		ProductID: req.GetProductId(),
		CostPrice: req.GetCostPrice(),
		Currency:  "RUB",
	})
	if err != nil {
		return nil, mapError(err)
	}

	return &vendorpb.UpsertProductCostResponse{Product: productToProto(product)}, nil
}

func (h *Handler) ExportSalesReport(ctx context.Context, req *vendorpb.ExportSalesReportRequest) (*vendorpb.ExportSalesReportResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request is required")
	}

	report, err := h.service.ExportSalesReport(ctx, req.GetVendorId(), req.GetFrom(), req.GetTo(), req.GetFormat())
	if err != nil {
		return nil, mapError(err)
	}

	return &vendorpb.ExportSalesReportResponse{
		Content:     report.Content,
		Filename:    report.Filename,
		ContentType: report.ContentType,
	}, nil
}

func mapError(err error) error {
	switch {
	case errors.Is(err, context.Canceled):
		return status.Error(codes.Canceled, "request was canceled")
	case errors.Is(err, context.DeadlineExceeded):
		return status.Error(codes.DeadlineExceeded, "deadline exceeded")
	case errors.Is(err, analyticsservice.ErrInvalidArgument):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, analyticsservice.ErrNotFound):
		return status.Error(codes.NotFound, "resource not found")
	case errors.Is(err, analyticsservice.ErrReportTooLarge):
		return status.Error(codes.InvalidArgument, "report is too large, narrow the period")
	default:
		return status.Error(codes.Internal, "internal server error")
	}
}

func kpiToProto(kpi domain.KPI) *vendorpb.AnalyticsKpi {
	return &vendorpb.AnalyticsKpi{
		DemandUnits:         kpi.DemandUnits,
		SoldUnits:           kpi.SoldUnits,
		OrdersCount:         kpi.OrdersCount,
		SalesCount:          kpi.SalesCount,
		ProductViews:        kpi.ProductViews,
		Revenue:             kpi.Revenue,
		GrossProfit:         kpi.GrossProfit,
		MarginPercent:       kpi.MarginPercent,
		CostCoveragePercent: kpi.CostCoveragePercent,
		MarketplaceFee:      kpi.MarketplaceFee,
		NetProfit:           kpi.NetProfit,
		NetMarginPercent:    kpi.NetMarginPercent,
	}
}

func trendToProto(trend []domain.DailyTrendPoint) []*vendorpb.DailyTrendPoint {
	result := make([]*vendorpb.DailyTrendPoint, 0, len(trend))
	for _, point := range trend {
		result = append(result, &vendorpb.DailyTrendPoint{
			Day:            point.Day,
			DemandUnits:    point.DemandUnits,
			SoldUnits:      point.SoldUnits,
			ProductViews:   point.ProductViews,
			Revenue:        point.Revenue,
			GrossProfit:    point.GrossProfit,
			MarketplaceFee: point.MarketplaceFee,
			NetProfit:      point.NetProfit,
		})
	}
	return result
}

func productTrendsToProto(trends []domain.ProductDailyTrend) []*vendorpb.ProductDailyTrend {
	result := make([]*vendorpb.ProductDailyTrend, 0, len(trends))
	for _, trend := range trends {
		points := make([]*vendorpb.ProductDailyTrendPoint, 0, len(trend.Points))
		for _, point := range trend.Points {
			points = append(points, &vendorpb.ProductDailyTrendPoint{
				Day:        point.Day,
				SoldUnits:  point.SoldUnits,
				ViewsCount: point.ViewsCount,
			})
		}
		result = append(result, &vendorpb.ProductDailyTrend{
			ProductId:   trend.ProductID,
			ProductName: trend.ProductName,
			Points:      points,
		})
	}
	return result
}

func nichesToProto(niches []domain.NicheMetric) []*vendorpb.NicheMetric {
	result := make([]*vendorpb.NicheMetric, 0, len(niches))
	for _, niche := range niches {
		result = append(result, &vendorpb.NicheMetric{
			CategoryId:             niche.CategoryID,
			CategoryName:           niche.CategoryName,
			MarketDemandUnits:      niche.MarketDemandUnits,
			MarketSoldUnits:        niche.MarketSoldUnits,
			MarketRevenue:          niche.MarketRevenue,
			ActiveProducts:         niche.ActiveProducts,
			StockCount:             niche.StockCount,
			VendorsCount:           niche.VendorsCount,
			OpportunityScore:       niche.OpportunityScore,
			VendorDemandUnits:      niche.VendorDemandUnits,
			VendorSoldUnits:        niche.VendorSoldUnits,
			VendorRevenue:          niche.VendorRevenue,
			VendorGrossProfit:      niche.VendorGrossProfit,
			VendorMarginPercent:    niche.VendorMarginPercent,
			VendorMarketplaceFee:   niche.VendorMarketplaceFee,
			VendorNetProfit:        niche.VendorNetProfit,
			VendorNetMarginPercent: niche.VendorNetMarginPercent,
		})
	}
	return result
}

func productsToProto(products []domain.ProductMetric) []*vendorpb.ProductMetric {
	result := make([]*vendorpb.ProductMetric, 0, len(products))
	for _, product := range products {
		result = append(result, productToProto(product))
	}
	return result
}

func productToProto(product domain.ProductMetric) *vendorpb.ProductMetric {
	return &vendorpb.ProductMetric{
		ProductId:        product.ProductID,
		ProductName:      product.ProductName,
		CategoryId:       product.CategoryID,
		CategoryName:     product.CategoryName,
		StockCount:       product.StockCount,
		Price:            product.Price,
		CostPrice:        product.CostPrice,
		HasCost:          product.HasCost,
		DemandUnits:      product.DemandUnits,
		SoldUnits:        product.SoldUnits,
		ViewsCount:       product.ViewsCount,
		Revenue:          product.Revenue,
		GrossProfit:      product.GrossProfit,
		MarginPercent:    product.MarginPercent,
		MarketplaceFee:   product.MarketplaceFee,
		NetProfit:        product.NetProfit,
		NetMarginPercent: product.NetMarginPercent,
	}
}

func tariffToProto(tariff domain.Tariff) *vendorpb.Tariff {
	if tariff.ID == 0 {
		return nil
	}
	return &vendorpb.Tariff{
		Id:                tariff.ID,
		Name:              tariff.Name,
		CommissionPercent: tariff.CommissionPercent,
		IsDefault:         tariff.IsDefault,
		AssignedVendors:   tariff.AssignedVendors,
		CreatedAt:         tariff.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:         tariff.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
}
