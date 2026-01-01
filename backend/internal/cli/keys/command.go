package keys

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Anvoria/authly/internal/config"
	"github.com/Anvoria/authly/internal/domain/auth"
	"github.com/lestrrat-go/jwx/v3/jwk"
)

// Command implements the keys management command
type Command struct{}

func (c *Command) Name() string {
	return "keys"
}

func (c *Command) Description() string {
	return "Manage cryptographic keys (generate, list, set-active)"
}

func (c *Command) Run(args []string) error {
	if len(args) < 1 {
		c.printUsage()
		return fmt.Errorf("subcommand required")
	}

	subcmd := args[0]
	switch subcmd {
	case "generate":
		return c.runGenerate(args[1:])
	case "list":
		return c.runList(args[1:])
	case "set-active":
		return c.runSetActive(args[1:])
	default:
		c.printUsage()
		return fmt.Errorf("unknown subcommand: %s", subcmd)
	}
}

func (c *Command) printUsage() {
	fmt.Fprintf(os.Stderr, "Usage: authly-cli keys <subcommand> [args]\n\n")
	fmt.Fprintf(os.Stderr, "Subcommands:\n")
	fmt.Fprintf(os.Stderr, "  generate              Generate a new RSA key pair\n")
	fmt.Fprintf(os.Stderr, "    -kid <id>           Key ID (required)\n")
	fmt.Fprintf(os.Stderr, "    -bits <size>        Key size: 2048, 3072, or 4096 (default: 2048)\n")
	fmt.Fprintf(os.Stderr, "    -path <dir>         Custom keys directory (overrides config)\n")
	fmt.Fprintf(os.Stderr, "  list                  List all available keys\n")
	fmt.Fprintf(os.Stderr, "  set-active <kid>      Set active key ID\n")
}

func (c *Command) runGenerate(args []string) error {
	fs := flag.NewFlagSet("generate", flag.ExitOnError)
	kid := fs.String("kid", "", "Key ID (required)")
	bits := fs.Int("bits", 2048, "Key size in bits (2048, 3072, or 4096)")
	customPath := fs.String("path", "", "Custom keys directory path (overrides config)")

	if err := fs.Parse(args); err != nil {
		return err
	}

	if *kid == "" {
		return fmt.Errorf("key ID is required")
	}
	if *bits != 2048 && *bits != 3072 && *bits != 4096 {
		return fmt.Errorf("key size must be 2048, 3072, or 4096")
	}

	// Load config to get default path
	envConfig := config.LoadEnv()
	cfg, err := config.Load(envConfig.ConfigPath)
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	keysPath := cfg.Auth.KeysPath
	if *customPath != "" {
		keysPath = *customPath
	}

	return generateKey(keysPath, *kid, *bits)
}

func (c *Command) runList(args []string) error {
	fs := flag.NewFlagSet("list", flag.ExitOnError)
	customPath := fs.String("path", "", "Custom keys directory path (overrides config)")

	if err := fs.Parse(args); err != nil {
		return err
	}

	envConfig := config.LoadEnv()
	cfg, err := config.Load(envConfig.ConfigPath)
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	keysPath := cfg.Auth.KeysPath
	if *customPath != "" {
		keysPath = *customPath
	}

	return listKeys(keysPath, cfg.Auth.ActiveKID)
}

func (c *Command) runSetActive(args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("key ID required")
	}
	kid := args[0]

	envConfig := config.LoadEnv()
	cfg, err := config.Load(envConfig.ConfigPath)
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	return setActiveKey(cfg, kid)
}

func generateKey(keysPath, kid string, bits int) error {
	if err := os.MkdirAll(keysPath, 0700); err != nil {
		return fmt.Errorf("failed to create keys directory: %w", err)
	}

	privPath := filepath.Join(keysPath, fmt.Sprintf("private-%s.pem", kid))
	pubPath := filepath.Join(keysPath, fmt.Sprintf("public-%s.pem", kid))

	if _, err := os.Stat(privPath); err == nil {
		return fmt.Errorf("key with ID %s already exists at %s", kid, privPath)
	}

	// Test write permissions before generating keys
	testFile := filepath.Join(keysPath, ".write-test")
	if testF, err := os.OpenFile(testFile, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600); err != nil {
		return fmt.Errorf("no write permission to keys directory %s: %w", keysPath, err)
	} else {
		if err := testF.Close(); err != nil {
			return fmt.Errorf("failed to close write-test file %s: %w", testFile, err)
		}
		if err := os.Remove(testFile); err != nil {
			return fmt.Errorf("failed to remove write-test file %s: %w", testFile, err)
		}
	}

	fmt.Printf("Generating %d-bit RSA key pair...\n", bits)
	privateKey, err := rsa.GenerateKey(rand.Reader, bits)
	if err != nil {
		return fmt.Errorf("failed to generate RSA key: %w", err)
	}

	privateKeyPEM := &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(privateKey),
	}
	fPriv, err := os.OpenFile(privPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return err
	}
	if err := pem.Encode(fPriv, privateKeyPEM); err != nil {
		return err
	}
	if err := fPriv.Close(); err != nil {
		return err
	}

	publicKeyBytes, err := x509.MarshalPKIXPublicKey(&privateKey.PublicKey)
	if err != nil {
		return err
	}
	publicKeyPEM := &pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: publicKeyBytes,
	}
	fPub, err := os.OpenFile(pubPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0644)
	if err != nil {
		return err
	}
	if err := pem.Encode(fPub, publicKeyPEM); err != nil {
		if cerr := fPub.Close(); cerr != nil {
			return fmt.Errorf("failed to encode public key: %w (additionally, failed to close file: %v)", err, cerr)
		}
		return err
	}
	if err := fPub.Close(); err != nil {
		return err
	}

	fmt.Printf("Key pair generated successfully\n")
	fmt.Printf("  Key ID: %s\n", kid)
	return nil
}

func listKeys(keysPath, activeKID string) error {
	info, err := os.Stat(keysPath)
	if err != nil || !info.IsDir() {
		return fmt.Errorf("keys directory invalid: %s", keysPath)
	}

	keyStore, err := auth.LoadKeys(keysPath, activeKID)
	if err != nil {
		return fmt.Errorf("failed to load keys: %w", err)
	}

	keySet := keyStore.JWKS()
	if keySet.Len() == 0 {
		fmt.Printf("No keys found in %s\n", keysPath)
		return nil
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
			} else {
				fmt.Fprintf(os.Stderr, "  %s: skipped (not an RSA key)\n", kid)
			}
		} else {
			fmt.Fprintf(os.Stderr, "  %s: skipped (export failed: %v)\n", kid, err)
		}
	}

	fmt.Printf("Active KID: %s\n", activeKID)
	return nil
}

func setActiveKey(cfg *config.Config, kid string) error {
	keyStore, err := auth.LoadKeys(cfg.Auth.KeysPath, cfg.Auth.ActiveKID)
	if err != nil {
		return err
	}

	keyID := fmt.Sprintf("key-%s", kid)
	if _, ok := keyStore.KeySet.LookupKeyID(keyID); !ok {
		return fmt.Errorf("key with ID %s not found", kid)
	}

	fmt.Printf("To set active key, update config.yaml:\n\n")
	fmt.Printf("  auth:\n")
	fmt.Printf("    active_kid: %s\n", kid)
	return nil
}
