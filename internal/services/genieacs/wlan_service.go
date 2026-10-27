package genieacs

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/asepcepot6140-glitch/netcore/internal/models"
)

const wlanProjection = "_id,_deviceId._Manufacturer,InternetGatewayDevice.LANDevice.1.WLANConfiguration"

func isFiberHome(device map[string]any) bool {
	if strings.Contains(strings.ToLower(getString(device, "_deviceId._Manufacturer")), "fiberhome") {
		return true
	}
	id, _ := device["_id"].(string)
	return strings.Contains(strings.ToUpper(id), "-FHTT")
}

func isNokia(device map[string]any) bool {
	manufacturer := strings.ToLower(getString(device, "_deviceId._Manufacturer"))
	return manufacturer == "alcl" || strings.Contains(manufacturer, "nokia") ||
		strings.Contains(manufacturer, "alcatel")
}

func beaconTypeOptions(device map[string]any) []string {
	switch {
	case isFiberHome(device):
		return []string{"None", "WPA", "11i", "WPAand11i"}
	case isNokia(device):
		return []string{"Basic", "11i", "WPAand11i", "WPA3", "11iandWPA3"}
	}
	return []string{"Basic", "WPA", "11i", "WPAand11i"}
}

var wlanPasswordKeys = []string{"PreSharedKey.1.KeyPassphrase", "KeyPassphrase"}

// nokiaPasswordSpecials are the only symbols the Nokia accepted when written live; every other symbol got fault 9003.
const nokiaPasswordSpecials = "!@#-_=+:,./"

type PasswordPolicy struct {
	MinLength int    `json:"min_length"`
	MaxLength int    `json:"max_length"`
	Specials  string `json:"specials"`
}

// isNokiaWLAN detects Nokia by its vendor parameters, since a projected node carries no manufacturer.
func isNokiaWLAN(config map[string]any) bool {
	for key := range config {
		if strings.HasPrefix(key, "X_ALU") {
			return true
		}
	}
	return false
}

func passwordPolicy(config map[string]any) *PasswordPolicy {
	if isNokiaWLAN(config) {
		return &PasswordPolicy{MinLength: 8, MaxLength: 63, Specials: nokiaPasswordSpecials}
	}
	return nil
}

// minWLANPasswordLength applies to every vendor; Nokia adds its own stricter policy.
const minWLANPasswordLength = 8

// validatePassword requires upper case, lower case, a digit and an allowed symbol when the device
// has a policy (Nokia); otherwise only the minimum length is checked.
func validatePassword(policy *PasswordPolicy, password string) error {
	if policy == nil {
		if utf8.RuneCountInString(password) < minWLANPasswordLength {
			return fmt.Errorf("%w: key_passphrase must be at least %d characters", ErrInvalidParameter, minWLANPasswordLength)
		}
		return nil
	}
	var upper, lower, digit, special bool
	for i, r := range password {
		switch {
		case r >= 'A' && r <= 'Z':
			upper = true
		case r >= 'a' && r <= 'z':
			lower = true
		case r >= '0' && r <= '9':
			digit = true
		case strings.ContainsRune(policy.Specials, r):
			if i == 0 {
				return fmt.Errorf("%w: key_passphrase must start with a letter or digit", ErrInvalidParameter)
			}
			special = true
		default:
			return fmt.Errorf("%w: key_passphrase may only contain letters, digits and %s", ErrInvalidParameter, policy.Specials)
		}
	}
	if n := len(password); n < policy.MinLength || n > policy.MaxLength {
		return fmt.Errorf("%w: key_passphrase must be %d-%d characters", ErrInvalidParameter, policy.MinLength, policy.MaxLength)
	}
	if !upper || !lower || !digit || !special {
		return fmt.Errorf("%w: key_passphrase needs upper case, lower case, a digit and one of %s", ErrInvalidParameter, policy.Specials)
	}
	return nil
}

var wlanMaxClientKeys = []string{
	"X_HW_AssociateNum", "X_FH_AssociateNum", "X_ZTE-COM_MaxUserNum", "X_ALU-COM_VirtualIfCfg_MaxAssoc",
}

