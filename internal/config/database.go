package config

import (
	"fmt"
	"net"
	"net/url"
	"os"
	"strings"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var DB *gorm.DB

func DatabaseDSN() (string, error) {
	host := strings.TrimSpace(os.Getenv("DB_HOST"))
	user := os.Getenv("DB_USER")
	name := os.Getenv("DB_NAME")
	if host == "" || user == "" || name == "" {
		return "", fmt.Errorf("DB_HOST, DB_USER, and DB_NAME must be configured")
	}

	port := strings.TrimSpace(os.Getenv("DB_PORT"))
	if port == "" {
		port = "5432"
	}

	query := url.Values{
		"sslmode":  {"disable"},
		"TimeZone": {"Asia/Jakarta"},
	}
	if rootCert := strings.TrimSpace(os.Getenv("DB_SSLROOTCERT")); rootCert != "" {
		query.Set("sslrootcert", rootCert)
	}

	dsn := url.URL{
		Scheme:   "postgres",
		User:     url.UserPassword(user, os.Getenv("DB_PASSWORD")),
		Host:     net.JoinHostPort(host, port),
		Path:     "/" + name,
		RawQuery: strings.ReplaceAll(query.Encode(), "TimeZone=Asia%2FJakarta", "TimeZone=Asia/Jakarta"),
	}
	return dsn.String(), nil
}

func ConnectDatabase() error {
	dsn, err := DatabaseDSN()
	if err != nil {
		return err
	}

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}

	DB = db
	return nil
}
