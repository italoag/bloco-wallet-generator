package main

import (
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
