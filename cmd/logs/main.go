package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func main() {
	// Create directories if they don't exist
	createDirIfNotExists("uelog")
	createDirIfNotExists("pm")

	// Use a catch-all handler that logs everything and routes appropriately
	http.HandleFunc("/", logRequest(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path

		// Handle /uelog/ paths
		if strings.HasPrefix(path, "/uelog/") {
			handleFileUpload("uelog")(w, r)
			return
		}

		// Handle /pm/ paths
		if strings.HasPrefix(path, "/pm/") {
			handleFileUpload("pm")(w, r)
			return
		}

		// Handle root path
		if path == "/" {
			w.WriteHeader(http.StatusOK)
			fmt.Fprintf(w, "File upload server ready. Use PUT method on /uelog/ or /pm/ endpoints.\n")
			return
		}

		// Handle any other path
		http.NotFound(w, r)
	}))

	// Start server
	port := ":8182" // Changed to match your curl command
	fmt.Printf("Server starting on port %s\n", port)
	fmt.Println("Upload files using PUT method:")
	fmt.Println("\n=== REQUEST LOGGING ENABLED ===")

	log.Fatal(http.ListenAndServe(port, nil))
}

// logRequest is a middleware that logs detailed information about each request
func logRequest(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		fmt.Println("\n" + strings.Repeat("=", 80))
		fmt.Printf("REQUEST: %s\n", time.Now().Format("2006-01-02 15:04:05"))
		fmt.Printf("Method: %s\n", r.Method)
		fmt.Printf("URL: %s\n", r.URL.String())
		fmt.Printf("Path: %s\n", r.URL.Path)
		fmt.Printf("Raw Query: %s\n", r.URL.RawQuery)
		fmt.Printf("Remote Addr: %s\n", r.RemoteAddr)
		fmt.Printf("User Agent: %s\n", r.UserAgent())
		fmt.Printf("Content Length: %d\n", r.ContentLength)
		fmt.Printf("Content Type: %s\n", r.Header.Get("Content-Type"))
		fmt.Printf("Host: %s\n", r.Host)

		// Print all headers
		fmt.Println("Headers:")
		for name, values := range r.Header {
			for _, value := range values {
				fmt.Printf("  %s: %s\n", name, value)
			}
		}

		// Print query parameters if any
		if len(r.URL.Query()) > 0 {
			fmt.Println("Query Parameters:")
			for key, values := range r.URL.Query() {
				for _, value := range values {
					fmt.Printf("  %s: %s\n", key, value)
				}
			}
		}

		fmt.Println(strings.Repeat("-", 80))

		// Call the actual handler
		next(w, r)

		// Log response time
		duration := time.Since(start)
		fmt.Printf("RESPONSE: Completed in %v\n", duration)
		fmt.Println(strings.Repeat("=", 80))
	}
}

// createDirIfNotExists creates a directory if it doesn't already exist
func createDirIfNotExists(dir string) {
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		err := os.MkdirAll(dir, 0755)
		if err != nil {
			log.Fatalf("Failed to create directory %s: %v", dir, err)
		}
		fmt.Printf("Created directory: %s\n", dir)
	}
}

// handleFileUpload returns a handler function for the specified directory
func handleFileUpload(baseDir string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Extract filename from URL path
		path := r.URL.Path
		prefix := "/" + baseDir + "/"
		if !strings.HasPrefix(path, prefix) {
			http.Error(w, "Invalid path", http.StatusBadRequest)
			return
		}

		filename := strings.TrimPrefix(path, prefix)

		// Handle different HTTP methods
		switch r.Method {
		case http.MethodPut:
			// Handle file upload
			if filename == "" {
				http.Error(w, "No filename specified for upload", http.StatusBadRequest)
				return
			}

			// Prevent directory traversal attacks
			if strings.Contains(filename, "..") || strings.Contains(filename, "/") {
				http.Error(w, "Invalid filename", http.StatusBadRequest)
				return
			}

			// Create full file path
			fullPath := filepath.Join(baseDir, filename)

			// Create the file
			file, err := os.Create(fullPath)
			if err != nil {
				log.Printf("Error creating file %s: %v", fullPath, err)
				http.Error(w, "Failed to create file", http.StatusInternalServerError)
				return
			}
			defer file.Close()

			// Copy request body to file
			bytesWritten, err := io.Copy(file, r.Body)
			if err != nil {
				log.Printf("Error writing to file %s: %v", fullPath, err)
				http.Error(w, "Failed to write file", http.StatusInternalServerError)
				return
			}

			// Log successful upload
			log.Printf("File uploaded successfully: %s (%d bytes)", fullPath, bytesWritten)

			// Send success response
			w.WriteHeader(http.StatusCreated)
			fmt.Fprintf(w, "File uploaded successfully: %s (%d bytes)\n", filename, bytesWritten)

		case http.MethodGet:
			// Handle GET requests - show available files or file listing
			if filename == "" {
				// List files in directory
				files, err := os.ReadDir(baseDir)
				if err != nil {
					http.Error(w, "Cannot read directory", http.StatusInternalServerError)
					return
				}

				w.Header().Set("Content-Type", "text/plain")
				fmt.Fprintf(w, "Files in /%s/:\n", baseDir)
				for _, file := range files {
					if !file.IsDir() {
						fmt.Fprintf(w, "- %s\n", file.Name())
					}
				}
			} else {
				// Serve specific file
				fullPath := filepath.Join(baseDir, filename)
				http.ServeFile(w, r, fullPath)
			}

		case http.MethodDelete:
			// Handle file deletion
			if filename == "" {
				http.Error(w, "No filename specified for deletion", http.StatusBadRequest)
				return
			}

			fullPath := filepath.Join(baseDir, filename)
			err := os.Remove(fullPath)
			if err != nil {
				http.Error(w, "Failed to delete file", http.StatusInternalServerError)
				return
			}

			fmt.Fprintf(w, "File deleted: %s\n", filename)

		default:
			// Handle any other method
			w.WriteHeader(http.StatusOK)
			fmt.Fprintf(w, "Received %s request to %s\n", r.Method, r.URL.Path)
			fmt.Fprintf(w, "Supported methods: GET (list/download), PUT (upload), DELETE (remove)\n")
		}
	}
}
