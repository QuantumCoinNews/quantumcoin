package p2p

import (
	"bufio"
	"bytes"
	"encoding/gob"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"quantumcoin/blockchain"
	"quantumcoin/config"
	"quantumcoin/wallet"
)

func syncFixture(t *testing.T, count int, padding bool) *blockchain.Blockchain {
	t.Helper()
	chain := blockchain.NewBlockchain(config.Default().InitialReward, config.Default().TotalSupply)
	for i := 0; i < count; i++ {
		if _, err := chain.MineBlock(wallet.NewWallet().GetAddress(), 4); err != nil {
			t.Fatal(err)
		}
		// Legacy metadata is informational; populated before sharing the fixture.
		if padding {
			chain.Blocks[len(chain.Blocks)-1].Metadata["test-padding"] = strings.Repeat("x", 96<<10)
		}
	}
	return chain
}

// This helper runs as an independent node process, including under -race.
func TestSyncNodeProcess(t *testing.T) {
	role := os.Getenv("QC_SYNC_TEST_ROLE")
	if role == "" {
		t.Skip("subprocess helper")
	}
	data, err := os.ReadFile(os.Getenv("QC_SYNC_TEST_INPUT"))
	if err != nil {
		t.Fatal(err)
	}
	chain := blockchain.DeserializeBlockchain(data)
	if chain == nil || !chain.IsValidChainSynchronized() {
		t.Fatal("invalid test node fixture")
	}
	stop := make(chan struct{})
	go func() { _, _ = bufio.NewReader(os.Stdin).ReadByte(); close(stop) }()
	if role == "source" {
		listener, err := net.Listen("tcp", "127.0.0.1:0")
		if err != nil {
			t.Fatal(err)
		}
		defer listener.Close()
		go func() { <-stop; _ = listener.Close(); CloseAllPeersV1() }()
		fmt.Println("READY " + listener.Addr().String())
		var mining sync.Once
		var workers sync.WaitGroup
		for {
			conn, err := listener.Accept()
			if err != nil {
				break
			}
			registerPeer(conn)
			workers.Add(1)
			go func() { defer workers.Done(); HandleConnection(conn, chain) }()
			sendToPeer(conn, HandshakeMessage(chain))
			mining.Do(func() {
				workers.Add(1)
				go func() {
					defer workers.Done()
					for i := 0; i < 4; i++ {
						if _, err := chain.MineBlock(wallet.NewWallet().GetAddress(), 4); err != nil {
							t.Error(err)
							return
						}
						time.Sleep(15 * time.Millisecond)
					}
				}()
			})
		}
		workers.Wait()
		return
	}
	if role != "target" {
		t.Fatal("unknown node role")
	}
	defer CloseAllPeersV1()
	expected, _ := strconv.Atoi(os.Getenv("QC_SYNC_TEST_HEIGHT"))
	saved := false
	for {
		select {
		case <-stop:
			return
		case <-time.After(100 * time.Millisecond):
			if chain.BestHeightSnapshot() >= expected && !saved {
				if err := chain.SaveToFile(os.Getenv("QC_SYNC_TEST_RESULT")); err != nil {
					t.Fatal(err)
				}
				saved = true
			}
			if !saved && GetPeerCount() == 0 {
				ConnectToPeer("", os.Getenv("QC_SYNC_TEST_REMOTE"), chain)
			}
		}
	}
}

func scanSyncNodeReadiness(output io.Reader, ready chan<- string) error {
	scanner := bufio.NewScanner(output)
	for scanner.Scan() {
		if strings.HasPrefix(scanner.Text(), "READY ") {
			ready <- strings.TrimPrefix(scanner.Text(), "READY ")
		}
	}
	return scanner.Err()
}

