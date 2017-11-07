// jc package to produce hash values from passwords
package main

import (
	"crypto/sha512"
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"
)

var portVar int
var debugVar bool
var hashWaitVar int

func init() {
	// Constant defaults for command-line flag parser.
	const (
		shorthand            = " (shorthand)"
		defaultPort          = 8080
		portUsage            = "Port number for HTTP listener"
		defaultDebug         = false
		debugUsage           = "Enable debug output"
		defaultHashWait      = 5
		defaultHashWaitUsage = "Time in seconds to wait for hash to be computed"
	)

	// Parse command-line flags
	flag.IntVar(&portVar, "port", defaultPort, portUsage)
	flag.IntVar(&portVar, "p", defaultPort, portUsage + shorthand)
	flag.BoolVar(&debugVar, "debug", defaultDebug, debugUsage)
	flag.BoolVar(&debugVar, "d", defaultDebug, debugUsage + shorthand)
	flag.IntVar(&hashWaitVar, "hash_wait", defaultHashWait, defaultHashWaitUsage)
	flag.IntVar(&hashWaitVar, "hw", defaultHashWait, defaultHashWaitUsage + shorthand)
}

// Broadcast channel that is closed when the application has been gracefully
// terminated.
var shutdown = make(chan struct{})

func shutdownPending() bool {
	select {
	case <-shutdown:
		return true
	default:
		return false
	}
}

type entry struct {
	id int
    hash string
    duration int
}

type stats struct {
	total int
	average int
}

var addJobId = make(chan int)
var addEntry = make(chan entry)
var getJobId = make(chan int)
var getHash = make(chan string)
var getStats = make(chan stats)

func getHashHandler() http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        if shutdownPending() {
            http.Error(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)
            return
        }

        // Validate GET parameter input. If we fail to convert
        // the GET parameter to an integer, return an error.
        jobid, err := strconv.Atoi(r.URL.Path[len("/hash/"):])
        if err != nil {
            http.Error(w, "Invalid Job ID", http.StatusBadRequest)
            return
        }
		
		getJobId <- jobid
		select {
			case hash := <-getHash:
				if hash == "" {
					http.Error(w, "Password has not been hashed", http.StatusNotFound)
					return
				}
				fmt.Fprintf(w, base64.StdEncoding.EncodeToString([]byte(hash)))
		}
    }
}

func postHashHandler() http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        if shutdownPending() {
            http.Error(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)
            return
        }

        pw := r.PostFormValue("password")
        if pw == "" {
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return
        }

		jobid := <-addJobId
		if debugVar {
			fmt.Printf("post handler jobId = %d\n", jobid)
		}
		
        // Respond immediately with the jobId.
        fmt.Fprintf(w, strconv.Itoa(jobid))
		
		// TODO: Move this to the hashManager? Use a WaitGroup to keep track of how many.
		go generateHash(jobid, []byte(pw))
    }
}

func getStatsHandler() http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        if shutdownPending() {
            http.Error(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)
            return
        }
		stats := <-getStats
		jsonData := map[string]int{"total": stats.total, "average": stats.average}
		s, err := json.Marshal(jsonData)
		if err != nil {
			http.Error(w, "Unable to get statistics", http.StatusInternalServerError)
			return
		}
        fmt.Fprintf(w, string(s))
    }
}

func main() {
	flag.Parse()
	
    if debugVar {
        fmt.Printf("Listening on port %d\n", portVar)
    }

	// Handle SIGINT and SIGTERM. You can trigger a graceful
	// shutdown via Ctrl-C from the terminal in which we are
	// launched or via kill -2(INT)/-15(TERM) <pid>.
	ch := make(chan os.Signal)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM, syscall.SIGKILL)
	go func() {
		<-ch
		close(shutdown)
		fmt.Println("Shutting down...")
		fmt.Println("Waiting for outstanding hash requests to complete...")
		fmt.Println("Shutdown complete.")
		os.Exit(1)
	}()

	go hashManager()
	
	http.Handle("/hash/", getHashHandler())
	http.Handle("/hash", postHashHandler())
	http.Handle("/stats", getStatsHandler())

	err := http.ListenAndServe(":" + strconv.Itoa(portVar), nil)
	if err != nil {
		log.Fatal("ListenAndServe:", err)
		os.Exit(1)
	}
}

func hashManager() {
	var jobIds int // Count of jobIds is confined to hashManager goroutine
	entries := make(map[int]string) // Map of jobIds->hashes is confined to hashManager goroutine
	var stats stats // Stats is confined to hashManager goroutine
	for {
		select {
		case addJobId <- jobIds:
			jobIds += 1
		case entry := <-addEntry:
			if debugVar {
				fmt.Printf("Entry id = %d; hash = %s; duration = %d\n", entry.id, entry.hash,
				entry.duration)
			}
			entries[entry.id] = entry.hash
			stats.total = len(entries)
			stats.average = (stats.average * (stats.total - 1) + entry.duration) / stats.total
			if debugVar {
				fmt.Printf("Hash average = %d\n", stats.average)
			}
		case id := <-getJobId:
			if debugVar {
				fmt.Printf("Checking for job id = %d\n", id)
			}
			value, _ := entries[id]
			getHash <- value
		case getStats <- stats:
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
	end := time.Now().UTC()
    var duration time.Duration = end.Sub(start)
    e.duration = int(duration.Nanoseconds() % 1e6 / 1e3)
	if debugVar {
		fmt.Printf("Hash generation duration = %v, %d\n", duration, e.duration)
	}
	addEntry <- *e
}
