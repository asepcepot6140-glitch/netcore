package iface

import (
	"log"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/asepcepot6140-glitch/netcore/internal/utils"

	"github.com/go-routeros/routeros/v3"
	"github.com/go-routeros/routeros/v3/proto"
)

const (
	interfacePath = "/interface"
	ethernetPath  = "/interface/ethernet"
	vlanPath      = "/interface/vlan"
	pppoePath     = "/interface/pppoe-server"
	pppoeServer   = "/interface/pppoe-server/server"

	interfaceProperties = ".id,name,type,running,disabled,dynamic,rx-byte,tx-byte,last-link-up-time,last-link-down-time"
	ethernetProperties  = "name,speed,auto-negotiation,full-duplex"
	vlanProperties      = "name,interface"
	pppoeProperties     = "" // RouterOS 6 menghilangkan "service" bila .proplist memuat properti yang tidak dikenalnya ("interface")
	pppoeServerProps    = "service-name,interface,invalid"

	// RouterOS menolak monitor-traffic untuk lebih dari 100 interface dalam satu perintah.
	monitorBatchSize = 100

	maxNameLength = 255

	linkTimeLayout = "2006-01-02 15:04:05"
)

type Interface struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Type     string `json:"type"`
	Running  bool   `json:"running"`
	Disabled bool   `json:"disabled"`
	Dynamic  bool   `json:"dynamic"`
	// Parent adalah nama interface induk (untuk VLAN); kosong bila interface berdiri sendiri.
	Parent string `json:"parent"`
	// Rate dalam bit per detik, Bytes dalam byte (sejak counter router di-reset).
	TxRate  int64 `json:"tx_rate"`
	RxRate  int64 `json:"rx_rate"`
	TxBytes int64 `json:"tx_bytes"`
	RxBytes int64 `json:"rx_bytes"`
	// Ethernet bernilai true bila interface punya pengaturan di /interface/ethernet;
	// hanya untuk interface itu Speed, AutoNegotiation, dan FullDuplex bermakna.
	Ethernet bool `json:"ethernet"`
	// Speed adalah pengaturan (dipakai saat auto-negotiation mati; kosong di RouterOS 7 bila belum
	// diatur), sedangkan Rate adalah kecepatan link sebenarnya hasil negosiasi ("1Gbps"); kosong bila no-link.
	Speed           string `json:"speed"`
	Rate            string `json:"rate"`
	AutoNegotiation bool   `json:"auto_negotiation"`
	FullDuplex      bool   `json:"full_duplex"`
	LastLinkUp      string `json:"last_link_up"`
	LastLinkDown    string `json:"last_link_down"`
}

// Input: nil = tidak diubah. Speed, AutoNegotiation, dan FullDuplex hanya berlaku untuk ethernet.
type Input struct {
	Name            *string `json:"name"`
	AutoNegotiation *bool   `json:"auto_negotiation"`
	Speed           *string `json:"speed"`
	FullDuplex      *bool   `json:"full_duplex"`
	Disabled        *bool   `json:"disabled"`
}

var speedPattern = regexp.MustCompile(`^\d+(?:\.\d+)?(?:Mbps|Gbps)$`)

func parseInt(value string) int64 {
	n, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return 0
	}
	return n
}

// FormatLinkTime menyeragamkan waktu link ("oct/26/2026 14:54:47" atau "2026-10-26 14:54:47",
// tergantung versi RouterOS) menjadi satu format; teks yang tidak dikenali dikembalikan apa adanya.
func FormatLinkTime(value string) string {
	value = strings.TrimSpace(value)
	if t, err := time.Parse(linkTimeLayout, value); err == nil {
		return t.Format(linkTimeLayout)
	}
	if t, ok := utils.ParseRouterOSTime(value); ok {
		return t.Format(linkTimeLayout)
	}
	return value
}

