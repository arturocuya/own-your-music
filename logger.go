package main

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"time"
)

// Global file pointer
var logFile *os.File

func deleteEmptyFiles(dir string) error {
	// Walk the directory recursively.
	return filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		// If there's an error accessing a path, log and skip it.
		if err != nil {
			log.Printf("Error accessing %s: %v", path, err)
			return nil
		}

		// Skip directories.
		if info.IsDir() {
			return nil
		}

		// Check if file size is zero.
		if info.Size() == 0 {
			log.Printf("Deleting empty file: %s", path)
			if removeErr := os.Remove(path); removeErr != nil {
				log.Printf("Failed to delete %s: %v", path, removeErr)
			}
		}
		return nil
	})
}

func initLogger() {
	logDir := "logs"
	if err := os.MkdirAll(logDir, 0755); err != nil {
		log.Fatalf("Failed to create logs directory: %v", err)
	}

	deleteEmptyFiles(logDir)

	timestamp := time.Now().Format("2006-01-02_15-04-05_MST")
	filename := fmt.Sprintf("%s/output_%s.log", logDir, timestamp)

	var err error
	logFile, err = os.OpenFile(filename, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalf("Failed to open log file %s: %v", filename, err)
	}

	// Create a MultiWriter to write to both stdout and the file.
	mw := io.MultiWriter(os.Stdout, logFile)
	log.SetOutput(mw)
}

func shutdownLogger() {
	if logFile != nil {
		if err := logFile.Close(); err != nil {
			log.Printf("Error closing log file: %v", err)
		}
	}
}
