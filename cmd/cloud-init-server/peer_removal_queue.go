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
	GetPublicKey(peerName string) (string, bool)
}

type PeerRemovalQueue struct {
	remover peerRemover
	jobs    chan removalJob
}

type removalJob struct {
	name string
	key  string
}

func NewPeerRemovalQueue(remover peerRemover) *PeerRemovalQueue {
	return newPeerRemovalQueue(remover, defaultPeerRemovalWorkers, defaultPeerRemovalBuffer)
}

func newPeerRemovalQueue(remover peerRemover, workers int, buffer int) *PeerRemovalQueue {
	queue := &PeerRemovalQueue{
		remover: remover,
		jobs:    make(chan removalJob, buffer),
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
	// Capture the current public key for the peer (if any).
	key, _ := q.remover.GetPublicKey(peerName)
	job := removalJob{name: peerName, key: key}
	select {
	case q.jobs <- job:
		return true
	default:
		return false
	}
}

func (q *PeerRemovalQueue) work() {
	for job := range q.jobs {
		// Verify that the public key hasn't changed since enqueue time.
		currentKey, ok := q.remover.GetPublicKey(job.name)
		if ok && job.key != "" && currentKey != job.key {
			// Stale removal – skip and log.
			log.Warn().Str("peer", job.name).Str("queuedKey", job.key).Str("currentKey", currentKey).
				Msg("stale removal job skipped")
			continue
		}
		if err := q.remover.RemovePeer(job.name); err != nil {
			log.Error().Err(err).Str("peer", job.name).Msg("failed to remove WireGuard peer")
		}
	}
}
