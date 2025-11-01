package crypto

import (
	"testing"
)

func BenchmarkGenerateKeyPair(b *testing.B) {
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, err := GenerateKeyPair()
		if err != nil {
			b.Fatalf("GenerateKeyPair failed: %v", err)
		}
	}
}

func BenchmarkEncapsulate(b *testing.B) {
	// Setup: Generate a recipient keypair once
	recipientKP, err := GenerateKeyPair()
	if err != nil {
		b.Fatalf("Failed to generate recipient keypair: %v", err)
	}

	publicKey := recipientKP.PublicKey()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, _, err := Encapsulate(publicKey)
		if err != nil {
			b.Fatalf("Encapsulate failed: %v", err)
		}
	}
}

func BenchmarkDecapsulate(b *testing.B) {
	// Setup: Generate keypair and ciphertext once
	recipientKP, err := GenerateKeyPair()
	if err != nil {
		b.Fatalf("Failed to generate recipient keypair: %v", err)
	}

	_, ciphertext, err := Encapsulate(recipientKP.PublicKey())
	if err != nil {
		b.Fatalf("Failed to encapsulate: %v", err)
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, err := Decapsulate(recipientKP, ciphertext)
		if err != nil {
			b.Fatalf("Decapsulate failed: %v", err)
		}
	}
}

func BenchmarkEncapsulateDecapsulateRoundtrip(b *testing.B) {
	// This benchmark measures the complete roundtrip time
	// which is critical for the <50ms acceptance criterion

	// Setup: Generate recipient keypair once
	recipientKP, err := GenerateKeyPair()
	if err != nil {
		b.Fatalf("Failed to generate recipient keypair: %v", err)
	}

	publicKey := recipientKP.PublicKey()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		// Encapsulate
		sharedSecret1, ciphertext, err := Encapsulate(publicKey)
		if err != nil {
			b.Fatalf("Encapsulate failed: %v", err)
		}

		// Decapsulate
		sharedSecret2, err := Decapsulate(recipientKP, ciphertext)
		if err != nil {
			b.Fatalf("Decapsulate failed: %v", err)
		}

		// Verify (this is part of a real-world usage)
		if len(sharedSecret1) != len(sharedSecret2) {
			b.Fatal("Shared secret length mismatch")
		}
	}
}

func BenchmarkKEMPublicKeyBytes(b *testing.B) {
	// Setup: Generate keypair once
	kp, err := GenerateKeyPair()
	if err != nil {
		b.Fatalf("Failed to generate keypair: %v", err)
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = kp.PublicKeyBytes()
	}
}

func BenchmarkParsePublicKey(b *testing.B) {
	// Setup: Generate keypair and serialize public key once
	kp, err := GenerateKeyPair()
	if err != nil {
		b.Fatalf("Failed to generate keypair: %v", err)
	}

	pubBytes := kp.PublicKeyBytes()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, err := ParsePublicKey(pubBytes)
		if err != nil {
			b.Fatalf("ParsePublicKey failed: %v", err)
		}
	}
}

func BenchmarkDeriveSharedSecret(b *testing.B) {
	kemSecret := make([]byte, 32)
	ecdhSecret := make([]byte, 32)

	// Fill with some data
	for i := range kemSecret {
		kemSecret[i] = byte(i)
		ecdhSecret[i] = byte(i * 2)
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = deriveSharedSecret(kemSecret, ecdhSecret)
	}
}

func BenchmarkFullKeyExchangeProtocol(b *testing.B) {
	// This benchmark simulates a complete protocol flow:
	// 1. Alice generates keypair
	// 2. Alice sends public key to Bob
	// 3. Bob encapsulates to Alice's public key
	// 4. Bob sends ciphertext to Alice
	// 5. Alice decapsulates ciphertext
	// This is the most realistic benchmark for actual usage

	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		// Alice generates keypair
		aliceKP, err := GenerateKeyPair()
		if err != nil {
			b.Fatalf("Failed to generate Alice's keypair: %v", err)
		}

		// Alice sends public key to Bob (serialize/deserialize)
		alicePubBytes := aliceKP.PublicKeyBytes()
		alicePub, err := ParsePublicKey(alicePubBytes)
		if err != nil {
			b.Fatalf("Failed to parse Alice's public key: %v", err)
		}

		// Bob encapsulates
		sharedSecret1, ciphertext, err := Encapsulate(alicePub)
		if err != nil {
			b.Fatalf("Failed to encapsulate: %v", err)
		}

		// Alice decapsulates
		sharedSecret2, err := Decapsulate(aliceKP, ciphertext)
		if err != nil {
			b.Fatalf("Failed to decapsulate: %v", err)
		}

		// Verify
		if len(sharedSecret1) != len(sharedSecret2) {
			b.Fatal("Shared secret length mismatch")
		}
	}
}

// BenchmarkParallelEncapsulate tests concurrent encapsulation performance
func BenchmarkParallelEncapsulate(b *testing.B) {
	recipientKP, err := GenerateKeyPair()
	if err != nil {
		b.Fatalf("Failed to generate recipient keypair: %v", err)
	}

	publicKey := recipientKP.PublicKey()

	b.ResetTimer()
	b.ReportAllocs()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, _, err := Encapsulate(publicKey)
			if err != nil {
				b.Fatalf("Encapsulate failed: %v", err)
			}
		}
	})
}

// BenchmarkParallelDecapsulate tests concurrent decapsulation performance
func BenchmarkParallelDecapsulate(b *testing.B) {
	recipientKP, err := GenerateKeyPair()
	if err != nil {
		b.Fatalf("Failed to generate recipient keypair: %v", err)
	}

	_, ciphertext, err := Encapsulate(recipientKP.PublicKey())
	if err != nil {
		b.Fatalf("Failed to encapsulate: %v", err)
	}

	b.ResetTimer()
	b.ReportAllocs()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, err := Decapsulate(recipientKP, ciphertext)
			if err != nil {
				b.Fatalf("Decapsulate failed: %v", err)
			}
		}
	})
}
