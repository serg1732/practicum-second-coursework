package utils

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"io"

	"golang.org/x/crypto/argon2"
	"golang.org/x/crypto/chacha20poly1305"
	"golang.org/x/crypto/pbkdf2"
)

const (
	encryptionVersion byte = 1

	saltSize = 16

	argonTime    uint32 = 3
	argonMemory  uint32 = 64 * 1024 // 64 MB
	argonThreads uint8  = 4
	argonKeyLen  uint32 = chacha20poly1305.KeySize

	keyLen  = 32
	keyIter = 100_000
)

// Encrypt шифрование данных.
func Encrypt(plaintext string, secretKey []byte) (string, error) {
	salt := make([]byte, saltSize)
	if _, err := io.ReadFull(rand.Reader, salt); err != nil {
		return "", err
	}

	key := deriveEncryptionKey(secretKey, salt)

	aead, err := chacha20poly1305.NewX(key)
	if err != nil {
		return "", err
	}

	nonce := make([]byte, aead.NonceSize())
	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}

	encryptedData := aead.Seal(nil, nonce, []byte(plaintext), nil)

	result := make([]byte, 0, 1+len(salt)+len(nonce)+len(encryptedData))
	result = append(result, encryptionVersion)
	result = append(result, salt...)
	result = append(result, nonce...)
	result = append(result, encryptedData...)
	return base64.StdEncoding.EncodeToString(result), nil
}

// Decrypt расшифровка данных.
func Decrypt(ciphertext string, secretKey []byte) (string, error) {
	rawCiphertext, err := base64.StdEncoding.DecodeString(ciphertext)
	if err != nil {
		return "", err
	}

	nonceSize := chacha20poly1305.NonceSizeX

	minSize := 1 + saltSize + nonceSize + chacha20poly1305.Overhead
	if len(rawCiphertext) < minSize {
		return "", errors.New("размер шифротекста слишком мал")
	}

	if rawCiphertext[0] != encryptionVersion {
		return "", errors.New("версия шифрования не поддерживается")
	}

	offset := 1
	salt := rawCiphertext[offset : offset+saltSize]
	offset += saltSize

	nonce := rawCiphertext[offset : offset+nonceSize]
	offset += nonceSize

	encryptedData := rawCiphertext[offset:]
	key := deriveEncryptionKey(secretKey, salt)
	aead, err := chacha20poly1305.NewX(key)
	if err != nil {
		return "", err
	}

	plaintext, err := aead.Open(nil, nonce, encryptedData, nil)
	if err != nil {
		return "", err
	}
	return string(plaintext), nil
}

// deriveEncryptionKey получение ключа шифрования.
func deriveEncryptionKey(secretKey []byte, salt []byte) []byte {
	return argon2.IDKey(
		secretKey,
		salt,
		argonTime,
		argonMemory,
		argonThreads,
		argonKeyLen,
	)
}

// HashPassword - генератор хэша пароля.
func HashPassword(password string) (string, error) {
	h := hmac.New(sha256.New, []byte(password))
	_, err := h.Write([]byte(password))
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

// Pbkdf2KeySecureRandom - генерация секрета.
func Pbkdf2KeySecureRandom(keyword []byte) (key []byte) {
	h := sha256.New()
	h.Write(keyword)
	salt := h.Sum(nil)
	return pbkdf2.Key(
		keyword,
		salt,
		keyIter,
		keyLen,
		sha256.New,
	)
}
