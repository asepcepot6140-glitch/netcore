package iface

import (
	"context"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/asepcepot6140-glitch/netcore/internal/services/livestream"
	"github.com/asepcepot6140-glitch/netcore/internal/services/mikrotikconn"

	"github.com/go-routeros/routeros/v3"
)

const (
	// streamInterval adalah jeda antar kiriman ke browser; router sendiri mengirim rate tiap detik.
	streamInterval = time.Second
	// Status (running, disabled, bytes, kecepatan link) dibaca tiap statusEveryTicks kiriman.
	statusEveryTicks = 2
	// Batas satu pembacaan status; sesi tanpa deadline bisa menggantung bila router berhenti menjawab.
	statusCallTimeout = 10 * time.Second
	// Perubahan daftar interface baru ditanggapi setelah stream berumur segini, supaya interface
	// yang naik-turun terus tidak membuat browser menyambung ulang tanpa henti.
	streamMinAge = 10 * time.Second
	// Stream ditutup berkala supaya daftar interface (mis. sesi PPPoE baru) dan token login
	// diperiksa ulang; browser langsung menyambung lagi.
	streamMaxAge = 60 * time.Second
	// Router yang diam sepanjang ini dianggap macet.
	streamStallAfter = 15 * time.Second
	// Batas menyiapkan sesi (login, daftar interface, memulai stream).
	streamSetupTimeout = 20 * time.Second

	maxStreamNames = 500
)

// Rate adalah trafik sesaat satu interface dalam bit per detik.
type Rate struct {
	Name   string `json:"name"`
	TxRate int64  `json:"tx_rate"`
	RxRate int64  `json:"rx_rate"`
}

// RateFromRow membaca satu baris monitor-traffic; false bila baris tidak punya nama.
func RateFromRow(row map[string]string) (Rate, bool) {
	name := row["name"]
	if name == "" {
		return Rate{}, false
	}
	return Rate{
		Name:   name,
		TxRate: parseInt(row["tx-bits-per-second"]),
		RxRate: parseInt(row["rx-bits-per-second"]),
	}, true
}

// Status adalah keadaan satu interface yang berubah lebih lambat dari rate. Rate adalah kecepatan
// link ("1Gbps") dan hanya bermakna untuk interface ethernet; kosong bila tidak ada link.
type Status struct {
	Name     string `json:"name"`
	Running  bool   `json:"running"`
	Disabled bool   `json:"disabled"`
	TxBytes  int64  `json:"tx_bytes"`
	RxBytes  int64  `json:"rx_bytes"`
	Rate     string `json:"rate"`
}

// Live adalah satu kiriman stream; slice kosong berarti tidak ada yang berubah. Changed berarti
// daftar interface di router berubah (ada yang baru, hilang, atau mulai running): stream ditutup
// setelah kiriman ini dan browser perlu memuat ulang daftar lalu menyambung lagi.
type Live struct {
	Rates   []Rate   `json:"rates"`
	Status  []Status `json:"status"`
	Changed bool     `json:"changed"`
}

// NamesChanged memberi tahu apakah stream perlu disambung ulang: ada interface baru atau hilang
// (dibanding all), atau ada interface running yang belum dipantau rate-nya (monitored). Interface
// yang berhenti running tidak dihitung karena stream-nya tetap valid.
func NamesChanged(all, monitored []string, statuses []Status) bool {
	if len(all) != len(statuses) {
		return true
	}
	allSet := make(map[string]struct{}, len(all))
	for _, name := range all {
		allSet[name] = struct{}{}
	}
	monitoredSet := make(map[string]struct{}, len(monitored))
	for _, name := range monitored {
		monitoredSet[name] = struct{}{}
	}

	// Di atas batas, sebagian interface memang tidak dipantau, jadi tidak bisa dibandingkan.
	full := len(monitored) >= maxStreamNames
	for _, status := range statuses {
		if _, ok := allSet[status.Name]; !ok {
			return true
		}
		if full || !status.Running || status.Name == "" || strings.Contains(status.Name, ",") {
			continue
		}
		if _, ok := monitoredSet[status.Name]; !ok {
			return true
		}
	}
	return false
}

// StatusFromRow membaca satu baris /interface/print; false bila baris tidak punya nama.
func StatusFromRow(row map[string]string) (Status, bool) {
	name := row["name"]
	if name == "" {
		return Status{}, false
	}
	return Status{
		Name:     name,
		Running:  row["running"] == "true",
		Disabled: row["disabled"] == "true",
		TxBytes:  parseInt(row["tx-byte"]),
		RxBytes:  parseInt(row["rx-byte"]),
	}, true
}

// pollStatus membaca status semua interface lewat sesi yang sudah terbuka. Kecepatan link ethernet
// bersifat pelengkap: kegagalannya hanya mengosongkan Rate, sedangkan kegagalan sesi sudah
// ketahuan dari pembacaan pertama.
func pollStatus(client *routeros.Client, ethernet []string) ([]Status, error) {
	reply, err := client.Run(interfacePath+"/print", "=.proplist=name,running,disabled,rx-byte,tx-byte")
	if err != nil {
		return nil, err
	}

	statuses := make([]Status, 0, len(reply.Re))
	index := make(map[string]int, len(reply.Re))
	for _, row := range reply.Re {
		if status, ok := StatusFromRow(row.Map); ok {
			index[status.Name] = len(statuses)
			statuses = append(statuses, status)
		}
	}

	for batch := range slices.Chunk(ethernet, monitorBatchSize) {
		monitor, err := client.Run(ethernetPath+"/monitor", "=numbers="+strings.Join(batch, ","), "=once=", "=.proplist=name,rate")
		if err != nil {
			break
		}
		for _, row := range monitor.Re {
			if i, ok := index[row.Map["name"]]; ok {
				statuses[i].Rate = row.Map["rate"]
			}
		}
	}
	return statuses, nil
}

