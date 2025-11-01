package crypto

import (
	"testing"
)

// BenchmarkKeyGeneration benchmarks hybrid key generation
func BenchmarkKeyGeneration(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := GenerateSigningKey()
		if err != nil {
			b.Fatalf("Failed to generate key: %v", err)
		}
	}
}

// BenchmarkSign benchmarks hybrid signature generation
// Target: <10ms per operation
func BenchmarkSign(b *testing.B) {
	// Setup: generate key pair once
	signingKey, err := GenerateSigningKey()
	if err != nil {
		b.Fatalf("Failed to generate signing key: %v", err)
	}

	message := []byte("This is a test message for benchmarking hybrid signatures in ShadowMesh")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := Sign(signingKey, message)
		if err != nil {
			b.Fatalf("Failed to sign message: %v", err)
		}
	}
}

// BenchmarkVerify benchmarks hybrid signature verification
// Target: <5ms per operation
func BenchmarkVerify(b *testing.B) {
	// Setup: generate key pair and signature once
	signingKey, err := GenerateSigningKey()
	if err != nil {
		b.Fatalf("Failed to generate signing key: %v", err)
	}

	verifyKey := signingKey.PublicKey()
	message := []byte("This is a test message for benchmarking hybrid signatures in ShadowMesh")

	signature, err := Sign(signingKey, message)
	if err != nil {
		b.Fatalf("Failed to sign message: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := Verify(verifyKey, message, signature)
		if err != nil {
			b.Fatalf("Failed to verify signature: %v", err)
		}
	}
}

// BenchmarkSignAndVerify benchmarks the full sign+verify cycle
func BenchmarkSignAndVerify(b *testing.B) {
	// Setup: generate key pair once
	signingKey, err := GenerateSigningKey()
	if err != nil {
		b.Fatalf("Failed to generate signing key: %v", err)
	}

	verifyKey := signingKey.PublicKey()
	message := []byte("This is a test message for benchmarking hybrid signatures in ShadowMesh")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		signature, err := Sign(signingKey, message)
		if err != nil {
			b.Fatalf("Failed to sign message: %v", err)
		}

		err = Verify(verifyKey, message, signature)
		if err != nil {
			b.Fatalf("Failed to verify signature: %v", err)
		}
	}
}

// BenchmarkPublicKeyHash benchmarks public key hash generation
func BenchmarkPublicKeyHash(b *testing.B) {
	// Setup: generate key pair once
	signingKey, err := GenerateSigningKey()
	if err != nil {
		b.Fatalf("Failed to generate signing key: %v", err)
	}

	verifyKey := signingKey.PublicKey()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = PublicKeyHash(verifyKey)
	}
}

// BenchmarkVerifyKeyBytes benchmarks public key serialization
func BenchmarkVerifyKeyBytes(b *testing.B) {
	// Setup: generate key pair once
	signingKey, err := GenerateSigningKey()
	if err != nil {
		b.Fatalf("Failed to generate signing key: %v", err)
	}

	verifyKey := signingKey.PublicKey()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := verifyKey.Bytes()
		if err != nil {
			b.Fatalf("Failed to serialize public key: %v", err)
		}
	}
}

// BenchmarkSignSmallMessage benchmarks signing small messages (32 bytes)
func BenchmarkSignSmallMessage(b *testing.B) {
	signingKey, err := GenerateSigningKey()
	if err != nil {
		b.Fatalf("Failed to generate signing key: %v", err)
	}

	message := make([]byte, 32)
	for i := range message {
		message[i] = byte(i)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := Sign(signingKey, message)
		if err != nil {
			b.Fatalf("Failed to sign message: %v", err)
		}
	}
}

// BenchmarkSignLargeMessage benchmarks signing large messages (1 MB)
func BenchmarkSignLargeMessage(b *testing.B) {
	signingKey, err := GenerateSigningKey()
	if err != nil {
		b.Fatalf("Failed to generate signing key: %v", err)
	}

	message := make([]byte, 1024*1024)
	for i := range message {
		message[i] = byte(i % 256)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := Sign(signingKey, message)
		if err != nil {
			b.Fatalf("Failed to sign message: %v", err)
		}
	}
}

// BenchmarkVerifySmallMessage benchmarks verifying small messages (32 bytes)
func BenchmarkVerifySmallMessage(b *testing.B) {
	signingKey, err := GenerateSigningKey()
	if err != nil {
		b.Fatalf("Failed to generate signing key: %v", err)
	}

	verifyKey := signingKey.PublicKey()

	message := make([]byte, 32)
	for i := range message {
		message[i] = byte(i)
	}

	signature, err := Sign(signingKey, message)
	if err != nil {
		b.Fatalf("Failed to sign message: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := Verify(verifyKey, message, signature)
		if err != nil {
			b.Fatalf("Failed to verify signature: %v", err)
		}
	}
}

// BenchmarkVerifyLargeMessage benchmarks verifying large messages (1 MB)
func BenchmarkVerifyLargeMessage(b *testing.B) {
	signingKey, err := GenerateSigningKey()
	if err != nil {
		b.Fatalf("Failed to generate signing key: %v", err)
	}

	verifyKey := signingKey.PublicKey()

	message := make([]byte, 1024*1024)
	for i := range message {
		message[i] = byte(i % 256)
	}

	signature, err := Sign(signingKey, message)
	if err != nil {
		b.Fatalf("Failed to sign message: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := Verify(verifyKey, message, signature)
		if err != nil {
			b.Fatalf("Failed to verify signature: %v", err)
		}
	}
}

// BenchmarkPublicKeyExtraction benchmarks extracting public key from signing key
func BenchmarkPublicKeyExtraction(b *testing.B) {
	signingKey, err := GenerateSigningKey()
	if err != nil {
		b.Fatalf("Failed to generate signing key: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = signingKey.PublicKey()
	}
}

// BenchmarkParallelSign benchmarks signing with parallel goroutines
func BenchmarkParallelSign(b *testing.B) {
	signingKey, err := GenerateSigningKey()
	if err != nil {
		b.Fatalf("Failed to generate signing key: %v", err)
	}

	message := []byte("Parallel signing benchmark for ShadowMesh")

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, err := Sign(signingKey, message)
			if err != nil {
				b.Fatalf("Failed to sign message: %v", err)
			}
		}
	})
}

// BenchmarkParallelVerify benchmarks verification with parallel goroutines
func BenchmarkParallelVerify(b *testing.B) {
	signingKey, err := GenerateSigningKey()
	if err != nil {
		b.Fatalf("Failed to generate signing key: %v", err)
	}

	verifyKey := signingKey.PublicKey()
	message := []byte("Parallel verification benchmark for ShadowMesh")

	signature, err := Sign(signingKey, message)
	if err != nil {
		b.Fatalf("Failed to sign message: %v", err)
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			err := Verify(verifyKey, message, signature)
			if err != nil {
				b.Fatalf("Failed to verify signature: %v", err)
			}
		}
	})
}
