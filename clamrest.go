package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"iwfwebsolutions/av-rest/scanner"
	"iwfwebsolutions/av-rest/scanner/avast"
)

var opts map[string]string
var sc scanner.Scanner

func init() {
	log.SetOutput(io.Discard)
}

// versionHandler returns the scanner engine version as JSON.
func versionHandler(w http.ResponseWriter, r *http.Request) {
	v, err := sc.Version()
	if err != nil {
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	fmt.Fprintf(w, `{"Version":%s}`, jsonString(v))
}

// healthcheck returns 200 when the scanner is healthy, 420 when signatures are
// too old, and 503 when the scanner is unreachable.
func healthcheck(w http.ResponseWriter, r *http.Request) {
	maxAge, _ := strconv.ParseInt(opts["HEALTHCHECK_MAX_SIGNATURE_AGE"], 10, 64)
	if err := sc.IsHealthy(maxAge); err != nil {
		msg := err.Error()
		if strings.Contains(msg, "too old") {
			http.Error(w, msg, 420)
		} else {
			http.Error(w, msg, http.StatusServiceUnavailable)
		}
		return
	}
	w.WriteHeader(http.StatusOK)
}

// scanFileHandler scans a file whose absolute path is given via ?path=.
func scanFileHandler(w http.ResponseWriter, r *http.Request) {
	paths, ok := r.URL.Query()["path"]
	if !ok || len(paths[0]) < 1 {
		http.Error(w, "URL param 'path' is missing", http.StatusBadRequest)
		return
	}
	path := paths[0]

	log.Printf("Started scanning %v\n", path)
	result, err := sc.ScanFile(path)
	if err != nil {
		http.Error(w, fmt.Sprintf("scanner error: %v", err), http.StatusInternalServerError)
		return
	}

	resp := scanResponse{
		Status:      result.Status,
		Description: result.Description,
		FileName:    result.FileName,
		httpStatus:  httpStatusForResult(result),
	}

	body, err := json.Marshal(resp)
	if err != nil {
		http.Error(w, "failed to marshal response", http.StatusInternalServerError)
		return
	}

	log.Printf("Finished scanning %v: %v\n", path, result.Status)
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(resp.httpStatus)
	w.Write(body)
}

// httpStatusForResult maps a ScanResult status to an HTTP status code.
func httpStatusForResult(r *scanner.ScanResult) int {
	switch r.Status {
	case scanner.StatusOK:
		return http.StatusOK // 200
	case scanner.StatusFound:
		return http.StatusNotAcceptable // 406
	case scanner.StatusError:
		return http.StatusBadRequest // 400
	default:
		return http.StatusNotImplemented // 501
	}
}

// jsonString returns a JSON-encoded string literal (with quotes).
func jsonString(s string) string {
	b, _ := json.Marshal(s)
	return string(b)
}

func main() {
	opts = make(map[string]string)

	log.SetFlags(0)
	log.SetOutput(new(logWriter))

	for _, e := range os.Environ() {
		pair := strings.SplitN(e, "=", 2)
		if len(pair) == 2 {
			opts[pair[0]] = pair[1]
		}
	}

	// Defaults
	if opts["PORT"] == "" {
		opts["PORT"] = "9000"
	}
	if opts["HEALTHCHECK_MAX_SIGNATURE_AGE"] == "" {
		opts["HEALTHCHECK_MAX_SIGNATURE_AGE"] = "48"
	}

	log.Println("Using Avast scanner")
	sc = avast.New(
		opts["AVAST_SCAN_BIN"],
		opts["AVAST_VDF_DIR"],
		opts["AVAST_REPORT_PUP"] == "1",
		opts["AVAST_REPORT_TOOLS"] == "1",
	)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /scanFile", scanFileHandler)
	mux.HandleFunc("GET /version", versionHandler)
	mux.HandleFunc("GET /healthcheck", healthcheck)

	// HTTP server with h2c support
	var protocols http.Protocols
	protocols.SetHTTP1(true)
	protocols.SetUnencryptedHTTP2(true)
	protocols.SetHTTP2(true)

	httpServer := &http.Server{
		Addr:      fmt.Sprintf(":%s", opts["PORT"]),
		Handler:   mux,
		Protocols: &protocols,
	}

	log.Printf("Starting antivirus-rest on :%s (scanner: %s)\n", opts["PORT"], opts["SCANNER"])
	log.Fatal(httpServer.ListenAndServe())
}

// logWriter formats log output with a timestamp matching the ClamAV log format.
type logWriter struct{}

func (w logWriter) Write(b []byte) (int, error) {
	return fmt.Printf("%v -> %v", time.Now().Format(time.ANSIC), string(b))
}