func launchSyncNode(t *testing.T, role, input, remote, result string, height int) string {
	t.Helper()
	cmd := exec.Command(os.Args[0], "-test.run=^TestSyncNodeProcess$", "-test.timeout=90s")
	cmd.Env = append(os.Environ(), "QC_SYNC_TEST_ROLE="+role, "QC_SYNC_TEST_INPUT="+input,
		"QC_SYNC_TEST_REMOTE="+remote, "QC_SYNC_TEST_RESULT="+result,
		"QC_SYNC_TEST_HEIGHT="+strconv.Itoa(height), "QC_NODE_DIR="+t.TempDir())
	stdin, err := cmd.StdinPipe()
	if err != nil {
		t.Fatal(err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	readDone := make(chan struct{})
	var readErr error
	ready := make(chan string, 1)
	go func() {
		defer close(readDone)
		readErr = scanSyncNodeReadiness(stdout, ready)
	}()
	t.Cleanup(func() {
		_ = stdin.Close()
		// Drain StdoutPipe before Wait closes it, and preserve read failures.
		select {
		case <-readDone:
		case <-time.After(10 * time.Second):
			_ = cmd.Process.Kill()
			<-readDone
			t.Errorf("%s node output did not close", role)
		}
		if readErr != nil {
			t.Errorf("%s node output failed: %v", role, readErr)
		}
		done := make(chan error, 1)
		go func() { done <- cmd.Wait() }()
		select {
		case err := <-done:
			if err != nil {
				t.Errorf("%s node failed: %v\n%s", role, err, stderr.String())
			}
		case <-time.After(10 * time.Second):
			_ = cmd.Process.Kill()
			<-done
			t.Errorf("%s node did not shut down", role)
		}
		<-readDone
	})
	if role == "source" {
		select {
		case address := <-ready:
			return address
		case <-time.After(15 * time.Second):
			t.Fatal("source node did not become ready")
		}
	}
	return ""
}

func TestTwoProcessSyncLargeForkReconnectAndMining(t *testing.T) {
	t.Setenv("QC_DIFFICULTY_BITS", "4")
	t.Setenv("QC_NODE_DIR", t.TempDir())
	full := syncFixture(t, 20, true)
	fork := syncFixture(t, 3, false)
	serialized := blockchain.SerializeBlockchain(full)
	if len(serialized) <= MaxMessageSize {
		t.Fatal("fixture does not exceed legacy message limit")
	}
	directory := t.TempDir()
	sourcePath, targetPath, result := filepath.Join(directory, "source.fixture"), filepath.Join(directory, "target.fixture"), filepath.Join(directory, "result.dat")
	if err := os.WriteFile(sourcePath, serialized, 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(targetPath, blockchain.SerializeBlockchain(fork), 0600); err != nil {
		t.Fatal(err)
	}
	address := launchSyncNode(t, "source", sourcePath, "", "", 0)
	proxy, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = proxy.Close() })
	var mu sync.Mutex
	connections, chunks, maxBytes, resumed := 0, 0, 0, false
	var connectionsToClose []net.Conn
	t.Cleanup(func() {
		mu.Lock()
		defer mu.Unlock()
		for _, conn := range connectionsToClose {
			_ = conn.Close()
		}
	})
	go func() {
		for {
			client, err := proxy.Accept()
			if err != nil {
				return
			}
			server, err := net.DialTimeout("tcp", address, 5*time.Second)
			if err != nil {
				_ = client.Close()
				return
			}
			mu.Lock()
			connections++
			number := connections
			connectionsToClose = append(connectionsToClose, client, server)
			mu.Unlock()
			pump := func(from, to net.Conn, fromSource bool) {
				defer client.Close()
				defer server.Close()
				decoder, encoder := gob.NewDecoder(from), gob.NewEncoder(to)
				for {
					var msg Message
					if err := decoder.Decode(&msg); err != nil {
						return
					}
					mu.Lock()
					if len(msg.Data) > maxBytes {
						maxBytes = len(msg.Data)
					}
					if !fromSource && msg.Type == MsgSyncRequest && number > 1 {
						var request syncRequest
						if gob.NewDecoder(bytes.NewReader(msg.Data)).Decode(&request) == nil && request.Start > 0 {
							resumed = true
						}
					}
					if fromSource && msg.Type == MsgSyncChunk {
						chunks++
					}
					cut := fromSource && msg.Type == MsgSyncChunk && number == 1 && chunks == 2
					mu.Unlock()
					if err := encoder.Encode(msg); err != nil {
						return
					}
					if cut {
						return
					}
				}
			}
			go pump(server, client, true)
			go pump(client, server, false)
		}
	}()
	launchSyncNode(t, "target", targetPath, proxy.Addr().String(), result, 20)
	deadline := time.Now().Add(60 * time.Second)
	for time.Now().Before(deadline) {
		if _, err := os.Stat(result); err == nil {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	loaded, err := blockchain.LoadBlockchainFromFile(result)
	if err != nil {
		t.Fatalf("two-node sync did not finish: %v", err)
	}
	blocks := loaded.BlocksSnapshot()
	if len(blocks) < 21 || !bytes.Equal(blocks[20].Hash, full.Blocks[20].Hash) {
		t.Fatal("wrong branch installed")
	}
	for _, block := range blocks[1:21] {
		if len(block.Metadata["test-padding"]) != 96<<10 {
			t.Fatal("history content lost")
		}
	}
	mu.Lock()
	defer mu.Unlock()
	if connections < 2 || !resumed || chunks < 3 || maxBytes > MaxMessageSize {
		t.Fatalf("connections=%d resumed=%v chunks=%d maxPayload=%d", connections, resumed, chunks, maxBytes)
	}
	t.Logf("two processes: %d history bytes; %d connections; %d chunks; largest payload %d; resumed=%v", len(serialized), connections, chunks, maxBytes, resumed)
}

func syncTestConnection(t *testing.T, chain *blockchain.Blockchain) (net.Conn, syncRequest) {
	t.Helper()
	left, right := net.Pipe()
	registerPeer(left)
	t.Cleanup(func() {
		_ = right.Close()
		unregisterPeer(left)
		syncDownloads.Lock()
		delete(syncDownloads.chains, chain)
		syncDownloads.Unlock()
	})
	done := make(chan error, 1)
	go func() { done <- startChainSync(chain, left) }()
	var message Message
	if err := gob.NewDecoder(right).Decode(&message); err != nil {
		t.Fatal(err)
	}
	if err := <-done; err != nil {
		t.Fatal(err)
	}
	var request syncRequest
	if err := gob.NewDecoder(bytes.NewReader(message.Data)).Decode(&request); err != nil {
		t.Fatal(err)
	}
	return left, request
}

func TestSyncRejectsMalformedReplayAndWrongGenesis(t *testing.T) {
	t.Setenv("QC_DIFFICULTY_BITS", "4")
	t.Setenv("QC_NODE_DIR", t.TempDir())
	for _, kind := range []string{"id", "position", "count", "genesis", "pow", "tip"} {
		t.Run(kind, func(t *testing.T) {
			local, source := syncFixture(t, 0, false), syncFixture(t, 1, false)
			conn, request := syncTestConnection(t, local)
			blocks := source.BlocksSnapshot()
			chunk := syncChunk{ID: request.ID, Start: 0, TipHeight: 1, TipHash: blocks[1].Hash, Blocks: blocks}
			switch kind {
			case "id":
				chunk.ID[0] ^= 1
			case "position":
				chunk.Start = 1
			case "count":
				chunk.Blocks = make([]*blockchain.Block, 9)
				for i := range chunk.Blocks {
					chunk.Blocks[i] = blocks[0]
				}
			case "genesis":
				chunk.Blocks[0].Hash[0] ^= 1
			case "pow":
				chunk.Blocks[1].Hash[0] ^= 1
			case "tip":
				chunk.TipHash = append([]byte(nil), chunk.TipHash...)
				chunk.TipHash[0] ^= 1
			}
			msg, err := encodeSync(MsgSyncChunk, chunk)
			if err != nil {
				t.Fatal(err)
			}
			if err := acceptSyncChunk(local, conn, msg.Data); err == nil {
				t.Fatal("invalid chunk accepted")
			}
			if local.BestHeightSnapshot() != 0 {
				t.Fatal("invalid candidate changed canonical chain")
			}
		})
	}
	local, source := syncFixture(t, 0, false), syncFixture(t, 1, false)
	conn, request := syncTestConnection(t, local)
	blocks := source.BlocksSnapshot()
	msg, err := encodeSync(MsgSyncChunk, syncChunk{ID: request.ID, TipHeight: 1, TipHash: blocks[1].Hash, Blocks: blocks})
	if err != nil {
		t.Fatal(err)
	}
	if err := acceptSyncChunk(local, conn, msg.Data); err != nil {
		t.Fatal(err)
	}
	if err := acceptSyncChunk(local, conn, msg.Data); err == nil {
		t.Fatal("completed chunk replay accepted")
	}
	if err := ValidateP2PMessageSizeStrictV1(Message{Type: MsgSyncChunk, Data: make([]byte, MaxMessageSize+1)}); err == nil {
		t.Fatal("oversized chunk accepted")
	}
}
