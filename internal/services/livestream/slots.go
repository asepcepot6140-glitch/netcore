// Package livestream berisi bagian yang dipakai bersama oleh semua stream realtime ke router:
// batas jumlah stream, siklus hidup sesi tetap, dan pengiriman hanya baris yang berubah.
package livestream

import (
	"errors"
	"sync"
)

const (
	maxStreamsPerUser   = 3
	maxStreamsPerRouter = 2 // semua user dan semua menu digabung; tiap stream membebani CPU router
)

var (
	ErrTooManyStreams    = errors.New("too many live streams")
	ErrRouterStreamsFull = errors.New("router has too many live streams")
)

var (
	mu              sync.Mutex
	streamsByUser   = map[uint]int{}
	streamsByRouter = map[uint64]int{}
)

// AcquireStream mencadangkan satu slot stream (tiap stream = satu sesi tetap ke router) dengan
// batas per user dan per router. Panggil release saat selesai; err berisi ErrTooManyStreams bila
// user sudah memakai semua slotnya, atau ErrRouterStreamsFull bila router sudah penuh.
func AcquireStream(userID uint, mikrotikID uint64) (release func(), err error) {
	mu.Lock()
	defer mu.Unlock()

	if streamsByUser[userID] >= maxStreamsPerUser {
		return nil, ErrTooManyStreams
	}
	if streamsByRouter[mikrotikID] >= maxStreamsPerRouter {
		return nil, ErrRouterStreamsFull
	}
	streamsByUser[userID]++
	streamsByRouter[mikrotikID]++

	var once sync.Once
	return func() {
		once.Do(func() {
			mu.Lock()
			defer mu.Unlock()
			if streamsByUser[userID]--; streamsByUser[userID] <= 0 {
				delete(streamsByUser, userID)
			}
			if streamsByRouter[mikrotikID]--; streamsByRouter[mikrotikID] <= 0 {
				delete(streamsByRouter, mikrotikID)
			}
		})
	}, nil
}
