package genieacs

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/asepcepot6140-glitch/netcore/internal/models"
)

var (
	ErrInvalidParameter      = errors.New("invalid parameter")
	ErrParameterNotSupported = errors.New("parameter not supported by this device")
	ErrTargetNotFound        = errors.New("target object not found on device")
	ErrDeviceNotFound        = errors.New("device not found")
	// ErrDeviceRejected means the device answered the task with a TR-069 fault.
	ErrDeviceRejected = errors.New("device rejected the change")
)

const maxParameterValueLength = 256

// paramSpec maps an API field to TR-069 parameter names relative to the target object.
type paramSpec struct {
	// names are candidates (vendor extensions differ); the first one present on the device is used.
	names []string
	// xsdType is a fallback for when the device does not report _type.
	xsdType string
}

type targetSpec struct {
	// path restricts which object the client may write to.
	path *regexp.Regexp
	// fixedPath overrides the client-supplied path for singleton objects.
	fixedPath string
	fields    map[string]paramSpec
}

func param(xsdType string, names ...string) paramSpec {
	return paramSpec{names: names, xsdType: xsdType}
}

const (
	typeString   = "xsd:string"
	typeBoolean  = "xsd:boolean"
	typeUnsigned = "xsd:unsignedInt"
)

const dhcpObjectPath = "InternetGatewayDevice.LANDevice.1.LANHostConfigManagement"

// Only writable parameters are listed; read-only ones (status, MAC address, hosts) are intentionally absent.
var targetSpecs = map[string]targetSpec{
	"wan": {
		path: regexp.MustCompile(`^InternetGatewayDevice\.WANDevice\.\d+\.WANConnectionDevice\.\d+\.(WANIPConnection|WANPPPConnection)\.\d+$`),
		fields: map[string]paramSpec{
			"enable":      param(typeBoolean, "Enable"),
			"conn_name":   param(typeString, "Name"),
			"enable_vlan": param(typeBoolean, vlanEnableKeys...),
			"conn_type":   param(typeString, "ConnectionType"),
			"service":     param(typeString, serviceKeys...),
			"username":    param(typeString, "Username"),
			"password":    param(typeString, "Password"),
			"nat":         param(typeBoolean, "NATEnabled"),
			"vlan":        param(typeUnsigned, vlanKeys...),
		},
	},
	"lan/dhcp": {
		path:      regexp.MustCompile(`^` + regexp.QuoteMeta(dhcpObjectPath) + `$`),
		fixedPath: dhcpObjectPath,
		fields: map[string]paramSpec{
			"server_enable":      param(typeBoolean, "DHCPServerEnable"),
			"min_address":        param(typeString, "MinAddress"),
			"max_address":        param(typeString, "MaxAddress"),
			"subnet_mask":        param(typeString, "SubnetMask"),
			"ip_routers":         param(typeString, "IPRouters"),
			"dns_servers":        param(typeString, "DNSServers"),
			"domain_name":        param(typeString, "DomainName"),
			"lease_time":         param(typeUnsigned, "DHCPLeaseTime"),
			"reserved_addresses": param(typeString, "ReservedAddresses"),
		},
	},
	"lan/ports": {
		path: regexp.MustCompile(`^InternetGatewayDevice\.LANDevice\.1\.LANEthernetInterfaceConfig\.\d+$`),
		fields: map[string]paramSpec{
			"enable":       param(typeBoolean, "Enable"),
			"max_bit_rate": param(typeString, "MaxBitRate"),
			"duplex_mode":  param(typeString, "DuplexMode"),
			"l3_enable":    param(typeBoolean, "X_HW_L3Enable"),
		},
	},
	"wlan": {
		path: regexp.MustCompile(`^InternetGatewayDevice\.LANDevice\.1\.WLANConfiguration\.\d+$`),
		fields: map[string]paramSpec{
			"ssid":                       param(typeString, "SSID"),
			"enable":                     param(typeBoolean, "Enable"),
			"ssid_advertisement_enabled": param(typeBoolean, "SSIDAdvertisementEnabled"),
			"auto_channel_enable":        param(typeBoolean, "AutoChannelEnable"),
			"channel":                    param(typeUnsigned, "Channel"),
			"operating_frequency_band":   param(typeString, "OperatingFrequencyBand"),
			"standard":                   param(typeString, "Standard"),
			"transmit_power":             param(typeUnsigned, "TransmitPower"),
			"beacon_type":                param(typeString, "BeaconType"),
			"key_passphrase":             param(typeString, wlanPasswordKeys...),
			"wmm_enable":                 param(typeBoolean, "WMMEnable"),
			"max_clients":                param(typeUnsigned, wlanMaxClientKeys...),
		},
	},
}