type WLAN struct {
	Path                            string                 `json:"path"`
	Index                           string                 `json:"index"`
	SSID                            string                 `json:"ssid"`
	Enable                          string                 `json:"enable"`
	Channel                         string                 `json:"channel"`
	ChannelOptions                  []string               `json:"channel_options"`
	ChannelWidth                    string                 `json:"channel_width"`
	ChannelWidthOptions             []string               `json:"channel_width_options"`
	ChannelWidthByStandard          map[string][]string    `json:"channel_width_by_standard,omitempty"`
	MaxClients                      string                 `json:"max_clients"`
	AutoChannelEnable               string                 `json:"auto_channel_enable"`
	OperatingFrequencyBand          string                 `json:"operating_frequency_band"`
	Standard                        string                 `json:"standard"`
	StandardOptions                 []string               `json:"standard_options"`
	TransmitPower                   string                 `json:"transmit_power"`
	BeaconType                      string                 `json:"beacon_type"`
	BeaconTypeOptions               []string               `json:"beacon_type_options"`
	KeyPassphrase                   string                 `json:"key_passphrase"`
	PasswordPolicy                  *PasswordPolicy        `json:"password_policy,omitempty"`
	SSIDAdvertisementEnabled        string                 `json:"ssid_advertisement_enabled"`
	WMMEnable                       string                 `json:"wmm_enable"`
	AssociatedDeviceNumberOfEntries string                 `json:"associated_device_number_of_entries"`
	AssociatedDevices               []WLANAssociatedDevice `json:"associated_devices"`
}

var (
	assocNameKeys = []string{
		"X_ZTE-COM_AssociatedDeviceName", "X_FH_DeviceName", "X_HW_AssociatedDevicedescriptions",
	}
	assocMACKeys  = []string{"AssociatedDeviceMACAddress", "MACAddress"}
	assocIPKeys   = []string{"AssociatedDeviceIPAddress", "IPAddress"}
	assocAuthKeys = []string{"AssociatedDeviceAuthenticationState", "AuthenticationState"}
	assocRSSIKeys = []string{
		"X_ZTE-COM_WLAN_RSSI", "X_FH_WLAN_RSSI", "AssociatedDeviceRSSI",
		"X_FH_WLAN_ClientSignalStrength", "X_HW_RSSI", "SignalStrength", "RSSI",
	}
	assocTxRateKeys = []string{
		"X_ZTE-COM_WLAN_ClientTxRate", "X_FH_WLAN_ClientTxRate", "AssociatedDeviceTxRate",
		"X_HW_TxRate", "LastDataTransmitRate", "X_ALU-COM_CurrentTxRate", "LastDataDownlinkRate",
	}
	assocRxRateKeys = []string{
		"X_ZTE-COM_WLAN_ClientRxRate", "X_FH_WLAN_ClientRxRate", "X_HW_RxRate", "LastDataUplinkRate",
	}
	assocBandwidthKeys = []string{
		"AssociatedDeviceBandWidth", "X_FH_WLAN_ClientChannelWidth", "X_HW_FrequencyWidth", "X_ALU-COM_OperatingBW",
	}
	assocModeKeys = []string{
		"X_ZTE-COM_WLAN_ClientMode", "X_FH_WLAN_ClientMode", "X_HW_WorkingMode",
		"OperatingStandard", "X_ALU-COM_Standards",
	}
	assocRadioKeys    = []string{"X_ZTE-COM_WLAN_Radio", "X_FH_WLAN_Radio"}
	assocStayTimeKeys = []string{"X_ZTE-COM_StayTime", "X_FH_WLAN_UpTime", "X_HW_Uptime", "AssociationTime"}
	assocBytesRxKeys  = []string{
		"X_ZTE-COM_BytesReceived", "X_FH_WLAN_BytesReceived", "StaDeviceStats.BytesReceived", "Stats.BytesReceived",
	}
	assocBytesTxKeys = []string{
		"X_ZTE-COM_BytesSent", "X_FH_WLAN_BytesSent", "StaDeviceStats.BytesSent", "Stats.BytesSent",
	}
)

type WLANAssociatedDevice struct {
	Path                string `json:"path"`
	Name                string `json:"name"`
	MACAddress          string `json:"mac_address"`
	IPAddress           string `json:"ip_address"`
	AuthenticationState string `json:"authentication_state"`
	RSSI                string `json:"rssi"`
	TxRate              string `json:"tx_rate"`
	RxRate              string `json:"rx_rate"`
	Bandwidth           string `json:"bandwidth"`
	Mode                string `json:"mode"`
	Radio               string `json:"radio"`
	StayTime            string `json:"stay_time"`
	BytesReceived       string `json:"bytes_received"`
	BytesSent           string `json:"bytes_sent"`
}

func GetWLAN(ctx context.Context, item *models.GenieACS, deviceID string) ([]WLAN, error) {
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
	params.Set("projection", wlanProjection)

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

	wlans := make([]WLAN, 0)
	for _, device := range devices {
		wlans = append(wlans, parseWLANs(device)...)
	}

	return wlans, nil
}

