package crypto

import (
	"crypto/rand"
	"fmt"
	"testing"
)

// Benchmark encryption of 1500-byte Ethernet frames (typical MTU)
// Target: 1+ Gbps throughput = ~83,000 frames/second = ~12μs per frame
func BenchmarkEncrypt1500(b *testing.B) {
	key := generateTestKey()
	encryptor, err := NewFrameEncryptor(key)
	if err != nil {
		b.Fatalf("NewFrameEncryptor failed: %v", err)
	}

	// Typical Ethernet frame size (1500 bytes MTU)
	plaintext := make([]byte, 1500)
	rand.Read(plaintext)

	b.ResetTimer()
	b.SetBytes(1500) // Track throughput

	for i := 0; i < b.N; i++ {
		_, err := encryptor.Encrypt(plaintext)
		if err != nil {
			b.Fatalf("Encrypt failed: %v", err)
		}
	}

	b.StopTimer()

	// Calculate and report throughput
	nsPerOp := float64(b.Elapsed().Nanoseconds()) / float64(b.N)
	framesPerSec := 1e9 / nsPerOp
	gbps := (framesPerSec * 1500 * 8) / 1e9

	b.ReportMetric(framesPerSec, "frames/sec")
	b.ReportMetric(gbps, "Gbps")
}

// Benchmark decryption of 1500-byte Ethernet frames
func BenchmarkDecrypt1500(b *testing.B) {
	key := generateTestKey()
	encryptor, err := NewFrameEncryptor(key)
	if err != nil {
		b.Fatalf("NewFrameEncryptor failed: %v", err)
	}

	plaintext := make([]byte, 1500)
	rand.Read(plaintext)

	// Pre-encrypt frames to avoid measuring encryption time
	ciphertexts := make([][]byte, b.N)
	for i := 0; i < b.N; i++ {
		ct, err := encryptor.Encrypt(plaintext)
		if err != nil {
			b.Fatalf("Encrypt failed: %v", err)
		}
		ciphertexts[i] = ct
	}

	b.ResetTimer()
	b.SetBytes(1500)

	for i := 0; i < b.N; i++ {
		_, err := encryptor.Decrypt(ciphertexts[i])
		if err != nil {
			b.Fatalf("Decrypt failed: %v", err)
		}
	}

	b.StopTimer()

	// Calculate and report throughput
	nsPerOp := float64(b.Elapsed().Nanoseconds()) / float64(b.N)
	framesPerSec := 1e9 / nsPerOp
	gbps := (framesPerSec * 1500 * 8) / 1e9

	b.ReportMetric(framesPerSec, "frames/sec")
	b.ReportMetric(gbps, "Gbps")
}

// Benchmark encryption/decryption roundtrip
func BenchmarkRoundtrip1500(b *testing.B) {
	key := generateTestKey()
	encryptor, err := NewFrameEncryptor(key)
	if err != nil {
		b.Fatalf("NewFrameEncryptor failed: %v", err)
	}

	plaintext := make([]byte, 1500)
	rand.Read(plaintext)

	b.ResetTimer()
	b.SetBytes(1500)

	for i := 0; i < b.N; i++ {
		ciphertext, err := encryptor.Encrypt(plaintext)
		if err != nil {
			b.Fatalf("Encrypt failed: %v", err)
		}

		_, err = encryptor.Decrypt(ciphertext)
		if err != nil {
			b.Fatalf("Decrypt failed: %v", err)
		}
	}

	b.StopTimer()

	nsPerOp := float64(b.Elapsed().Nanoseconds()) / float64(b.N)
	framesPerSec := 1e9 / nsPerOp
	gbps := (framesPerSec * 1500 * 8) / 1e9

	b.ReportMetric(framesPerSec, "frames/sec")
	b.ReportMetric(gbps, "Gbps")
}

// Benchmark with various frame sizes
func BenchmarkEncryptVariousSizes(b *testing.B) {
	key := generateTestKey()
	encryptor, err := NewFrameEncryptor(key)
	if err != nil {
		b.Fatalf("NewFrameEncryptor failed: %v", err)
	}

	sizes := []int{64, 128, 256, 512, 1024, 1500, 4096, 9000}

	for _, size := range sizes {
		b.Run(fmt.Sprintf("%dB", size), func(b *testing.B) {
			plaintext := make([]byte, size)
			rand.Read(plaintext)

			b.ResetTimer()
			b.SetBytes(int64(size))

			for i := 0; i < b.N; i++ {
				_, err := encryptor.Encrypt(plaintext)
				if err != nil {
					b.Fatalf("Encrypt failed: %v", err)
				}
			}

			b.StopTimer()

			nsPerOp := float64(b.Elapsed().Nanoseconds()) / float64(b.N)
			framesPerSec := 1e9 / nsPerOp
			gbps := (framesPerSec * float64(size) * 8) / 1e9

			b.ReportMetric(framesPerSec, "frames/sec")
			b.ReportMetric(gbps, "Gbps")
		})
	}
}

