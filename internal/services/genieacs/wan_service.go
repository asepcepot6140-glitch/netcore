package genieacs

import (
	"context"
	"encoding/json"
	"fmt"
	"maps"
	"net/http"
	"net/url"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/asepcepot6140-glitch/netcore/internal/models"
)

var (
	vlanKeys       = []string{"X_CT-COM_VLANIDMark", "X_ZTE-COM_VLANID", "X_FH_VLANID", "X_CMCC_VLANIDMark", "X_HW_VLAN", "VLANID"}
	vlanEnableKeys = []string{"X_CT-COM_VLANEnable", "X_ZTE-COM_VLANEnable", "X_FH_VLANEnable", "X_CMCC_VLANEnable", "X_HW_VLANEnable", "VLANEnable"}
	serviceKeys    = []string{"X_CT-COM_ServiceList", "X_ZTE-COM_ServiceList", "X_FH_ServiceList", "X_CMCC_ServiceList", "X_HW_SERVICELIST", "X_D0542D_ServiceList", "ServiceList"}
	lanBindKeys    = []string{"X_CT-COM_LanInterface", "X_ZTE-COM_LanInterface", "X_FH_LanInterface", "X_CMCC_LanInterface", "X_ALU-COM_LanInterface"}
	linkConfigKeys = []string{"X_CT-COM_WANGponLinkConfig"}
	linkVLANKeys   = []string{"VLANIDMark", "VLANID"}

	// nameLockedKeys mark FiberHome, Huawei and Nokia connections, whose Name the device does not let us change.
	nameLockedKeys = []string{"X_FH_ServiceList", "X_HW_SERVICELIST", "X_HW_LANBIND", "X_D0542D_ServiceList"}

	// serviceOptions are the values seen on real devices, keyed by the vendor's ServiceList parameter.
	serviceOptions = map[string][]string{
		"X_HW_SERVICELIST":      {"INTERNET", "TR069_VOIP", "OTHER"},
		"X_D0542D_ServiceList":  {"INTERNET", "TR069", "OTHER"},
		"X_FH_ServiceList":      {"INTERNET", "TR069", "TR069,VOIP", "OTHER"},
		"X_ZTE-COM_ServiceList": {"INTERNET", "TR069", "VoIP_TR069", "INTERNET_VoIP_TR069", "OTHER"},
		"X_CT-COM_ServiceList":  {"INTERNET", "TR069", "TR069_VOIP_INTERNET", "INTERNET_VoIP_TR069", "OTHER"},
		"X_CMCC_ServiceList":    {"INTERNET", "TR069", "TR069_VOIP_INTERNET", "INTERNET_VoIP_TR069", "OTHER"},
	}
)

const wanProjection = "_id,InternetGatewayDevice.WANDevice," + lanBindLANPath + "," + lanBindWLANPath + "," + portBindingPath

const (
	lanBindLANPath  = "InternetGatewayDevice.LANDevice.1.LANEthernetInterfaceConfig"
	lanBindWLANPath = "InternetGatewayDevice.LANDevice.1.WLANConfiguration"
)

type WAN struct {
	Path       string `json:"path"`
	ConnDevice string `json:"conn_device"`
	ConnIndex  string `json:"conn_instance"`
	Kind       string `json:"kind"`
	Enable     string `json:"enable"`
	ConnName   string `json:"conn_name"`
	EnableVLAN bool   `json:"enable_vlan"`
	VLAN       string `json:"vlan"`
	ConnType   string `json:"conn_type"`
	Service    string `json:"service"`
	Username   string `json:"username"`
	Password   string `json:"password"`
	NAT        string `json:"nat"`
	LanBind    string `json:"lan_bind"`
	// LanBindOptions are the LAN/SSID ports this device has; LanBindDisabled are those that cannot be toggled.
	LanBindOptions  []string `json:"lan_bind_options"`
	LanBindDisabled []string `json:"lan_bind_disabled"`
	ConnTypeOptions []string `json:"conn_type_options"`
	ServiceOptions  []string `json:"service_options"`
	NameLocked      bool     `json:"name_locked"`
	Status          string   `json:"status"`
}

