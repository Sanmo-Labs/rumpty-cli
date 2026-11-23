package ssh

import (
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/cloverworks-hq/rumpty-cli/internal/api"
	"github.com/cloverworks-hq/rumpty-cli/internal/credentials"
	rumptylog "github.com/cloverworks-hq/rumpty-cli/internal/log"
)

const certReuseSkew = 30 * time.Second

const certStoreVersion = 1

var sessionCache = &certCache{entries: make(map[string]cachedSession)}

type certCache struct {
	mu      sync.Mutex
	entries map[string]cachedSession
}

type cachedSession struct {
	key       KeyPair
	session   api.CertResponse
	expiresAt time.Time
}

type certStoreFile struct {
	Version int              `json:"version"`
	Entries []certStoreEntry `json:"entries"`
}

type certStoreEntry struct {
	APIURL        string           `json:"api_url"`
	Workspace     string           `json:"workspace"`
	VMSlug        string           `json:"vm_slug"`
	GuestUser     string           `json:"guest_user,omitempty"`
	ExpiresAt     string           `json:"expires_at"`
	PrivateKeyB64 string           `json:"private_key,omitempty"` //nolint:gosec // ephemeral SSH key, only when no keychain is available
	KeyInKeychain bool             `json:"private_key_in_keychain,omitempty"`
	PublicKeyB64  string           `json:"public_key"`
	Session       api.CertResponse `json:"session"`
}

func certCacheKey(apiURL, workspace, vmSlug, guestUser string) string {
	return apiURL + "\x00" + workspace + "\x00" + vmSlug + "\x00" + guestUser
}

func certKeyAccount(apiURL, workspace, vmSlug, guestUser string) string {
	sum := sha256.Sum256([]byte(certCacheKey(apiURL, workspace, vmSlug, guestUser)))
	return "ssh-key:" + hex.EncodeToString(sum[:12])
}

func (e *certStoreEntry) account() string {
	return certKeyAccount(e.APIURL, e.Workspace, e.VMSlug, e.GuestUser)
}

func (e *certStoreEntry) forgetKey() {
	if e.KeyInKeychain {
		credentials.DeleteSecret(e.account())
	}
}

func (c *certCache) get(apiURL, workspace, vmSlug, guestUser string) (KeyPair, api.CertResponse, bool) {
	key := certCacheKey(apiURL, workspace, vmSlug, guestUser)

	c.mu.Lock()
	if entry, ok := c.entries[key]; ok && certStillValid(entry.expiresAt) {
		cached := entry
		c.mu.Unlock()
		return cached.key, cached.session, true
	}
	c.mu.Unlock()

	entry, ok, err := loadCertStoreEntry(apiURL, workspace, vmSlug, guestUser)
	if err != nil {
		return KeyPair{}, api.CertResponse{}, false
	}
	if !ok {
		return KeyPair{}, api.CertResponse{}, false
	}

	c.mu.Lock()
	c.entries[key] = entry
	c.mu.Unlock()
	return entry.key, entry.session, true
}

func (c *certCache) put(apiURL, workspace, vmSlug, guestUser string, key KeyPair, session *api.CertResponse) {
	entry := cachedSession{
		key:       key,
		session:   *session,
		expiresAt: certExpiry(session),
	}

	cacheKey := certCacheKey(apiURL, workspace, vmSlug, guestUser)
	c.mu.Lock()
	c.entries[cacheKey] = entry
	c.mu.Unlock()

	if err := saveCertStoreEntry(apiURL, workspace, vmSlug, guestUser, key, session); err != nil {
		rumptylog.Debug("unable to persist ssh certificate cache", "error", err)
	} else if path, err := certStorePath(); err == nil {
		rumptylog.Debug("persisted ssh certificate cache", "path", path)
	}
}

func certStillValid(expiresAt time.Time) bool {
	return time.Until(expiresAt) > certReuseSkew
}

func certExpiry(session *api.CertResponse) time.Time {
	if t, err := time.Parse(time.RFC3339, session.ExpiresAt); err == nil {
		return t
	}
	return time.Now().Add(4 * time.Minute)
}

func certStorePath() (string, error) {
	dir, err := credentials.Dir()
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", fmt.Errorf("create rumpty config dir: %w", err)
	}
	return filepath.Join(dir, "ssh-certs.json"), nil
}

