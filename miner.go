package main

import (
	"crypto/sha256"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

const IPFSDownloadURL = "http://127.0.0.1:8080/ipfs/"

// Transaction represents a transaction in the blockchain
type Transaction struct {
	ID   string // The IP address or unique identifier of the transaction
	Data string // The result or output of the computation
}

// Block represents a block in the blockchain
type Block struct {
	PrevHash     string        // Hash of the previous block in the chain
	Transactions []Transaction // List of transactions included in this block
	Nonce        int           // Nonce for proof-of-work
	Hash         string        // Hash of the current block
	PrevCID      string        // IPFS CID of the previous block
	BlockNumber  int           // The block number in the chain (0 for genesis block)
	Timestamp    int64         // Unix timestamp of when the block was created
	Creator      string        // Identifier of the node that created the block
	Difficulty   int           // Mining difficulty level
}

var transactionPool []Transaction
var mutex sync.Mutex   // Mutex to synchronize access to the transaction pool
var currentBlock Block // Each miner has their own current block

var previousBlockCID string = "-1"  // Genesis block's PrevCID will be -1 initially
var previousBlockHash string = "-1" // Genesis block's PrevHash will be empty initially

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

// proofOfWork performs the proof-of-work algorithm to find a valid nonce
func proofOfWork(block Block, difficulty int) int {
	nonce := 0
	var hash string
	for {
		// Generate the hash of the block with the current nonce
		hash = generateHash(block, nonce)

		// Check if the hash satisfies the difficulty condition
		if validProof(hash, difficulty) {
			break
		}

		nonce++
	}
	return nonce
}

// validProof validates the proof of work by checking if the hash has the required number of leading zeros
func validProof(hash string, difficulty int) bool {
	prefix := strings.Repeat("0", difficulty)
	return strings.HasPrefix(hash, prefix)
}

// generateHash generates a SHA256 hash for the block with the given nonce
func generateHash(block Block, nonce int) string {
	block.Nonce = nonce
	blockData := fmt.Sprintf("%s%d%d%s", block.PrevHash, block.BlockNumber, nonce, block.Transactions)
	return fmt.Sprintf("%x", sha256.Sum256([]byte(blockData)))
}

// mineBlock mines a new block using proof of work and adds it to the local chain
func mineBlock(miner string, difficulty int) {
	mutex.Lock()
	defer mutex.Unlock()

	if len(transactionPool) >= 3 {
		// Create a new block
		block := Block{
			PrevHash:     previousBlockHash,            // The hash of the previous block (starting with -1 for the genesis block)
			PrevCID:      previousBlockCID,             // Set the PrevCID of the previous block
			BlockNumber:  currentBlock.BlockNumber + 1, // Increment BlockNumber
			Transactions: transactionPool[:3],          // Take the first 3 transactions
			Timestamp:    time.Now().Unix(),            // Set the current timestamp
			Creator:      miner,                        // Set the creator to the miner's identifier
			Difficulty:   difficulty,                   // Set the difficulty
		}

		// Run Proof of Work in a Goroutine
		go func() {
			nonce := proofOfWork(block, difficulty)
			block.Nonce = nonce
			block.Hash = generateHash(block, nonce)

			// Add the mined block to the local chain (after uploading it to IPFS)
			// Save the block's CID after it's uploaded to IPFS
			go uploadBlockToIPFS(block)

			// Update the previous block's CID to this block's CID after successful upload
			mutex.Lock()
			previousBlockHash = block.Hash
			previousBlockCID = block.PrevCID
			currentBlock = block // Update current block to the mined one
			mutex.Unlock()

			// Broadcast the block to other miners
			go broadcastBlock(block)

			// Clear the processed transactions from the pool
			mutex.Lock()
			transactionPool = transactionPool[3:] // Remove processed transactions
			mutex.Unlock()
		}()
	}
}

// broadcastBlock broadcasts the mined block to other miners for validation
func broadcastBlock(block Block) {
	// Send the block to all other miners for validation
	fmt.Printf("\n\nBroadcasting Block:\n\n %+v\n\n", block)
	// Implement your broadcasting logic here (e.g., send it over a network)
}

// uploadBlockToIPFS uploads the mined block to IPFS
func uploadBlockToIPFS(block Block) {
	// Implement your IPFS upload logic here
	//fmt.Printf("Uploading Block to IPFS: %+v\n", block)
	// Here you would typically use a Go IPFS client to upload the block to IPFS
	// Example: ipfs.AddBlock(block)
}

// addTransaction adds a new transaction to the transaction pool
func addTransaction(transaction Transaction) {
	mutex.Lock()
	defer mutex.Unlock()
	transactionPool = append(transactionPool, transaction)
}

// handleReceive handles incoming requests with transaction hashes
func handleReceive(w http.ResponseWriter, r *http.Request) {
	// Log the client's IP address
	clientIP := strings.Split(r.RemoteAddr, ":")[0] // Extract IP address only
	fmt.Printf("Received request from IP: %s\n", clientIP)

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

	// Add transaction to pool
	addTransaction(Transaction{ID: clientIP, Data: result})

	// Start mining the block
	go mineBlock(clientIP, 4)

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
