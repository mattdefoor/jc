package main

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
	AddJobID = make(chan int)
	AddEntry = make(chan entry)
	GetJobID = make(chan int)
	GetHash  = make(chan string)
	GetStats = make(chan Stats)
)

func init() {
	go hashManager()
}

func hashManager() {
	var jobIDs = 1                  // Count of jobIDs is confined to hashManager goroutine
	entries := make(map[int]string) // Map of jobIds->hashes is confined to hashManager goroutine
	var stats Stats                 // Stats is confined to hashManager goroutine
	for {
		select {
		case AddJobID <- jobIDs:
			jobIDs++
		case entry := <-AddEntry:
			debugLog("Entry id = %d; hash = %s; duration = %d\n", entry.jobID, entry.hash, entry.duration)
			entries[entry.jobID] = entry.hash
			stats.Total = len(entries)
			stats.Average = (stats.Average*(stats.Total-1) + entry.duration) / stats.Total
			debugLog("Hash average = %d\n", stats.Average)
		case id := <-GetJobID:
			debugLog("Checking for job id = %d\n", id)
			value, _ := entries[id]
			GetHash <- value
		case GetStats <- stats:
		}
	}
}
