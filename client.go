package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// IPFSUploadResponse represents the response from IPFS
type IPFSUploadResponse struct {
	Hash string `json:"Hash"`
}

// uploadToIPFS uploads a file to IPFS and returns the file hash
func uploadToIPFS(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	var requestBody bytes.Buffer
	writer := multipart.NewWriter(&requestBody)
	part, err := writer.CreateFormFile("file", file.Name())
	if err != nil {
		return "", fmt.Errorf("failed to create form file: %w", err)
	}
	_, err = io.Copy(part, file)
	if err != nil {
		return "", fmt.Errorf("failed to copy file content: %w", err)
	}
	writer.Close()

	resp, err := http.Post("http://localhost:5001/api/v0/add", writer.FormDataContentType(), &requestBody)
	if err != nil {
		return "", fmt.Errorf("failed to upload to IPFS: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("IPFS upload failed with status %d: %s", resp.StatusCode, string(body))
	}

	var ipfsResponse IPFSUploadResponse
	if err := json.NewDecoder(resp.Body).Decode(&ipfsResponse); err != nil {
		return "", fmt.Errorf("failed to decode IPFS response: %w", err)
	}

	return ipfsResponse.Hash, nil
}

// getTailscalePeers retrieves the list of Tailscale-connected peers
func getTailscalePeers() ([]string, error) {
	cmd := exec.Command("tailscale", "status")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to execute 'tailscale status': %w", err)
	}

	lines := strings.Split(string(output), "\n")
	peers := []string{}
	for _, line := range lines {
		fields := strings.Fields(line)
		if len(fields) >= 2 && strings.Contains(fields[0], ".") { // Assuming valid IP address is in the first field
			peers = append(peers, fields[0])
		}
	}

	// Print peers
	fmt.Println("Tailscale peers: ", peers)

	return peers, nil
}

// sendHashToTailscalePeers sends the concatenated hash string to all Tailscale-connected peers
func sendHashToTailscalePeers(hashes string, peers []string) {
	for _, peer := range peers {
		url := fmt.Sprintf("http://%s:8080/receive", peer) // Assuming peers listen on port 8080
		resp, err := http.Post(url, "text/plain", strings.NewReader(hashes))
		if err != nil {
			fmt.Printf("Error sending hash to %s: %v\n", peer, err)
			continue
		}
		resp.Body.Close()
		if resp.StatusCode == http.StatusOK {
			fmt.Printf("Successfully sent hash to %s\n", peer)
		} else {
			fmt.Printf("Failed to send hash to %s, status: %d\n", peer, resp.StatusCode)
		}
	}
}

func main() {
	// List of files to upload
	files := []string{"algo.py", "data.txt"}
	fileHashes := make(map[string]string)

	for _, filePath := range files {
		hash, err := uploadToIPFS(filePath)
		if err != nil {
			fmt.Printf("Error uploading %s: %v\n", filePath, err)
			continue
		}
		fileHashes[filePath] = hash
		fmt.Printf("Uploaded %s to IPFS with hash: %s\n", filePath, hash)
	}

	// Concatenate hashes with extensions into a single comma-separated string
	hashList := []string{}
	for filePath, hash := range fileHashes {
		extension := filepath.Ext(filePath) // Extract the file extension
		hashList = append(hashList, fmt.Sprintf("%s|%s", hash, extension))
	}
	hashes := strings.Join(hashList, ",")

	// Retrieve Tailscale-connected peers
	peers, err := getTailscalePeers()
	if err != nil {
		fmt.Printf("Error retrieving Tailscale peers: %v\n", err)
		return
	}

	// Send hashes to all peers
	sendHashToTailscalePeers(hashes, peers)
}