func FromRow(row map[string]string) Interface {
	return Interface{
		ID:           row[".id"],
		Name:         row["name"],
		Type:         row["type"],
		Running:      row["running"] == "true",
		Disabled:     row["disabled"] == "true",
		Dynamic:      row["dynamic"] == "true",
		TxBytes:      parseInt(row["tx-byte"]),
		RxBytes:      parseInt(row["rx-byte"]),
		LastLinkUp:   FormatLinkTime(row["last-link-up-time"]),
		LastLinkDown: FormatLinkTime(row["last-link-down-time"]),
	}
}

// ApplyEthernet mengisi data /interface/ethernet ke interface dengan nama yang sama.
func ApplyEthernet(items []Interface, rows []map[string]string) {
	byName := make(map[string]map[string]string, len(rows))
	for _, row := range rows {
		byName[row["name"]] = row
	}

	for i := range items {
		row, ok := byName[items[i].Name]
		if !ok {
			continue
		}
		items[i].Ethernet = true
		items[i].Speed = row["speed"]
		items[i].AutoNegotiation = row["auto-negotiation"] == "true"
		items[i].FullDuplex = row["full-duplex"] == "true"
	}
}

// ApplyVlan mengisi interface induk dari /interface/vlan ke interface dengan nama yang sama.
func ApplyVlan(items []Interface, rows []map[string]string) {
	parents := make(map[string]string, len(rows))
	for _, row := range rows {
		parents[row["name"]] = row["interface"]
	}

	for i := range items {
		// Interface tidak boleh menjadi induk bagi dirinya sendiri.
		if parent := parents[items[i].Name]; parent != items[i].Name {
			items[i].Parent = parent
		}
	}
}

// ApplyPPPoE mengisi induk untuk interface PPPoE dinamis (<pppoe-user>). RouterOS 7 memberi
// "interface" langsung; RouterOS 6 hanya memberi nama service, sehingga induknya dicari dari
// PPPoE server dengan service-name yang sama (dilewati bila ambigu atau server-nya invalid).
func ApplyPPPoE(items []Interface, sessions, servers []map[string]string) {
	known := make(map[string]struct{}, len(items))
	for _, item := range items {
		known[item.Name] = struct{}{}
	}

	byService := make(map[string]string, len(servers))
	for _, server := range servers {
		parent := server["interface"]
		if server["invalid"] == "true" || parent == "" {
			continue
		}
		name := server["service-name"]
		if current, ok := byService[name]; ok && current != parent {
			byService[name] = "" // dua server berbeda memakai service-name yang sama
			continue
		}
		byService[name] = parent
	}

	parents := make(map[string]string, len(sessions))
	for _, session := range sessions {
		parent := session["interface"]
		if parent == "" {
			parent = byService[session["service"]]
		}
		parents[session["name"]] = parent
	}

	for i := range items {
		if items[i].Parent != "" || !items[i].Dynamic || items[i].Type != "pppoe-in" {
			continue
		}
		if parent := parents[items[i].Name]; parent != items[i].Name {
			if _, ok := known[parent]; ok {
				items[i].Parent = parent
			}
		}
	}
}

// EthernetNames memilih interface ethernet untuk dicek kecepatan link-nya.
func EthernetNames(items []Interface) []string {
	names := make([]string, 0, len(items))
	for _, item := range items {
		if item.Ethernet && item.Name != "" && !strings.Contains(item.Name, ",") {
			names = append(names, item.Name)
		}
	}
	return names
}

// ApplyLinkRate mengisi kecepatan link dari hasil /interface/ethernet/monitor (dicocokkan lewat nama).
func ApplyLinkRate(items []Interface, rows []map[string]string) {
	byName := make(map[string]map[string]string, len(rows))
	for _, row := range rows {
		byName[row["name"]] = row
	}

	for i := range items {
		if row, ok := byName[items[i].Name]; ok {
			items[i].Rate = row["rate"]
		}
	}
}

// MonitorNames memilih interface yang perlu diukur trafiknya. Interface yang tidak running pasti
// 0 bps, dan nama berkoma tidak bisa dikirim karena daftar interface dipisah koma.
func MonitorNames(items []Interface) []string {
	names := make([]string, 0, len(items))
	for _, item := range items {
		if item.Running && item.Name != "" && !strings.Contains(item.Name, ",") {
			names = append(names, item.Name)
		}
	}
	return names
}