func GetWAN(ctx context.Context, item *models.GenieACS, deviceID string) ([]WAN, error) {
	base, err := baseURL(item)
	if err != nil {
		return nil, err
	}

	deviceID = strings.TrimSpace(deviceID)
	if deviceID == "" {
		return nil, fmt.Errorf("device ID is empty")
	}

	query, err := json.Marshal(map[string]string{"_id": deviceID})
	if err != nil {
		return nil, err
	}

	params := url.Values{}
	params.Set("query", string(query))
	params.Set("projection", wanProjection)

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodGet,
		base+"/devices?"+params.Encode(),
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
		return nil, fmt.Errorf("GenieACS returned status %d", resp.StatusCode)
	}

	var devices []map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&devices); err != nil {
		return nil, fmt.Errorf("failed to read GenieACS response: %w", err)
	}

	wans := make([]WAN, 0)
	for _, device := range devices {
		wans = append(wans, parseWANs(device)...)
	}

	return wans, nil
}

func parseWANs(device map[string]any) []WAN {
	rootPath := "InternetGatewayDevice.WANDevice"
	wanDevices, ok := lookup(device, rootPath).(map[string]any)
	if !ok {
		return []WAN{}
	}

	var wans []WAN
	choices := lanBindChoices(device)
	for _, wanDeviceIndex := range numericKeys(wanDevices) {
		wanDevice, _ := wanDevices[wanDeviceIndex].(map[string]any)
		connDevices, _ := wanDevice["WANConnectionDevice"].(map[string]any)

		for _, connDeviceIndex := range numericKeys(connDevices) {
			connDevice, _ := connDevices[connDeviceIndex].(map[string]any)
			connDevicePath := fmt.Sprintf(
				"%s.%s.WANConnectionDevice.%s",
				rootPath,
				wanDeviceIndex,
				connDeviceIndex,
			)

			for _, kind := range []string{"WANIPConnection", "WANPPPConnection"} {
				connections, _ := connDevice[kind].(map[string]any)
				for _, connIndex := range numericKeys(connections) {
					connection, _ := connections[connIndex].(map[string]any)
					wan := parseWAN(
						connection,
						connDevice,
						connDevicePath,
						kind,
						connIndex,
					)
					wan.LanBindOptions, wan.LanBindDisabled = huaweiBindChoices(device, connection, choices)
					if table, ok := lookup(device, portBindingPath).(map[string]any); ok && usesPortBinding(connection) {
						_, entry := portBindingEntry(table, wan.Path)
						wan.LanBind = getValue(entry, "LANInterface")
						wan.LanBindOptions = choices
					}
					// Huawei (marked by X_HW_LANBIND) IPoE also allows bridged; Huawei PPPoE keeps free text.
					_, huawei := connection["X_HW_LANBIND"]
					switch {
					case kind == "WANIPConnection":
						wan.ConnTypeOptions = []string{"IP_Routed"}
						if huawei {
							wan.ConnTypeOptions = append(wan.ConnTypeOptions, "IP_Bridged")
						}
					case !huawei:
						wan.ConnTypeOptions = []string{"IP_Routed", "PPPoE_Bridged"}
					}
					wan.ServiceOptions = serviceChoices(connection)
					wan.NameLocked = nameLocked(connection)
					wans = append(wans, wan)
				}
			}
		}
	}

	return wans
}

// serviceChoices returns the known Service values for the vendor parameter this connection has; nil for unknown vendors.
func serviceChoices(connection map[string]any) []string {
	for _, key := range serviceKeys {
		if _, ok := connection[key]; ok {
			return serviceOptions[key]
		}
	}
	return nil
}

// nameLocked reports whether the connection belongs to a vendor whose Name cannot be edited.
func nameLocked(connection map[string]any) bool {
	for _, key := range nameLockedKeys {
		if _, ok := connection[key]; ok {
			return true
		}
	}
	return false
}

// lanBindChoices lists the LAN ports, then SSIDs, that exist on the device.
func lanBindChoices(device map[string]any) []string {
	choices := make([]string, 0, 12)
	lan, _ := lookup(device, lanBindLANPath).(map[string]any)
	for _, index := range numericKeys(lan) {
		choices = append(choices, "LAN"+index)
	}
	wlan, _ := lookup(device, lanBindWLANPath).(map[string]any)
	for _, index := range numericKeys(wlan) {
		choices = append(choices, "SSID"+index)
	}
	return choices
}

