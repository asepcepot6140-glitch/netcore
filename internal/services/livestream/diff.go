package livestream

import (
	"reflect"
	"slices"
	"strconv"
	"time"
)

// Delta adalah perubahan satu tabel sejak kiriman sebelumnya. Upsert memuat baris utuh yang baru
// atau berubah. Order memuat seluruh ID dengan urutan tampil, hanya bila daftar atau urutannya
// berubah (nil = tidak berubah); ID yang tidak ada di Order berarti baris sudah hilang.
type Delta[T any] struct {
	Upsert []T      `json:"upsert"`
	Order  []string `json:"order"`
}

// Differ mengingat apa yang sudah dikirim untuk satu tabel dan hanya mengembalikan selisihnya.
// Kiriman pertama selalu berisi semua baris (snapshot).
type Differ[T any] struct {
	key    func(T) string
	stable func(T) T
	sent   map[string]T
	order  []string
	primed bool
}

// NewDiffer membuat Differ. key mengambil ID baris. stable (boleh nil) mengembalikan salinan
// baris dengan kolom yang berubah terus (uptime, idle) dikosongkan; kolom itu diabaikan saat
// membandingkan, kecuali pada siklus full.
func NewDiffer[T any](key func(T) string, stable func(T) T) *Differ[T] {
	return &Differ[T]{key: key, stable: stable, sent: map[string]T{}}
}

func (d *Differ[T]) same(previous, current T, full bool) bool {
	if !full && d.stable != nil {
		previous, current = d.stable(previous), d.stable(current)
	}
	return reflect.DeepEqual(previous, current)
}

// Diff membandingkan items (urutan tampil) dengan kiriman terakhir; nil bila tidak ada perubahan.
// Baris tanpa ID atau dengan ID ganda dilewati.
func (d *Differ[T]) Diff(items []T, full bool) *Delta[T] {
	order := make([]string, 0, len(items))
	upsert := make([]T, 0)
	next := make(map[string]T, len(items))

	for _, item := range items {
		id := d.key(item)
		if _, dup := next[id]; id == "" || dup {
			continue
		}
		order = append(order, id)

		previous, known := d.sent[id]
		if known && d.same(previous, item, full) {
			next[id] = previous
			continue
		}
		next[id] = item
		upsert = append(upsert, item)
	}

	orderChanged := !d.primed || !slices.Equal(order, d.order)
	d.sent, d.order, d.primed = next, order, true

	if len(upsert) == 0 && !orderChanged {
		return nil
	}
	delta := &Delta[T]{Upsert: upsert}
	if orderChanged {
		delta.Order = order
	}
	return delta
}

type counterSample struct {
	at      time.Time
	in, out int64
}

// RateTracker menghitung kecepatan (bit per detik) dari selisih counter byte antar siklus.
// Pakai Observe untuk tiap baris dalam satu siklus, lalu Commit di akhir siklus.
type RateTracker struct {
	last map[string]counterSample
	next map[string]counterSample
}

func NewRateTracker() *RateTracker {
	return &RateTracker{last: map[string]counterSample{}, next: map[string]counterSample{}}
}

// Observe mencatat counter id dan mengembalikan kecepatan keluar (tx, dari out) dan masuk (rx,
// dari in); nil bila belum ada sampel pembanding. Counter yang turun (sesi di-reset) dihitung nol.
func (t *RateTracker) Observe(id string, in, out int64, now time.Time) (tx, rx *int64) {
	t.next[id] = counterSample{at: now, in: in, out: out}

	before, ok := t.last[id]
	seconds := now.Sub(before.at).Seconds()
	if !ok || seconds <= 0 {
		return nil, nil
	}
	rate := func(current, previous int64) *int64 {
		value := int64(float64(max(current-previous, 0)) * 8 / seconds)
		return &value
	}
	return rate(out, before.out), rate(in, before.in)
}

// Commit menjadikan sampel siklus ini pembanding siklus berikutnya dan membuang baris yang hilang.
func (t *RateTracker) Commit() {
	t.last, t.next = t.next, make(map[string]counterSample, len(t.next))
}

// ParseInt membaca counter RouterOS; nilai kosong atau rusak dianggap 0.
func ParseInt(value string) int64 {
	n, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return 0
	}
	return n
}
