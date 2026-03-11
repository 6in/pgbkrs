package restore

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// Logger manages three log files for a restore operation: drop.log, restore.log, summary.log.
type Logger struct {
	dropLog    *os.File
	restoreLog *os.File
	summaryLog *os.File
	dropCount    int
	restoreCount int
	warnCount    int
	startTime    time.Time
}

// NewLogger creates a new Logger rooted at logDir/restore_YYYYMMDD_HHMMSS/.
func NewLogger(logDir string) (*Logger, error) {
	ts := time.Now().Format("20060102_150405")
	dir := filepath.Join(logDir, "restore_"+ts)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("create log dir %s: %w", dir, err)
	}

	dropLog, err := os.Create(filepath.Join(dir, "drop.log"))
	if err != nil {
		return nil, fmt.Errorf("create drop.log: %w", err)
	}

	restoreLog, err := os.Create(filepath.Join(dir, "restore.log"))
	if err != nil {
		dropLog.Close()
		return nil, fmt.Errorf("create restore.log: %w", err)
	}

	summaryLog, err := os.Create(filepath.Join(dir, "summary.log"))
	if err != nil {
		dropLog.Close()
		restoreLog.Close()
		return nil, fmt.Errorf("create summary.log: %w", err)
	}

	return &Logger{
		dropLog:    dropLog,
		restoreLog: restoreLog,
		summaryLog: summaryLog,
		startTime:  time.Now(),
	}, nil
}

// LogDrop writes a DROP log entry for the given object.
func (l *Logger) LogDrop(schema, kind, name string, err error) {
	l.dropCount++
	status := "OK"
	if err != nil {
		status = "ERROR: " + err.Error()
	}
	fmt.Fprintf(l.dropLog, "[%s] DROP  schema=%s kind=%-20s name=%s → %s\n",
		time.Now().Format("2006-01-02 15:04:05"), schema, kind, name, status)
}

// LogRestore writes a RESTORE log entry for the given object.
func (l *Logger) LogRestore(schema, kind, name string, err error) {
	l.restoreCount++
	status := "OK"
	if err != nil {
		status = "ERROR: " + err.Error()
	}
	fmt.Fprintf(l.restoreLog, "[%s] RESTORE schema=%s kind=%-20s name=%s → %s\n",
		time.Now().Format("2006-01-02 15:04:05"), schema, kind, name, status)
}

// LogLeakWarning writes a WARN entry for a leaked object to the drop log.
func (l *Logger) LogLeakWarning(schema, kind, name string) {
	l.warnCount++
	fmt.Fprintf(l.dropLog, "[%s] WARN  schema=%s kind=%-20s name=%s → LEAK DETECTED\n",
		time.Now().Format("2006-01-02 15:04:05"), schema, kind, name)
}

// WriteSummary writes the summary log with operation totals.
func (l *Logger) WriteSummary(totalDrop, totalRestore, totalWarn int, elapsed time.Duration, restoreErr error) {
	status := "SUCCESS"
	if restoreErr != nil {
		status = "FAILED: " + restoreErr.Error()
	}
	fmt.Fprintf(l.summaryLog,
		"Restore Summary\n"+
			"  Status:          %s\n"+
			"  Duration:        %s\n"+
			"  Objects dropped: %d\n"+
			"  Objects restored:%d\n"+
			"  Leak warnings:   %d\n",
		status, elapsed.Round(time.Millisecond), totalDrop, totalRestore, totalWarn)
}

// Close closes all three log files.
func (l *Logger) Close() error {
	var errs []error
	if err := l.dropLog.Close(); err != nil {
		errs = append(errs, err)
	}
	if err := l.restoreLog.Close(); err != nil {
		errs = append(errs, err)
	}
	if err := l.summaryLog.Close(); err != nil {
		errs = append(errs, err)
	}
	if len(errs) > 0 {
		return fmt.Errorf("close logger: %v", errs)
	}
	return nil
}