// Benchmark decryption with various frame sizes
func BenchmarkDecryptVariousSizes(b *testing.B) {
	key := generateTestKey()
	encryptor, err := NewFrameEncryptor(key)
	if err != nil {
		b.Fatalf("NewFrameEncryptor failed: %v", err)
	}

	sizes := []int{64, 128, 256, 512, 1024, 1500, 4096, 9000}

	for _, size := range sizes {
		b.Run(fmt.Sprintf("%dB", size), func(b *testing.B) {
			plaintext := make([]byte, size)
			rand.Read(plaintext)

			// Pre-encrypt frames
			ciphertexts := make([][]byte, b.N)
			for i := 0; i < b.N; i++ {
				ct, err := encryptor.Encrypt(plaintext)
				if err != nil {
					b.Fatalf("Encrypt failed: %v", err)
				}
				ciphertexts[i] = ct
			}

			b.ResetTimer()
			b.SetBytes(int64(size))

			for i := 0; i < b.N; i++ {
				_, err := encryptor.Decrypt(ciphertexts[i])
				if err != nil {
					b.Fatalf("Decrypt failed: %v", err)
				}
			}

			b.StopTimer()

			nsPerOp := float64(b.Elapsed().Nanoseconds()) / float64(b.N)
			framesPerSec := 1e9 / nsPerOp
			gbps := (framesPerSec * float64(size) * 8) / 1e9

			b.ReportMetric(framesPerSec, "frames/sec")
			b.ReportMetric(gbps, "Gbps")
		})
	}
}

// Benchmark nonce generation (should be negligible overhead)
func BenchmarkNonceGeneration(b *testing.B) {
	key := generateTestKey()
	encryptor, err := NewFrameEncryptor(key)
	if err != nil {
		b.Fatalf("NewFrameEncryptor failed: %v", err)
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = encryptor.generateNonce()
	}
}

// Benchmark encryption overhead (encryption vs plaintext copy)
func BenchmarkEncryptionOverhead(b *testing.B) {
	key := generateTestKey()
	encryptor, err := NewFrameEncryptor(key)
	if err != nil {
		b.Fatalf("NewFrameEncryptor failed: %v", err)
	}

	plaintext := make([]byte, 1500)
	rand.Read(plaintext)

	b.Run("Encrypt", func(b *testing.B) {
		b.SetBytes(1500)
		for i := 0; i < b.N; i++ {
			_, err := encryptor.Encrypt(plaintext)
			if err != nil {
				b.Fatalf("Encrypt failed: %v", err)
			}
		}
	})

	b.Run("PlaintextCopy", func(b *testing.B) {
		dst := make([]byte, 1500)
		b.SetBytes(1500)
		for i := 0; i < b.N; i++ {
			copy(dst, plaintext)
		}
	})
}

// Benchmark parallel encryption (simulating multi-core usage)
func BenchmarkEncryptParallel(b *testing.B) {
	key := generateTestKey()
	plaintext := make([]byte, 1500)
	rand.Read(plaintext)

	b.SetBytes(1500)
	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		// Each goroutine needs its own encryptor (not thread-safe)
		encryptor, err := NewFrameEncryptor(key)
		if err != nil {
			b.Fatalf("NewFrameEncryptor failed: %v", err)
		}

		for pb.Next() {
			_, err := encryptor.Encrypt(plaintext)
			if err != nil {
				b.Fatalf("Encrypt failed: %v", err)
			}
		}
	})

	b.StopTimer()

	// Calculate aggregate throughput
	nsPerOp := float64(b.Elapsed().Nanoseconds()) / float64(b.N)
	framesPerSec := 1e9 / nsPerOp
	gbps := (framesPerSec * 1500 * 8) / 1e9

	b.ReportMetric(framesPerSec, "frames/sec")
	b.ReportMetric(gbps, "Gbps")
}

// Benchmark parallel decryption
func BenchmarkDecryptParallel(b *testing.B) {
	key := generateTestKey()
	plaintext := make([]byte, 1500)
	rand.Read(plaintext)

	// Pre-generate encrypted frames (using one encryptor)
	encryptor, err := NewFrameEncryptor(key)
	if err != nil {
		b.Fatalf("NewFrameEncryptor failed: %v", err)
	}

	ciphertext, err := encryptor.Encrypt(plaintext)
	if err != nil {
		b.Fatalf("Encrypt failed: %v", err)
	}

	b.SetBytes(1500)
	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		// Each goroutine needs its own encryptor
		dec, err := NewFrameEncryptor(key)
		if err != nil {
			b.Fatalf("NewFrameEncryptor failed: %v", err)
		}

		for pb.Next() {
			_, err := dec.Decrypt(ciphertext)
			if err != nil {
				b.Fatalf("Decrypt failed: %v", err)
			}
		}
	})

	b.StopTimer()

	nsPerOp := float64(b.Elapsed().Nanoseconds()) / float64(b.N)
	framesPerSec := 1e9 / nsPerOp
	gbps := (framesPerSec * 1500 * 8) / 1e9

	b.ReportMetric(framesPerSec, "frames/sec")
	b.ReportMetric(gbps, "Gbps")
}