func wlanChannel(config map[string]any) string {
	channel := strings.TrimSpace(getValue(config, "Channel"))
	if channel != "" && channel != "0" {
		return channel
	}
	if inUse := strings.TrimSpace(getValue(config, "ChannelsInUse")); inUse != "" {
		return inUse
	}
	return channel
}

func parseChannelList(raw string) []string {
	first, _, _ := strings.Cut(raw, ";")
	var channels []string
	for part := range strings.SplitSeq(first, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		low, high, isRange := strings.Cut(part, "-")
		if !isRange {
			if _, err := strconv.Atoi(part); err == nil {
				channels = append(channels, part)
			}
			continue
		}
		start, startErr := strconv.Atoi(strings.TrimSpace(low))
		end, endErr := strconv.Atoi(strings.TrimSpace(high))
		if startErr != nil || endErr != nil || start > end || end-start > 200 {
			continue
		}
		for channel := start; channel <= end; channel++ {
			channels = append(channels, strconv.Itoa(channel))
		}
	}
	return channels
}

type channelWidthSpec struct {
	key     string
	codes   map[string]string
	codes2g map[string]string
}

var channelWidthSpecs = []channelWidthSpec{
	{key: "X_HW_HT20", codes: map[string]string{"20": "1", "40": "2", "80": "3"}},
	{
		key:     "X_FH_ChannelWidth",
		codes:   map[string]string{"20": "0", "40": "1", "80": "3"},
		codes2g: map[string]string{"20": "0", "20/40": "2"},
	},
	{key: "X_ALU_COM_ChannelBandWidthExtend"},
	{key: "X_ZTE-COM_OperatingChannelBandwidth"},
	{key: "BandWidth"},
}

var widthOrder = []string{"20", "20/40", "40", "80"}

func validWidth(mhz string) bool {
	return slices.Contains(widthOrder, mhz)
}

func (s channelWidthSpec) table(band string) map[string]string {
	if band == "2" && s.codes2g != nil {
		return s.codes2g
	}
	return s.codes
}

func (s channelWidthSpec) raw(mhz, band string) (string, bool) {
	if !validWidth(mhz) {
		return "", false
	}
	codes := s.table(band)
	if codes == nil {
		if mhz == "20/40" {
			return "", false
		}
		return mhz + "MHz", true
	}
	raw, ok := codes[mhz]
	return raw, ok
}

func (s channelWidthSpec) mhz(raw, band string) string {
	raw = strings.TrimSpace(raw)
	codes := s.table(band)
	if codes == nil {
		raw = strings.TrimSpace(strings.TrimSuffix(raw, "MHz"))
		if validWidth(raw) {
			return raw
		}
		return ""
	}
	for mhz, code := range codes {
		if code == raw {
			return mhz
		}
	}
	return ""
}

func wlanChannelWidth(config map[string]any, band string) string {
	for _, spec := range channelWidthSpecs {
		if _, ok := lookup(config, spec.key).(map[string]any); ok {
			return spec.mhz(getValue(config, spec.key), band)
		}
	}
	return ""
}

// huaweiWidthLimit is the widest width (MHz) a HG8145V5 accepted per Standard when probed live;
// 160 MHz and 11ax were rejected, 11b/g/bg/a stay at 20 and the 11n family at 40.
var huaweiWidthLimit = map[string]int{
	"11b": 20, "11g": 20, "11bg": 20, "11a": 20,
	"11n": 40, "11gn": 40, "11bgn": 40,
	"11ac": 80,
}

func limitWidths(options []string, standard string) []string {
	limit, known := huaweiWidthLimit[standard]
	if !known {
		return options
	}
	var kept []string
	for _, option := range options {
		if mhz, err := strconv.Atoi(option); err == nil && mhz <= limit {
			kept = append(kept, option)
		}
	}
	return kept
}

func channelWidthOptions(config map[string]any, band, standard string) []string {
	var spec channelWidthSpec
	found := false
	for _, candidate := range channelWidthSpecs {
		if object, ok := lookup(config, candidate.key).(map[string]any); ok {
			writable, known := object["_writable"].(bool)
			if known && !writable {
				return nil
			}
			spec, found = candidate, true
			break
		}
	}
	if !found {
		return nil
	}

	codes := spec.table(band)
	var options []string
	for _, width := range widthOrder {
		if band == "2" && width == "80" {
			continue
		}
		if _, ok := codes[width]; codes != nil && !ok {
			continue
		}
		if codes == nil && width == "20/40" {
			continue
		}
		options = append(options, width)
	}
	if spec.key == "X_HW_HT20" {
		return limitWidths(options, standard)
	}
	return options
}

