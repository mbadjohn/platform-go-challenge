package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/flowchartsman/swaggerui"
	amem "github.com/gwi/platform-go-challenge/assets/adapters/memory"
	"github.com/gwi/platform-go-challenge/config"
	"github.com/gwi/platform-go-challenge/favourites/adapters/assetcatalog"
	fmem "github.com/gwi/platform-go-challenge/favourites/adapters/memory"
	"github.com/gwi/platform-go-challenge/favourites/application"
	httppkg "github.com/gwi/platform-go-challenge/http"
)

func main() {
	env := os.Getenv("APP_ENV")
	if env == "" {
		env = "dev"
	}
	configPath := os.Getenv("CONFIG_PATH")
	if configPath == "" {
		configPath = filepath.Join("config", env+".yaml")
	}
	cfg, err := config.Load(configPath)
	if err != nil {
		log.Fatalf("config: %v", err)
	}
	log.Printf("loaded config from %s (APP_ENV=%s)", configPath, env)

	signer, verifier, err := httppkg.LoadJWTAuth()
	if err != nil {
		log.Fatalf("JWT auth: %v", err)
	}
	internalEventSecret := os.Getenv("INTERNAL_EVENT_SECRET")

	repo := fmem.NewRepository()
	catalog := amem.NewInMemoryCatalog()
	assetCatalogClient := assetcatalog.NewClient(catalog)
	svc := application.NewFavouritesService(repo, assetCatalogClient)
	openAPIServer := &httppkg.OpenAPIServer{
		UC:           svc,
		EventsUC:     svc,
		EventsSecret: internalEventSecret,
		Signer:       signer,
		Verifier:     verifier,
		TokenExpiry:  cfg.TokenExpiry(),
	}

	specPath := os.Getenv("OPENAPI_SPEC")
	if specPath == "" {
		specPath = "swagger.yaml"
	}
	spec, err := os.ReadFile(specPath)
	if err != nil {
		if os.IsNotExist(err) {
			alt := filepath.Join(filepath.Dir(configPath), "..", "swagger.yaml")
			spec, err = os.ReadFile(alt)
		}
		if err != nil {
			log.Printf("warning: could not load OpenAPI spec from %s: %v (Swagger UI disabled)", specPath, err)
			spec = nil
		}
	}

	mux := http.NewServeMux()
	if len(spec) > 0 {
		mux.Handle("/swagger/", http.StripPrefix("/swagger", swaggerui.Handler(spec)))
		mux.HandleFunc("GET /swagger", func(w http.ResponseWriter, r *http.Request) {
			http.Redirect(w, r, "/swagger/", http.StatusMovedPermanently)
		})
	}
	handler := httppkg.HandlerFromMuxWithBaseURL(openAPIServer, mux, "/v1")

	srv := &http.Server{
		Addr:    cfg.Addr(),
		Handler: handler,
	}

	go func() {
		log.Printf("server listening on %s", cfg.Addr())
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit

	log.Print("shutting down server...")
	ctx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout())
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("server shutdown: %v", err)
	}
	log.Print("server stopped")
}