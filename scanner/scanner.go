package scanner

// Status constants returned by scanner implementations.
const (
	StatusOK         = "OK"
	StatusFound      = "FOUND"
	StatusError      = "ERROR"
	StatusParseError = "PARSE ERROR"
	StatusSizeLimit  = "SIZE LIMIT EXCEEDED"
)

// ScanResult holds the result of a single file scan.
type ScanResult struct {
	Status      string
	Description string
	FileName    string
}

// Scanner is the interface every antivirus backend must implement.
type Scanner interface {
	// ScanFile scans the file at the given absolute path.
	ScanFile(path string) (*ScanResult, error)

	// Version returns a human-readable version string of the scanner engine.
	Version() (string, error)

	// IsHealthy returns nil when the scanner is operational and its signature
	// database is not older than maxSignatureAgeHours. Returns a descriptive
	// error otherwise.
	IsHealthy(maxSignatureAgeHours int64) error
}
