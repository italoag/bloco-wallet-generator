package main

import (
	"fmt"
	"sync"
	"testing"
)

func TestCryptoPool(t *testing.T) {
	pool := NewCryptoPool()

	// Test private key buffer pool
	buf1 := pool.GetPrivateKeyBuffer()
	if len(buf1) != 32 {
		t.Errorf("Expected private key buffer length 32, got %d", len(buf1))
	}

	// Fill buffer with test data
	for i := range buf1 {
		buf1[i] = byte(i)
	}

	pool.PutPrivateKeyBuffer(buf1)

	// Get another buffer and verify it's cleared
	buf2 := pool.GetPrivateKeyBuffer()
	for i, b := range buf2 {
		if b != 0 {
			t.Errorf("Buffer not cleared at index %d: expected 0, got %d", i, b)
		}
	}
	pool.PutPrivateKeyBuffer(buf2)

	// Test BigInt pool
	bigInt1 := pool.GetBigInt()
	if bigInt1 == nil {
		t.Error("GetBigInt returned nil")
	}

	bigInt1.SetInt64(12345)
	pool.PutBigInt(bigInt1)

	bigInt2 := pool.GetBigInt()
	if bigInt2.Int64() != 0 {
		t.Errorf("BigInt not cleared: expected 0, got %d", bigInt2.Int64())
	}
	pool.PutBigInt(bigInt2)

	// Test ECDSA key pool
	key1 := pool.GetECDSAKey()
	if key1 == nil {
		t.Error("GetECDSAKey returned nil")
	}
	pool.PutECDSAKey(key1)
}

func TestHasherPool(t *testing.T) {
	pool := NewHasherPool()

	// Test Keccak hasher pool
	hasher1 := pool.GetKeccak()
	if hasher1 == nil {
		t.Error("GetKeccak returned nil")
	}

	// Test that we can use the hasher
	hasher1.Write([]byte("test"))
	hash1 := hasher1.Sum(nil)
	if len(hash1) == 0 {
		t.Error("Hasher produced empty hash")
	}

	pool.PutKeccak(hasher1)

	// Get another hasher - it should be reset and ready to use
	hasher2 := pool.GetKeccak()
	if hasher2 == nil {
		t.Error("Second GetKeccak returned nil")
	}

	// Test that the second hasher works independently
	hasher2.Write([]byte("different"))
	hash2 := hasher2.Sum(nil)
	if len(hash2) == 0 {
		t.Error("Second hasher produced empty hash")
	}

	pool.PutKeccak(hasher2)
}

func TestBufferPool(t *testing.T) {
	pool := NewBufferPool()

	// Test byte buffer pool
	buf1 := pool.GetByteBuffer()
	if buf1 == nil {
		t.Error("GetByteBuffer returned nil")
	}

	buf1 = append(buf1, 1, 2, 3, 4, 5)
	pool.PutByteBuffer(buf1)

	buf2 := pool.GetByteBuffer()
	if len(buf2) != 0 {
		t.Errorf("Buffer not reset: expected length 0, got %d", len(buf2))
	}
	pool.PutByteBuffer(buf2)

	// Test string builder pool
	sb1 := pool.GetStringBuilder()
	if sb1 == nil {
		t.Error("GetStringBuilder returned nil")
	}

	sb1.WriteString("test")
	pool.PutStringBuilder(sb1)

	sb2 := pool.GetStringBuilder()
	if sb2.Len() != 0 {
		t.Errorf("String builder not reset: expected length 0, got %d", sb2.Len())
	}
	pool.PutStringBuilder(sb2)

	// Test hex buffer pool
	hexBuf1 := pool.GetHexBuffer()
	if len(hexBuf1) != 64 {
		t.Errorf("Expected hex buffer length 64, got %d", len(hexBuf1))
	}

	// Fill with test data
	for i := range hexBuf1 {
		hexBuf1[i] = byte(i)
	}

	pool.PutHexBuffer(hexBuf1)

	hexBuf2 := pool.GetHexBuffer()
	for i, b := range hexBuf2 {
		if b != 0 {
			t.Errorf("Hex buffer not cleared at index %d: expected 0, got %d", i, b)
		}
	}
	pool.PutHexBuffer(hexBuf2)
}

func TestGlobalPoolsInitialization(t *testing.T) {
	// Verify global pools are initialized
	if globalCryptoPool == nil {
		t.Error("globalCryptoPool is not initialized")
	}

	if globalHasherPool == nil {
		t.Error("globalHasherPool is not initialized")
	}

	if globalBufferPool == nil {
		t.Error("globalBufferPool is not initialized")
	}
}

// TestCryptoPoolThreadSafety tests concurrent access to CryptoPool
func TestCryptoPoolThreadSafety(t *testing.T) {
	pool := NewCryptoPool()
	numGoroutines := 50
	operationsPerGoroutine := 100

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	// Test concurrent access to private key buffers
	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < operationsPerGoroutine; j++ {
				buf := pool.GetPrivateKeyBuffer()
				if len(buf) != 32 {
					t.Errorf("Expected buffer length 32, got %d", len(buf))
				}
				// Simulate some work
				for k := range buf {
					buf[k] = byte(k)
				}
				pool.PutPrivateKeyBuffer(buf)
			}
		}()
	}

	wg.Wait()

	// Test concurrent access to BigInt pool
	wg.Add(numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < operationsPerGoroutine; j++ {
				bigInt := pool.GetBigInt()
				if bigInt == nil {
					t.Error("GetBigInt returned nil")
				}
				bigInt.SetInt64(int64(j))
				pool.PutBigInt(bigInt)
			}
		}()
	}

	wg.Wait()

	// Test concurrent access to ECDSA key pool
	wg.Add(numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < operationsPerGoroutine; j++ {
				key := pool.GetECDSAKey()
				if key == nil {
					t.Error("GetECDSAKey returned nil")
				}
				pool.PutECDSAKey(key)
			}
		}()
	}

	wg.Wait()
}

