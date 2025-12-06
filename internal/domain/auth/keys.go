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
		return nil, &ErrKeysDirectoryNotAccessible{Path: path, Err: err}
	}
	if !info.IsDir() {
		return nil, &ErrKeysPathNotDirectory{Path: path}
	}

	ks := &KeyStore{
		ActiveKid: activeKid,
		Keys:      make(map[string]*SigningKey),
	}

	files, err := os.ReadDir(path)
	if err != nil {
		return nil, &ErrFailedToReadKeysDirectory{Err: err}
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
			return nil, &ErrFailedToReadPrivateKeyFile{FileName: fileName, Err: err}
		}

		block, _ := pem.Decode(privData)
		if block == nil {
			return nil, &ErrFailedToDecodePrivateKeyPEM{FileName: fileName}
		}

		priv, err := x509.ParsePKCS1PrivateKey(block.Bytes)
		if err != nil {
			pkcs8Key, err2 := x509.ParsePKCS8PrivateKey(block.Bytes)
			if err2 != nil {
				return nil, &ErrFailedToParsePrivateKey{FileName: fileName, Err: err}
			}
			rsaKey, ok := pkcs8Key.(*rsa.PrivateKey)
			if !ok {
				return nil, &ErrPrivateKeyNotRSA{FileName: fileName}
			}
			priv = rsaKey
		}

		pubFileName := fmt.Sprintf("public-%s.pem", kid)
		pubPath := filepath.Join(path, pubFileName)
		pubData, err := os.ReadFile(pubPath)
		if err != nil {
			return nil, &ErrFailedToReadPublicKeyFile{FileName: pubFileName, Err: err}
		}

		pubBlock, _ := pem.Decode(pubData)
		if pubBlock == nil {
			return nil, &ErrFailedToDecodePublicKeyPEM{FileName: pubFileName}
		}

		pub, err := x509.ParsePKIXPublicKey(pubBlock.Bytes)
		if err != nil {
			return nil, &ErrFailedToParsePublicKey{FileName: pubFileName, Err: err}
		}

		rsaPub, ok := pub.(*rsa.PublicKey)
		if !ok {
			return nil, &ErrPublicKeyNotRSA{FileName: pubFileName}
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
