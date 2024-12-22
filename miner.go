package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const IPFSDownloadURL = "http://127.0.0.1:8080/ipfs/"

// downloadFromIPFS downloads a file from IPFS using the provided hash
func downloadFromIPFS(hash, filename string) error {
	url := IPFSDownloadURL + hash
	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("failed to download file from IPFS: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to download file, status: %d", resp.StatusCode)
	}

	file, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	_, err = io.Copy(file, resp.Body)
	return err
}

// executePythonFile executes the specified Python file with an argument and displays the output
func executePythonFile(filename, arg string) (string, error) {
	cmd := exec.Command("python", filename, arg) // Use python explicitly
	output, err := cmd.CombinedOutput()          // Capture both stdout and stderr
	if err != nil {
		return "", fmt.Errorf("File execution failed: %v, output: %s", err, string(output))
	}
	return string(output), nil
}

// removeFile removes a file from the filesystem
func removeFile(filename string) error {
	err := os.Remove(filename)
	if err != nil {
		return fmt.Errorf("failed to remove file: %v", err)
	}
	return nil
}

func handleReceive(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read request body", http.StatusInternalServerError)
		return
	}
	defer r.Body.Close()

	hashes := strings.Split(string(body), ",")
	if len(hashes) != 2 {
		http.Error(w, "Expected two hashes: one for the Python file and one for the text file", http.StatusBadRequest)
		return
	}

	// Parse hashes with extensions
	pythonParts := strings.SplitN(hashes[0], "|", 2)
	txtParts := strings.SplitN(hashes[1], "|", 2)

	if len(pythonParts) != 2 || len(txtParts) != 2 {
		http.Error(w, "Hashes must include file extensions", http.StatusBadRequest)
		return
	}

	pythonHash, pythonExt := strings.TrimSpace(pythonParts[0]), strings.TrimSpace(pythonParts[1])
	txtHash, txtExt := strings.TrimSpace(txtParts[0]), strings.TrimSpace(txtParts[1])

	// Ensure the Python file has the .py extension
	if !strings.HasSuffix(pythonExt, ".py") {
		pythonExt = ".py"
	}

	// Ensure the text file has the .txt extension
	if !strings.HasSuffix(txtExt, ".txt") {
		txtExt = ".txt"
	}

	// Save files in the current directory
	currentDir, err := os.Getwd()
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get current directory: %v", err), http.StatusInternalServerError)
		return
	}

	pythonFilename := filepath.Join(currentDir, fmt.Sprintf("%s%s", pythonHash, pythonExt))
	txtFilename := filepath.Join(currentDir, fmt.Sprintf("%s%s", txtHash, txtExt))

	fmt.Printf("Downloading Python file with hash: %s\n", pythonHash)
	if err := downloadFromIPFS(pythonHash, pythonFilename); err != nil {
		http.Error(w, fmt.Sprintf("Failed to download Python file: %v", err), http.StatusInternalServerError)
		return
	}

	fmt.Printf("Downloading text file with hash: %s\n", txtHash)
	if err := downloadFromIPFS(txtHash, txtFilename); err != nil {
		http.Error(w, fmt.Sprintf("Failed to download text file: %v", err), http.StatusInternalServerError)
		return
	}

	fmt.Printf("Executing Python file: %s with argument: %s\n", pythonFilename, txtFilename)
	result, err := executePythonFile(pythonFilename, txtFilename)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to execute Python file: %v", err), http.StatusInternalServerError)
		return
	}

	// Print Python script output
	fmt.Println("Python script output:", result)

	// Remove the downloaded files after processing
	if err := removeFile(pythonFilename); err != nil {
		http.Error(w, fmt.Sprintf("Failed to remove Python file: %v", err), http.StatusInternalServerError)
		return
	}
	if err := removeFile(txtFilename); err != nil {
		http.Error(w, fmt.Sprintf("Failed to remove text file: %v", err), http.StatusInternalServerError)
		return
	}

	fmt.Println("Hashes processed successfully")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Hashes processed successfully"))
}

func main() {
	http.HandleFunc("/receive", handleReceive)
	fmt.Println("Server is listening on port 8080...")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		fmt.Printf("Error starting server: %v\n", err)
	}
}
