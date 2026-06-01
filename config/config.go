package config

import (
	"github.com/martketplace-vkr/pkg/build/components/pgxsqlxcomponent"
	"github.com/martketplace-vkr/pkg/server/grpc"
	"github.com/martketplace-vkr/pkg/utils/duration"
)

type Config struct {
	Grpc            grpc.Config             `validate:"required"`
	Postgres        pgxsqlxcomponent.Config `validate:"required"`
	OrderPostgres   pgxsqlxcomponent.Config `validate:"required"`
	CatalogPostgres pgxsqlxcomponent.Config `validate:"required"`
	AuthPostgres    pgxsqlxcomponent.Config `validate:"required"`
	Refresh         RefreshConfig           `validate:"required"`
}

type RefreshConfig struct {
	Interval duration.Seconds `validate:"required" default:"900"`
}