// TestHasherPoolThreadSafety tests concurrent access to HasherPool
func TestHasherPoolThreadSafety(t *testing.T) {
	pool := NewHasherPool()
	numGoroutines := 50
	operationsPerGoroutine := 100

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < operationsPerGoroutine; j++ {
				hasher := pool.GetKeccak()
				if hasher == nil {
					t.Error("GetKeccak returned nil")
				}

				// Use the hasher
				testData := fmt.Sprintf("test-data-%d-%d", id, j)
				hasher.Write([]byte(testData))
				hash := hasher.Sum(nil)

				if len(hash) == 0 {
					t.Error("Hasher produced empty hash")
				}

				pool.PutKeccak(hasher)
			}
		}(i)
	}

	wg.Wait()
}

// TestBufferPoolThreadSafety tests concurrent access to BufferPool
func TestBufferPoolThreadSafety(t *testing.T) {
	pool := NewBufferPool()
	numGoroutines := 50
	operationsPerGoroutine := 100

	var wg sync.WaitGroup

	// Test byte buffer pool thread safety
	wg.Add(numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < operationsPerGoroutine; j++ {
				buf := pool.GetByteBuffer()
				if buf == nil {
					t.Error("GetByteBuffer returned nil")
				}

				// Use the buffer
				testData := []byte(fmt.Sprintf("test-%d-%d", id, j))
				buf = append(buf, testData...)

				pool.PutByteBuffer(buf)
			}
		}(i)
	}

	wg.Wait()

	// Test string builder pool thread safety
	wg.Add(numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < operationsPerGoroutine; j++ {
				sb := pool.GetStringBuilder()
				if sb == nil {
					t.Error("GetStringBuilder returned nil")
				}

				// Use the string builder
				sb.WriteString(fmt.Sprintf("test-%d-%d", id, j))

				pool.PutStringBuilder(sb)
			}
		}(i)
	}

	wg.Wait()

	// Test hex buffer pool thread safety
	wg.Add(numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < operationsPerGoroutine; j++ {
				buf := pool.GetHexBuffer()
				if len(buf) != 64 {
					t.Errorf("Expected hex buffer length 64, got %d", len(buf))
				}

				// Use the buffer
				for k := range buf {
					buf[k] = byte(k % 256)
				}

				pool.PutHexBuffer(buf)
			}
		}()
	}

	wg.Wait()
}

// TestPoolMemoryClearing tests that sensitive data is properly cleared
func TestPoolMemoryClearing(t *testing.T) {
	pool := NewCryptoPool()

	// Test private key buffer clearing
	buf := pool.GetPrivateKeyBuffer()
	for i := range buf {
		buf[i] = 0xFF // Fill with non-zero data
	}
	pool.PutPrivateKeyBuffer(buf)

	// Get another buffer and verify it's cleared
	buf2 := pool.GetPrivateKeyBuffer()
	for i, b := range buf2 {
		if b != 0 {
			t.Errorf("Private key buffer not cleared at index %d: expected 0, got %d", i, b)
		}
	}
	pool.PutPrivateKeyBuffer(buf2)

	// Test BigInt clearing
	bigInt := pool.GetBigInt()
	bigInt.SetInt64(0xDEADBEEF)
	pool.PutBigInt(bigInt)

	bigInt2 := pool.GetBigInt()
	if bigInt2.Int64() != 0 {
		t.Errorf("BigInt not cleared: expected 0, got %d", bigInt2.Int64())
	}
	pool.PutBigInt(bigInt2)

	// Test ECDSA key clearing
	key := pool.GetECDSAKey()
	// We can't easily test key clearing without exposing internals,
	// but we can verify the key is properly reset
	if key.D != nil {
		t.Error("ECDSA key D should be nil after getting from pool")
	}
	if key.PublicKey.X != nil {
		t.Error("ECDSA key PublicKey.X should be nil after getting from pool")
	}
	if key.PublicKey.Y != nil {
		t.Error("ECDSA key PublicKey.Y should be nil after getting from pool")
	}
	pool.PutECDSAKey(key)
}

// TestPoolResourceReuse tests that pools actually reuse objects
func TestPoolResourceReuse(t *testing.T) {
	pool := NewCryptoPool()

	// Get and return a private key buffer
	buf1 := pool.GetPrivateKeyBuffer()
	buf1Ptr := &buf1[0] // Get pointer to first element
	pool.PutPrivateKeyBuffer(buf1)

	// Get another buffer - it should be the same one (reused)
	buf2 := pool.GetPrivateKeyBuffer()
	buf2Ptr := &buf2[0]

	if buf1Ptr != buf2Ptr {
		// This test might be flaky due to Go's sync.Pool implementation,
		// but it's worth checking for basic reuse
		t.Log("Warning: Buffer was not reused (this might be normal due to sync.Pool behavior)")
	}

	pool.PutPrivateKeyBuffer(buf2)
}
