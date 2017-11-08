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
	"path"
	"strconv"
	"sync"
	"syscall"
	"time"
)

var (
	portVar     int
	debugVar    bool
	hashWaitVar int
	shutdown    = make(chan struct{})
	WG          sync.WaitGroup
)

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
	flag.IntVar(&portVar, "p", defaultPort, portUsage+shorthand)
	flag.BoolVar(&debugVar, "debug", defaultDebug, debugUsage)
	flag.BoolVar(&debugVar, "d", defaultDebug, debugUsage+shorthand)
	flag.IntVar(&hashWaitVar, "hash_wait", defaultHashWait, defaultHashWaitUsage)
	flag.IntVar(&hashWaitVar, "hw", defaultHashWait, defaultHashWaitUsage+shorthand)
}

func shutdownPending() bool {
	select {
	case <-shutdown:
		return true
	default:
		return false
	}
}

func generateHash(queuedEntry QueuedEntry) {
	debugLog("Waiting %d seconds to generate hash for JobID = %d\n", hashWaitVar, queuedEntry.JobID)

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
	debugLog("JobID %d hash generation duration = %v, %d\n", e.jobID, duration, e.duration)
	AddEntry <- *e
	WG.Done()
}

func hashHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if shutdownPending() {
			http.Error(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)
			return
		}

		switch r.Method {
		case http.MethodGet:
			base := path.Base(r.URL.Path)
			if base == "." {
				http.Error(w, "Invalid Job ID", http.StatusBadRequest)
				return
			}

			// Validate GET parameter input. If we fail to convert
			// the GET parameter to an integer, return an error.
			jobID, err := strconv.Atoi(base)
			if err != nil {
				http.Error(w, "Invalid Job ID", http.StatusBadRequest)
				return
			}

			GetJobID <- jobID
			select {
			case hash := <-GetHash:
				if hash == "" {
					http.Error(w, "Password has not been hashed", http.StatusNotFound)
					return
				}
				fmt.Fprintf(w, base64.StdEncoding.EncodeToString([]byte(hash)))
			}
		case http.MethodPost:
			pw := r.PostFormValue("password")
			if pw == "" {
				http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
				return
			}

			jobID := <-AddJobID
			debugLog("Post Handler jobID = %d\n", jobID)

			// Respond immediately with the jobID.
			fmt.Fprintf(w, strconv.Itoa(jobID))

			// Create queuedEntry and send it to the QueueEntry channel.
			var entry QueuedEntry
			entry.JobID = jobID
			entry.Data = []byte(pw)
			WG.Add(1)
			go generateHash(entry)
		default:
			http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		}
	}
}

func statsHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if shutdownPending() {
			http.Error(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)
			return
		}

		if r.Method != http.MethodGet {
			http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
			return
		}

		stats := <-GetStats
		jsonData := map[string]int{"total": stats.Total, "average": stats.Average}
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

	// Handle SIGINT and SIGTERM. You can trigger a graceful
	// shutdown via Ctrl-C from the terminal in which we are
	// launched or via kill -2(INT)/-15(TERM) <pid>.
	ch := make(chan os.Signal)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM, syscall.SIGKILL)
	go func() {
		<-ch
		close(shutdown)
		log.Println("Shutting down...")
		log.Println("Waiting for outstanding hash requests to complete...")
		WG.Wait()
		log.Println("Outstanding hash requests finished.")
		log.Println("Shutdown complete.")
		os.Exit(1)
	}()

	log.Println("Registering handlers...")

	http.Handle("/hash", hashHandler())
	http.Handle("/hash/", hashHandler())
	http.Handle("/stats", statsHandler())

	log.Println("Setting up listener...")

	err := http.ListenAndServe(":"+strconv.Itoa(portVar), nil)
	if err != nil {
		log.Fatal("ListenAndServe:", err)
		os.Exit(1)
	}
}

func debugLog(format string, args ...interface{}) {
	if debugVar {
		fmt.Printf(format, args...)
	}
}
