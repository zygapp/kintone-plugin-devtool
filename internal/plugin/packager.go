package plugin

import (
	"archive/zip"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha1"
	"crypto/x509"
	"io"
	"os"
	"path/filepath"

	"github.com/kintone/kpdev/internal/config"
	"github.com/kintone/kpdev/internal/generator"
)

// PackageDevPlugin は開発用プラグインZIPを作成する
func PackageDevPlugin(projectDir string) (string, error) {
	managedDir := filepath.Join(config.GetConfigDir(projectDir), "managed")
	devPluginDir := filepath.Join(managedDir, "dev-plugin")
	zipPath := filepath.Join(managedDir, "dev-plugin.zip")

	// 秘密鍵を読み込み
	keyPath := generator.GetDevKeyPath(projectDir)
	privateKey, err := generator.LoadPrivateKey(keyPath)
	if err != nil {
		return "", err
	}

	// プラグインZIPを作成
	if err := createPluginZip(devPluginDir, zipPath, privateKey); err != nil {
		return "", err
	}

	return zipPath, nil
}

// PackageProdPlugin は本番用プラグインZIPを作成する
func PackageProdPlugin(projectDir, distDir string) (string, error) {
	// 秘密鍵を読み込み
	keyPath := generator.GetProdKeyPath(projectDir)
	privateKey, err := generator.LoadPrivateKey(keyPath)
	if err != nil {
		return "", err
	}

	// manifest.json からバージョンを取得
	// TODO: バージョン取得

	zipPath := filepath.Join(projectDir, "dist", "plugin.zip")
	if err := createPluginZip(distDir, zipPath, privateKey); err != nil {
		return "", err
	}

	return zipPath, nil
}

func createPluginZip(srcDir, dstPath string, privateKey *rsa.PrivateKey) error {
	// 1. contents.zip を作成
	contentsZipPath := dstPath + ".contents"
	if err := createContentsZip(srcDir, contentsZipPath); err != nil {
		return err
	}
	defer os.Remove(contentsZipPath)

	// contents.zip を読み込み
	contentsData, err := os.ReadFile(contentsZipPath)
	if err != nil {
		return err
	}

	// 2. 署名を作成
	hash := sha1.Sum(contentsData)
	signature, err := rsa.SignPKCS1v15(rand.Reader, privateKey, crypto.SHA1, hash[:])
	if err != nil {
		return err
	}

	// 3. 公開鍵をエクスポート
	pubKeyDer, err := x509.MarshalPKIXPublicKey(&privateKey.PublicKey)
	if err != nil {
		return err
	}

	// 4. 最終ZIPを作成
	zipFile, err := os.Create(dstPath)
	if err != nil {
		return err
	}
	defer zipFile.Close()

	zipWriter := zip.NewWriter(zipFile)
	defer zipWriter.Close()

	// contents.zip
	contentsWriter, err := zipWriter.Create("contents.zip")
	if err != nil {
		return err
	}
	if _, err := contentsWriter.Write(contentsData); err != nil {
		return err
	}

	// PUBKEY
	pubkeyWriter, err := zipWriter.Create("PUBKEY")
	if err != nil {
		return err
	}
	if _, err := pubkeyWriter.Write(pubKeyDer); err != nil {
		return err
	}

	// SIGNATURE
	sigWriter, err := zipWriter.Create("SIGNATURE")
	if err != nil {
		return err
	}
	if _, err := sigWriter.Write(signature); err != nil {
		return err
	}

	return nil
}

func createContentsZip(srcDir, dstPath string) error {
	zipFile, err := os.Create(dstPath)
	if err != nil {
		return err
	}
	defer zipFile.Close()

	zipWriter := zip.NewWriter(zipFile)
	defer zipWriter.Close()

	return filepath.Walk(srcDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		relPath, err := filepath.Rel(srcDir, path)
		if err != nil {
			return err
		}

		// Zipエントリを作成
		writer, err := zipWriter.Create(relPath)
		if err != nil {
			return err
		}

		// ファイルを読み込んで書き込み
		file, err := os.Open(path)
		if err != nil {
			return err
		}
		defer file.Close()

		_, err = io.Copy(writer, file)
		return err
	})
}
