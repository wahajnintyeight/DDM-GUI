package main

import "sync"

// NetworkSampler turns the wrapping adapter counters into a steadily growing byte total the interface can poll.
type networkSampler struct {
	mu    sync.Mutex
	last  uint32
	total uint64
	ready bool
}

var network networkSampler

// NetworkReceivedBytes returns the bytes received by this computer's network adapters since the first call,
// or -1 where that is not available. The screen turns two readings into a download speed. It counts all
// traffic on the machine, not only the downloader's.
func (a *App) NetworkReceivedBytes() int64 {
	counter, ok := systemReceivedBytes()
	if !ok {
		return -1
	}
	network.mu.Lock()
	defer network.mu.Unlock()
	if network.ready {
		network.total += uint64(counter - network.last) // unsigned subtraction handles counter wrap-around
	}
	network.last, network.ready = counter, true
	return int64(network.total)
}