// standard is the Standard that will be in effect after the write (requested one, else current).
func channelWidthUpdate(path string, node map[string]any, mhz, standard string) ([]any, error) {
	mhz = strings.TrimSpace(mhz)
	band := wlanBand(node, parseChannelList(getValue(node, "PossibleChannels")))
	// Same list the UI offers, so the API cannot send widths the dropdown hides (e.g. 80 on 2.4 GHz).
	if !slices.Contains(channelWidthOptions(node, band, strings.TrimSpace(standard)), mhz) {
		return nil, fmt.Errorf("%w: channel_width %q is not supported on this device or band", ErrInvalidParameter, mhz)
	}
	for _, spec := range channelWidthSpecs {
		name, xsdType, found := resolveParameter(node, param(typeString, spec.key))
		if !found {
			continue
		}
		raw, ok := spec.raw(mhz, band)
		if !ok {
			return nil, fmt.Errorf("%w: channel_width %q is not supported on this device or band", ErrInvalidParameter, mhz)
		}
		value, err := normalizeValue("channel_width", xsdType, raw)
		if err != nil {
			return nil, err
		}
		return []any{path + "." + name, value, xsdType}, nil
	}
	return nil, fmt.Errorf("%w: channel_width", ErrParameterNotSupported)
}

func wlanBand(config map[string]any, channels []string) string {
	band := firstValue(config, []string{"X_HW_RFBand", "X_ZTE-COM_RFBand"})
	switch {
	case strings.HasPrefix(band, "5"):
		return "5"
	case strings.HasPrefix(band, "2"):
		return "2"
	}
	for _, channel := range channels {
		if number, err := strconv.Atoi(channel); err == nil && number > 14 {
			return "5"
		}
	}
	if len(channels) > 0 {
		return "2"
	}
	return ""
}

// huaweiStandards lists the Standard values a HG8145V5 accepted when written live, with the
// radio standards each one needs. Anything else (e.g. 11n on 5 GHz) is answered with
// fault 9002, so it is not offered. Values are further limited by X_HW_SupportedStandards.
var huaweiStandards = map[string][]struct {
	value string
	needs []string
}{
	"2": {
		{"11b", []string{"b"}}, {"11g", []string{"g"}}, {"11bg", []string{"b", "g"}},
		{"11n", []string{"n"}}, {"11gn", []string{"g", "n"}}, {"11bgn", []string{"b", "g", "n"}},
	},
	"5": {
		{"11a", []string{"a"}}, {"11ac", []string{"ac"}},
	},
}

func huaweiStandardOptions(config map[string]any, band string) []string {
	supported := strings.Split(getValue(config, "X_HW_SupportedStandards"), ",")
	for i := range supported {
		supported[i] = strings.TrimSpace(supported[i])
	}
	var options []string
	for _, candidate := range huaweiStandards[band] {
		ok := true
		for _, need := range candidate.needs {
			if !slices.Contains(supported, need) {
				ok = false
				break
			}
		}
		if ok {
			options = append(options, candidate.value)
		}
	}
	return options
}

// vendorStandards lists the Standard values each model accepted when written live and read back;
// others were rejected (9002/9003) or silently ignored (Nokia ac/ax). Keyed by a model-specific parameter.
var vendorStandards = []struct {
	key    string
	byBand map[string][]string
}{
	{"X_ZTE-COM_OperatingChannelBandwidth", map[string][]string{
		"2": {"b,g,n", "b,g", "n"},
		"5": {"a,n,ac", "ac"},
	}},
	{"X_ALU_COM_ChannelBandWidthExtend", map[string][]string{
		"2": {"b,g,n", "n"},
		"5": {"a,n,ac"},
	}},
}

func vendorStandardOptions(config map[string]any, band string) []string {
	for _, vendor := range vendorStandards {
		if _, ok := lookup(config, vendor.key).(map[string]any); ok {
			return vendor.byBand[band]
		}
	}
	return nil
}

func standardOptions(wlans []WLAN, bands []string, index int, known []string) []string {
	current := strings.TrimSpace(wlans[index].Standard)
	if current == "" {
		return nil
	}
	if len(known) > 0 {
		if !slices.Contains(known, current) {
			known = append([]string{current}, known...)
		}
		return known
	}

	options := []string{current}
	if bands[index] == "" {
		return options
	}
	seen := map[string]bool{current: true}
	for i, other := range wlans {
		value := strings.TrimSpace(other.Standard)
		if bands[i] != bands[index] || value == "" || seen[value] {
			continue
		}
		seen[value] = true
		options = append(options, value)
	}
	return options
}