// SetParameters writes the given fields to one object on the device.
// applied is false when GenieACS queued the task because the device was unreachable.
func SetParameters(
	ctx context.Context,
	item *models.GenieACS,
	deviceID string,
	target string,
	path string,
	values map[string]string,
) (applied bool, err error) {
	spec, ok := targetSpecs[target]
	if !ok {
		return false, fmt.Errorf("%w: unknown target", ErrInvalidParameter)
	}
	if spec.fixedPath != "" {
		path = spec.fixedPath
	}
	path = strings.TrimSpace(path)
	if !spec.path.MatchString(path) {
		return false, fmt.Errorf("%w: invalid object path", ErrInvalidParameter)
	}
	if len(values) == 0 {
		return false, fmt.Errorf("%w: no values provided", ErrInvalidParameter)
	}

	base, err := baseURL(item)
	if err != nil {
		return false, err
	}
	deviceID = strings.TrimSpace(deviceID)
	if deviceID == "" {
		return false, fmt.Errorf("device ID is empty")
	}

	node, err := fetchObject(ctx, base, deviceID, path)
	if err != nil {
		return false, err
	}

	fields := make([]string, 0, len(values))
	for field := range values {
		fields = append(fields, field)
	}
	sort.Strings(fields)

	updates := make([][]any, 0, len(fields))
	var portBind string
	var hasPortBind bool
	for _, field := range fields {
		// ZTE PortBinding only accepts a connection that already has a type, so it is written after the other fields.
		if target == "wan" && field == "lan_bind" && usesPortBinding(node) {
			portBind, hasPortBind = values[field], true
			continue
		}

		// LanBind is vendor specific (path list or per-port flags), so it has its own writer.
		if target == "wan" && field == "lan_bind" {
			bindUpdates, bindErr := lanBindUpdates(path, node, values[field])
			if bindErr != nil {
				return false, bindErr
			}
			updates = append(updates, bindUpdates...)
			continue
		}

		// Channel width is stored differently per vendor (text or numeric code), so it has its own writer.
		if target == "wlan" && field == "channel_width" {
			standard := values["standard"]
			if strings.TrimSpace(standard) == "" {
				standard = getValue(node, "Standard")
			}
			widthUpdate, err := channelWidthUpdate(path, node, values[field], standard)
			if err != nil {
				return false, err
			}
			updates = append(updates, widthUpdate)
			continue
		}

		fieldSpec, ok := spec.fields[field]
		if !ok {
			return false, fmt.Errorf("%w: %s cannot be edited", ErrInvalidParameter, field)
		}
		if target == "wan" && field == "conn_name" && nameLocked(node) {
			return false, fmt.Errorf("%w: %s cannot be edited on this device", ErrInvalidParameter, field)
		}

		name, xsdType, found := resolveParameter(node, fieldSpec)
		if !found {
			return false, fmt.Errorf("%w: %s", ErrParameterNotSupported, field)
		}
		if target == "wlan" && field == "key_passphrase" {
			if err := validatePassword(passwordPolicy(node), values[field]); err != nil {
				return false, err
			}
		}

		raw := values[field]
		// Some devices (e.g. ZTE F609/F660) report a flag as unsignedInt 0/1; send it as a number.
		if fieldSpec.xsdType == typeBoolean && xsdType != typeBoolean {
			raw, err = booleanAsNumber(field, raw)
			if err != nil {
				return false, err
			}
		}

		value, err := normalizeValue(field, xsdType, raw)
		if err != nil {
			return false, err
		}
		updates = append(updates, []any{path + "." + name, value, xsdType})
	}
	if len(updates) == 0 && !hasPortBind {
		return false, fmt.Errorf("%w: no changes to apply", ErrInvalidParameter)
	}

	// Reject before anything is written when the binding cannot succeed.
	if hasPortBind {
		connType := strings.TrimSpace(values["conn_type"])
		if connType == "" {
			connType = getValue(node, "ConnectionType")
		}
		if err := checkPortBindingReady(connType); err != nil {
			return false, err
		}
	}

	applied = true
	if len(updates) > 0 {
		applied, err = postTask(ctx, base, deviceID, map[string]any{
			"name":            "setParameterValues",
			"parameterValues": updates,
		})
		if err != nil || !applied {
			return applied, err
		}
	}

	if hasPortBind {
		bindUpdates, err := portBindingUpdates(ctx, base, deviceID, path, portBind)
		if err != nil {
			return false, err
		}
		if len(bindUpdates) > 0 {
			applied, err = postTask(ctx, base, deviceID, map[string]any{
				"name":            "setParameterValues",
				"parameterValues": bindUpdates,
			})
			if err != nil || !applied {
				return applied, err
			}
		}
	}

	if target == "wan" {
		refreshWAN(ctx, item, deviceID, path, hasPortBind)
	}
	return true, nil
}

