package totp

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base32"
	"encoding/hex"
	"fmt"
	"io"
	"time"

	"github.com/pquerna/otp"
	"github.com/pquerna/otp/totp"
)

type TOTPManager struct {
	issuer        string
	encryptionKey []byte
}

type SecretInfo struct {
	Secret   string
	QRCode   string
	OTPAuth  string
}

func NewTOTPManager(issuer string, encryptionKeyHex string) (*TOTPManager, error) {
	keyBytes, err := hex.DecodeString(encryptionKeyHex)
	if err != nil {
		return nil, fmt.Errorf("invalid encryption key hex: %w", err)
	}

	if len(keyBytes) != 32 {
		return nil, fmt.Errorf("encryption key must be 32 bytes (256 bits), got %d", len(keyBytes))
	}

	return &TOTPManager{
		issuer:        issuer,
		encryptionKey: keyBytes,
	}, nil
}

func (m *TOTPManager) GenerateSecret(email string) (*SecretInfo, error) {
	key, err := totp.Generate(totp.GenerateOpts{
		Issuer:      m.issuer,
		AccountName: email,
		Period:      30,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to generate TOTP key: %w", err)
	}

	secretString := key.Secret()
	otpauthURL := key.URL()

	return &SecretInfo{
		Secret:  secretString,
		QRCode:  otpauthURL,
		OTPAuth: otpauthURL,
	}, nil
}

func (m *TOTPManager) Verify(encryptedSecret, code string) (bool, error) {
	secret, err := m.DecryptSecret(encryptedSecret)
	if err != nil {
		return false, err
	}

	valid, err := totp.ValidateCustom(code, secret, time.Now(), totp.ValidateOpts{
		Period:    30,
		Skew:      1,
		Digits:    6,
		Algorithm: otp.AlgorithmSHA1,
	})

	return valid, err
}

func (m *TOTPManager) EncryptSecret(secret string) (string, error) {
	secretBytes := []byte(secret)

	block, err := aes.NewCipher(m.encryptionKey)
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}

	ciphertext := gcm.Seal(nonce, nonce, secretBytes, nil)

	return hex.EncodeToString(ciphertext), nil
}

func (m *TOTPManager) DecryptSecret(encryptedSecret string) (string, error) {
	ciphertext, err := hex.DecodeString(encryptedSecret)
	if err != nil {
		return "", fmt.Errorf("invalid encrypted secret hex: %w", err)
	}

	block, err := aes.NewCipher(m.encryptionKey)
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return "", fmt.Errorf("ciphertext too short")
	}

	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]

	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", fmt.Errorf("failed to decrypt secret: %w", err)
	}

	return string(plaintext), nil
}

func GenerateBackupCodes(count int) ([]string, error) {
	codes := make([]string, count)
	for i := 0; i < count; i++ {
		randomBytes := make([]byte, 4)
		if _, err := rand.Read(randomBytes); err != nil {
			return nil, err
		}
		codes[i] = base32.StdEncoding.EncodeToString(randomBytes)[:6]
	}
	return codes, nil
}
