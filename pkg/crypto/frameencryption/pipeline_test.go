package frameencryption

import (
	"context"
	"crypto/rand"
	"testing"
	"time"

	"github.com/shadowmesh/shadowmesh/pkg/crypto/symmetric"
	"github.com/shadowmesh/shadowmesh/pkg/layer2"
)

// TestPipelineCreation tests pipeline initialization
func TestPipelineCreation(t *testing.T) {
	key := generateTestKey()

	config := &PipelineConfig{
		Key:        key,
		BufferSize: 100,
	}

	pipeline, err := NewEncryptionPipeline(config)
	if err != nil {
		t.Fatalf("Failed to create pipeline: %v", err)
	}
	defer pipeline.Stop()

	if pipeline.bufferSize != 100 {
		t.Errorf("Expected buffer size 100, got %d", pipeline.bufferSize)
	}

	if pipeline.nonceGen == nil {
		t.Error("Nonce generator not initialized")
	}
}

// TestEncryptionDecryption tests end-to-end frame encryption and decryption
func TestEncryptionDecryption(t *testing.T) {
	key := generateTestKey()

	config := &PipelineConfig{
		Key:        key,
		BufferSize: 10,
	}

	pipeline, err := NewEncryptionPipeline(config)
	if err != nil {
		t.Fatalf("Failed to create pipeline: %v", err)
	}
	defer pipeline.Stop()

	// Start pipeline
	pipeline.Start()

	// Create test frame
	testFrame := createTestFrame()

	// Send frame for encryption
	if !pipeline.SendFrame(testFrame) {
		t.Fatal("Failed to send frame for encryption")
	}

	// Receive encrypted frame
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	encryptedFrame, err := pipeline.ReceiveEncryptedFrame(ctx)
	if err != nil {
		t.Fatalf("Failed to receive encrypted frame: %v", err)
	}

	// Verify encryption added authentication tag (16 bytes)
	if len(encryptedFrame.Frame.Ciphertext) != len(testFrame.Serialize())+symmetric.TagSize {
		t.Errorf("Expected ciphertext size %d, got %d",
			len(testFrame.Serialize())+symmetric.TagSize,
			len(encryptedFrame.Frame.Ciphertext))
	}

	// Send encrypted frame for decryption
	if !pipeline.SendEncryptedFrame(encryptedFrame) {
		t.Fatal("Failed to send encrypted frame for decryption")
	}

	// Receive decrypted frame
	decryptedBytes, err := pipeline.ReceiveDecryptedFrame(ctx)
	if err != nil {
		t.Fatalf("Failed to receive decrypted frame: %v", err)
	}

	// Verify decrypted frame matches original
	originalBytes := testFrame.Serialize()
	if len(decryptedBytes) != len(originalBytes) {
		t.Errorf("Decrypted frame size mismatch: got %d, expected %d",
			len(decryptedBytes), len(originalBytes))
	}

	for i := range originalBytes {
		if decryptedBytes[i] != originalBytes[i] {
			t.Errorf("Byte %d mismatch: got %02x, expected %02x",
				i, decryptedBytes[i], originalBytes[i])
			break
		}
	}
}

// TestInvalidTagRejection tests that frames with invalid authentication tags are dropped
func TestInvalidTagRejection(t *testing.T) {
	key := generateTestKey()

	config := &PipelineConfig{
		Key:        key,
		BufferSize: 10,
	}

	pipeline, err := NewEncryptionPipeline(config)
	if err != nil {
		t.Fatalf("Failed to create pipeline: %v", err)
	}
	defer pipeline.Stop()

	pipeline.Start()

	// Create encrypted frame with tampered ciphertext
	testFrame := createTestFrame()
	plaintext := testFrame.Serialize()

	nonce, _ := pipeline.nonceGen.GenerateNonce()
	encrypted, _ := symmetric.Encrypt(plaintext, key, nonce)

	// Tamper with ciphertext (flip a bit)
	encrypted.Ciphertext[0] ^= 0x01

	// Send tampered frame for decryption
	tamperedFrame := &EncryptedEthernetFrame{
		Frame:     encrypted,
		Timestamp: time.Now(),
	}

	if !pipeline.SendEncryptedFrame(tamperedFrame) {
		t.Fatal("Failed to send tampered frame")
	}

	// Try to receive decrypted frame (should timeout - frame was dropped)
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	_, err = pipeline.ReceiveDecryptedFrame(ctx)
	if err == nil {
		t.Error("Expected timeout for tampered frame, but received frame")
	}

	// Verify dropped count increased
	metrics := pipeline.GetMetrics()
	if metrics.DroppedCount != 1 {
		t.Errorf("Expected 1 dropped frame, got %d", metrics.DroppedCount)
	}
}

// TestReplayProtection tests that nonces are unique (frame counter monotonically increases)
func TestReplayProtection(t *testing.T) {
	key := generateTestKey()

	config := &PipelineConfig{
		Key:        key,
		BufferSize: 100,
	}

	pipeline, err := NewEncryptionPipeline(config)
	if err != nil {
		t.Fatalf("Failed to create pipeline: %v", err)
	}
	defer pipeline.Stop()

	pipeline.Start()

	// Track nonces to verify uniqueness
	seenNonces := make(map[[symmetric.NonceSize]byte]bool)

	testFrame := createTestFrame()

	// Send multiple frames
	numFrames := 100
	for i := 0; i < numFrames; i++ {
		if !pipeline.SendFrame(testFrame) {
			t.Fatalf("Failed to send frame %d", i)
		}
	}

	// Receive and verify nonces are unique
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	for i := 0; i < numFrames; i++ {
		encFrame, err := pipeline.ReceiveEncryptedFrame(ctx)
		if err != nil {
			t.Fatalf("Failed to receive encrypted frame %d: %v", i, err)
		}

		nonce := encFrame.Frame.Nonce
		if seenNonces[nonce] {
			t.Errorf("Duplicate nonce detected at frame %d", i)
		}
		seenNonces[nonce] = true
	}

	if len(seenNonces) != numFrames {
		t.Errorf("Expected %d unique nonces, got %d", numFrames, len(seenNonces))
	}
}

