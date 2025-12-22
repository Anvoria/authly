package main

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/Anvoria/authly/internal/config"
	"github.com/Anvoria/authly/internal/domain/auth"
	"github.com/lestrrat-go/jwx/v3/jwk"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	envConfig := config.LoadEnv()
	cfg, err := config.Load(envConfig.ConfigPath)
	if err != nil {
		slog.Error("Failed to load configuration", "error", err)
		os.Exit(1)
	}

	command := os.Args[1]
	switch command {
	case "generate":
		generateKey(cfg.Auth.KeysPath)
	case "list":
		listKeys(cfg.Auth.KeysPath, cfg.Auth.ActiveKID)
	case "set-active":
		if len(os.Args) < 3 {
			fmt.Fprintf(os.Stderr, "Error: key ID required\n")
			fmt.Fprintf(os.Stderr, "Usage: %s set-active <key-id>\n", os.Args[0])
			os.Exit(1)
		}
		setActiveKey(cfg, os.Args[2])
	default:
		fmt.Fprintf(os.Stderr, "Error: unknown command %s\n", command)
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Fprintf(os.Stderr, "Usage: %s <command> [options]\n\n", os.Args[0])
	fmt.Fprintf(os.Stderr, "Commands:\n")
	fmt.Fprintf(os.Stderr, "  generate              Generate a new RSA key pair\n")
	fmt.Fprintf(os.Stderr, "    -kid <id>           Key ID (required)\n")
	fmt.Fprintf(os.Stderr, "    -bits <size>        Key size: 2048, 3072, or 4096 (default: 2048)\n")
	fmt.Fprintf(os.Stderr, "    -path <dir>         Custom keys directory (overrides config)\n")
	fmt.Fprintf(os.Stderr, "  list                  List all available keys\n")
	fmt.Fprintf(os.Stderr, "  set-active <kid>      Set active key ID\n")
}

func generateKey(keysPath string) {
	fs := flag.NewFlagSet("generate", flag.ExitOnError)
	kid := fs.String("kid", "", "Key ID (required)")
	bits := fs.Int("bits", 2048, "Key size in bits (2048, 3072, or 4096)")
	customPath := fs.String("path", "", "Custom keys directory path (overrides config)")
	fs.Parse(os.Args[2:])

	if *kid == "" {
		fmt.Fprintf(os.Stderr, "Error: key ID is required\n")
		fmt.Fprintf(os.Stderr, "Usage: %s generate -kid <key-id> [-bits 2048] [-path /custom/path]\n", os.Args[0])
		os.Exit(1)
	}

	if *bits != 2048 && *bits != 3072 && *bits != 4096 {
		fmt.Fprintf(os.Stderr, "Error: key size must be 2048, 3072, or 4096\n")
		os.Exit(1)
	}

	// Use custom path if provided
	if *customPath != "" {
		keysPath = *customPath
	}

	// Ensure keys directory exists
	if err := os.MkdirAll(keysPath, 0700); err != nil {
		slog.Error("Failed to create keys directory", "error", err, "path", keysPath)
		fmt.Fprintf(os.Stderr, "\nTip: Use -path flag to specify a different directory:\n")
		fmt.Fprintf(os.Stderr, "  %s generate -kid %s -path ./keys\n", os.Args[0], *kid)
		os.Exit(1)
	}

	// Check if key already exists
	privPath := filepath.Join(keysPath, fmt.Sprintf("private-%s.pem", *kid))
	pubPath := filepath.Join(keysPath, fmt.Sprintf("public-%s.pem", *kid))

	if _, err := os.Stat(privPath); err == nil {
		fmt.Fprintf(os.Stderr, "Error: key with ID %s already exists at %s\n", *kid, privPath)
		os.Exit(1)
	}

	// Test write permissions before generating keys
	testFile := filepath.Join(keysPath, ".write-test")
	if testF, err := os.OpenFile(testFile, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600); err != nil {
		slog.Error("No write permission to keys directory", "error", err, "path", keysPath)
		fmt.Fprintf(os.Stderr, "\nTip: Use -path flag to specify a writable directory:\n")
		fmt.Fprintf(os.Stderr, "  %s generate -kid %s -path ./keys\n", os.Args[0], *kid)
		os.Exit(1)
	} else {
		testF.Close()
		os.Remove(testFile)
	}

	// Generate RSA key pair
	fmt.Printf("Generating %d-bit RSA key pair...\n", *bits)
	privateKey, err := rsa.GenerateKey(rand.Reader, *bits)
	if err != nil {
		slog.Error("Failed to generate RSA key", "error", err)
		os.Exit(1)
	}

	// Encode private key to PEM (PKCS1 format)
	privateKeyPEM := &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(privateKey),
	}

	privateKeyFile, err := os.OpenFile(privPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		slog.Error("Failed to create private key file", "error", err, "path", privPath)
		os.Exit(1)
	}
	defer privateKeyFile.Close()

	if err := pem.Encode(privateKeyFile, privateKeyPEM); err != nil {
		slog.Error("Failed to encode private key", "error", err)
		os.Exit(1)
	}

	// Encode public key to PEM (PKIX format)
	publicKeyBytes, err := x509.MarshalPKIXPublicKey(&privateKey.PublicKey)
	if err != nil {
		slog.Error("Failed to marshal public key", "error", err)
		os.Exit(1)
	}

	publicKeyPEM := &pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: publicKeyBytes,
	}

	publicKeyFile, err := os.OpenFile(pubPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0644)
	if err != nil {
		slog.Error("Failed to create public key file", "error", err, "path", pubPath)
		os.Exit(1)
	}
	defer publicKeyFile.Close()

	if err := pem.Encode(publicKeyFile, publicKeyPEM); err != nil {
		slog.Error("Failed to encode public key", "error", err)
		os.Exit(1)
	}

	fmt.Printf("Key pair generated successfully\n")
	fmt.Printf("  Private key: %s\n", privPath)
	fmt.Printf("  Public key:  %s\n", pubPath)
	fmt.Printf("  Key ID:      %s\n", *kid)
	fmt.Printf("  Key size:    %d bits\n", *bits)
}

