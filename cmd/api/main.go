package main

import (
	"log"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/asepcepot6140-glitch/netcore/internal/config"
	"github.com/asepcepot6140-glitch/netcore/internal/middleware"
	"github.com/asepcepot6140-glitch/netcore/internal/models"
	"github.com/asepcepot6140-glitch/netcore/internal/routes"
	"github.com/asepcepot6140-glitch/netcore/internal/utils"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func main() {

	if err := godotenv.Load(); err != nil {
		log.Println(".env file not found")
	}

	if err := utils.ValidateJWTSecret(); err != nil {
		log.Fatal("Invalid JWT configuration:", err)
	}

	if err := config.ConnectDatabase(); err != nil {
		log.Fatal("Database connection failed:", err)
	}

	err := config.DB.AutoMigrate(
		&models.User{},
		&models.PasswordResetOTP{},
		&models.RefreshToken{},
		&models.Mikrotik{},
		&models.GenieACS{},
	)

	if err != nil {
		log.Fatal("Migration failed:", err)
	}

	// Release mode kecuali GIN_MODE=debug dipilih eksplisit.
	if os.Getenv("GIN_MODE") != "debug" {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.New()
	r.Use(gin.Logger(), gin.Recovery(), middleware.SecurityHeaders())

	// Tanpa TRUSTED_PROXIES, ClientIP memakai alamat koneksi langsung (header XFF diabaikan).
	var proxies []string
	for _, p := range strings.Split(os.Getenv("TRUSTED_PROXIES"), ",") {
		if p = strings.TrimSpace(p); p != "" {
			proxies = append(proxies, p)
		}
	}
	if err := r.SetTrustedProxies(proxies); err != nil {
		log.Fatal("Invalid TRUSTED_PROXIES:", err)
	}

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

	srv := &http.Server{
		Addr:              address,
		Handler:           r,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      60 * time.Second,
		IdleTimeout:       60 * time.Second,
		MaxHeaderBytes:    1 << 20,
	}

	if err := srv.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}
