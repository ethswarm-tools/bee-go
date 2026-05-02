// swarm-keyring is a passphrase-encrypted secp256k1 key store.
//
// Generate, import, list, export, and "use" Swarm signing keys
// without keeping them in plaintext on disk. Each key is encrypted
// with AES-256-GCM using a key derived from the user's passphrase
// via scrypt, and stored in keyring.json.
//
// Usage:
//
//	swarm-keyring new      <name>          # generate + store
//	swarm-keyring import   <name> <hex>    # store an existing key
//	swarm-keyring list                     # name → eth address
//	swarm-keyring export   <name>          # decrypt + print hex
//	swarm-keyring address  <name>          # print eth address only
//
// Passphrase is read from BEE_KEYRING_PASS; for production CLI use,
// swap to a TTY prompt (golang.org/x/term ReadPassword).
package main

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"

	"golang.org/x/crypto/scrypt"

	"github.com/ethswarm-tools/bee-go/pkg/swarm"
)

const keyringFile = "keyring.json"

type encrypted struct {
	Name      string `json:"name"`
	Address   string `json:"address"`
	SaltHex   string `json:"salt_hex"`
	NonceHex  string `json:"nonce_hex"`
	CipherHex string `json:"cipher_hex"`
	LogN      uint8  `json:"log_n"`
	R         int    `json:"r"`
	P         int    `json:"p"`
}

type keyring struct {
	Keys []encrypted `json:"keys"`
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	args := os.Args[1:]
	if len(args) == 0 {
		return fmt.Errorf("usage: swarm-keyring <new|import|list|export|address>")
	}
	switch args[0] {
	case "new":
		if len(args) < 2 {
			return fmt.Errorf("usage: swarm-keyring new <name>")
		}
		return cmdNew(args[1])
	case "import":
		if len(args) < 3 {
			return fmt.Errorf("usage: swarm-keyring import <name> <hex>")
		}
		return cmdImport(args[1], args[2])
	case "list":
		return cmdList()
	case "export":
		if len(args) < 2 {
			return fmt.Errorf("usage: swarm-keyring export <name>")
		}
		return cmdExport(args[1])
	case "address":
		if len(args) < 2 {
			return fmt.Errorf("usage: swarm-keyring address <name>")
		}
		return cmdAddress(args[1])
	default:
		return fmt.Errorf("unknown command: %s", args[0])
	}
}

func cmdNew(name string) error {
	pass, err := passphrase()
	if err != nil {
		return err
	}
	var bytes [32]byte
	if _, err := rand.Read(bytes[:]); err != nil {
		return fmt.Errorf("rand: %w", err)
	}
	pk, err := swarm.NewPrivateKey(bytes[:])
	if err != nil {
		return fmt.Errorf("private key: %w", err)
	}
	if err := insert(name, pk, pass); err != nil {
		return err
	}
	fmt.Printf("created %s: %s\n", name, pk.PublicKey().Address().Hex())
	return nil
}

func cmdImport(name, hexStr string) error {
	pk, err := swarm.PrivateKeyFromHex(hexStr)
	if err != nil {
		return fmt.Errorf("invalid hex: %w", err)
	}
	pass, err := passphrase()
	if err != nil {
		return err
	}
	if err := insert(name, pk, pass); err != nil {
		return err
	}
	fmt.Printf("imported %s: %s\n", name, pk.PublicKey().Address().Hex())
	return nil
}

func cmdList() error {
	kr := load()
	if len(kr.Keys) == 0 {
		fmt.Println("(empty keyring)")
		return nil
	}
	fmt.Printf("%-20s  %s\n", "name", "address")
	for _, k := range kr.Keys {
		fmt.Printf("%-20s  %s\n", k.Name, k.Address)
	}
	return nil
}

func cmdExport(name string) error {
	pass, err := passphrase()
	if err != nil {
		return err
	}
	pk, err := decrypt(name, pass)
	if err != nil {
		return err
	}
	fmt.Println(pk.Hex())
	return nil
}

func cmdAddress(name string) error {
	kr := load()
	for _, k := range kr.Keys {
		if k.Name == name {
			fmt.Println(k.Address)
			return nil
		}
	}
	return fmt.Errorf("no key named %s", name)
}

func insert(name string, pk swarm.PrivateKey, pass string) error {
	kr := load()
	for _, k := range kr.Keys {
		if k.Name == name {
			return fmt.Errorf("key %s already exists", name)
		}
	}
	var salt [16]byte
	if _, err := rand.Read(salt[:]); err != nil {
		return err
	}
	var nonce [12]byte
	if _, err := rand.Read(nonce[:]); err != nil {
		return err
	}
	logN := uint8(14)
	r := 8
	p := 1
	key, err := deriveKey(pass, salt[:], logN, r, p)
	if err != nil {
		return err
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return fmt.Errorf("aes: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return fmt.Errorf("gcm: %w", err)
	}
	ct := gcm.Seal(nil, nonce[:], pk.Raw(), nil)

	kr.Keys = append(kr.Keys, encrypted{
		Name:      name,
		Address:   pk.PublicKey().Address().Hex(),
		SaltHex:   hex.EncodeToString(salt[:]),
		NonceHex:  hex.EncodeToString(nonce[:]),
		CipherHex: hex.EncodeToString(ct),
		LogN:      logN,
		R:         r,
		P:         p,
	})
	return save(kr)
}

func decrypt(name, pass string) (swarm.PrivateKey, error) {
	kr := load()
	var entry *encrypted
	for i := range kr.Keys {
		if kr.Keys[i].Name == name {
			entry = &kr.Keys[i]
			break
		}
	}
	if entry == nil {
		return swarm.PrivateKey{}, fmt.Errorf("no key named %s", name)
	}
	salt, err := hex.DecodeString(entry.SaltHex)
	if err != nil {
		return swarm.PrivateKey{}, fmt.Errorf("salt: %w", err)
	}
	nonce, err := hex.DecodeString(entry.NonceHex)
	if err != nil {
		return swarm.PrivateKey{}, fmt.Errorf("nonce: %w", err)
	}
	ct, err := hex.DecodeString(entry.CipherHex)
	if err != nil {
		return swarm.PrivateKey{}, fmt.Errorf("cipher: %w", err)
	}
	key, err := deriveKey(pass, salt, entry.LogN, entry.R, entry.P)
	if err != nil {
		return swarm.PrivateKey{}, err
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return swarm.PrivateKey{}, fmt.Errorf("aes: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return swarm.PrivateKey{}, fmt.Errorf("gcm: %w", err)
	}
	pt, err := gcm.Open(nil, nonce, ct, nil)
	if err != nil {
		return swarm.PrivateKey{}, fmt.Errorf("decryption failed (wrong passphrase?)")
	}
	return swarm.NewPrivateKey(pt)
}

func deriveKey(pass string, salt []byte, logN uint8, r, p int) ([]byte, error) {
	N := 1 << logN
	return scrypt.Key([]byte(pass), salt, N, r, p, 32)
}

func passphrase() (string, error) {
	v := os.Getenv("BEE_KEYRING_PASS")
	if v == "" {
		return "", fmt.Errorf("BEE_KEYRING_PASS not set (32-char passphrase recommended)")
	}
	return v, nil
}

func load() *keyring {
	bytes, err := os.ReadFile(keyringFile)
	if err != nil {
		return &keyring{}
	}
	var kr keyring
	if err := json.Unmarshal(bytes, &kr); err != nil {
		return &keyring{}
	}
	return &kr
}

func save(kr *keyring) error {
	bytes, err := json.MarshalIndent(kr, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(keyringFile, bytes, 0600)
}
