package main

import (
	"context"
	"fmt"
	"log"

	//	"github.com/gin-gonic/gin"
	//	"gorm.io/gorm"

	"backend/internal/app"
	"backend/internal/redisdb"
	"backend/internal/routes"
)

func main() {
	// 1. Connect to SQLite with GORM
	db, err := app.InitializeDBAll()
	if err != nil {
		fmt.Println(err)
		return
	}
	//redis init

	redisCtx := context.Background()
	rds, err := redisdb.InitRedis(redisCtx)
	redisdb.ClearRedis(rds)
	if err != nil {
		fmt.Println(err)
		return
	}

	r := routes.SetupRouter(db, rds)

	// 6. Start the server
	log.Println("?? Server running on http://localhost:8080")
	if err := r.Run(":8080"); err != nil {
		log.Fatal("? failed to start server:", err)
	}
}
