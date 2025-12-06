package auth

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type SigningKey struct {
	kid        string
	PrivateKey *rsa.PrivateKey
	PublicKey  *rsa.PublicKey
}

type KeyStore struct {
	ActiveKid string
	Keys      map[string]*SigningKey
}

func LoadKeys(path, activeKid string) (*KeyStore, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("keys directory does not exist or is not accessible: %w", err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("keys path is not a directory: %s", path)
	}

	ks := &KeyStore{
		ActiveKid: activeKid,
		Keys:      make(map[string]*SigningKey),
	}

	files, err := os.ReadDir(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read keys directory: %w", err)
	}

	for _, file := range files {
		if file.IsDir() {
			continue
		}

		fileName := file.Name()

		if !strings.HasPrefix(fileName, "private") || filepath.Ext(fileName) != ".pem" {
			continue
		}

		kid := strings.TrimPrefix(fileName, "private-")
		kid = strings.TrimSuffix(kid, ".pem")
		if kid == "" {
			continue
		}

		privPath := filepath.Join(path, fileName)
		privData, err := os.ReadFile(privPath)
		if err != nil {
			return nil, fmt.Errorf("failed to read private key file %s: %w", fileName, err)
		}

		block, _ := pem.Decode(privData)
		if block == nil {
			return nil, fmt.Errorf("failed to decode PEM block from private key file: %s", fileName)
		}

		priv, err := x509.ParsePKCS1PrivateKey(block.Bytes)
		if err != nil {
			pkcs8Key, err2 := x509.ParsePKCS8PrivateKey(block.Bytes)
			if err2 != nil {
				return nil, fmt.Errorf("failed to parse private key from %s (tried PKCS1 and PKCS8): %w", fileName, err)
			}
			rsaKey, ok := pkcs8Key.(*rsa.PrivateKey)
			if !ok {
				return nil, fmt.Errorf("private key in %s is not an RSA key", fileName)
			}
			priv = rsaKey
		}

		pubFileName := fmt.Sprintf("public-%s.pem", kid)
		pubPath := filepath.Join(path, pubFileName)
		pubData, err := os.ReadFile(pubPath)
		if err != nil {
			return nil, fmt.Errorf("failed to read public key file %s: %w", pubFileName, err)
		}

		pubBlock, _ := pem.Decode(pubData)
		if pubBlock == nil {
			return nil, fmt.Errorf("failed to decode PEM block from public key file: %s", pubFileName)
		}

		pub, err := x509.ParsePKIXPublicKey(pubBlock.Bytes)
		if err != nil {
			return nil, fmt.Errorf("failed to parse public key from %s: %w", pubFileName, err)
		}

		rsaPub, ok := pub.(*rsa.PublicKey)
		if !ok {
			return nil, fmt.Errorf("public key in %s is not an RSA key", pubFileName)
		}

		keyID := fmt.Sprintf("key-%s", kid)
		ks.Keys[keyID] = &SigningKey{
			kid:        keyID,
			PrivateKey: priv,
			PublicKey:  rsaPub,
		}
	}

	return ks, nil
}

func (ks *KeyStore) GetActiveKey() *SigningKey {
	activeKid := ks.ActiveKid
	if !strings.HasPrefix(activeKid, "key-") {
		activeKid = fmt.Sprintf("key-%s", activeKid)
	}
	return ks.Keys[activeKid]
}
