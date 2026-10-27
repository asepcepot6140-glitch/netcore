package simplequeue

import (
	"context"

	"github.com/asepcepot6140-glitch/netcore/internal/services/livestream"

	"github.com/go-routeros/routeros/v3"
)

// Live adalah perubahan tabel queue sejak kiriman sebelumnya; Queues nil bila tidak ada yang
// berubah, yang berarti sekadar tanda hidup.
type Live struct {
	Queues *livestream.Delta[Queue] `json:"queues,omitempty"`
}

// StreamLive membaca queue lewat satu sesi tetap ke router tiap livestream.FastInterval (rate dan
// bytes berubah tiap detik) lalu memanggil emit dengan baris yang berubah. Fungsi kembali nil saat
// ctx selesai atau umur stream habis.
func StreamLive(ctx context.Context, userID uint, mikrotikID uint64, emit func(Live) error) error {
	queues := livestream.NewDiffer(func(q Queue) string { return q.ID }, nil)

	return livestream.RunFast(ctx, userID, mikrotikID, func(client *routeros.Client, tick livestream.Tick) error {
		list, err := fetchQueues(client)
		if err != nil {
			return err
		}
		return emit(Live{Queues: queues.Diff(list, tick.Full)})
	})
}