// ApplyTraffic mengisi rate dari hasil /interface/monitor-traffic (dicocokkan lewat nama).
func ApplyTraffic(items []Interface, rows []map[string]string) {
	byName := make(map[string]map[string]string, len(rows))
	for _, row := range rows {
		byName[row["name"]] = row
	}

	for i := range items {
		if row, ok := byName[items[i].Name]; ok {
			items[i].TxRate = parseInt(row["tx-bits-per-second"])
			items[i].RxRate = parseInt(row["rx-bits-per-second"])
		}
	}
}

// monitorTraffic mengukur rate sesaat per batch dan mengisinya ke items.
func monitorTraffic(client *routeros.Client, items []Interface) error {
	for batch := range slices.Chunk(MonitorNames(items), monitorBatchSize) {
		reply, err := client.Run(interfacePath+"/monitor-traffic", "=interface="+strings.Join(batch, ","), "=once=")
		if err != nil {
			return err
		}

		rows := make([]map[string]string, 0, len(reply.Re))
		for _, row := range reply.Re {
			rows = append(rows, row.Map)
		}
		ApplyTraffic(items, rows)
	}
	return nil
}

// monitorLinkRate membaca kecepatan link ethernet. Seperti monitorTraffic, kegagalan hanya
// membuat Rate kosong.
func monitorLinkRate(client *routeros.Client, items []Interface) {
	for batch := range slices.Chunk(EthernetNames(items), monitorBatchSize) {
		reply, err := client.Run(ethernetPath+"/monitor", "=numbers="+strings.Join(batch, ","), "=once=")
		if err != nil {
			log.Printf("interface ethernet monitor error: %v", err)
			return
		}

		rows := make([]map[string]string, 0, len(reply.Re))
		for _, row := range reply.Re {
			rows = append(rows, row.Map)
		}
		ApplyLinkRate(items, rows)
	}
}

// Detail satu baris hasil /interface/ethernet/monitor (urutan sesuai keluaran router).
type Detail struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

// DetailsFromRow mengubah satu baris monitor menjadi daftar berurutan; nilai kosong dibuang.
// "eeprom" (dump hex modul SFP) tidak berguna untuk dibaca sehingga tidak ditampilkan.
func DetailsFromRow(pairs []proto.Pair) []Detail {
	details := make([]Detail, 0, len(pairs))
	for _, pair := range pairs {
		if pair.Key == ".id" || pair.Key == "name" || pair.Key == "eeprom" || pair.Value == "" {
			continue
		}
		value := pair.Value
		// API memberi true/false, sedangkan CLI dan Winbox menampilkan yes/no.
		switch value {
		case "true":
			value = "yes"
		case "false":
			value = "no"
		}
		details = append(details, Detail{Key: pair.Key, Value: value})
	}
	return details
}

// generalDetailKeys adalah field "/interface print detail" yang ditampilkan (urutan sama dengan CLI).
var generalDetailKeys = []string{"mtu", "actual-mtu", "l2mtu", "max-l2mtu", "mac-address", "link-downs"}

// Monitor mengembalikan detail interface: field umum (mtu, mac, dll.), hasil ethernet monitor
// (status link, negosiasi, modul SFP) bila ethernet, lalu alamat IP yang terpasang.
func Monitor(userID uint, mikrotikID uint64, id string) ([]Detail, error) {
	id = strings.TrimSpace(id)
	if !ValidID(id) {
		return nil, ErrInvalidInput
	}

	client, err := connect(userID, mikrotikID)
	if err != nil {
		return nil, err
	}
	defer client.Close()

	// Tanpa .proplist: RouterOS 6 menjatuhkan properti sah bila daftar memuat properti yang tidak dikenal.
	current, err := client.Run(interfacePath+"/print", "?.id="+id)
	if err != nil {
		return nil, mapRouterError(err)
	}
	if len(current.Re) == 0 {
		return nil, ErrNotFound
	}
	row := current.Re[0].Map
	name := row["name"]

	details := []Detail{}
	for _, key := range generalDetailKeys {
		if value := row[key]; value != "" {
			details = append(details, Detail{Key: key, Value: value})
		}
	}

	// Koma memisahkan banyak nama pada =numbers=, jadi nama berkoma tidak bisa dicek dengan aman.
	if name != "" && !strings.Contains(name, ",") {
		ethernetDetails, err := ethernetMonitor(client, name)
		if err != nil {
			return nil, err
		}
		details = append(details, ethernetDetails...)
	}

	// Alamat IP bersifat tambahan; kegagalannya (mis. paket ip tidak ada) tidak menggagalkan Show.
	if name != "" {
		if addresses, err := client.Run("/ip/address/print", "?interface="+name, "=.proplist=address"); err == nil {
			for _, address := range addresses.Re {
				if value := address.Map["address"]; value != "" {
					details = append(details, Detail{Key: "address", Value: value})
				}
			}
		}
	}
	return details, nil
}

