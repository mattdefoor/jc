package main

import (
	"crypto/sha512"
	"fmt"
	"log"
	"sync"
	"time"
)

type entry struct {
	jobID    int
	hash     string
	duration int
}

// QueuedEntry represents a JobID and Data (password) that are waiting to be hashed and stored.
type QueuedEntry struct {
	JobID int
	Data  []byte
}

// Stats reports the total number and average hash time of map entries
type Stats struct {
	Total   int
	Average int
}

var (
	AddJobID   = make(chan int)
	AddEntry   = make(chan entry)
	GetJobID   = make(chan int)
	GetHash    = make(chan string)
	GetStats   = make(chan Stats)
	QueueEntry = make(chan QueuedEntry)
	WG         sync.WaitGroup
)

func init() {
	go hashManager()
}

func hashManager() {
	var jobIDs int                  // Count of jobIDs is confined to hashManager goroutine
	entries := make(map[int]string) // Map of jobIds->hashes is confined to hashManager goroutine
	var stats Stats                 // Stats is confined to hashManager goroutine
	for {
		select {
		case AddJobID <- jobIDs:
			jobIDs++
		case entry := <-AddEntry:
			if debugVar {
				log.Printf("Entry id = %d; hash = %s; duration = %d\n", entry.jobID, entry.hash,
					entry.duration)
			}
			entries[entry.jobID] = entry.hash
			stats.Total = len(entries)
			stats.Average = (stats.Average*(stats.Total-1) + entry.duration) / stats.Total
			if debugVar {
				log.Printf("Hash average = %d\n", stats.Average)
			}
		case id := <-GetJobID:
			if debugVar {
				log.Printf("Checking for job id = %d\n", id)
			}
			value, _ := entries[id]
			GetHash <- value
		case GetStats <- stats:
		case queuedEntry := <-QueueEntry:
			WG.Add(1)
			go generateHash(queuedEntry)
		}
	}
}

func generateHash(queuedEntry QueuedEntry) {
	log.Printf("Waiting %d seconds to generate hash for JobID = %d\n", hashWaitVar, queuedEntry.JobID)

	// Wait the appropriate amount of time specified by hashWaitVar.
	time.Sleep(time.Duration(hashWaitVar) * time.Second)

	// New hash entry for use.
	e := new(entry)
	e.jobID = queuedEntry.JobID

	// Start calculating how long it takes to generate a hash.
	start := time.Now().UTC()
	e.hash = fmt.Sprintf("%x", sha512.Sum512(queuedEntry.Data))
	duration := time.Since(start)
	e.duration = int(duration.Nanoseconds() % 1e6 / 1e3)
	log.Printf("JobID %d hash generation duration = %v, %d\n", e.jobID, duration, e.duration)
	AddEntry <- *e
	WG.Done()
}
