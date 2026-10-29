package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	//	"github.com/gin-gonic/gin"
	//	"gorm.io/gorm"

	"backend/internal/app"
	"backend/internal/redisdb"
	"backend/internal/routes"
	"backend/utils"
)

func main() {
	if err := utils.ValidateJWTConfig(); err != nil {
		log.Fatal(err)
	}
	// 1. Connect to SQLite with GORM
	db, err := app.InitializeDBAll()
	if err != nil {
		fmt.Println(err)
		return
	}
	sqlDB, err := db.DB()
	if err != nil {
		log.Println(err)
		return
	}
	defer sqlDB.Close()
	//redis init

	redisCtx := context.Background()
	rds, err := redisdb.InitRedis(redisCtx)
	if err != nil {
		fmt.Println(err)
		return
	}

	defer rds.Close()
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()
	stopMessages := routes.StartMessageRuntime(ctx, db)
	defer stopMessages()
	server := &http.Server{Addr: ":8080", Handler: routes.SetupRouter(db, rds), ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 10 * time.Second, WriteTimeout: 10 * time.Second, IdleTimeout: 60 * time.Second}
	shutdownDone := make(chan struct{})
	go func() {
		defer close(shutdownDone)
		<-ctx.Done()
		shutdown, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := server.Shutdown(shutdown); err != nil {
			_ = server.Close()
		}
	}()
	log.Println("Server running on http://localhost:8080")
	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Println("HTTP server failed:", err)
	}
	cancel()
	<-shutdownDone
}
