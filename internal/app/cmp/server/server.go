package server

import (
	"context"
	"net"
	"time"

	vendorpb "github.com/martketplace-vkr/analytics/pkg/api/grpc/v1/vendor"
	grpcServer "github.com/martketplace-vkr/pkg/server/grpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

const cmpName = "GRPC server"

type Server struct {
	cfg        grpcServer.Config
	grpcServer *grpc.Server
	vendor     vendorpb.AnalyticsVendorServiceServer
}

func New(cfg grpcServer.Config, vendor vendorpb.AnalyticsVendorServiceServer) *Server {
	return &Server{
		cfg:    cfg,
		vendor: vendor,
	}
}

func (s *Server) Start(ctx context.Context) error {
	server, err := grpcServer.New(ctx, s.cfg, nil)
	if err != nil {
		return err
	}

	s.grpcServer = server.Grpc
	reflection.Register(s.grpcServer)
	vendorpb.RegisterAnalyticsVendorServiceServer(s.grpcServer, s.vendor)

	listener, err := net.Listen("tcp", s.cfg.Host)
	if err != nil {
		return err
	}

	errCh := make(chan error, 1)
	go func() {
		if err := s.grpcServer.Serve(listener); err != nil {
			errCh <- err
		}
	}()

	select {
	case err := <-errCh:
		return err
	case <-time.After(s.cfg.StartTimeout.Duration):
		return nil
	}
}

func (s *Server) Stop(_ context.Context) error {
	stopCh := make(chan struct{}, 1)
	go func() {
		s.grpcServer.GracefulStop()
		stopCh <- struct{}{}
	}()

	select {
	case <-time.After(s.cfg.StopTimeout.Duration):
		return nil
	case <-stopCh:
		return nil
	}
}

func (s *Server) GetName() string {
	return cmpName
}

func (s *Server) GetShutdownDelay() time.Duration {
	return time.Second
}

func (s *Server) GetStartTimeout() time.Duration {
	return 5 * time.Second
}

func (s *Server) GetStopTimeout() time.Duration {
	return 5 * time.Second
}
