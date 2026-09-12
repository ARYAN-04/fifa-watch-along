package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"

	"github.com/fifa-watch-along/fifa-hub/internal/api"
	"github.com/fifa-watch-along/fifa-hub/internal/config"
	"github.com/fifa-watch-along/fifa-hub/internal/inference"
	"github.com/fifa-watch-along/fifa-hub/internal/poller"
	"github.com/fifa-watch-along/fifa-hub/internal/source"
	"github.com/fifa-watch-along/fifa-hub/internal/store"
	"github.com/fifa-watch-along/fifa-hub/internal/web"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	st, err := store.Open(cfg.DBPath)
	if err != nil {
		log.Fatalf("store: %v", err)
	}
	defer st.Close()

	inf, err := inference.New("ml/export/model.json")
	if err != nil {
		log.Fatalf("inference: %v", err)
	}

	mux := http.NewServeMux()
	api.Register(mux, api.Deps{
		Store:   st,
		Predict: inf.Predict,
		Mocks:   cfg.DevMocks,
	})
	if err := web.Mount(mux); err != nil {
		log.Fatalf("web: %v", err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	var src source.DataSource
	if !cfg.DevMocks && cfg.FootballDataAPIKey != "" {
		src = source.NewFootballData(cfg.FootballDataAPIKey)
	}

	go poller.Run(ctx, poller.Deps{
		Store:   st,
		Source:  src,
		Predict: inf.Predict,
		Every:   cfg.PollInterval,
	})

	log.Printf("fifa-hub listening on :%s (db=%s mocks=%v)", cfg.Port, cfg.DBPath, cfg.DevMocks)
	if err := http.ListenAndServe(":"+cfg.Port, mux); err != nil && ctx.Err() == nil {
		log.Fatalf("listen: %v", err)
	}
}