// DeleteWANConnection removes one WAN IP/PPP connection instance from the device.
// applied is false when GenieACS queued the task because the device was unreachable.
func DeleteWANConnection(
	ctx context.Context,
	item *models.GenieACS,
	deviceID string,
	path string,
) (applied bool, err error) {
	path = strings.TrimSpace(path)
	if !targetSpecs["wan"].path.MatchString(path) {
		return false, fmt.Errorf("%w: invalid object path", ErrInvalidParameter)
	}

	base, err := baseURL(item)
	if err != nil {
		return false, err
	}
	deviceID = strings.TrimSpace(deviceID)
	if deviceID == "" {
		return false, fmt.Errorf("device ID is empty")
	}

	if _, err := fetchObject(ctx, base, deviceID, path); err != nil {
		return false, err
	}

	applied, err = postTask(ctx, base, deviceID, map[string]any{
		"name":       "deleteObject",
		"objectName": path,
	})
	if err != nil || !applied {
		return applied, err
	}

	// The connection is already gone; a failed cleanup must not turn the delete into an error.
	bindingRemoved, bindErr := removePortBindings(ctx, base, deviceID, path)
	if bindErr != nil {
		log.Printf("genieacs wan port binding cleanup error: %v", bindErr)
	}
	if cleanupErr := removeEmptyConnectionDevice(ctx, base, deviceID, path); cleanupErr != nil {
		log.Printf("genieacs wan cleanup error: %v", cleanupErr)
	}
	refreshWAN(ctx, item, deviceID, path, bindingRemoved)
	return true, nil
}

var connectionDevicePath = regexp.MustCompile(`^(InternetGatewayDevice\.WANDevice\.\d+)\.WANConnectionDevice\.\d+`)

// removeEmptyConnectionDevice deletes the parent WANConnectionDevice left empty by a delete.
// Only FiberHome, Huawei and Nokia create one per connection; ZTE shares WANConnectionDevice.1.
func removeEmptyConnectionDevice(
	ctx context.Context,
	base, deviceID, connectionPath string,
) error {
	match := connectionDevicePath.FindStringSubmatch(connectionPath)
	if match == nil {
		return nil
	}

	device, err := deviceIdentity(ctx, base, deviceID)
	if err != nil {
		return err
	}
	if !needsConnectionDevice(device) {
		return nil
	}

	// Another connection under the same WANConnectionDevice keeps it alive.
	parent := match[0]
	remaining, err := fetchObject(ctx, base, deviceID, parent)
	if err != nil && !errors.Is(err, ErrTargetNotFound) {
		return err
	}
	for _, kind := range []string{"WANIPConnection", "WANPPPConnection"} {
		if connections, _ := remaining[kind].(map[string]any); len(numericKeys(connections)) > 0 {
			return nil
		}
	}

	if remaining != nil {
		if _, err := postTask(ctx, base, deviceID, map[string]any{
			"name":       "deleteObject",
			"objectName": parent,
		}); err != nil {
			return err
		}
	}
	return nil
}

