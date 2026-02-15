package application

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/psds-microservice/helpy/db"
	"github.com/psds-microservice/operator-directory-service/internal/client"
	"github.com/psds-microservice/operator-directory-service/internal/config"
	"github.com/psds-microservice/operator-directory-service/internal/database"
	"github.com/psds-microservice/operator-directory-service/internal/handler"
	"github.com/psds-microservice/operator-directory-service/internal/router"
	"github.com/psds-microservice/operator-directory-service/internal/service"
)

type API struct {
	cfg *config.Config
	srv *http.Server
}

func NewAPI(cfg *config.Config) (*API, error) {
	if err := database.MigrateUp(cfg.DatabaseURL()); err != nil {
		return nil, fmt.Errorf("migrate: %w", err)
	}
	conn, err := db.Open(cfg.DSN())
	if err != nil {
		return nil, fmt.Errorf("db: %w", err)
	}
	poolClient := client.NewPoolClient(cfg.OperatorPoolURL, &client.PoolClientOpts{
		Timeout:      cfg.PoolTimeout,
		MaxRetries:   cfg.PoolMaxRetries,
		RetryBackoff: time.Duration(cfg.PoolRetryBackoffMs) * time.Millisecond,
	})
	directorySvc := service.NewDirectoryService(conn, poolClient)
	directoryHandler := handler.NewDirectoryHandler(directorySvc)
	r := router.New(directoryHandler)
	addr := cfg.AppHost + ":" + cfg.HTTPPort
	srv := &http.Server{
		Addr:              addr,
		Handler:           r,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
	return &API{cfg: cfg, srv: srv}, nil
}

func (a *API) Run(ctx context.Context) error {
	host := a.cfg.AppHost
	if host == "0.0.0.0" {
		host = "localhost"
	}
	base := "http://" + host + ":" + a.cfg.HTTPPort
	log.Printf("operator-directory-service HTTP listening on %s", a.srv.Addr)
	log.Printf("  Health: %s/health  Ready: %s/ready  Swagger: %s/swagger/index.html  API: %s/api/v1/operators", base, base, base, base)
	go func() {
		if err := a.srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("http: %v", err)
		}
	}()
	<-ctx.Done()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := a.srv.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("http shutdown: %w", err)
	}
	return nil
}