// TestPipelineThroughput tests pipeline performance (>10,000 frames/sec requirement)
func TestPipelineThroughput(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping throughput test in short mode")
	}

	key := generateTestKey()

	config := &PipelineConfig{
		Key:        key,
		BufferSize: 1000, // Larger buffer for throughput test
	}

	pipeline, err := NewEncryptionPipeline(config)
	if err != nil {
		t.Fatalf("Failed to create pipeline: %v", err)
	}
	defer pipeline.Stop()

	pipeline.Start()

	testFrame := createTestFrame()
	numFrames := 10000

	// Send and receive concurrently to prevent blocking
	ctx := context.Background()
	start := time.Now()

	// Goroutine to send frames
	sendDone := make(chan bool)
	go func() {
		for i := 0; i < numFrames; i++ {
			for !pipeline.SendFrame(testFrame) {
				time.Sleep(1 * time.Microsecond)
			}
		}
		close(sendDone)
	}()

	// Goroutine to receive encrypted frames
	receiveDone := make(chan bool)
	go func() {
		for i := 0; i < numFrames; i++ {
			_, err := pipeline.ReceiveEncryptedFrame(ctx)
			if err != nil {
				t.Errorf("Failed to receive frame %d: %v", i, err)
				break
			}
		}
		close(receiveDone)
	}()

	// Wait for both to complete
	<-sendDone
	<-receiveDone

	elapsed := time.Since(start)
	throughput := float64(numFrames) / elapsed.Seconds()

	t.Logf("Encrypted %d frames in %v", numFrames, elapsed)
	t.Logf("Throughput: %.0f frames/sec", throughput)

	// AC #6: Pipeline must process >10,000 frames/second
	if throughput < 10000 {
		t.Errorf("Throughput too low: %.0f frames/sec (expected >10,000)", throughput)
	} else {
		t.Logf("✅ Throughput requirement met: %.0f frames/sec", throughput)
	}
}

// TestPipelineMetrics tests pipeline metrics collection
func TestPipelineMetrics(t *testing.T) {
	key := generateTestKey()

	config := &PipelineConfig{
		Key:        key,
		BufferSize: 10,
	}

	pipeline, err := NewEncryptionPipeline(config)
	if err != nil {
		t.Fatalf("Failed to create pipeline: %v", err)
	}
	defer pipeline.Stop()

	pipeline.Start()

	testFrame := createTestFrame()

	// Send and process 10 frames
	numFrames := 10
	for i := 0; i < numFrames; i++ {
		pipeline.SendFrame(testFrame)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	for i := 0; i < numFrames; i++ {
		encFrame, _ := pipeline.ReceiveEncryptedFrame(ctx)
		pipeline.SendEncryptedFrame(encFrame)
	}

	for i := 0; i < numFrames; i++ {
		_, _ = pipeline.ReceiveDecryptedFrame(ctx)
	}

	// Check metrics
	metrics := pipeline.GetMetrics()

	if metrics.EncryptedCount != uint64(numFrames) {
		t.Errorf("Expected %d encrypted frames, got %d", numFrames, metrics.EncryptedCount)
	}

	if metrics.DecryptedCount != uint64(numFrames) {
		t.Errorf("Expected %d decrypted frames, got %d", numFrames, metrics.DecryptedCount)
	}

	throughput := metrics.GetThroughput()
	t.Logf("Throughput: %.0f frames/sec", throughput)

	if throughput == 0 {
		t.Error("Throughput calculation returned 0")
	}
}

// TestPipelineGracefulShutdown tests graceful pipeline shutdown
func TestPipelineGracefulShutdown(t *testing.T) {
	key := generateTestKey()

	config := &PipelineConfig{
		Key:        key,
		BufferSize: 10,
	}

	pipeline, err := NewEncryptionPipeline(config)
	if err != nil {
		t.Fatalf("Failed to create pipeline: %v", err)
	}

	pipeline.Start()

	// Send a few frames
	testFrame := createTestFrame()
	for i := 0; i < 5; i++ {
		pipeline.SendFrame(testFrame)
	}

	// Stop pipeline gracefully
	pipeline.Stop()

	// Verify receive returns error after stop (channels closed)
	ctx := context.Background()
	_, err = pipeline.ReceiveEncryptedFrame(ctx)
	if err == nil {
		t.Error("Expected error when receiving after stop")
	}

	// Note: SendFrame() will panic after Stop() since channels are closed
	// This is expected behavior - users should not call SendFrame() after Stop()
}

// Helper functions

func generateTestKey() [symmetric.KeySize]byte {
	var key [symmetric.KeySize]byte
	rand.Read(key[:])
	return key
}

func createTestFrame() *layer2.EthernetFrame {
	return &layer2.EthernetFrame{
		DestinationMAC: [6]byte{0x00, 0x11, 0x22, 0x33, 0x44, 0x55},
		SourceMAC:      [6]byte{0xAA, 0xBB, 0xCC, 0xDD, 0xEE, 0xFF},
		EtherType:      0x0800, // IPv4
		Payload:        []byte("Test payload data for encryption"),
	}
}
