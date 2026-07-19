package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/alexedwards/scs/goredisstore"
	"github.com/alexedwards/scs/v2"
	"github.com/redis/go-redis/v9"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"shreelance/internal/config"
	"shreelance/internal/models"
	"shreelance/internal/web"
	"shreelance/internal/worker"
)

func main() {
	cfg := config.LoadConfig()

	// 1. Connect to PostgreSQL
	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=%s",
		cfg.DBHost, cfg.DBUser, cfg.DBPassword, cfg.DBName, cfg.DBPort, cfg.DBSSLMode)
	
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	log.Println("Successfully connected to PostgreSQL")

	// Run AutoMigrations
	err = db.AutoMigrate(&models.User{}, &models.Order{}, &models.Bid{}, &models.ProcessedDonation{})
	if err != nil {
		log.Fatalf("Database migration failed: %v", err)
	}
	log.Println("Database migration completed")

	// 2. Connect to Valkey (Redis-compatible)
	valkeyClient := redis.NewClient(&redis.Options{
		Addr:     cfg.ValkeyAddr,
		Password: cfg.ValkeyPassword,
		DB:       0,
	})

	// Quick connection test
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := valkeyClient.Ping(ctx).Err(); err != nil {
		log.Printf("Warning: Valkey ping failed (is it running?): %v", err)
	} else {
		log.Println("Successfully connected to Valkey")
	}

	// 3. Initialize SCS Session Manager
	sessionManager := scs.New()
	sessionManager.Lifetime = 24 * time.Hour
	sessionManager.Store = goredisstore.New(valkeyClient)

	// 4. Setup Router
	router := web.NewRouter(cfg, db, sessionManager)

	// Start DonationAlerts Polling Worker
	worker.StartDonationWorker(db, valkeyClient, cfg)

	// 5. Start Server
	srvAddr := ":" + cfg.Port
	log.Printf("Starting server on %s", srvAddr)
	if err := http.ListenAndServe(srvAddr, router); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}
