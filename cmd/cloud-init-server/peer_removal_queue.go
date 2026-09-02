// SPDX-FileCopyrightText: Copyright © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package main

import "github.com/rs/zerolog/log"

const (
	defaultPeerRemovalWorkers = 2
	defaultPeerRemovalBuffer  = 64
)

type peerRemover interface {
	RemovePeer(peerName string) error
}

type PeerRemovalQueue struct {
	remover peerRemover
	jobs    chan string
}

func NewPeerRemovalQueue(remover peerRemover) *PeerRemovalQueue {
	return newPeerRemovalQueue(remover, defaultPeerRemovalWorkers, defaultPeerRemovalBuffer)
}

func newPeerRemovalQueue(remover peerRemover, workers int, buffer int) *PeerRemovalQueue {
	queue := &PeerRemovalQueue{
		remover: remover,
		jobs:    make(chan string, buffer),
	}
	for range workers {
		go queue.work()
	}
	return queue
}

func (q *PeerRemovalQueue) TryEnqueue(peerName string) bool {
	if q == nil || q.remover == nil {
		return true
	}
	select {
	case q.jobs <- peerName:
		return true
	default:
		return false
	}
}

func (q *PeerRemovalQueue) work() {
	for peerName := range q.jobs {
		if err := q.remover.RemovePeer(peerName); err != nil {
			log.Error().Err(err).Str("peer", peerName).Msg("failed to remove WireGuard peer")
		}
	}
}
