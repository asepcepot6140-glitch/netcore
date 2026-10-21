package genieacs

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/asepcepot6140-glitch/netcore/internal/models"
)

type Device struct {
	ID            string     `json:"id"`
	SerialNumber  string     `json:"serial_number"`
	Manufacturer  string     `json:"manufacturer"`
	ProductClass  string     `json:"product_class"`
	LastInform    *time.Time `json:"last_inform"`
	Status        string     `json:"status"`
	PPPoEUsername string     `json:"pppoe_username"`
	RXPower       string     `json:"rx_power"`
	SSID          string     `json:"ssid"`
}

const (
	ssidPath     = "InternetGatewayDevice.LANDevice.1.WLANConfiguration.1.SSID"
	onlineWindow = 5 * time.Minute
)

var httpClient = &http.Client{
	Timeout: 15 * time.Second,
}

var rxPowerPaths = []string{
	"InternetGatewayDevice.WANDevice.*.X_ZTE-COM_WANPONInterfaceConfig.RXPower",
	"InternetGatewayDevice.WANDevice.*.X_GponInterafceConfig.RXPower",
	"InternetGatewayDevice.WANDevice.*.X_FH_GponInterfaceConfig.RXPower",
	"InternetGatewayDevice.X_ALU_OntOpticalParam.RXPower",
	"InternetGatewayDevice.WANDevice.*.X_CT-COM_GponInterfaceConfig.RXPower",
	"InternetGatewayDevice.WANDevice.*.X_CT-COM_EponInterfaceConfig.RXPower",
	"InternetGatewayDevice.WANDevice.1.X_CMCC_GponInterfaceConfig.RXPower",
	"InternetGatewayDevice.WANDevice.1.X_CMCC_EponInterfaceConfig.RXPower",
	"InternetGatewayDevice.WANDevice.1.X_CU_WANEPONInterfaceConfig.OpticalTransceiver.RXPower",
	"InternetGatewayDevice.WANDevice.1.WANEponInterfaceConfig.RXPower",
}

func pppoeUsernamePaths() []string {
	paths := make([]string, 0, 16)

	for conn := 1; conn <= 4; conn++ {
		for instance := 1; instance <= 4; instance++ {
			paths = append(paths, fmt.Sprintf(
				"InternetGatewayDevice.WANDevice.1.WANConnectionDevice.%d.WANPPPConnection.%d.Username",
				conn,
				instance,
			))
		}
	}

	return paths
}

func expandWildcardPath(path string) []string {
	if !strings.Contains(path, ".*.") {
		return []string{path}
	}
	paths := make([]string, 0, 4)
	for i := 1; i <= 4; i++ {
		paths = append(paths, strings.Replace(
			path,
			".*.",
			fmt.Sprintf(".%d.", i),
			1,
		))
	}
	return paths
}

func projection() string {
	fields := []string{
		"_id",
		"_deviceId._SerialNumber",
		"_deviceId._Manufacturer",
		"_deviceId._ProductClass",
		"_lastInform",
		ssidPath,
	}

	for _, path := range rxPowerPaths {
		fields = append(fields, expandWildcardPath(path)...)
	}

	fields = append(fields, pppoeUsernamePaths()...)

	return strings.Join(fields, ",")
}

func baseURL(item *models.GenieACS) (string, error) {
	if item == nil {
		return "", fmt.Errorf("konfigurasi GenieACS kosong")
	}

	host := strings.TrimRight(strings.TrimSpace(item.Host), "/")
	if host == "" {
		return "", fmt.Errorf("host GenieACS kosong")
	}

	parsed, err := url.Parse(host)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return "", fmt.Errorf("format host GenieACS tidak valid")
	}

	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return "", fmt.Errorf("skema host GenieACS harus http atau https")
	}

	if parsed.User != nil {
		return "", fmt.Errorf("host GenieACS tidak boleh berisi kredensial URL")
	}

	if parsed.Port() == "" {
		if item.Port <= 0 || item.Port > 65535 {
			return "", fmt.Errorf("port GenieACS tidak valid")
		}

		parsed.Host = fmt.Sprintf("%s:%d", parsed.Host, item.Port)
	}

	parsed.Path = strings.TrimRight(parsed.Path, "/")
	parsed.RawQuery = ""
	parsed.Fragment = ""

	return strings.TrimRight(parsed.String(), "/"), nil
}

func GetDevices(ctx context.Context, item *models.GenieACS) ([]Device, error) {
	base, err := baseURL(item)
	if err != nil {
		return nil, err
	}

	query := url.Values{}
	query.Set("projection", projection())

	endpoint := base + "/devices?" + query.Encode()

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodGet,
		endpoint,
		nil,
	)
	if err != nil {
		return nil, err
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("gagal menghubungi GenieACS: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf(
			"GenieACS mengembalikan status %d",
			resp.StatusCode,
		)
	}

	var raw []map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return nil, fmt.Errorf("gagal membaca respons GenieACS: %w", err)
	}

	devices := make([]Device, 0, len(raw))

	for _, d := range raw {
		devices = append(devices, toDevice(d))
	}

	return devices, nil
}

