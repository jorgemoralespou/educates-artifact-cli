package utils

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

// CleanupManager manages cleanup operations and temporary resources
type CleanupManager struct {
	tempFiles    []string
	tempDirs     []string
	cleanupFuncs []func() error
	mu           sync.Mutex
}

// Global cleanup manager instance
var globalCleanup = &CleanupManager{
	tempFiles:    make([]string, 0),
	tempDirs:     make([]string, 0),
	cleanupFuncs: make([]func() error, 0),
}

// AddTempFile adds a temporary file to be cleaned up
func (cm *CleanupManager) AddTempFile(filePath string) {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	cm.tempFiles = append(cm.tempFiles, filePath)
}

// AddTempDir adds a temporary directory to be cleaned up
func (cm *CleanupManager) AddTempDir(dirPath string) {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	cm.tempDirs = append(cm.tempDirs, dirPath)
}

// AddCleanupFunc adds a cleanup function to be called
func (cm *CleanupManager) AddCleanupFunc(cleanupFunc func() error) {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	cm.cleanupFuncs = append(cm.cleanupFuncs, cleanupFunc)
}

// Cleanup performs all cleanup operations
func (cm *CleanupManager) Cleanup() error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	var errors []error

	// Run cleanup functions first
	for _, cleanupFunc := range cm.cleanupFuncs {
		if err := cleanupFunc(); err != nil {
			errors = append(errors, fmt.Errorf("cleanup function failed: %w", err))
		}
	}

	// Remove temporary files
	for _, filePath := range cm.tempFiles {
		if err := os.Remove(filePath); err != nil && !os.IsNotExist(err) {
			errors = append(errors, fmt.Errorf("failed to remove temp file %s: %w", filePath, err))
		}
	}

	// Remove temporary directories
	for _, dirPath := range cm.tempDirs {
		if err := os.RemoveAll(dirPath); err != nil && !os.IsNotExist(err) {
			errors = append(errors, fmt.Errorf("failed to remove temp dir %s: %w", dirPath, err))
		}
	}

	// Clear the lists
	cm.tempFiles = cm.tempFiles[:0]
	cm.tempDirs = cm.tempDirs[:0]
	cm.cleanupFuncs = cm.cleanupFuncs[:0]

	if len(errors) > 0 {
		return fmt.Errorf("cleanup errors: %v", errors)
	}

	return nil
}

// Global cleanup functions
func AddTempFile(filePath string) {
	globalCleanup.AddTempFile(filePath)
}

func AddTempDir(dirPath string) {
	globalCleanup.AddTempDir(dirPath)
}

func AddCleanupFunc(cleanupFunc func() error) {
	globalCleanup.AddCleanupFunc(cleanupFunc)
}

func Cleanup() error {
	return globalCleanup.Cleanup()
}

// CreateTempDir creates a temporary directory and registers it for cleanup
func CreateTempDir(pattern string) (string, error) {
	tempDir, err := os.MkdirTemp("", pattern)
	if err != nil {
		return "", fmt.Errorf("failed to create temp directory: %w", err)
	}

	AddTempDir(tempDir)
	return tempDir, nil
}

// CreateTempFile creates a temporary file and registers it for cleanup
func CreateTempFile(pattern string) (*os.File, error) {
	tempFile, err := os.CreateTemp("", pattern)
	if err != nil {
		return nil, fmt.Errorf("failed to create temp file: %w", err)
	}

	AddTempFile(tempFile.Name())
	return tempFile, nil
}

// ContextWithTimeout creates a cancellable context with the specified timeout
func ContextWithTimeout(timeout time.Duration) (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), timeout)
}

