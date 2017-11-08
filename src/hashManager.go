package main

import (
	"crypto/sha512"
	"fmt"
	"log"
	"time"
)

type entry struct {
	id       int
	hash     string
	duration int
}

// Stats reports the total number and average hash time of map entries
type Stats struct {
	Total   int
	Average int
}

var AddJobID = make(chan int)
var AddEntry = make(chan entry)
var GetJobID = make(chan int)
var GetHash = make(chan string)
var GetStats = make(chan Stats)

func hashManager() {
	var jobIds int                  // Count of jobIds is confined to hashManager goroutine
	entries := make(map[int]string) // Map of jobIds->hashes is confined to hashManager goroutine
	var stats Stats                 // Stats is confined to hashManager goroutine
	for {
		select {
		case AddJobID <- jobIds:
			jobIds++
		case entry := <-AddEntry:
			if debugVar {
				log.Printf("Entry id = %d; hash = %s; duration = %d\n", entry.id, entry.hash,
					entry.duration)
			}
			entries[entry.id] = entry.hash
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
		}
	}
}

func generateHash(jobid int, data []byte) {
	// Wait the appropriate amount of time specified by hashWaitVar.
	time.Sleep(time.Duration(hashWaitVar) * time.Second)

	// New hash entry for use.
	e := new(entry)
	e.id = jobid

	// Start calculating how long it takes to generate a hash.
	start := time.Now().UTC()
	e.hash = fmt.Sprintf("%x", sha512.Sum512(data))
	duration := time.Since(start)
	e.duration = int(duration.Nanoseconds() % 1e6 / 1e3)
	if debugVar {
		log.Printf("Hash generation duration = %v, %d\n", duration, e.duration)
	}
	AddEntry <- *e
}