// deviceIdentity loads _deviceId so vendor checks can run on the same shape as the device list.
func deviceIdentity(ctx context.Context, base, deviceID string) (map[string]any, error) {
	identity, err := fetchObject(ctx, base, deviceID, "_deviceId")
	if err != nil {
		return nil, err
	}
	return map[string]any{"_id": deviceID, "_deviceId": identity}, nil
}

// AddWANConnection creates an empty IPoE or PPPoE connection instance on the device.
// applied is false when GenieACS queued the task because the device was unreachable.
func AddWANConnection(
	ctx context.Context,
	item *models.GenieACS,
	deviceID string,
	kind string,
) (applied bool, err error) {
	if kind != "WANIPConnection" && kind != "WANPPPConnection" {
		return false, fmt.Errorf("%w: kind must be WANIPConnection or WANPPPConnection", ErrInvalidParameter)
	}

	base, err := baseURL(item)
	if err != nil {
		return false, err
	}
	deviceID = strings.TrimSpace(deviceID)
	if deviceID == "" {
		return false, fmt.Errorf("device ID is empty")
	}

	const wanDevice = "InternetGatewayDevice.WANDevice.1"
	const connDevicesPath = wanDevice + ".WANConnectionDevice"

	device, err := deviceIdentity(ctx, base, deviceID)
	if err != nil {
		return false, err
	}

	// ZTE hosts every connection under WANConnectionDevice.1.
	if !needsConnectionDevice(device) {
		parent := connDevicesPath + ".1." + kind
		applied, err = postTask(ctx, base, deviceID, map[string]any{
			"name":       "addObject",
			"objectName": parent,
		})
		if err != nil || !applied {
			return applied, err
		}
		refreshWAN(ctx, item, deviceID, parent, false)
		return true, nil
	}

	// FiberHome, Huawei and Nokia need a fresh WANConnectionDevice first; the new connection goes under it.
	before, err := fetchObject(ctx, base, deviceID, connDevicesPath)
	if err != nil && !errors.Is(err, ErrTargetNotFound) {
		return false, err
	}
	known := make(map[string]bool)
	for _, index := range numericKeys(before) {
		known[index] = true
	}

	applied, err = postTask(ctx, base, deviceID, map[string]any{
		"name":       "addObject",
		"objectName": connDevicesPath,
	})
	if err != nil || !applied {
		// A queued first step cannot be followed up until the device reports the new instance.
		return false, err
	}

	newIndex, err := waitNewInstance(ctx, base, deviceID, connDevicesPath, known)
	if err != nil {
		return false, err
	}

	// Nokia only accepts a connection under a link whose Mode and VLAN are already set.
	if isNokia(device) {
		link := connDevicesPath + "." + newIndex + ".X_CT-COM_WANGponLinkConfig"
		applied, err = postTask(ctx, base, deviceID, map[string]any{
			"name": "setParameterValues",
			"parameterValues": [][]any{
				{link + ".Mode", nokiaLinkMode, typeUnsigned},
				{link + ".VLANIDMark", nokiaLinkVLAN, typeUnsigned},
			},
		})
		if err != nil || !applied {
			return false, err
		}
	}

	parent := connDevicesPath + "." + newIndex + "." + kind
	applied, err = postTask(ctx, base, deviceID, map[string]any{
		"name":       "addObject",
		"objectName": parent,
	})
	if err != nil || !applied {
		return applied, err
	}
	refreshWAN(ctx, item, deviceID, parent, false)
	return true, nil
}

