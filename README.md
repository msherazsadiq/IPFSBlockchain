# IPFS-Based Blockchain with Distributed Computation

## Description
This project implements a decentralized system that integrates IPFS, blockchain, and distributed computation. The client uploads a Python algorithm and input data to IPFS and sends the content hashes to peer nodes over a Tailscale network. Miner nodes download the files from IPFS, execute the computation, store the output as transactions, and mine blocks using a Proof-of-Work (PoW) mechanism. The system demonstrates decentralized storage, peer-to-peer communication, and blockchain-based result verification.

---

## Features
- Upload files to IPFS and retrieve CIDs
- Peer-to-peer communication using Tailscale
- Remote execution of Python computation
- Transaction pool management
- Proof-of-Work based block mining
- Distributed miner nodes


---

## How It Works
1. Client uploads `algo.py` and `data.txt` to IPFS.
2. IPFS returns content hashes (CIDs).
3. Client sends hashes to Tailscale-connected peers.
4. Miner nodes download files from IPFS.
5. Miner executes the Python script on the input file.
6. Output is stored as a transaction.
7. After collecting transactions, a block is mined using PoW.

---

## Requirements
- Go (1.18+)
- Python 3
- IPFS (daemon running)
- Tailscale (for peer networking)

---


