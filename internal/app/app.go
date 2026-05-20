package app

import (
	"context"

	"github.com/martketplace-vkr/analytics/config"
	"github.com/martketplace-vkr/analytics/internal/app/cmp/refresher"
	"github.com/martketplace-vkr/analytics/internal/app/cmp/server"
	repository "github.com/martketplace-vkr/analytics/internal/repository/pg"
	analyticsService "github.com/martketplace-vkr/analytics/internal/service"
	adminTransport "github.com/martketplace-vkr/analytics/internal/transport/grpc/v1/admin"
	vendorTransport "github.com/martketplace-vkr/analytics/internal/transport/grpc/v1/vendor"
	"github.com/martketplace-vkr/pkg/build"
	"github.com/martketplace-vkr/pkg/build/components/pgxsqlxcomponent"
)

func Run(ctx context.Context, cfg *config.Config) error {
	analyticsPG := pgxsqlxcomponent.New(cfg.Postgres)
	orderPG := pgxsqlxcomponent.New(cfg.OrderPostgres)
	catalogPG := pgxsqlxcomponent.New(cfg.CatalogPostgres)

	repo := repository.New(analyticsPG.DB, orderPG.DB, catalogPG.DB)
	service := analyticsService.New(repo)
	adminHandler := adminTransport.New(service)
	vendorHandler := vendorTransport.New(service)

	grpcServer := server.New(cfg.Grpc, adminHandler, vendorHandler)
	refreshCmp := refresher.New(cfg.Refresh.Interval.Duration, service)

	cmps := build.Components{
		analyticsPG,
		orderPG,
		catalogPG,
		refreshCmp,
		grpcServer,
	}

	app, err := build.NewApp(cmps)
	if err != nil {
		return err
	}

	return build.Run(ctx, app)
}