func toDevice(d map[string]any) Device {
	device := Device{
		ID:            getString(d, "_id"),
		SerialNumber:  getString(d, "_deviceId._SerialNumber"),
		Manufacturer:  getString(d, "_deviceId._Manufacturer"),
		ProductClass:  getString(d, "_deviceId._ProductClass"),
		Status:        "offline",
		PPPoEUsername: getPPPoEUsername(d),
		RXPower:       getRXPower(d),
		SSID:          getValue(d, ssidPath),
	}

	if lastInform := getString(d, "_lastInform"); lastInform != "" {
		if t, err := time.Parse(time.RFC3339Nano, lastInform); err == nil {
			device.LastInform = &t

			if age := time.Since(t); age >= 0 && age <= onlineWindow {
				device.Status = "online"
			}
		}
	}

	return device
}

func getPPPoEUsername(data map[string]any) string {
	for _, path := range pppoeUsernamePaths() {
		if username := strings.TrimSpace(getValue(data, path)); username != "" {
			return username
		}
	}

	return ""
}

func getRXPower(data map[string]any) string {
	for _, pattern := range rxPowerPaths {
		for _, path := range expandWildcardPath(pattern) {
			if value := strings.TrimSpace(getValue(data, path)); value != "" {
				return calculateRXPower(value)
			}
		}
	}

	return "N/A"
}

func calculateRXPower(value string) string {
	value = strings.TrimSpace(value)
	if strings.HasSuffix(strings.ToLower(value), "dbm") {
		value = strings.TrimSpace(value[:len(value)-3])
		if _, err := strconv.ParseFloat(value, 64); err != nil {
			return "N/A"
		}
		return value
	}
	raw, err := strconv.ParseFloat(value, 64)
	if err != nil || math.IsNaN(raw) || math.IsInf(raw, 0) {
		return "N/A"
	}
	if raw < 0 {
		return strconv.FormatFloat(raw, 'f', -1, 64)
	}
	if raw == 0 {
		return "N/A"
	}
	result := math.Ceil(10 * math.Log10(raw/10000))
	return strconv.FormatFloat(result, 'f', -1, 64)
}

func lookup(data map[string]any, path string) any {
	var current any = data
	for key := range strings.SplitSeq(path, ".") {
		object, ok := current.(map[string]any)
		if !ok {
			return nil
		}

		current = object[key]
	}
	return current
}

func getString(data map[string]any, path string) string {
	value, _ := lookup(data, path).(string)
	return value
}

func getValue(data map[string]any, path string) string {
	object, ok := lookup(data, path).(map[string]any)
	if !ok {
		return ""
	}

	switch value := object["_value"].(type) {
	case string:
		return value
	case float64:
		return strconv.FormatFloat(value, 'f', -1, 64)
	case json.Number:
		return value.String()
	case bool:
		return strconv.FormatBool(value)
	default:
		return ""
	}
}

const summonTimeout = 30 * time.Second

var summonClient = &http.Client{
	Timeout: summonTimeout,
}

func SummonDevice(ctx context.Context, item *models.GenieACS, deviceID string) error {
	base, err := baseURL(item)
	if err != nil {
		return err
	}
	deviceID = strings.TrimSpace(deviceID)
	if deviceID == "" {
		return fmt.Errorf("device ID kosong")
	}
	endpoint := fmt.Sprintf(
		"%s/devices/%s/tasks?connection_request",
		base,
		url.PathEscape(deviceID),
	)
	objects := []string{
		"InternetGatewayDevice.DeviceInfo.",
		"InternetGatewayDevice.WANDevice.",
		"InternetGatewayDevice.LANDevice.1.WLANConfiguration.1.",
	}
	for _, object := range objects {
		payload, err := json.Marshal(map[string]any{
			"name":       "refreshObject",
			"objectName": object,
		})
		if err != nil {
			return err
		}
		req, err := http.NewRequestWithContext(
			ctx,
			http.MethodPost,
			endpoint,
			bytes.NewReader(payload),
		)
		if err != nil {
			return err
		}
		req.Header.Set("Content-Type", "application/json")
		resp, err := summonClient.Do(req)
		if err != nil {
			return fmt.Errorf("gagal refresh object %s: %w", object, err)
		}
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		resp.Body.Close()
		if resp.StatusCode != http.StatusOK &&
			resp.StatusCode != http.StatusCreated &&
			resp.StatusCode != http.StatusAccepted {
			return fmt.Errorf(
				"gagal refresh object %s, status %d: %s",
				object,
				resp.StatusCode,
				string(body),
			)
		}
	}
	return nil
}