func parseWLANs(device map[string]any) []WLAN {
	const wlanPath = "InternetGatewayDevice.LANDevice.1.WLANConfiguration"

	configurations, _ := lookup(device, wlanPath).(map[string]any)
	beaconOptions := beaconTypeOptions(device)
	wlans := make([]WLAN, 0, len(configurations))
	bands := make([]string, 0, len(configurations))
	for _, index := range numericKeys(configurations) {
		config, _ := configurations[index].(map[string]any)
		path := fmt.Sprintf("%s.%s", wlanPath, index)
		associated := parseWLANAssociatedDevices(config, path)
		channels := parseChannelList(getValue(config, "PossibleChannels"))
		band := wlanBand(config, channels)
		wlans = append(wlans, WLAN{
			Path:                            path,
			Index:                           index,
			SSID:                            getValue(config, "SSID"),
			Enable:                          getValue(config, "Enable"),
			Channel:                         wlanChannel(config),
			ChannelOptions:                  channels,
			ChannelWidth:                    wlanChannelWidth(config, band),
			ChannelWidthOptions:             channelWidthOptions(config, band, getValue(config, "Standard")),
			MaxClients:                      firstValue(config, wlanMaxClientKeys),
			AutoChannelEnable:               getValue(config, "AutoChannelEnable"),
			OperatingFrequencyBand:          getValue(config, "OperatingFrequencyBand"),
			Standard:                        getValue(config, "Standard"),
			TransmitPower:                   getValue(config, "TransmitPower"),
			BeaconType:                      getValue(config, "BeaconType"),
			BeaconTypeOptions:               beaconOptions,
			KeyPassphrase:                   firstValue(config, wlanPasswordKeys),
			PasswordPolicy:                  passwordPolicy(config),
			SSIDAdvertisementEnabled:        getValue(config, "SSIDAdvertisementEnabled"),
			WMMEnable:                       getValue(config, "WMMEnable"),
			AssociatedDeviceNumberOfEntries: getValue(config, "AssociatedDeviceNumberOfEntries"),
			AssociatedDevices:               associated,
		})
		bands = append(bands, band)
	}

	for i, index := range numericKeys(configurations) {
		config, _ := configurations[index].(map[string]any)
		known := vendorStandardOptions(config, bands[i])
		_, huawei := lookup(config, "X_HW_SupportedStandards").(map[string]any)
		if huawei {
			known = huaweiStandardOptions(config, bands[i])
		}
		wlans[i].StandardOptions = standardOptions(wlans, bands, i, known)
		if huawei && known != nil {
			wlans[i].ChannelWidthByStandard = make(map[string][]string, len(wlans[i].StandardOptions))
			for _, standard := range wlans[i].StandardOptions {
				wlans[i].ChannelWidthByStandard[standard] = channelWidthOptions(config, bands[i], standard)
			}
		}
	}

	return wlans
}

func parseWLANAssociatedDevices(
	config map[string]any,
	wlanPath string,
) []WLANAssociatedDevice {
	devices, _ := config["AssociatedDevice"].(map[string]any)
	associated := make([]WLANAssociatedDevice, 0, len(devices))
	for _, index := range numericKeys(devices) {
		device, _ := devices[index].(map[string]any)
		associated = append(associated, WLANAssociatedDevice{
			Path:                fmt.Sprintf("%s.AssociatedDevice.%s", wlanPath, index),
			Name:                firstValue(device, assocNameKeys),
			MACAddress:          firstValue(device, assocMACKeys),
			IPAddress:           firstValue(device, assocIPKeys),
			AuthenticationState: firstValue(device, assocAuthKeys),
			RSSI:                firstValue(device, assocRSSIKeys),
			TxRate:              firstValue(device, assocTxRateKeys),
			RxRate:              firstValue(device, assocRxRateKeys),
			Bandwidth:           firstValue(device, assocBandwidthKeys),
			Mode:                firstValue(device, assocModeKeys),
			Radio:               firstValue(device, assocRadioKeys),
			StayTime:            firstValue(device, assocStayTimeKeys),
			BytesReceived:       firstValue(device, assocBytesRxKeys),
			BytesSent:           firstValue(device, assocBytesTxKeys),
		})
	}

	return associated
}