func loadCertStoreEntry(apiURL, workspace, vmSlug, guestUser string) (cachedSession, bool, error) {
	path, err := certStorePath()
	if err != nil {
		return cachedSession{}, false, err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return cachedSession{}, false, nil
		}
		return cachedSession{}, false, err
	}

	var store certStoreFile
	if err := json.Unmarshal(data, &store); err != nil {
		return cachedSession{}, false, nil
	}

	var kept []certStoreEntry
	var match *certStoreEntry
	for i := range store.Entries {
		entry := store.Entries[i]
		expiresAt, err := time.Parse(time.RFC3339, entry.ExpiresAt)
		if err != nil || !certStillValid(expiresAt) {
			entry.forgetKey()
			continue
		}
		kept = append(kept, entry)
		if entry.APIURL == apiURL && entry.Workspace == workspace && entry.VMSlug == vmSlug && entry.GuestUser == guestUser {
			copy := entry
			match = &copy
		}
	}
	if len(kept) != len(store.Entries) {
		_ = writeCertStore(kept)
	}

	if match == nil {
		return cachedSession{}, false, nil
	}

	privateB64 := match.PrivateKeyB64
	if match.KeyInKeychain {
		privateB64, _ = credentials.LoadSecret(match.account())
	}
	key, err := decodeStoredKeyPair(privateB64, match.PublicKeyB64)
	if err != nil {
		return cachedSession{}, false, nil
	}

	expiresAt, _ := time.Parse(time.RFC3339, match.ExpiresAt)
	return cachedSession{
		key:       key,
		session:   match.Session,
		expiresAt: expiresAt,
	}, true, nil
}

func saveCertStoreEntry(apiURL, workspace, vmSlug, guestUser string, key KeyPair, session *api.CertResponse) error {
	path, err := certStorePath()
	if err != nil {
		return err
	}

	var store certStoreFile
	if data, err := os.ReadFile(path); err == nil {
		_ = json.Unmarshal(data, &store)
	}
	if store.Version == 0 {
		store.Version = certStoreVersion
	}

	var kept []certStoreEntry
	for i := range store.Entries {
		entry := store.Entries[i]
		if entry.APIURL == apiURL && entry.Workspace == workspace && entry.VMSlug == vmSlug && entry.GuestUser == guestUser {
			continue
		}
		expiresAt, err := time.Parse(time.RFC3339, entry.ExpiresAt)
		if err != nil || !certStillValid(expiresAt) {
			entry.forgetKey()
			continue
		}
		kept = append(kept, entry)
	}

	entry := certStoreEntry{
		APIURL:       apiURL,
		Workspace:    workspace,
		VMSlug:       vmSlug,
		GuestUser:    guestUser,
		ExpiresAt:    session.ExpiresAt,
		PublicKeyB64: base64.StdEncoding.EncodeToString(key.Public),
		Session:      *session,
	}
	privB64 := base64.StdEncoding.EncodeToString(key.Private)
	if credentials.StoreSecret(entry.account(), privB64) {
		entry.KeyInKeychain = true
	} else {
		credentials.DeleteSecret(entry.account())
		entry.PrivateKeyB64 = privB64
	}
	kept = append(kept, entry)

	return writeCertStore(kept)
}

func writeCertStore(entries []certStoreEntry) error {
	path, err := certStorePath()
	if err != nil {
		return err
	}

	data, err := json.MarshalIndent(certStoreFile{
		Version: certStoreVersion,
		Entries: entries,
	}, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return credentials.WriteSecretFile(path, data)
}

func decodeStoredKeyPair(privateB64, publicB64 string) (KeyPair, error) {
	priv, err := base64.StdEncoding.DecodeString(privateB64)
	if err != nil {
		return KeyPair{}, err
	}
	pub, err := base64.StdEncoding.DecodeString(publicB64)
	if err != nil {
		return KeyPair{}, err
	}
	if len(priv) != ed25519.PrivateKeySize || len(pub) != ed25519.PublicKeySize {
		return KeyPair{}, fmt.Errorf("invalid cached key size")
	}
	if !ed25519.PublicKey(pub).Equal(ed25519.PrivateKey(priv).Public()) {
		return KeyPair{}, fmt.Errorf("cached private key does not match its certificate")
	}
	return KeyPair{
		Public:  ed25519.PublicKey(pub),
		Private: ed25519.PrivateKey(priv),
	}, nil
}

func resetCertCacheForTest() {
	sessionCache.mu.Lock()
	defer sessionCache.mu.Unlock()
	sessionCache.entries = make(map[string]cachedSession)
}

func resetCertStoreForTest() error {
	path, err := certStorePath()
	if err != nil {
		return err
	}
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

// ClearCertCache removes cached SSH certificates and their private keys,
// including keys kept in the OS keychain.
func ClearCertCache() error {
	resetCertCacheForTest()
	if path, err := certStorePath(); err == nil {
		if data, err := os.ReadFile(path); err == nil {
			var store certStoreFile
			if json.Unmarshal(data, &store) == nil {
				for i := range store.Entries {
					store.Entries[i].forgetKey()
				}
			}
		}
	}
	return resetCertStoreForTest()
}
