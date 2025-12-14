package generator

import (
	"os"
	"os/exec"
	"path/filepath"

	"github.com/kintone/kpdev/internal/config"
)

func GenerateCerts(projectDir string) error {
	certsDir := filepath.Join(config.GetConfigDir(projectDir), "certs")
	if err := os.MkdirAll(certsDir, 0755); err != nil {
		return err
	}

	keyPath := filepath.Join(certsDir, "localhost-key.pem")
	certPath := filepath.Join(certsDir, "localhost.pem")

	// 既存の証明書があればスキップ
	if _, err := os.Stat(keyPath); err == nil {
		if _, err := os.Stat(certPath); err == nil {
			return nil
		}
	}

	// OpenSSL設定ファイルを一時作成
	opensslConf := `[req]
default_bits = 2048
prompt = no
default_md = sha256
distinguished_name = dn
x509_extensions = v3_req

[dn]
CN = localhost

[v3_req]
subjectAltName = @alt_names

[alt_names]
DNS.1 = localhost
IP.1 = 127.0.0.1
IP.2 = ::1
`
	confPath := filepath.Join(certsDir, "openssl.cnf")
	if err := os.WriteFile(confPath, []byte(opensslConf), 0644); err != nil {
		return err
	}
	defer os.Remove(confPath)

	// openssl で自己署名証明書を生成
	cmd := exec.Command("openssl", "req",
		"-x509",
		"-newkey", "rsa:2048",
		"-keyout", keyPath,
		"-out", certPath,
		"-days", "365",
		"-nodes",
		"-config", confPath,
	)

	return cmd.Run()
}
