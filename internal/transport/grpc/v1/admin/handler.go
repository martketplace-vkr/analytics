package admin

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/martketplace-vkr/analytics/domain"
	analyticsservice "github.com/martketplace-vkr/analytics/internal/service"
	adminpb "github.com/martketplace-vkr/analytics/pkg/api/grpc/v1/admin"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type Handler struct {
	service analyticsService
	adminpb.UnimplementedAnalyticsAdminServiceServer
}

func New(service analyticsService) *Handler {
	return &Handler{service: service}
}

func (h *Handler) ListTariffs(ctx context.Context, _ *adminpb.ListTariffsRequest) (*adminpb.ListTariffsResponse, error) {
	tariffs, err := h.service.ListTariffs(ctx)
	if err != nil {
		return nil, mapError(err)
	}

	result := make([]*adminpb.Tariff, 0, len(tariffs))
	for _, tariff := range tariffs {
		result = append(result, tariffToProto(tariff))
	}
	return &adminpb.ListTariffsResponse{Tariffs: result}, nil
}

func (h *Handler) CreateTariff(ctx context.Context, req *adminpb.CreateTariffRequest) (*adminpb.TariffResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request is required")
	}
	tariff, err := h.service.CreateTariff(ctx, domain.Tariff{
		Name:              req.GetName(),
		CommissionPercent: req.GetCommissionPercent(),
	})
	if err != nil {
		return nil, mapError(err)
	}
	return &adminpb.TariffResponse{Tariff: tariffToProto(tariff)}, nil
}

func (h *Handler) UpdateTariff(ctx context.Context, req *adminpb.UpdateTariffRequest) (*adminpb.TariffResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request is required")
	}
	tariff, err := h.service.UpdateTariff(ctx, domain.Tariff{
		ID:                req.GetTariffId(),
		Name:              req.GetName(),
		CommissionPercent: req.GetCommissionPercent(),
	})
	if err != nil {
		return nil, mapError(err)
	}
	return &adminpb.TariffResponse{Tariff: tariffToProto(tariff)}, nil
}

func (h *Handler) SetDefaultTariff(ctx context.Context, req *adminpb.SetDefaultTariffRequest) (*adminpb.TariffResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request is required")
	}
	tariff, err := h.service.SetDefaultTariff(ctx, req.GetTariffId())
	if err != nil {
		return nil, mapError(err)
	}
	return &adminpb.TariffResponse{Tariff: tariffToProto(tariff)}, nil
}

func (h *Handler) AssignVendorTariff(ctx context.Context, req *adminpb.AssignVendorTariffRequest) (*adminpb.VendorTariffResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request is required")
	}
	assignment, err := h.service.AssignVendorTariff(ctx, req.GetVendorId(), req.GetTariffId())
	if err != nil {
		return nil, mapError(err)
	}
	return &adminpb.VendorTariffResponse{
		VendorId: assignment.VendorID,
		Tariff:   tariffToProto(assignment.Tariff),
	}, nil
}

func (h *Handler) GetVendorTariff(ctx context.Context, req *adminpb.GetVendorTariffRequest) (*adminpb.VendorTariffResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "request is required")
	}
	tariff, err := h.service.GetVendorTariff(ctx, req.GetVendorId())
	if err != nil {
		return nil, mapError(err)
	}
	return &adminpb.VendorTariffResponse{
		VendorId: req.GetVendorId(),
		Tariff:   tariffToProto(tariff),
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
	case errors.Is(err, analyticsservice.ErrNotFound), errors.Is(err, sql.ErrNoRows):
		return status.Error(codes.NotFound, "resource not found")
	default:
		return status.Error(codes.Internal, "internal server error")
	}
}

func tariffToProto(tariff domain.Tariff) *adminpb.Tariff {
	if tariff.ID == 0 {
		return nil
	}
	return &adminpb.Tariff{
		Id:                tariff.ID,
		Name:              tariff.Name,
		CommissionPercent: tariff.CommissionPercent,
		IsDefault:         tariff.IsDefault,
		AssignedVendors:   tariff.AssignedVendors,
		CreatedAt:         formatTime(tariff.CreatedAt),
		UpdatedAt:         formatTime(tariff.UpdatedAt),
	}
}

func formatTime(value time.Time) string {
	if value.IsZero() {
		return ""
	}
	return value.UTC().Format(time.RFC3339)
}
