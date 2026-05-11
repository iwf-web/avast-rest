package avast

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"iwfwebsolutions/avast-rest/scanner"
)

const (
	// Default path to the Avast scan CLI binary.
	defaultScanBin = "/usr/bin/scan"
	// Default directory containing Avast virus definitions (mtime used for age check).
	defaultVDFDir = "/var/lib/avast"

	// scan exit codes
	exitClean    = 0
	exitInfected = 1
	exitError    = 2
)

// Scanner uses Avast Business Antivirus for Linux (scan CLI) for file scanning.
type Scanner struct {
	scanBin     string
	vdfDir      string
	reportPUP   bool
	reportTools bool
}

// New creates an Avast Scanner. Pass empty strings to use the defaults.
// reportPUP enables the -u flag (report potentially unwanted programs).
// reportTools enables the -T flag (report tools).
func New(scanBin, vdfDir string, reportPUP, reportTools bool) *Scanner {
	if scanBin == "" {
		scanBin = defaultScanBin
	}
	if vdfDir == "" {
		vdfDir = defaultVDFDir
	}
	return &Scanner{
		scanBin:     scanBin,
		vdfDir:      vdfDir,
		reportPUP:   reportPUP,
		reportTools: reportTools,
	}
}

// ScanFile scans the file at path using the Avast scan CLI.
func (s *Scanner) ScanFile(path string) (*scanner.ScanResult, error) {
	args := []string{}
	if s.reportPUP {
		args = append(args, "-u")
	}
	if s.reportTools {
		args = append(args, "-T")
	}
	args = append(args, path)
	cmd := exec.Command(s.scanBin, args...)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out

	err := cmd.Run()
	exitCode := 0
	if exitErr, ok := err.(*exec.ExitError); ok {
		exitCode = exitErr.ExitCode()
	} else if err != nil {
		return nil, fmt.Errorf("failed to run avast scan: %w", err)
	}

	output := strings.TrimSpace(out.String())

	switch exitCode {
	case exitClean:
		return &scanner.ScanResult{
			Status:   scanner.StatusOK,
			FileName: path,
		}, nil
	case exitInfected:
		// Output format: "PATH\tINFECTION_NAME"
		description := output
		if parts := strings.SplitN(output, "\t", 2); len(parts) == 2 {
			description = parts[1]
		}
		return &scanner.ScanResult{
			Status:      scanner.StatusFound,
			Description: description,
			FileName:    path,
		}, nil
	default:
		return &scanner.ScanResult{
			Status:      scanner.StatusError,
			Description: cleanAvastErrorOutput(output, path),
			FileName:    path,
		}, nil
	}
}

// cleanAvastErrorOutput strips the "avast: " prefix from each line, removes the
// redundant temp-file path (already in FileName), and deduplicates identical lines.
// Line format from the scan binary: "avast: <path>[|><archive_path>]\t<code>: <msg>"
func cleanAvastErrorOutput(output, filePath string) string {
	lines := strings.Split(output, "\n")
	seen := make(map[string]struct{})
	var result []string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		line = strings.TrimPrefix(line, "avast: ")
		parts := strings.SplitN(line, "\t", 2)
		var cleaned string
		if len(parts) == 2 && filePath != "" {
			archivePath := parts[0]
			if strings.HasPrefix(archivePath, filePath+"|>") {
				archivePath = archivePath[len(filePath)+2:]
			} else if archivePath == filePath {
				archivePath = ""
			}
			if archivePath != "" {
				cleaned = archivePath + "\t" + parts[1]
			} else {
				cleaned = parts[1]
			}
		} else {
			cleaned = line
		}
		if _, ok := seen[cleaned]; !ok {
			seen[cleaned] = struct{}{}
			result = append(result, cleaned)
		}
	}
	return strings.Join(result, "\n")
}

// Version returns the Avast virus definitions version from the running daemon.
func (s *Scanner) Version() (string, error) {
	cmd := exec.Command(s.scanBin, "-V")
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("failed to get Avast version: %w", err)
	}
	return strings.TrimSpace(out.String()), nil
}

// IsHealthy verifies that the Avast daemon is reachable and, if
// maxSignatureAgeHours > 0, that the VDF directory is not older than
// maxSignatureAgeHours.
func (s *Scanner) IsHealthy(maxSignatureAgeHours int64) error {
	if _, err := s.Version(); err != nil {
		return fmt.Errorf("Avast scanner unavailable: %w", err)
	}

	if maxSignatureAgeHours == 0 {
		return nil
	}

	fi, err := os.Stat(s.vdfDir)
	if err != nil {
		// If the directory doesn't exist we can't check the age; treat as healthy.
		return nil
	}
	ageHours := time.Since(fi.ModTime()).Hours()
	if ageHours > float64(maxSignatureAgeHours) {
		return fmt.Errorf("Avast signatures too old: %.0fh (max %dh)", ageHours, maxSignatureAgeHours)
	}
	return nil
}
