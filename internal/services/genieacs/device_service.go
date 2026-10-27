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
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/asepcepot6140-glitch/netcore/internal/models"
	"github.com/asepcepot6140-glitch/netcore/internal/utils"
)

type Device struct {
	ID            string     `json:"id"`
	SerialNumber  string     `json:"serial_number"`
	Manufacturer  string     `json:"manufacturer"`
	ProductClass  string     `json:"product_class"`
	LastInform    *time.Time `json:"last_inform"`
	Registered    *time.Time `json:"registered"`
	Status        string     `json:"status"`
	PPPoEUsername string     `json:"pppoe_username"`
	RXPower       string     `json:"rx_power"`
	SSID          string     `json:"ssid"`
}

const (
	ssidPath     = "InternetGatewayDevice.LANDevice.1.WLANConfiguration.1.SSID"
	onlineWindow = 5 * time.Minute
)

var httpClient = utils.NewSafeHTTPClient(15 * time.Second)

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
		"_registered",
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
		return "", fmt.Errorf("GenieACS configuration is empty")
	}

	host := strings.TrimRight(strings.TrimSpace(item.Host), "/")
	if host == "" {
		return "", fmt.Errorf("GenieACS host is empty")
	}

	host = WithScheme(host, item.Port)

	parsed, err := url.Parse(host)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return "", fmt.Errorf("invalid GenieACS host format")
	}

	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return "", fmt.Errorf("GenieACS host scheme must be http or https")
	}

	if parsed.User != nil {
		return "", fmt.Errorf("GenieACS host must not contain URL credentials")
	}

	if parsed.Port() == "" {
		if item.Port <= 0 || item.Port > 65535 {
			return "", fmt.Errorf("invalid GenieACS port")
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
		return nil, fmt.Errorf("failed to reach GenieACS: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf(
			"GenieACS returned status %d",
			resp.StatusCode,
		)
	}

	var raw []map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return nil, fmt.Errorf("failed to read GenieACS response: %w", err)
	}

	devices := make([]Device, 0, len(raw))

	for _, d := range raw {
		devices = append(devices, ToDevice(d))
	}

	SortByNewestRegistration(devices)

	return devices, nil
}

// SortByNewestRegistration orders newest registration first; devices without a registration time go last.
func SortByNewestRegistration(devices []Device) {
	sort.SliceStable(devices, func(i, j int) bool {
		a, b := devices[i].Registered, devices[j].Registered
		if a == nil || b == nil {
			return a != nil && b == nil
		}
		return a.After(*b)
	})
}

func ToDevice(d map[string]any) Device {
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

	if registered := getString(d, "_registered"); registered != "" {
		if t, err := time.Parse(time.RFC3339Nano, registered); err == nil {
			device.Registered = &t
		}
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
				return CalculateRXPower(value)
			}
		}
	}

	return "N/A"
}

func CalculateRXPower(value string) string {
	value = strings.TrimSpace(value)
	if strings.HasSuffix(strings.ToLower(value), "dbm") {
		value = strings.TrimSpace(value[:len(value)-3])
		parsed, err := strconv.ParseFloat(value, 64)
		if err != nil || math.IsNaN(parsed) || math.IsInf(parsed, 0) {
			return "N/A"
		}
		return formatRXPower(parsed)
	}
	raw, err := strconv.ParseFloat(value, 64)
	if err != nil || math.IsNaN(raw) || math.IsInf(raw, 0) {
		return "N/A"
	}
	if raw < 0 {
		return formatRXPower(raw)
	}
	if raw == 0 {
		return "N/A"
	}
	return formatRXPower(10 * math.Log10(raw/10000))
}

// formatRXPower rounds to 2 decimals and avoids a "-0.00" result.
func formatRXPower(dbm float64) string {
	text := strconv.FormatFloat(dbm, 'f', 2, 64)
	if text == "-0.00" {
		return "0.00"
	}
	return text
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

var summonClient = utils.NewSafeHTTPClient(summonTimeout)

func SummonDevice(ctx context.Context, item *models.GenieACS, deviceID string) (bool, error) {
	return summonObjects(ctx, item, deviceID, []string{
		"InternetGatewayDevice.DeviceInfo",
		"InternetGatewayDevice.WANDevice",
		"InternetGatewayDevice.LANDevice.1.WLANConfiguration.1",
	})
}

func SummonDeviceObject(
	ctx context.Context,
	item *models.GenieACS,
	deviceID string,
	scope string,
) (bool, error) {
	objects := map[string]string{
		"wan":  "InternetGatewayDevice.WANDevice.1.WANConnectionDevice",
		"lan":  "InternetGatewayDevice.LANDevice",
		"wlan": "InternetGatewayDevice.LANDevice.1.WLANConfiguration",
	}
	object, ok := objects[scope]
	if !ok {
		return false, fmt.Errorf("invalid summon scope")
	}

	return summonObjects(ctx, item, deviceID, []string{object})
}

// summonObjects refreshes objects on the device. applied is false when any task was only queued.
func summonObjects(
	ctx context.Context,
	item *models.GenieACS,
	deviceID string,
	objects []string,
) (bool, error) {
	base, err := baseURL(item)
	if err != nil {
		return false, err
	}
	deviceID = strings.TrimSpace(deviceID)
	if deviceID == "" {
		return false, fmt.Errorf("device ID is empty")
	}
	endpoint := fmt.Sprintf(
		"%s/devices/%s/tasks?connection_request",
		base,
		url.PathEscape(deviceID),
	)
	applied := true
	for _, object := range objects {
		payload, err := json.Marshal(map[string]any{
			"name":       "refreshObject",
			"objectName": object,
		})
		if err != nil {
			return false, err
		}
		req, err := http.NewRequestWithContext(
			ctx,
			http.MethodPost,
			endpoint,
			bytes.NewReader(payload),
		)
		if err != nil {
			return false, err
		}
		req.Header.Set("Content-Type", "application/json")
		resp, err := summonClient.Do(req)
		if err != nil {
			return false, fmt.Errorf("failed to refresh object %s: %w", object, err)
		}
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		resp.Body.Close()
		switch resp.StatusCode {
		case http.StatusOK, http.StatusCreated:
		case http.StatusAccepted:
			applied = false
		default:
			return false, fmt.Errorf(
				"failed to refresh object %s, status %d: %s",
				object,
				resp.StatusCode,
				string(body),
			)
		}
	}
	return applied, nil
}

// DeleteDevice removes the device record from GenieACS; it re-registers on its next inform.
func DeleteDevice(ctx context.Context, item *models.GenieACS, deviceID string) error {
	base, err := baseURL(item)
	if err != nil {
		return err
	}
	deviceID = strings.TrimSpace(deviceID)
	if deviceID == "" {
		return fmt.Errorf("device ID is empty")
	}

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodDelete,
		fmt.Sprintf("%s/devices/%s", base, url.PathEscape(deviceID)),
		nil,
	)
	if err != nil {
		return err
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to reach GenieACS: %w", err)
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK, http.StatusNoContent:
		return nil
	case http.StatusNotFound:
		return ErrDeviceNotFound
	default:
		return fmt.Errorf("GenieACS returned status %d", resp.StatusCode)
	}
}
