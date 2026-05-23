package ssh

import (
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/Sanmo-Labs/rumpty-cli/internal/api"
	"github.com/Sanmo-Labs/rumpty-cli/internal/credentials"
	rumptylog "github.com/Sanmo-Labs/rumpty-cli/internal/log"
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
	PrivateKeyB64 string           `json:"private_key"` //nolint:gosec // ephemeral SSH key in local cert cache only
	PublicKeyB64  string           `json:"public_key"`
	Session       api.CertResponse `json:"session"`
}

func certCacheKey(apiURL, workspace, vmSlug, guestUser string) string {
	return apiURL + "\x00" + workspace + "\x00" + vmSlug + "\x00" + guestUser
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

	key, err := decodeStoredKeyPair(match.PrivateKeyB64, match.PublicKeyB64)
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
		expiresAt, err := time.Parse(time.RFC3339, entry.ExpiresAt)
		if err != nil || !certStillValid(expiresAt) {
			continue
		}
		if entry.APIURL == apiURL && entry.Workspace == workspace && entry.VMSlug == vmSlug && entry.GuestUser == guestUser {
			continue
		}
		kept = append(kept, entry)
	}

	privB64 := base64.StdEncoding.EncodeToString(key.Private)
	pubB64 := base64.StdEncoding.EncodeToString(key.Public)
	kept = append(kept, certStoreEntry{
		APIURL:        apiURL,
		Workspace:     workspace,
		VMSlug:        vmSlug,
		GuestUser:     guestUser,
		ExpiresAt:     session.ExpiresAt,
		PrivateKeyB64: privB64,
		PublicKeyB64:  pubB64,
		Session:       *session,
	})

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
	return os.WriteFile(path, data, 0o600)
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