// ContextWithTimeoutFromString creates a cancellable context from a timeout string
// Supports formats like "30s", "5m", "1h", etc.
func ContextWithTimeoutFromString(timeoutStr string) (context.Context, context.CancelFunc, error) {
	if timeoutStr == "" {
		// No timeout specified, use background context
		return context.Background(), func() {}, nil
	}

	timeout, err := time.ParseDuration(timeoutStr)
	if err != nil {
		return nil, nil, fmt.Errorf("invalid timeout format '%s': %w", timeoutStr, err)
	}

	if timeout <= 0 {
		return nil, nil, fmt.Errorf("timeout must be positive, got: %v", timeout)
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	return ctx, cancel, nil
}

// ContextWithSignalHandling creates a context that handles SIGINT (Ctrl+C) gracefully
// This is the recommended function to use in commands
func ContextWithSignalHandling(timeoutStr string) (context.Context, context.CancelFunc, error) {
	// Create base context with timeout
	ctx, cancel, err := ContextWithTimeoutFromString(timeoutStr)
	if err != nil {
		return nil, nil, fmt.Errorf("invalid timeout: %w", err)
	}

	// If no timeout was specified, use default
	if timeoutStr == "" {
		ctx, cancel = ContextWithTimeout(DefaultTimeout)
	}

	// Create a new context that can be cancelled by signals
	signalCtx, signalCancel := context.WithCancel(ctx)

	// Set up signal handling
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		select {
		case sig := <-sigChan:
			fmt.Printf("\nOperation cancelled by user (signal: %v)\n", sig)

			// Perform cleanup
			if err := Cleanup(); err != nil {
				fmt.Printf("Warning: Cleanup failed: %v\n", err)
			}

			signalCancel()
		case <-signalCtx.Done():
			// Context was cancelled by timeout or other means
			return
		}
	}()

	// Return a combined cancel function
	combinedCancel := func() {
		signalCancel()
		cancel()
		signal.Stop(sigChan)
		close(sigChan)
	}

	return signalCtx, combinedCancel, nil
}

// ContextWithDeadline creates a cancellable context with a specific deadline
func ContextWithDeadline(deadline time.Time) (context.Context, context.CancelFunc) {
	return context.WithDeadline(context.Background(), deadline)
}

// ContextWithCancel creates a cancellable context without timeout
func ContextWithCancel() (context.Context, context.CancelFunc) {
	return context.WithCancel(context.Background())
}

// IsContextCancelled checks if a context has been cancelled
func IsContextCancelled(ctx context.Context) bool {
	select {
	case <-ctx.Done():
		return true
	default:
		return false
	}
}

// GetContextError returns the error from a cancelled context
func GetContextError(ctx context.Context) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}
	return nil
}

// IsCancelledByUser checks if the context was cancelled by user (SIGINT/SIGTERM)
func IsCancelledByUser(ctx context.Context) bool {
	select {
	case <-ctx.Done():
		return ctx.Err() == context.Canceled
	default:
		return false
	}
}

// HandleContextError handles context errors with appropriate messages
func HandleContextError(ctx context.Context, operation string) error {
	if ctx.Err() == nil {
		return nil
	}

	if IsCancelledByUser(ctx) {
		return fmt.Errorf("operation '%s' was cancelled by user", operation)
	}

	if ctx.Err() == context.DeadlineExceeded {
		return fmt.Errorf("operation '%s' timed out", operation)
	}

	return fmt.Errorf("operation '%s' failed: %w", operation, ctx.Err())
}

// DefaultTimeout is the default timeout for operations
const DefaultTimeout = 5 * time.Minute

// ParseTimeoutWithDefault parses a timeout string with a default fallback
func ParseTimeoutWithDefault(timeoutStr string, defaultTimeout time.Duration) (time.Duration, error) {
	if timeoutStr == "" {
		return defaultTimeout, nil
	}

	timeout, err := time.ParseDuration(timeoutStr)
	if err != nil {
		return 0, fmt.Errorf("invalid timeout format '%s': %w", timeoutStr, err)
	}

	if timeout <= 0 {
		return 0, fmt.Errorf("timeout must be positive, got: %v", timeout)
	}

	return timeout, nil
}