// ethernetMonitor mengembalikan hasil monitor ethernet; kosong bila interface bukan ethernet.
func ethernetMonitor(client *routeros.Client, name string) ([]Detail, error) {
	ethernet, err := client.Run(ethernetPath+"/print", "?name="+name, "=.proplist=.id")
	if err != nil {
		return nil, mapRouterError(err)
	}
	if len(ethernet.Re) == 0 {
		return nil, nil
	}
	reply, err := client.Run(ethernetPath+"/monitor", "=numbers="+name, "=once=")
	if err != nil {
		return nil, mapRouterError(err)
	}
	if len(reply.Re) == 0 {
		return nil, nil
	}
	return DetailsFromRow(reply.Re[0].List), nil
}

func hasType(items []Interface, typ string) bool {
	return slices.ContainsFunc(items, func(item Interface) bool { return item.Type == typ })
}

func runRows(client *routeros.Client, path, properties string) ([]map[string]string, error) {
	args := []string{path}
	if properties != "" {
		args = append(args, "=.proplist="+properties)
	}
	reply, err := client.Run(args...)
	if err != nil {
		return nil, err
	}
	rows := make([]map[string]string, 0, len(reply.Re))
	for _, row := range reply.Re {
		rows = append(rows, row.Map)
	}
	return rows, nil
}

func List(userID uint, mikrotikID uint64) ([]Interface, error) {
	client, err := connect(userID, mikrotikID)
	if err != nil {
		return nil, err
	}
	defer client.Close()

	reply, err := client.Run(interfacePath+"/print", "=.proplist="+interfaceProperties)
	if err != nil {
		return nil, err
	}
	items := make([]Interface, 0, len(reply.Re))
	for _, row := range reply.Re {
		items = append(items, FromRow(row.Map))
	}

	ethernet, err := client.Run(ethernetPath+"/print", "=.proplist="+ethernetProperties)
	if err != nil {
		return nil, err
	}
	rows := make([]map[string]string, 0, len(ethernet.Re))
	for _, row := range ethernet.Re {
		rows = append(rows, row.Map)
	}
	ApplyEthernet(items, rows)

	// Induk VLAN hanya pelengkap tampilan, jadi kegagalan di sini tidak menggagalkan daftar.
	if vlan, err := client.Run(vlanPath+"/print", "=.proplist="+vlanProperties); err != nil {
		log.Printf("interface: vlan print failed: %v", err)
	} else {
		vlanRows := make([]map[string]string, 0, len(vlan.Re))
		for _, row := range vlan.Re {
			vlanRows = append(vlanRows, row.Map)
		}
		ApplyVlan(items, vlanRows)
	}

	if hasType(items, "pppoe-in") {
		sessions, errSessions := runRows(client, pppoePath+"/print", pppoeProperties)
		servers, errServers := runRows(client, pppoeServer+"/print", pppoeServerProps)
		if errSessions != nil || errServers != nil {
			log.Printf("interface: pppoe parent lookup failed: %v %v", errSessions, errServers)
		} else {
			ApplyPPPoE(items, sessions, servers)
		}
	}

	monitorLinkRate(client, items)
	// Kegagalan rate tidak menggagalkan daftar: rate dibiarkan 0 supaya tabel tetap tampil.
	if err := monitorTraffic(client, items); err != nil {
		log.Printf("interface monitor-traffic error: %v", err)
	}
	return items, nil
}