// huaweiBindChoices lists ports that have an X_HW_LANBIND flag; LAN ports without X_HW_L3Enable true are
// returned as disabled. Other vendors keep all choices, none disabled.
func huaweiBindChoices(device, connection map[string]any, choices []string) (options, disabled []string) {
	disabled = []string{}
	bind, ok := connection["X_HW_LANBIND"].(map[string]any)
	if !ok {
		// Without any bind parameter (e.g. some ZTE F670L units) nothing can be bound.
		if !hasLanBindKey(connection) {
			return []string{}, disabled
		}
		return choices, disabled
	}

	options = make([]string, 0, len(choices))
	for _, choice := range choices {
		key := choice + "Enable"
		rest, isLAN := strings.CutPrefix(choice, "LAN")
		if isLAN {
			key = "Lan" + rest + "Enable"
		}
		if _, exists := bind[key].(map[string]any); !exists {
			continue
		}
		options = append(options, choice)
		if isLAN && getValue(device, lanBindLANPath+"."+rest+".X_HW_L3Enable") != "true" {
			disabled = append(disabled, choice)
		}
	}
	return options, disabled
}

func hasLanBindKey(connection map[string]any) bool {
	for _, key := range lanBindKeys {
		if _, ok := connection[key]; ok {
			return true
		}
	}
	return false
}

func parseHuaweiLANBind(connection map[string]any) string {
	bind, ok := connection["X_HW_LANBIND"].(map[string]any)
	if !ok {
		return ""
	}

	var enabled []string

	for i := 1; i <= 4; i++ {
		key := fmt.Sprintf("Lan%dEnable", i)
		if getValue(bind, key) == "1" {
			enabled = append(enabled, fmt.Sprintf("LAN%d", i))
		}
	}

	for i := 1; i <= 8; i++ {
		key := fmt.Sprintf("SSID%dEnable", i)
		if getValue(bind, key) == "1" {
			enabled = append(enabled, fmt.Sprintf("SSID%d", i))
		}
	}

	return strings.Join(enabled, ", ")
}

var lanBindLabel = regexp.MustCompile(`^(LAN|SSID)([1-9]\d?)$`)

// parseLanBindLabels turns "LAN1,SSID2" into a set of upper-case port labels.
func parseLanBindLabels(raw string) (map[string]bool, error) {
	selected := map[string]bool{}
	for part := range strings.SplitSeq(raw, ",") {
		label := strings.ToUpper(strings.TrimSpace(part))
		if label == "" {
			continue
		}
		if !lanBindLabel.MatchString(label) {
			return nil, fmt.Errorf("%w: lan_bind has an invalid port", ErrInvalidParameter)
		}
		selected[label] = true
	}
	return selected, nil
}

// lanBindUpdates turns labels like "LAN1,SSID2" into the writes the device's vendor format expects.
func lanBindUpdates(path string, node map[string]any, raw string) ([][]any, error) {
	selected, err := parseLanBindLabels(raw)
	if err != nil {
		return nil, err
	}

	if name, xsdType, found := resolveParameter(node, paramSpec{names: lanBindKeys, xsdType: typeString}); found {
		return [][]any{{path + "." + name, strings.Join(lanBindPaths(selected), ","), xsdType}}, nil
	}
	if bind, ok := node["X_HW_LANBIND"].(map[string]any); ok {
		return huaweiLANBindUpdates(path, bind, selected)
	}
	return nil, fmt.Errorf("%w: lan_bind", ErrParameterNotSupported)
}