// refreshWAN re-reads the WAN tree (and the ZTE PortBinding table when it changed) so the next read is current.
// A failure only delays fresh data, so the change itself stays successful.
func refreshWAN(ctx context.Context, item *models.GenieACS, deviceID, connectionPath string, withPortBinding bool) {
	root := "InternetGatewayDevice.WANDevice.1"
	if match := connectionDevicePath.FindStringSubmatch(connectionPath); match != nil {
		root = match[1]
	}
	objects := []string{root + ".WANConnectionDevice"}
	if withPortBinding {
		objects = append(objects, portBindingPath)
	}
	if _, err := summonObjects(ctx, item, deviceID, objects); err != nil {
		log.Printf("genieacs wan refresh error: %v", err)
	}
}

// Link values Nokia needs before a connection can be added under a new WANConnectionDevice.
const (
	nokiaLinkMode = 3
	nokiaLinkVLAN = 300
)

// needsConnectionDevice reports vendors that require a new WANConnectionDevice before adding a WAN connection.
func needsConnectionDevice(device map[string]any) bool {
	id, _ := device["_id"].(string)
	manufacturer := strings.ToLower(getString(device, "_deviceId._Manufacturer"))
	return isFiberHome(device) || isNokia(device) ||
		strings.Contains(manufacturer, "huawei") || strings.HasPrefix(strings.ToUpper(id), "00259E-")
}

// waitNewInstance polls the device tree until an index outside known appears under objectPath.
func waitNewInstance(
	ctx context.Context,
	base, deviceID, objectPath string,
	known map[string]bool,
) (string, error) {
	for attempt := 0; attempt < 20; attempt++ {
		node, err := fetchObject(ctx, base, deviceID, objectPath)
		if err != nil && !errors.Is(err, ErrTargetNotFound) {
			return "", err
		}
		for _, index := range numericKeys(node) {
			if !known[index] {
				return index, nil
			}
		}

		select {
		case <-ctx.Done():
			return "", ctx.Err()
		case <-time.After(500 * time.Millisecond):
		}
	}
	return "", errors.New("the device did not report the new WAN connection device in time")
}

func fetchObject(
	ctx context.Context,
	base string,
	deviceID string,
	path string,
) (map[string]any, error) {
	query, err := json.Marshal(map[string]string{"_id": deviceID})
	if err != nil {
		return nil, err
	}

	params := url.Values{}
	params.Set("query", string(query))
	params.Set("projection", path)

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
	if len(devices) == 0 {
		return nil, ErrTargetNotFound
	}

	node, ok := lookup(devices[0], path).(map[string]any)
	if !ok {
		return nil, ErrTargetNotFound
	}
	return node, nil
}

// resolveParameter picks the first candidate that exists on the device and its reported type.
func resolveParameter(node map[string]any, spec paramSpec) (name, xsdType string, found bool) {
	for _, candidate := range spec.names {
		object, ok := lookup(node, candidate).(map[string]any)
		if !ok {
			continue
		}
		if writable, known := object["_writable"].(bool); known && !writable {
			continue
		}
		xsdType = spec.xsdType
		if reported, _ := object["_type"].(string); reported != "" {
			xsdType = reported
		}
		return candidate, xsdType, true
	}
	return "", "", false
}

// booleanAsNumber converts "true"/"false" (or "1"/"0") to "1"/"0" for flags the device stores as numbers.
func booleanAsNumber(field, raw string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "true", "1":
		return "1", nil
	case "false", "0":
		return "0", nil
	}
	return "", fmt.Errorf("%w: %s must be true or false", ErrInvalidParameter, field)
}