// Normalize merapikan dan memvalidasi input; field nil dibiarkan.
func (in *Input) Normalize() error {
	if in.Name != nil {
		name := strings.TrimSpace(*in.Name)
		if name == "" {
			return &InputError{"Name tidak boleh kosong"}
		}
		if utf8.RuneCountInString(name) > maxNameLength {
			return &InputError{"Name terlalu panjang"}
		}
		in.Name = &name
	}

	if in.Speed != nil {
		speed := strings.Join(strings.Fields(*in.Speed), "")
		if speed == "" {
			in.Speed = nil
		} else if !speedPattern.MatchString(speed) {
			return &InputError{"Speed tidak valid. Gunakan contoh: 1Gbps"}
		} else {
			in.Speed = &speed
		}
	}

	if in.Name == nil && in.Disabled == nil && !in.hasEthernet() {
		return ErrInvalidInput
	}
	return nil
}

func (in *Input) hasEthernet() bool {
	return in.AutoNegotiation != nil || in.Speed != nil || in.FullDuplex != nil
}

// BuildInterfaceArgs menyusun /interface/set; nil bila tidak ada nama atau status yang diubah.
func BuildInterfaceArgs(id string, in Input) []string {
	if in.Name == nil && in.Disabled == nil {
		return nil
	}

	args := []string{interfacePath + "/set", "=.id=" + id}
	if in.Name != nil {
		args = append(args, "=name="+*in.Name)
	}
	if in.Disabled != nil {
		args = append(args, "=disabled="+strconv.FormatBool(*in.Disabled))
	}
	return args
}

// BuildEthernetArgs menyusun /interface/ethernet/set; nil bila tidak ada pengaturan ethernet.
func BuildEthernetArgs(id string, in Input) []string {
	if !in.hasEthernet() {
		return nil
	}

	args := []string{ethernetPath + "/set", "=.id=" + id}
	if in.AutoNegotiation != nil {
		args = append(args, "=auto-negotiation="+strconv.FormatBool(*in.AutoNegotiation))
	}
	if in.Speed != nil {
		args = append(args, "=speed="+*in.Speed)
	}
	if in.FullDuplex != nil {
		args = append(args, "=full-duplex="+strconv.FormatBool(*in.FullDuplex))
	}
	return args
}

func Update(userID uint, mikrotikID uint64, id string, input Input) error {
	id = strings.TrimSpace(id)
	if !ValidID(id) {
		return ErrInvalidInput
	}
	if err := input.Normalize(); err != nil {
		return err
	}

	client, err := connect(userID, mikrotikID)
	if err != nil {
		return err
	}
	defer client.Close()

	current, err := client.Run(interfacePath+"/print", "?.id="+id, "=.proplist=.id,name,dynamic")
	if err != nil {
		return mapRouterError(err)
	}
	if len(current.Re) == 0 {
		return ErrNotFound
	}
	if current.Re[0].Map["dynamic"] == "true" {
		return ErrDynamic
	}

	// ID ethernet dicari lebih dulu (berdasarkan nama lama) supaya gagal sebelum ada yang diubah.
	var ethernetArgs []string
	if input.hasEthernet() {
		reply, err := client.Run(ethernetPath+"/print", "?name="+current.Re[0].Map["name"], "=.proplist=.id")
		if err != nil {
			return mapRouterError(err)
		}
		if len(reply.Re) == 0 {
			return &InputError{"Interface ini tidak punya pengaturan ethernet"}
		}
		ethernetArgs = BuildEthernetArgs(reply.Re[0].Map[".id"], input)
	}

	// Nama dan status diubah lebih dulu: nama bentrok langsung ditolak tanpa mengubah ethernet.
	if args := BuildInterfaceArgs(id, input); args != nil {
		if _, err := client.Run(args...); err != nil {
			return mapRouterError(err)
		}
	}
	if ethernetArgs != nil {
		if _, err := client.Run(ethernetArgs...); err != nil {
			return mapRouterError(err)
		}
	}
	return nil
}
