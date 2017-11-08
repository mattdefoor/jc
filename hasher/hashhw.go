// jc package to produce hash values from passwords
package main

import (
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
	flag.IntVar(&portVar, "p", defaultPort, portUsage+shorthand)
	flag.BoolVar(&debugVar, "debug", defaultDebug, debugUsage)
	flag.BoolVar(&debugVar, "d", defaultDebug, debugUsage+shorthand)
	flag.IntVar(&hashWaitVar, "hash_wait", defaultHashWait, defaultHashWaitUsage)
	flag.IntVar(&hashWaitVar, "hw", defaultHashWait, defaultHashWaitUsage+shorthand)
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

func hashHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if shutdownPending() {
			http.Error(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)
			return
		}

		switch r.Method {
		case http.MethodGet:
			// Validate GET parameter input. If we fail to convert
			// the GET parameter to an integer, return an error.
			jobID, err := strconv.Atoi(r.URL.Path[len("/hash/"):])
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
			if debugVar {
				log.Printf("Post Handler jobID = %d\n", jobID)
			}

			// Respond immediately with the jobID.
			fmt.Fprintf(w, strconv.Itoa(jobID))

			// Create queuedEntry and send it to the QueueEntry channel.
			var entry QueuedEntry
			entry.JobID = jobID
			entry.Data = []byte(pw)
			QueueEntry <- entry
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
