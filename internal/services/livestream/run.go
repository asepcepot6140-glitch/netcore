package livestream

import (
	"context"
	"time"

	"github.com/asepcepot6140-glitch/netcore/internal/services/mikrotikconn"

	"github.com/go-routeros/routeros/v3"
)

const (
	// Interval adalah jeda antar pembacaan router (dan antar kiriman ke browser).
	Interval = 3 * time.Second
	// FastInterval adalah jeda untuk tabel yang berisi rate (RunFast); tabel lain tetap per Interval.
	FastInterval = time.Second
	// RefreshEvery: kolom yang berubah tiap detik (uptime, idle) hanya dikirim ulang tiap sekian
	// siklus, supaya semua baris tidak terkirim tiap siklus.
	RefreshEvery = 2
	// MaxAge: stream ditutup berkala supaya token login diperiksa ulang; browser langsung menyambung.
	MaxAge = 60 * time.Second
	// Batas satu pembacaan; sesi tanpa deadline bisa menggantung bila router berhenti menjawab.
	pollTimeout = 10 * time.Second
)

// Run membuka satu sesi tetap ke router (tanpa login ulang tiap siklus) lalu memanggil poll
// segera dan tiap Interval. full bernilai true tiap RefreshEvery siklus, termasuk siklus pertama.
// Fungsi kembali nil saat ctx selesai atau MaxAge habis; poll yang mengembalikan error
// menghentikan stream.
func Run(ctx context.Context, userID uint, mikrotikID uint64, poll func(client *routeros.Client, full bool) error) error {
	return run(ctx, userID, mikrotikID, Interval, func(client *routeros.Client, tick Tick) error {
		return poll(client, tick.Full)
	})
}

// Tick menjelaskan satu siklus RunFast.
type Tick struct {
	// Slow bernilai true sekali tiap Interval; hanya pada siklus ini tabel yang lambat berubah dibaca.
	Slow bool
	// Full bernilai true tiap RefreshEvery siklus Slow, termasuk siklus pertama.
	Full bool
}

// RunFast seperti Run, tetapi poll dipanggil tiap FastInterval. Tabel dengan rate dibaca di setiap
// siklus, sedangkan tabel lain hanya saat tick.Slow. Siklus yang terlambat karena router lambat
// dilewati, jadi pembacaan tidak menumpuk.
func RunFast(ctx context.Context, userID uint, mikrotikID uint64, poll func(client *routeros.Client, tick Tick) error) error {
	return run(ctx, userID, mikrotikID, FastInterval, poll)
}

func run(ctx context.Context, userID uint, mikrotikID uint64, interval time.Duration, poll func(client *routeros.Client, tick Tick) error) error {
	release, err := AcquireStream(userID, mikrotikID)
	if err != nil {
		return err
	}
	defer release()

	client, err := mikrotikconn.ConnectStream(userID, mikrotikID)
	if err != nil {
		return err
	}
	defer client.Close()

	// Client yang sedang menunggu jawaban router tidak melihat ctx; menutupnya membatalkan Run.
	stop := context.AfterFunc(ctx, func() { _ = client.Close() })
	defer stop()

	// Tabel lambat dibaca sekali tiap Interval walau siklus cepat ada yang terlewat.
	var lastSlow time.Time
	slowCycles := 0
	guarded := func(now time.Time) error {
		tick := Tick{}
		if lastSlow.IsZero() || now.Sub(lastSlow) >= Interval-interval/2 {
			lastSlow = now
			tick = Tick{Slow: true, Full: slowCycles%RefreshEvery == 0}
			slowCycles++
		}

		guard := time.AfterFunc(pollTimeout, func() { _ = client.Close() })
		defer guard.Stop()
		err := poll(client, tick)
		if err != nil && ctx.Err() != nil {
			return nil
		}
		return err
	}

	if err := guarded(time.Now()); err != nil {
		return err
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	expiry := time.NewTimer(MaxAge)
	defer expiry.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-expiry.C:
			return nil
		case at := <-ticker.C:
			// Tick yang menunggu karena pembacaan sebelumnya lambat sudah basi; lewati supaya
			// router tidak menerima pembacaan beruntun tanpa jeda.
			if time.Since(at) > interval/10 {
				continue
			}
			if err := guarded(at); err != nil {
				return err
			}
			if ctx.Err() != nil {
				return nil
			}
		}
	}
}
