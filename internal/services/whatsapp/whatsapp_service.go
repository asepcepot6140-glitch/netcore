package whatsapp

import (
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

func SendMessage(phone, message string) error {
	endpoint := os.Getenv("WHATSAPP_GATEWAY_URL")
	username := os.Getenv("WHATSAPP_GATEWAY_USERNAME")
	password := os.Getenv("WHATSAPP_GATEWAY_PASSWORD")

	if endpoint == "" || username == "" || password == "" {
		return fmt.Errorf("konfigurasi WhatsApp gateway belum lengkap")
	}

	form := url.Values{}
	form.Set("phone", phone)
	form.Set("message", message)

	req, err := http.NewRequest(
		http.MethodPost,
		endpoint,
		strings.NewReader(form.Encode()),
	)
	if err != nil {
		return err
	}

	req.SetBasicAuth(username, password)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{
		Timeout: 15 * time.Second,
	}

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("WhatsApp gateway mengembalikan HTTP %d", resp.StatusCode)
	}

	return nil
}
