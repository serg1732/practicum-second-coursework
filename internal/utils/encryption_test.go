package utils

import (
	"encoding/base64"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEncryptDecrypt(t *testing.T) {
	t.Run("должен зашифровать и расшифровать исходный текст", func(t *testing.T) {
		secretKey := []byte("super-secret-key")
		plaintext := "login: user@example.com; password: p@ssw0rd; привет"

		ciphertext, err := Encrypt(plaintext, secretKey)

		assert.NoError(t, err)
		assert.NotEmpty(t, ciphertext)
		assert.NotEqual(t, plaintext, ciphertext)

		rawCiphertext, err := base64.StdEncoding.DecodeString(ciphertext)
		assert.NoError(t, err)
		if err != nil {
			return
		}

		assert.NotEmpty(t, rawCiphertext)
		if len(rawCiphertext) == 0 {
			return
		}

		assert.Equal(t, encryptionVersion, rawCiphertext[0])

		decryptedText, err := Decrypt(ciphertext, secretKey)

		assert.NoError(t, err)
		assert.Equal(t, plaintext, decryptedText)
	})

	t.Run("должен создавать разный шифротекст для одинакового текста", func(t *testing.T) {
		secretKey := []byte("super-secret-key")
		plaintext := "same plaintext"

		firstCiphertext, err := Encrypt(plaintext, secretKey)
		assert.NoError(t, err)

		secondCiphertext, err := Encrypt(plaintext, secretKey)
		assert.NoError(t, err)

		assert.NotEqual(t, firstCiphertext, secondCiphertext)

		for _, ciphertext := range []string{firstCiphertext, secondCiphertext} {
			decryptedText, decryptErr := Decrypt(ciphertext, secretKey)

			assert.NoError(t, decryptErr)
			assert.Equal(t, plaintext, decryptedText)
		}
	})

	t.Run("должен вернуть ошибку при неверном ключе", func(t *testing.T) {
		ciphertext, err := Encrypt("secret data", []byte("correct-key"))
		assert.NoError(t, err)
		if err != nil {
			return
		}

		decryptedText, err := Decrypt(ciphertext, []byte("wrong-key"))

		assert.Error(t, err)
		assert.Empty(t, decryptedText)
	})

	t.Run("должен вернуть ошибку при невалидном base64", func(t *testing.T) {
		decryptedText, err := Decrypt("not-valid-base64!!!", []byte("key"))

		assert.Error(t, err)
		assert.Empty(t, decryptedText)
	})

	t.Run("должен вернуть ошибку при слишком маленьком шифротексте", func(t *testing.T) {
		ciphertext := base64.StdEncoding.EncodeToString([]byte{encryptionVersion})

		decryptedText, err := Decrypt(ciphertext, []byte("key"))

		assert.Error(t, err)
		assert.Empty(t, decryptedText)
		if err != nil {
			assert.Contains(t, err.Error(), "слишком мал")
		}
	})

	t.Run("должен вернуть ошибку при неподдерживаемой версии шифрования", func(t *testing.T) {
		rawCiphertext := make([]byte, 128)
		rawCiphertext[0] = encryptionVersion + 1
		ciphertext := base64.StdEncoding.EncodeToString(rawCiphertext)

		decryptedText, err := Decrypt(ciphertext, []byte("key"))

		assert.Error(t, err)
		assert.Empty(t, decryptedText)
		if err != nil {
			assert.Contains(t, err.Error(), "версия")
		}
	})

	t.Run("должен вернуть ошибку при измененном шифротексте", func(t *testing.T) {
		secretKey := []byte("super-secret-key")

		ciphertext, err := Encrypt("secret data", secretKey)
		assert.NoError(t, err)
		if err != nil {
			return
		}

		rawCiphertext, err := base64.StdEncoding.DecodeString(ciphertext)
		assert.NoError(t, err)
		if err != nil {
			return
		}

		assert.NotEmpty(t, rawCiphertext)
		if len(rawCiphertext) == 0 {
			return
		}

		rawCiphertext[len(rawCiphertext)-1] ^= 0xff
		tamperedCiphertext := base64.StdEncoding.EncodeToString(rawCiphertext)

		decryptedText, err := Decrypt(tamperedCiphertext, secretKey)

		assert.Error(t, err)
		assert.Empty(t, decryptedText)
	})
}

func TestHashPassword(t *testing.T) {
	t.Run("должен возвращать ожидаемый хэш пароля", func(t *testing.T) {
		firstHash, err := HashPassword("password")
		assert.NoError(t, err)

		secondHash, err := HashPassword("password")
		assert.NoError(t, err)

		const expectedHash = "507c4db58311630bdfa4ed5d4b8a562ca2f43370e03a3df411b3784a805681f7"
		assert.Equal(t, expectedHash, firstHash)
		assert.Equal(t, firstHash, secondHash)
	})
}

func TestPbkdf2KeySecureRandom(t *testing.T) {
	t.Run("должен возвращать ключ нужной длины", func(t *testing.T) {
		key := Pbkdf2KeySecureRandom([]byte("password"))

		assert.Len(t, key, keyLen)
	})

	t.Run("должен возвращать одинаковый ключ для одинакового пароля", func(t *testing.T) {
		firstKey := Pbkdf2KeySecureRandom([]byte("password"))
		secondKey := Pbkdf2KeySecureRandom([]byte("password"))

		assert.Equal(t, firstKey, secondKey)
	})

	t.Run("должен возвращать разные ключи для разных паролей", func(t *testing.T) {
		firstKey := Pbkdf2KeySecureRandom([]byte("password"))
		otherKey := Pbkdf2KeySecureRandom([]byte("other-password"))

		assert.NotEqual(t, firstKey, otherKey)
	})
}
