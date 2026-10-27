package hotspot

import (
	"context"
	"time"

	"github.com/asepcepot6140-glitch/netcore/internal/services/livestream"

	"github.com/go-routeros/routeros/v3"
)

// RatedActive adalah sesi aktif beserta kecepatannya (bit per detik); nil sampai ada dua sampel.
type RatedActive struct {
	Active
	TxRate *int64 `json:"tx_rate"`
	RxRate *int64 `json:"rx_rate"`
}

// RatedHost adalah host beserta kecepatannya (bit per detik); nil sampai ada dua sampel.
type RatedHost struct {
	Host
	TxRate *int64 `json:"tx_rate"`
	RxRate *int64 `json:"rx_rate"`
}

// Live adalah perubahan tabel hotspot sejak kiriman sebelumnya; tabel yang tidak berubah nil.
// Live kosong berarti sekadar tanda hidup.
type Live struct {
	Users  *livestream.Delta[User]        `json:"users,omitempty"`
	Active *livestream.Delta[RatedActive] `json:"active,omitempty"`
	Hosts  *livestream.Delta[RatedHost]   `json:"hosts,omitempty"`
}

// StreamLive membaca user, active, dan host lewat satu sesi tetap ke router. Active dan host (yang
// membawa rate) dibaca tiap livestream.FastInterval, user tiap livestream.Interval; emit dipanggil
// dengan tabel yang berubah. Fungsi kembali nil saat ctx selesai atau umur stream habis.
func StreamLive(ctx context.Context, userID uint, mikrotikID uint64, emit func(Live) error) error {
	users := livestream.NewDiffer(
		func(u User) string { return u.ID },
		func(u User) User {
			u.Uptime = "" // berubah tiap detik selama user online
			return u
		},
	)
	active := livestream.NewDiffer(
		func(a RatedActive) string { return a.ID },
		func(a RatedActive) RatedActive {
			a.Uptime, a.IdleTime, a.SessionTimeLeft = "", "", ""
			return a
		},
	)
	hosts := livestream.NewDiffer(
		func(h RatedHost) string { return h.ID },
		func(h RatedHost) RatedHost {
			h.Uptime, h.IdleTime = "", ""
			return h
		},
	)
	activeRates, hostRates := livestream.NewRateTracker(), livestream.NewRateTracker()

	return livestream.RunFast(ctx, userID, mikrotikID, func(client *routeros.Client, tick livestream.Tick) error {
		// Tabel dengan rate dibaca tiap siklus cepat; user (tanpa rate) hanya tiap siklus lambat.
		var userDelta *livestream.Delta[User]
		if tick.Slow {
			userList, err := fetchUsers(client)
			if err != nil {
				return err
			}
			userDelta = users.Diff(userList, tick.Full)
		}
		activeList, err := fetchActive(client)
		if err != nil {
			return err
		}
		hostList, err := fetchHosts(client)
		if err != nil {
			return err
		}

		now := time.Now()
		rated := make([]RatedActive, 0, len(activeList))
		for _, item := range activeList {
			tx, rx := activeRates.Observe(item.ID, livestream.ParseInt(item.BytesIn), livestream.ParseInt(item.BytesOut), now)
			rated = append(rated, RatedActive{Active: item, TxRate: tx, RxRate: rx})
		}
		activeRates.Commit()

		ratedHosts := make([]RatedHost, 0, len(hostList))
		for _, item := range hostList {
			tx, rx := hostRates.Observe(item.ID, livestream.ParseInt(item.BytesIn), livestream.ParseInt(item.BytesOut), now)
			ratedHosts = append(ratedHosts, RatedHost{Host: item, TxRate: tx, RxRate: rx})
		}
		hostRates.Commit()

		return emit(Live{
			Users:  userDelta,
			Active: active.Diff(rated, tick.Full),
			Hosts:  hosts.Diff(ratedHosts, tick.Full),
		})
	})
}
