package main

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io"
	"time"

	"github.com/redis/go-redis/v9"
)

type RedisTokenStorage struct {
	client *redis.Client
	gcm    cipher.AEAD
}

func NewRedisTokenStorage(redisURL, password string, encryptionKey []byte) (*RedisTokenStorage, error) {
	// Initialize Redis client
	opt, err := redis.ParseURL(redisURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse Redis URL: %w", err)
	}
	opt.Password = password

	client := redis.NewClient(opt)

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	// Initialize encryption
	block, err := aes.NewCipher(encryptionKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	return &RedisTokenStorage{
		client: client,
		gcm:    gcm,
	}, nil
}

func (s *RedisTokenStorage) encrypt(data []byte) ([]byte, error) {
	nonce := make([]byte, s.gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}

	return s.gcm.Seal(nonce, nonce, data, nil), nil
}

func (s *RedisTokenStorage) decrypt(data []byte) ([]byte, error) {
	if len(data) < s.gcm.NonceSize() {
		return nil, fmt.Errorf("ciphertext too short")
	}

	nonce := data[:s.gcm.NonceSize()]
	ciphertext := data[s.gcm.NonceSize():]

	return s.gcm.Open(nil, nonce, ciphertext, nil)
}

func (s *RedisTokenStorage) StoreRefreshToken(token string, info TokenInfo) error {
	fmt.Print("TOKEN HAS BEEN STORED: %w", token)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Serialize token info
	data, err := json.Marshal(info)
	if err != nil {
		return fmt.Errorf("failed to marshal token info: %w", err)
	}

	// Encrypt data
	encrypted, err := s.encrypt(data)
	if err != nil {
		return fmt.Errorf("failed to encrypt token info: %w", err)
	}

	// Store in Redis with expiration
	key := fmt.Sprintf("token:%s", token)
	expiration := time.Until(info.ExpiresAt)

	if err := s.client.Set(ctx, key, encrypted, expiration).Err(); err != nil {
		return fmt.Errorf("failed to store token in Redis: %w", err)
	}

	return nil
}

func (s *RedisTokenStorage) GetTokenInfo(token string) (*TokenInfo, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Get from Redis
	key := fmt.Sprintf("token:%s", token)
	encrypted, err := s.client.Get(ctx, key).Bytes()
	if err != nil {
		if err == redis.Nil {
			return nil, fmt.Errorf("token not found")
		}
		return nil, fmt.Errorf("failed to get token from Redis: %w", err)
	}

	// Decrypt data
	data, err := s.decrypt(encrypted)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt token info: %w", err)
	}

	// Deserialize token info
	var info TokenInfo
	if err := json.Unmarshal(data, &info); err != nil {
		return nil, fmt.Errorf("failed to unmarshal token info: %w", err)
	}

	return &info, nil
}

func (s *RedisTokenStorage) DeleteToken(token string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	key := fmt.Sprintf("token:%s", token)
	if err := s.client.Del(ctx, key).Err(); err != nil {
		return fmt.Errorf("failed to delete token from Redis: %w", err)
	}

	return nil
}

func (s *RedisTokenStorage) Close() error {
	return s.client.Close()
}