// huaweiLANBindUpdates writes only the per-port flags that differ from the device.
func huaweiLANBindUpdates(path string, bind map[string]any, selected map[string]bool) ([][]any, error) {
	type port struct{ label, key string }
	var ports []port
	for i := 1; i <= 4; i++ {
		ports = append(ports, port{fmt.Sprintf("LAN%d", i), fmt.Sprintf("Lan%dEnable", i)})
	}
	for i := 1; i <= 8; i++ {
		ports = append(ports, port{fmt.Sprintf("SSID%d", i), fmt.Sprintf("SSID%dEnable", i)})
	}

	unknown := maps.Clone(selected)
	var updates [][]any
	for _, p := range ports {
		object, ok := bind[p.key].(map[string]any)
		if !ok {
			continue
		}
		delete(unknown, p.label)

		if (getValue(bind, p.key) == "1") == selected[p.label] {
			continue
		}
		if writable, known := object["_writable"].(bool); known && !writable {
			return nil, fmt.Errorf("%w: lan_bind %s", ErrParameterNotSupported, p.label)
		}

		xsdType := typeUnsigned
		if reported, _ := object["_type"].(string); reported != "" {
			xsdType = reported
		}
		var value any
		switch {
		case xsdType == typeBoolean:
			value = selected[p.label]
		case selected[p.label]:
			value = 1
		default:
			value = 0
		}
		updates = append(updates, []any{path + ".X_HW_LANBIND." + p.key, value, xsdType})
	}

	// Selected ports without a flag on this device cannot be bound.
	if len(unknown) > 0 {
		return nil, fmt.Errorf("%w: lan_bind port", ErrParameterNotSupported)
	}
	return updates, nil
}

// lanBindPaths lists WLAN instances first, then LAN ports, each in numeric order.
func lanBindPaths(selected map[string]bool) []string {
	var wlan, lan []int
	for label := range selected {
		match := lanBindLabel.FindStringSubmatch(label)
		index, _ := strconv.Atoi(match[2])
		if match[1] == "SSID" {
			wlan = append(wlan, index)
		} else {
			lan = append(lan, index)
		}
	}
	sort.Ints(wlan)
	sort.Ints(lan)

	paths := make([]string, 0, len(wlan)+len(lan))
	for _, index := range wlan {
		paths = append(paths, fmt.Sprintf("InternetGatewayDevice.LANDevice.1.WLANConfiguration.%d", index))
	}
	for _, index := range lan {
		paths = append(paths, fmt.Sprintf("InternetGatewayDevice.LANDevice.1.LANEthernetInterfaceConfig.%d", index))
	}
	return paths
}

func parseWAN(connection, connDevice map[string]any, connDevicePath, kind, connIndex string) WAN {
	vlan := firstValue(connection, vlanKeys)

	if vlan == "" {
		vlan = linkConfigVLAN(connDevice)
	}

	enableVLAN, err := strconv.ParseBool(firstValue(connection, vlanEnableKeys))
	if err != nil {
		id, _ := strconv.Atoi(vlan)
		enableVLAN = id > 0
	}

	lanBind := firstValue(connection, lanBindKeys)
	if lanBind == "" {
		lanBind = parseHuaweiLANBind(connection)
	}

	return WAN{
		Path:       fmt.Sprintf("%s.%s.%s", connDevicePath, kind, connIndex),
		ConnDevice: connDevicePath,
		ConnIndex:  connIndex,
		Kind:       kind,
		Enable:     getValue(connection, "Enable"),
		ConnName:   getValue(connection, "Name"),
		EnableVLAN: enableVLAN,
		VLAN:       vlan,
		ConnType:   getValue(connection, "ConnectionType"),
		Service:    firstValue(connection, serviceKeys),
		Username:   getValue(connection, "Username"),
		Password:   getValue(connection, "Password"),
		NAT:        getValue(connection, "NATEnabled"),
		LanBind:    lanBind,
		Status:     getValue(connection, "ConnectionStatus"),
	}
}

func linkConfigVLAN(connDevice map[string]any) string {
	for _, key := range linkConfigKeys {
		linkConfig, ok := connDevice[key].(map[string]any)
		if !ok {
			continue
		}
		if vlan := firstValue(linkConfig, linkVLANKeys); vlan != "" {
			return vlan
		}
	}
	return ""
}

func firstValue(data map[string]any, keys []string) string {
	for _, key := range keys {
		if value := strings.TrimSpace(getValue(data, key)); value != "" {
			return value
		}
	}
	return ""
}

func numericKeys(data map[string]any) []string {
	keys := make([]string, 0, len(data))
	for key := range data {
		if _, err := strconv.Atoi(key); err == nil {
			keys = append(keys, key)
		}
	}
	sort.Slice(keys, func(i, j int) bool {
		left, _ := strconv.Atoi(keys[i])
		right, _ := strconv.Atoi(keys[j])
		return left < right
	})
	return keys
}
