package pppoe

import (
	"context"

	"github.com/asepcepot6140-glitch/netcore/internal/services/livestream"

	"github.com/go-routeros/routeros/v3"
)

// Live adalah perubahan ketiga tabel PPPoE sejak kiriman sebelumnya; tabel yang tidak berubah nil.
// Live kosong berarti sekadar tanda hidup.
type Live struct {
	Secrets *livestream.Delta[Secret]            `json:"secrets,omitempty"`
	Offline *livestream.Delta[OfflineConnection] `json:"offline,omitempty"`
	Active  *livestream.Delta[ActiveConnection]  `json:"active,omitempty"`
}

// OfflineFromSecrets menurunkan daftar offline: secret yang namanya tidak ada di active.
// secrets berurutan terbaru dulu (hasil ListSecrets) dan hasilnya diurutkan seperti ListOfflineConnections.
func OfflineFromSecrets(secrets []Secret, active []ActiveConnection) []OfflineConnection {
	online := make(map[string]struct{}, len(active))
	for _, connection := range active {
		online[connection.Name] = struct{}{}
	}

	offline := make([]OfflineConnection, 0)
	// Dibalik ke urutan RouterOS karena OrderOfflineNewestFirst membaliknya lagi.
	for i := len(secrets) - 1; i >= 0; i-- {
		secret := secrets[i]
		if _, ok := online[secret.Name]; ok {
			continue
		}
		offline = append(offline, OfflineConnection(secret))
	}

	OrderOfflineNewestFirst(offline)

	return offline
}

// StreamLive membaca secret dan active lewat satu sesi tetap ke router tiap livestream.Interval
// lalu memanggil emit dengan tabel yang berubah. Fungsi kembali nil saat ctx selesai atau umur
// stream habis.
func StreamLive(ctx context.Context, userID uint, mikrotikID uint64, emit func(Live) error) error {
	secrets := livestream.NewDiffer(func(s Secret) string { return s.ID }, nil)
	offline := livestream.NewDiffer(func(o OfflineConnection) string { return o.ID }, nil)
	active := livestream.NewDiffer(
		func(a ActiveConnection) string { return a.ID },
		func(a ActiveConnection) ActiveConnection {
			a.Uptime = "" // berubah tiap detik; dikirim ulang hanya pada siklus full
			return a
		},
	)

	return livestream.Run(ctx, userID, mikrotikID, func(client *routeros.Client, full bool) error {
		secretList, err := fetchSecrets(client)
		if err != nil {
			return err
		}
		activeList, err := fetchActive(client)
		if err != nil {
			return err
		}

		return emit(Live{
			Secrets: secrets.Diff(secretList, full),
			Offline: offline.Diff(OfflineFromSecrets(secretList, activeList), full),
			Active:  active.Diff(activeList, full),
		})
	})
}
