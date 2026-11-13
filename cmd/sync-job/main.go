package main

import (
	"context"
	"cron/internal/adapters/client"
	"cron/internal/adapters/database"
	"cron/internal/application"
	"database/sql"
	"fmt"
	"log"
	"os"
	_ "time/tzdata"

	_ "github.com/go-sql-driver/mysql"
	"github.com/joho/godotenv"
)

func main() {

	err := godotenv.Load()
	ctx := context.Background()
	if err != nil {
		log.Printf("Error loading .env file: %v", err)
	}

	jobSelection := os.Getenv("JOB_SELECTION")
	if jobSelection != "DATA_BATCH_SYNC" {
		log.Printf("No job selected or invalid job selection: %s", jobSelection)
		return
	}

	fmt.Print("[main] Starting sync job with Aegis-Go SDK")

	user := os.Getenv("USER")
	password := os.Getenv("PASSWORD")
	url := os.Getenv("URL")
	dbTimeZone := os.Getenv("TIMEZONE")
	if dbTimeZone == "" {
		dbTimeZone = "America%2FArgentina%2FBuenos_Aires"
	}

	client := client.NewClient(url)
	dsn := os.Getenv("DB_USER") + ":" + os.Getenv("DB_PASS") + "@tcp(" + os.Getenv("DB_HOST") + ":" + os.Getenv("DB_PORT") + ")/" + os.Getenv("DB_NAME") + "?parseTime=true&loc=" + dbTimeZone
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		fmt.Printf("Failed to connect to database: %v", err)
	}

	repositoryAdapter := database.NewMySQLRepository(db)
	service := application.NewSyncService(client, repositoryAdapter)

	storeInfos, err := repositoryAdapter.GetStoresIds(ctx)
	if err != nil {
		fmt.Printf("Failed to get store IDs: %v", err)
		return
	}

	service.RunMultiStoreSync(ctx, storeInfos, user, password)
	fmt.Printf("[main] Sync job completed successfully")
}
