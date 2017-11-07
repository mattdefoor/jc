// jc package to produce hash values from passwords
package main

import (
	//"crypto/sha512"
	"encoding/base64"
	//"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	//"time"
)

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

var portVar int
var debugVar bool
var hashWaitVar int

func init() {
	// Parse command-line flags
	flag.IntVar(&portVar, "port", defaultPort, portUsage)
	flag.IntVar(&portVar, "p", defaultPort, portUsage + shorthand)
	flag.BoolVar(&debugVar, "debug", defaultDebug, debugUsage)
	flag.BoolVar(&debugVar, "d", defaultDebug, debugUsage + shorthand)
	flag.IntVar(&hashWaitVar, "hash_wait", defaultHashWait, defaultHashWaitUsage)
	flag.IntVar(&hashWaitVar, "hw", defaultHashWait, defaultHashWaitUsage + shorthand)
	flag.Parse()
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

// TODO: Switch to using a channel to main to get the hash...
func getHashHandler(m map[int]string) http.HandlerFunc {
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
        value, ok := m[jobid]
        if !ok {
            http.Error(w, "Password has not been hashed", http.StatusNotFound)
            return
        }
        fmt.Fprintf(w, base64.StdEncoding.EncodeToString([]byte(value)))
    }
}

// TODO: Switch to using a channel to main to generate a jobId and generate
// the hash...
func postHashHandler(m map[int]string) http.HandlerFunc {
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

        // Use the increment channel to provide synchronization to jobId.
        //go incrementJobID(a, a.IncrementChan)
        //jobId := <-a.IncrementChan

        // Respond immediately with the jobId.
        //fmt.Fprintf(w, strconv.Itoa(jobId))

        //a.WG.Add(1)
        //go generateHash(a, jobid, []byte(r.PostFormValue("password")))
    }
}

// TODO: Switch to using a channel to get stats from main
func getStatsHandler(m map[int]string) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        if shutdownPending() {
            http.Error(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)
            return
        }

        //count := len(m)
        //average := 
        //stats := map[string]int{"total": count, "average": average}
        //s, err := json.Marshal(stats)
        //if err != nil {
            //http.Error(w, "Unable to get statistics", http.StatusInternalServerError)
            //return
        //}
        //fmt.Fprintf(w, string(s))
    }
}

func main() {
    if debugVar {
        fmt.Printf("Listening on port %d\n", portVar)
    }

	// Handle SIGINT and SIGTERM. You can trigger a graceful
	// shutdown via Ctrl-C from the terminal in which we are
	// launched or via kill -2(INT)/-15(TERM) <pid>.
	ch := make(chan os.Signal)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-ch
		close(shutdown)
		fmt.Println("Shutting down...")
		fmt.Println("Waiting for outstanding hash requests to complete...")
		fmt.Println("Shutdown complete.")
		os.Exit(1)
	}()

    m := make(map[int]string)

    gh := getHashHandler(m)
	http.Handle("/hash/", gh)

    ph := postHashHandler(m)
	http.Handle("/hash", ph)

    sh := getStatsHandler(m)
	http.Handle("/stats", sh)

	err := http.ListenAndServe(":" + strconv.Itoa(portVar), nil)
	if err != nil {
		log.Fatal("ListenAndServe:", err)
		os.Exit(1)
	}
}

