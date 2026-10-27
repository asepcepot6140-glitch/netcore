package genieacs

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/asepcepot6140-glitch/netcore/internal/models"
)

const (
	lanDevicePath = "InternetGatewayDevice.LANDevice.1"
	staticDNSPath = "InternetGatewayDevice.X_ALU-COM_AppCfg.DnsProxyCfg.X_ALU-COM_STATIC_DNS"
)

var (
	portSpeedKeys  = []string{"X_FH_LinkSpeed", "X_ALU_COM_CurMaxBitRate", "MaxBitRate"}
	portDuplexKeys = []string{"X_ALU_COM_CurDuplexMode", "DuplexMode"}
	portPVIDKeys   = []string{"X_ALU-COM_dot1qPvid"}
)

const lanProjection = "_id," + lanDevicePath + ",InternetGatewayDevice.X_ALU-COM_AppCfg.DnsProxyCfg"

type LAN struct {
	IPInterface IPInterface `json:"ip_interface"`
	DHCP        DHCPConfig  `json:"dhcp"`
	Ports       []LANPort   `json:"ports"`
	Hosts       []LANHost   `json:"hosts"`
	StaticDNS   []StaticDNS `json:"static_dns"`
}

type IPInterface struct {
	Path       string `json:"path"`
	IPAddress  string `json:"ip_address"`
	SubnetMask string `json:"subnet_mask"`
}

type DHCPConfig struct {
	ServerEnable string `json:"server_enable"`
	MinAddress   string `json:"min_address"`
	MaxAddress   string `json:"max_address"`
	SubnetMask   string `json:"subnet_mask"`
	IPRouters    string `json:"ip_routers"`
	DNSServers   string `json:"dns_servers"`
}

type LANPort struct {
	Path           string `json:"path"`
	Name           string `json:"name"`
	Enable         string `json:"enable"`
	Status         string `json:"status"`
	LinkSpeed      string `json:"link_speed"`
	DuplexMode     string `json:"duplex_mode"`
	PVID           string `json:"pvid"`
	MacLimitNumber string `json:"mac_limit_number"`
	L3Enable       string `json:"l3_enable"`
}

type LANHost struct {
	Path               string `json:"path"`
	IPAddress          string `json:"ip_address"`
	MACAddress         string `json:"mac_address"`
	HostName           string `json:"host_name"`
	AddressSource      string `json:"address_source"`
	InterfaceType      string `json:"interface_type"`
	Layer2Interface    string `json:"layer2_interface"`
	Active             string `json:"active"`
	LeaseTimeRemaining string `json:"lease_time_remaining"`
}

type StaticDNS struct {
	Path       string `json:"path"`
	DomainName string `json:"domain_name"`
	DomainAddr string `json:"domain_addr"`
}

func GetLAN(ctx context.Context, item *models.GenieACS, deviceID string) (*LAN, error) {
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
	params.Set("projection", lanProjection)

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
		return &LAN{
			Ports:     make([]LANPort, 0),
			Hosts:     make([]LANHost, 0),
			StaticDNS: make([]StaticDNS, 0),
		}, nil
	}

	return parseLAN(devices[0]), nil
}

func parseLAN(device map[string]any) *LAN {
	lanDevice, _ := lookup(device, lanDevicePath).(map[string]any)
	config, _ := lanDevice["LANHostConfigManagement"].(map[string]any)

	lan := &LAN{
		IPInterface: parseIPInterface(config),
		DHCP: DHCPConfig{
			ServerEnable: getValue(config, "DHCPServerEnable"),
			MinAddress:   getValue(config, "MinAddress"),
			MaxAddress:   getValue(config, "MaxAddress"),
			SubnetMask:   getValue(config, "SubnetMask"),
			IPRouters:    getValue(config, "IPRouters"),
			DNSServers:   getValue(config, "DNSServers"),
		},
		Ports:     make([]LANPort, 0),
		Hosts:     make([]LANHost, 0),
		StaticDNS: parseStaticDNS(device),
	}

	ports, _ := lanDevice["LANEthernetInterfaceConfig"].(map[string]any)
	for _, index := range numericKeys(ports) {
		port, _ := ports[index].(map[string]any)
		path := fmt.Sprintf("%s.LANEthernetInterfaceConfig.%s", lanDevicePath, index)
		lan.Ports = append(lan.Ports, LANPort{
			Path:           path,
			Name:           getValue(port, "Name"),
			Enable:         getValue(port, "Enable"),
			Status:         getValue(port, "Status"),
			LinkSpeed:      firstValue(port, portSpeedKeys),
			DuplexMode:     firstValue(port, portDuplexKeys),
			PVID:           firstValue(port, portPVIDKeys),
			MacLimitNumber: getValue(port, "X_FH_MacLimitNumber"),
			L3Enable:       getValue(port, "X_HW_L3Enable"),
		})
	}

	hostsRoot, _ := lanDevice["Hosts"].(map[string]any)
	hosts, _ := hostsRoot["Host"].(map[string]any)
	for _, index := range numericKeys(hosts) {
		host, _ := hosts[index].(map[string]any)
		path := fmt.Sprintf("%s.Hosts.Host.%s", lanDevicePath, index)
		lan.Hosts = append(lan.Hosts, LANHost{
			Path:               path,
			IPAddress:          getValue(host, "IPAddress"),
			MACAddress:         getValue(host, "MACAddress"),
			HostName:           getValue(host, "HostName"),
			AddressSource:      getValue(host, "AddressSource"),
			InterfaceType:      getValue(host, "InterfaceType"),
			Layer2Interface:    getValue(host, "Layer2Interface"),
			Active:             getValue(host, "Active"),
			LeaseTimeRemaining: getValue(host, "LeaseTimeRemaining"),
		})
	}

	return lan
}

func parseIPInterface(config map[string]any) IPInterface {
	interfaces, _ := config["IPInterface"].(map[string]any)
	for _, index := range numericKeys(interfaces) {
		iface, _ := interfaces[index].(map[string]any)
		return IPInterface{
			Path:       fmt.Sprintf("%s.LANHostConfigManagement.IPInterface.%s", lanDevicePath, index),
			IPAddress:  getValue(iface, "IPInterfaceIPAddress"),
			SubnetMask: getValue(iface, "IPInterfaceSubnetMask"),
		}
	}
	return IPInterface{}
}

func parseStaticDNS(device map[string]any) []StaticDNS {
	entries := make([]StaticDNS, 0)

	root, _ := lookup(device, staticDNSPath).(map[string]any)
	dnsInfo, _ := root["DnsInfo"].(map[string]any)
	for _, index := range numericKeys(dnsInfo) {
		info, _ := dnsInfo[index].(map[string]any)
		entries = append(entries, StaticDNS{
			Path:       fmt.Sprintf("%s.DnsInfo.%s", staticDNSPath, index),
			DomainName: getValue(info, "domainName"),
			DomainAddr: getValue(info, "domainAddr"),
		})
	}

	return entries
}
