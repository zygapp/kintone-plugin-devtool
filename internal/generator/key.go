package generator

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/hex"
	"encoding/pem"
	"os"
	"path/filepath"

	"github.com/kintone/kpdev/internal/config"
)

const (
	KeyBits     = 1024 // kintone プラグイン仕様
	DevKeyFile  = "private.dev.ppk"
	ProdKeyFile = "private.prod.ppk"
)

func GenerateKeys(projectDir string) error {
	keysDir := filepath.Join(config.GetConfigDir(projectDir), "keys")
	if err := os.MkdirAll(keysDir, 0755); err != nil {
		return err
	}

	// 開発用鍵
	devKeyPath := filepath.Join(keysDir, DevKeyFile)
	if _, err := os.Stat(devKeyPath); os.IsNotExist(err) {
		if err := generateKeyFile(devKeyPath); err != nil {
			return err
		}
	}

	// 本番用鍵
	prodKeyPath := filepath.Join(keysDir, ProdKeyFile)
	if _, err := os.Stat(prodKeyPath); os.IsNotExist(err) {
		if err := generateKeyFile(prodKeyPath); err != nil {
			return err
		}
	}

	return nil
}

func generateKeyFile(path string) error {
	privateKey, err := rsa.GenerateKey(rand.Reader, KeyBits)
	if err != nil {
		return err
	}

	keyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(privateKey),
	})

	return os.WriteFile(path, keyPEM, 0600)
}

func LoadPrivateKey(path string) (*rsa.PrivateKey, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	block, _ := pem.Decode(data)
	if block == nil {
		return nil, err
	}

	return x509.ParsePKCS1PrivateKey(block.Bytes)
}

// GeneratePluginID は秘密鍵からプラグインIDを生成する
// kintone の仕様に従い、公開鍵の SHA256 ハッシュの先頭32文字を変換
func GeneratePluginID(privateKey *rsa.PrivateKey) (string, error) {
	pubKeyDer, err := x509.MarshalPKIXPublicKey(&privateKey.PublicKey)
	if err != nil {
		return "", err
	}

	hash := sha256.Sum256(pubKeyDer)
	hexStr := hex.EncodeToString(hash[:])[:32]

	// 0-9a-f → a-p に変換
	result := make([]byte, 32)
	for i, c := range hexStr {
		if c >= '0' && c <= '9' {
			result[i] = byte('a' + (c - '0'))
		} else {
			result[i] = byte('k' + (c - 'a'))
		}
	}

	return string(result), nil
}

func GetDevKeyPath(projectDir string) string {
	return filepath.Join(config.GetConfigDir(projectDir), "keys", DevKeyFile)
}

func GetProdKeyPath(projectDir string) string {
	return filepath.Join(config.GetConfigDir(projectDir), "keys", ProdKeyFile)
}
