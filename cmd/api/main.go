package main

import (
	"log"
	"net"
	"os"

	"github.com/asepcepot6140-glitch/netcore/internal/config"
	"github.com/asepcepot6140-glitch/netcore/internal/models"
	"github.com/asepcepot6140-glitch/netcore/internal/routes"
	"github.com/asepcepot6140-glitch/netcore/internal/utils"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func main() {

	if err := godotenv.Load(); err != nil {
		log.Println("File .env tidak ditemukan")
	}

	if err := utils.ValidateJWTSecret(); err != nil {
		log.Fatal("Konfigurasi JWT tidak valid:", err)
	}

	if err := config.ConnectDatabase(); err != nil {
		log.Fatal("Koneksi database gagal:", err)
	}

	err := config.DB.AutoMigrate(
		&models.User{},
		&models.PasswordResetOTP{},
		&models.RefreshToken{},
		&models.Mikrotik{},
		&models.GenieACS{},
	)

	if err != nil {
		log.Fatal("Migration gagal:", err)
	}

	r := gin.Default()

	routes.SetupRoutes(r)

	port := os.Getenv("APP_PORT")

	if port == "" {
		port = "8080"
	}

	host := os.Getenv("APP_HOST")
	if host == "" {
		host = "127.0.0.1"
	}
	address := net.JoinHostPort(host, port)

	log.Println("Server running on " + address)

	if err := r.Run(address); err != nil {
		log.Fatal(err)
	}
}