// listKeys prints information about the RSA keys found in keysPath and indicates which key is active.
// 
// It reports and returns early when the path does not exist or is not a directory, or when keys cannot
// be loaded. If keys are present, it prints each key's full KID (marking the active key), the RSA key
// size in bits, and the expected private/public PEM filenames (private-<kid>.pem and public-<kid>.pem).
// Finally, it prints the provided activeKID.
func listKeys(keysPath, activeKID string) {
	// Check if directory exists
	info, err := os.Stat(keysPath)
	if err != nil {
		fmt.Printf("Keys directory does not exist: %s\n", keysPath)
		return
	}
	if !info.IsDir() {
		fmt.Printf("Keys path is not a directory: %s\n", keysPath)
		return
	}

	// Try to load keys
	keyStore, err := auth.LoadKeys(keysPath, activeKID)
	if err != nil {
		fmt.Printf("Error loading keys: %v\n", err)
		return
	}

	keySet := keyStore.JWKS()
	if keySet.Len() == 0 {
		fmt.Printf("No keys found in %s\n", keysPath)
		return
	}

	fmt.Printf("Keys in %s:\n\n", keysPath)

	normalizedActiveKID := activeKID
	if !strings.HasPrefix(normalizedActiveKID, "key-") {
		normalizedActiveKID = fmt.Sprintf("key-%s", normalizedActiveKID)
	}

	for i := 0; i < keySet.Len(); i++ {
		key, ok := keySet.Key(i)
		if !ok {
			continue
		}

		kid, _ := key.KeyID()
		active := ""
		if kid == normalizedActiveKID {
			active = " (ACTIVE)"
		}
		keyID := kid
		if len(kid) > 4 && kid[:4] == "key-" {
			keyID = kid[4:]
		}

		// Get key size from raw key
		var rawKey any
		if err := jwk.Export(key, &rawKey); err == nil {
			if rsaKey, ok := rawKey.(*rsa.PublicKey); ok {
				fmt.Printf("  %s%s\n", kid, active)
				fmt.Printf("    Key size: %d bits\n", rsaKey.N.BitLen())
				fmt.Printf("    Private:  private-%s.pem\n", keyID)
				fmt.Printf("    Public:   public-%s.pem\n", keyID)
				fmt.Println()
			}
		}
	}

	fmt.Printf("Active KID: %s\n", activeKID)
}

func setActiveKey(cfg *config.Config, kid string) {
	keyStore, err := auth.LoadKeys(cfg.Auth.KeysPath, cfg.Auth.ActiveKID)
	if err != nil {
		slog.Error("Failed to load keys", "error", err)
		os.Exit(1)
	}

	keyID := fmt.Sprintf("key-%s", kid)
	key, ok := keyStore.KeySet.LookupKeyID(keyID)
	if !ok {
		fmt.Fprintf(os.Stderr, "Error: key with ID %s not found\n", kid)
		os.Exit(1)
	}
	_ = key

	envConfig := config.LoadEnv()
	configPath := envConfig.ConfigPath

	fmt.Printf("To set active key, update %s:\n\n", configPath)
	fmt.Printf("  auth:\n")
	fmt.Printf("    keys_path: %s\n", cfg.Auth.KeysPath)
	fmt.Printf("    active_kid: %s\n", kid)
	fmt.Printf("\n")
}