func normalizeValue(field, xsdType, raw string) (any, error) {
	switch xsdType {
	case typeBoolean:
		switch strings.ToLower(strings.TrimSpace(raw)) {
		case "true", "1":
			return true, nil
		case "false", "0":
			return false, nil
		}
		return nil, fmt.Errorf("%w: %s must be true or false", ErrInvalidParameter, field)
	case typeUnsigned:
		number, err := strconv.ParseUint(strings.TrimSpace(raw), 10, 32)
		if err != nil {
			return nil, fmt.Errorf("%w: %s must be a non-negative whole number", ErrInvalidParameter, field)
		}
		return number, nil
	case "xsd:int":
		number, err := strconv.ParseInt(strings.TrimSpace(raw), 10, 32)
		if err != nil {
			return nil, fmt.Errorf("%w: %s must be a whole number", ErrInvalidParameter, field)
		}
		return number, nil
	}

	if len(raw) > maxParameterValueLength ||
		strings.IndexFunc(raw, unicode.IsControl) >= 0 {
		return nil, fmt.Errorf("%w: %s has an invalid value", ErrInvalidParameter, field)
	}
	return raw, nil
}

// rejectTaskOnFault looks for a fault GenieACS stored for the task we just created.
// A faulted task would be retried on every inform and block later tasks, so when the
// device rejected it, the task and fault are removed and the fault text is returned.
func rejectTaskOnFault(ctx context.Context, base, deviceID, taskID string) string {
	query, err := json.Marshal(map[string]string{"device": deviceID})
	if err != nil {
		return ""
	}
	params := url.Values{}
	params.Set("query", string(query))
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, base+"/faults/?"+params.Encode(), nil)
	if err != nil {
		return ""
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		return ""
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return ""
	}

	var faults []struct {
		ID      string `json:"_id"`
		Code    string `json:"code"`
		Message string `json:"message"`
		Detail  struct {
			SetParameterValuesFault []struct {
				ParameterName string `json:"parameterName"`
				FaultCode     string `json:"faultCode"`
				FaultString   string `json:"faultString"`
			} `json:"setParameterValuesFault"`
		} `json:"detail"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&faults); err != nil {
		return ""
	}

	for _, fault := range faults {
		if !strings.HasSuffix(fault.ID, ":task_"+taskID) {
			continue
		}
		detail := fault.Message
		if len(fault.Detail.SetParameterValuesFault) > 0 {
			parts := make([]string, 0, len(fault.Detail.SetParameterValuesFault))
			for _, item := range fault.Detail.SetParameterValuesFault {
				name := item.ParameterName
				if dot := strings.LastIndex(name, "."); dot >= 0 {
					name = name[dot+1:]
				}
				parts = append(parts, fmt.Sprintf("%s: %s", name, item.FaultString))
			}
			detail = strings.Join(parts, "; ")
		}
		if detail == "" {
			detail = "fault " + fault.Code
		}

		deleteResource(ctx, base+"/tasks/"+url.PathEscape(taskID))
		deleteResource(ctx, base+"/faults/"+strings.ReplaceAll(url.PathEscape(fault.ID), ":", "%3A"))
		return detail
	}
	return ""
}

func deleteResource(ctx context.Context, endpoint string) {
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, endpoint, nil)
	if err != nil {
		return
	}
	if resp, err := httpClient.Do(req); err == nil {
		resp.Body.Close()
	}
}

// postTask queues a task and asks the device to connect immediately.
func postTask(
	ctx context.Context,
	base string,
	deviceID string,
	task map[string]any,
) (bool, error) {
	payload, err := json.Marshal(task)
	if err != nil {
		return false, err
	}

	endpoint := fmt.Sprintf(
		"%s/devices/%s/tasks?connection_request",
		base,
		url.PathEscape(deviceID),
	)
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
		return false, fmt.Errorf("failed to reach GenieACS: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
	switch resp.StatusCode {
	case http.StatusOK, http.StatusCreated:
		return true, nil
	case http.StatusAccepted:
		// 202 is either an unreachable device or a device that answered with a fault.
		var created struct {
			ID string `json:"_id"`
		}
		if json.Unmarshal(body, &created) == nil && created.ID != "" {
			if detail := rejectTaskOnFault(ctx, base, deviceID, created.ID); detail != "" {
				return false, fmt.Errorf("%w: %s", ErrDeviceRejected, detail)
			}
		}
		return false, nil
	default:
		return false, fmt.Errorf(
			"GenieACS rejected the task, status %d: %s",
			resp.StatusCode,
			string(body),
		)
	}
}