// StreamLive mengalirkan data interface lewat satu sesi tetap ke router (tanpa login ulang):
// rate dari monitor-traffic mode stream, dan status (running, disabled, bytes, kecepatan link)
// dibaca tiap statusEveryTicks kiriman. emit dipanggil sekali per detik hanya dengan data yang
// berubah; Live kosong berarti sekadar tanda hidup. Fungsi kembali nil saat ctx selesai atau
// umur stream habis.
func StreamLive(ctx context.Context, userID uint, mikrotikID uint64, emit func(Live) error) error {
	release, err := livestream.AcquireStream(userID, mikrotikID)
	if err != nil {
		return err
	}
	defer release()

	client, err := mikrotikconn.ConnectStream(userID, mikrotikID)
	if err != nil {
		return err
	}

	// Sesi stream tidak punya deadline, jadi persiapannya dijaga watchdog.
	setup := time.AfterFunc(streamSetupTimeout, func() { _ = client.Close() })
	defer setup.Stop()

	reply, err := client.Run(interfacePath+"/print", "=.proplist=name,running")
	if err != nil {
		_ = client.Close()
		return mapRouterError(err)
	}
	items := make([]Interface, 0, len(reply.Re))
	known := make([]string, 0, len(reply.Re))
	for _, row := range reply.Re {
		items = append(items, Interface{Name: row.Map["name"], Running: row.Map["running"] == "true"})
		if row.Map["name"] != "" {
			known = append(known, row.Map["name"])
		}
	}
	// Nama ethernet hanya untuk kecepatan link; kegagalannya cukup membuat Rate kosong.
	var ethernet []string
	if reply, err := client.Run(ethernetPath+"/print", "=.proplist=name"); err == nil {
		for _, row := range reply.Re {
			if name := row.Map["name"]; name != "" && !strings.Contains(name, ",") {
				ethernet = append(ethernet, name)
			}
		}
	}
	names := MonitorNames(items)
	if len(names) > maxStreamNames {
		names = names[:maxStreamNames]
	}

	var (
		mu       sync.Mutex
		pending  = map[string]Rate{}
		sent     = map[string]Rate{}
		lastSeen = map[string]Status{}
		lastRow  atomic.Int64
		readers  sync.WaitGroup
		failed   = make(chan error, 1)
		listens  []*routeros.ListenReply
		closeAll = func() {
			// Menutup sesi mengakhiri semua listen; reader selesai setelah channel-nya tertutup.
			_ = client.Close()
			readers.Wait()
		}
	)
	lastRow.Store(time.Now().UnixNano())

	for batch := range slices.Chunk(names, monitorBatchSize) {
		listen, err := client.Listen(interfacePath+"/monitor-traffic", "=interface="+strings.Join(batch, ","))
		if err != nil {
			closeAll()
			return mapRouterError(err)
		}
		listens = append(listens, listen)

		readers.Add(1)
		go func() {
			defer readers.Done()
			// Channel wajib selalu dikuras, kalau tidak library menggantung saat sesi ditutup.
			for sentence := range listen.Chan() {
				rate, ok := RateFromRow(sentence.Map)
				if !ok {
					continue
				}
				lastRow.Store(time.Now().UnixNano())
				mu.Lock()
				pending[rate.Name] = rate
				mu.Unlock()
			}
			// Channel tertutup sebelum kita menutup sesi = router memutus atau menolak perintah.
			select {
			case failed <- listen.Err():
			default:
			}
		}()
	}
	setup.Stop()
	defer closeAll()

	if err := emit(Live{}); err != nil {
		return err
	}

	ticker := time.NewTicker(streamInterval)
	defer ticker.Stop()
	expiry := time.NewTimer(streamMaxAge)
	defer expiry.Stop()
	tick := 0
	started := time.Now()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-expiry.C:
			return nil
		case err := <-failed:
			if err == nil {
				return nil
			}
			return mapRouterError(err)
		case <-ticker.C:
			if len(listens) > 0 && time.Since(time.Unix(0, lastRow.Load())) > streamStallAfter {
				return ErrStreamStalled
			}

			mu.Lock()
			rates := make([]Rate, 0, len(pending))
			for name, rate := range pending {
				// Router mengirim semua interface tiap detik; kirim hanya yang berubah.
				if prev, seen := sent[name]; !seen || prev != rate {
					rates = append(rates, rate)
					sent[name] = rate
				}
			}
			clear(pending)
			mu.Unlock()

			update := Live{Rates: rates}
			if tick%statusEveryTicks == 0 {
				// Sesi tanpa deadline: watchdog menutupnya bila router berhenti menjawab.
				guard := time.AfterFunc(statusCallTimeout, func() { _ = client.Close() })
				statuses, err := pollStatus(client, ethernet)
				guard.Stop()
				if err != nil {
					return mapRouterError(err)
				}
				if time.Since(started) >= streamMinAge && NamesChanged(known, names, statuses) {
					return emit(Live{Changed: true})
				}
				for _, status := range statuses {
					if prev, seen := lastSeen[status.Name]; !seen || prev != status {
						update.Status = append(update.Status, status)
						lastSeen[status.Name] = status
					}
				}
			}
			tick++

			if err := emit(update); err != nil {
				return err
			}
		}
	}
}
