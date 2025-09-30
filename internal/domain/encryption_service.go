package domain

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"time"
)

// EncryptionService provides RSA-4096 encryption for contact tokens
type EncryptionService interface {
	// EncryptContactInfo encrypts contact information using current active key
	EncryptContactInfo(contactInfo ContactInfo) (*EncryptedContactInfo, error)

	// DecryptContactInfo decrypts contact information using the appropriate key
	DecryptContactInfo(encryptedInfo *EncryptedContactInfo) (*ContactInfo, error)

	// GenerateContactToken creates a secure token for contact exchange
	GenerateContactToken(contactInfo ContactInfo, expiresAt time.Time) (*ContactToken, error)

	// ValidateContactToken validates and decrypts a contact token
	ValidateContactToken(token *ContactToken) (*ContactInfo, error)

	// RotateKeys generates new key pair and marks current as old
	RotateKeys() error

	// GetActiveKeyFingerprint returns fingerprint of current active key
	GetActiveKeyFingerprint() string
}

// ContactInfo represents unencrypted contact information
type ContactInfo struct {
	Email           *string                  `json:"email,omitempty"`
	Phone           *string                  `json:"phone,omitempty"`
	PreferredMethod string                   `json:"preferred_method"`
	Message         *string                  `json:"message,omitempty"`
	Restrictions    *SharingRestrictions     `json:"restrictions,omitempty"`
}

// ContactToken represents an encrypted, time-limited contact exchange token
type ContactToken struct {
	Token           string    `json:"token"`            // Base64 encoded encrypted data
	KeyFingerprint  string    `json:"key_fingerprint"`  // Key used for encryption
	ExpiresAt       time.Time `json:"expires_at"`       // Token expiration
	CreatedAt       time.Time `json:"created_at"`       // Token creation time
	IntegrityHash   string    `json:"integrity_hash"`   // SHA256 hash for integrity
}

// EncryptionKey represents an RSA key pair with metadata
type EncryptionKey struct {
	ID           string    `json:"id"`            // Unique key identifier
	Fingerprint  string    `json:"fingerprint"`   // Key fingerprint for identification
	PrivateKey   string    `json:"private_key"`   // PEM encoded private key
	PublicKey    string    `json:"public_key"`    // PEM encoded public key
	IsActive     bool      `json:"is_active"`     // Whether this is the active key
	CreatedAt    time.Time `json:"created_at"`    // Key creation time
	ExpiresAt    *time.Time `json:"expires_at,omitempty"` // Optional key expiration
}

// EncryptionAuditLog represents audit trail for encryption operations
type EncryptionAuditLog struct {
	ID           string                 `json:"id"`
	Operation    EncryptionOperation    `json:"operation"`
	UserID       UserID                 `json:"user_id"`
	RequestID    *ContactExchangeRequestID `json:"request_id,omitempty"`
	KeyFingerprint string               `json:"key_fingerprint"`
	Success      bool                   `json:"success"`
	ErrorMessage *string                `json:"error_message,omitempty"`
	Timestamp    time.Time              `json:"timestamp"`
	IPAddress    *string                `json:"ip_address,omitempty"`
	UserAgent    *string                `json:"user_agent,omitempty"`
}

// EncryptionOperation represents types of encryption operations
type EncryptionOperation string

const (
	EncryptionOperationEncrypt     EncryptionOperation = "encrypt"
	EncryptionOperationDecrypt     EncryptionOperation = "decrypt"
	EncryptionOperationTokenCreate EncryptionOperation = "token_create"
	EncryptionOperationTokenValidate EncryptionOperation = "token_validate"
	EncryptionOperationKeyRotation EncryptionOperation = "key_rotation"
)

// RSAEncryptionService implements RSA-4096 encryption
type RSAEncryptionService struct {
	keyRepository KeyRepository
	auditLogger   EncryptionAuditLogger
	activeKey     *EncryptionKey
}

// NewRSAEncryptionService creates a new RSA encryption service
func NewRSAEncryptionService(keyRepo KeyRepository, auditLogger EncryptionAuditLogger) (*RSAEncryptionService, error) {
	service := &RSAEncryptionService{
		keyRepository: keyRepo,
		auditLogger:   auditLogger,
	}

	// Try to load active key
	activeKey, err := keyRepo.GetActiveKey()
	if err != nil {
		// If no active key exists, generate one
		if err := service.generateInitialKey(); err != nil {
			return nil, fmt.Errorf("failed to generate initial key: %w", err)
		}
		activeKey, err = keyRepo.GetActiveKey()
		if err != nil {
			return nil, fmt.Errorf("failed to load newly generated key: %w", err)
		}
	}

	service.activeKey = activeKey
	return service, nil
}

// EncryptContactInfo encrypts contact information using current active key
func (s *RSAEncryptionService) EncryptContactInfo(contactInfo ContactInfo) (*EncryptedContactInfo, error) {
	// Serialize contact info to JSON
	jsonData, err := json.Marshal(contactInfo)
	if err != nil {
		return nil, fmt.Errorf("failed to serialize contact info: %w", err)
	}

	// Encrypt using RSA-4096
	encryptedData, err := s.encryptWithActiveKey(jsonData)
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt contact info: %w", err)
	}

	// Create encrypted contact info
	encrypted := &EncryptedContactInfo{
		Email:           contactInfo.Email,
		Phone:           contactInfo.Phone,
		PreferredMethod: contactInfo.PreferredMethod,
		Message:         contactInfo.Message,
		SharingRestrictions: contactInfo.Restrictions,
	}

	// Store encrypted data as base64
	encryptedB64 := base64.StdEncoding.EncodeToString(encryptedData)
	encrypted.Email = &encryptedB64

	return encrypted, nil
}

// DecryptContactInfo decrypts contact information using the appropriate key
func (s *RSAEncryptionService) DecryptContactInfo(encryptedInfo *EncryptedContactInfo) (*ContactInfo, error) {
	if encryptedInfo.Email == nil {
		return nil, fmt.Errorf("no encrypted data found")
	}

	// Decode base64
	encryptedData, err := base64.StdEncoding.DecodeString(*encryptedInfo.Email)
	if err != nil {
		return nil, fmt.Errorf("failed to decode encrypted data: %w", err)
	}

	// Decrypt using RSA-4096
	decryptedData, err := s.decryptWithActiveKey(encryptedData)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt contact info: %w", err)
	}

	// Deserialize JSON
	var contactInfo ContactInfo
	if err := json.Unmarshal(decryptedData, &contactInfo); err != nil {
		return nil, fmt.Errorf("failed to deserialize contact info: %w", err)
	}

	return &contactInfo, nil
}

// GenerateContactToken creates a secure token for contact exchange
func (s *RSAEncryptionService) GenerateContactToken(contactInfo ContactInfo, expiresAt time.Time) (*ContactToken, error) {
	// Create token payload
	payload := map[string]interface{}{
		"contact_info": contactInfo,
		"expires_at":   expiresAt.Unix(),
		"created_at":   time.Now().Unix(),
		"nonce":        generateNonce(),
	}

	// Serialize payload
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to serialize token payload: %w", err)
	}

	// Encrypt payload
	encryptedData, err := s.encryptWithActiveKey(jsonData)
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt token payload: %w", err)
	}

	// Create integrity hash
	hash := sha256.Sum256(encryptedData)
	integrityHash := base64.StdEncoding.EncodeToString(hash[:])

	// Create contact token
	token := &ContactToken{
		Token:          base64.StdEncoding.EncodeToString(encryptedData),
		KeyFingerprint: s.activeKey.Fingerprint,
		ExpiresAt:      expiresAt,
		CreatedAt:      time.Now(),
		IntegrityHash:  integrityHash,
	}

	return token, nil
}

// ValidateContactToken validates and decrypts a contact token
func (s *RSAEncryptionService) ValidateContactToken(token *ContactToken) (*ContactInfo, error) {
	// Check expiration
	if time.Now().After(token.ExpiresAt) {
		return nil, fmt.Errorf("contact token has expired")
	}

	// Decode token
	encryptedData, err := base64.StdEncoding.DecodeString(token.Token)
	if err != nil {
		return nil, fmt.Errorf("failed to decode token: %w", err)
	}

	// Verify integrity
	hash := sha256.Sum256(encryptedData)
	expectedHash := base64.StdEncoding.EncodeToString(hash[:])
	if expectedHash != token.IntegrityHash {
		return nil, fmt.Errorf("token integrity check failed")
	}

	// Get appropriate key for decryption
	key, err := s.keyRepository.GetKeyByFingerprint(token.KeyFingerprint)
	if err != nil {
		return nil, fmt.Errorf("failed to get decryption key: %w", err)
	}

	// Decrypt payload
	decryptedData, err := s.decryptWithKey(encryptedData, key)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt token: %w", err)
	}

	// Deserialize payload
	var payload map[string]interface{}
	if err := json.Unmarshal(decryptedData, &payload); err != nil {
		return nil, fmt.Errorf("failed to deserialize token payload: %w", err)
	}

	// Extract contact info
	contactInfoData, ok := payload["contact_info"]
	if !ok {
		return nil, fmt.Errorf("contact info not found in token")
	}

	// Convert to ContactInfo
	jsonData, err := json.Marshal(contactInfoData)
	if err != nil {
		return nil, fmt.Errorf("failed to serialize contact info: %w", err)
	}

	var contactInfo ContactInfo
	if err := json.Unmarshal(jsonData, &contactInfo); err != nil {
		return nil, fmt.Errorf("failed to deserialize contact info: %w", err)
	}

	return &contactInfo, nil
}

// RotateKeys generates new key pair and marks current as old
func (s *RSAEncryptionService) RotateKeys() error {
	// Mark current active key as inactive
	if s.activeKey != nil {
		if err := s.keyRepository.MarkKeyInactive(s.activeKey.Fingerprint); err != nil {
			return fmt.Errorf("failed to mark current key inactive: %w", err)
		}
	}

	// Generate new key
	if err := s.generateInitialKey(); err != nil {
		return fmt.Errorf("failed to generate new key: %w", err)
	}

	// Load new active key
	activeKey, err := s.keyRepository.GetActiveKey()
	if err != nil {
		return fmt.Errorf("failed to load new active key: %w", err)
	}

	s.activeKey = activeKey
	return nil
}

// GetActiveKeyFingerprint returns fingerprint of current active key
func (s *RSAEncryptionService) GetActiveKeyFingerprint() string {
	if s.activeKey == nil {
		return ""
	}
	return s.activeKey.Fingerprint
}

// generateInitialKey generates the first RSA-4096 key pair
func (s *RSAEncryptionService) generateInitialKey() error {
	// Generate RSA-4096 key pair
	privateKey, err := rsa.GenerateKey(rand.Reader, 4096)
	if err != nil {
		return fmt.Errorf("failed to generate RSA key: %w", err)
	}

	// Encode private key to PEM
	privateKeyBytes := x509.MarshalPKCS1PrivateKey(privateKey)
	privateKeyPEM := &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: privateKeyBytes,
	}
	privateKeyString := string(pem.EncodeToMemory(privateKeyPEM))

	// Encode public key to PEM
	publicKeyBytes, err := x509.MarshalPKIXPublicKey(&privateKey.PublicKey)
	if err != nil {
		return fmt.Errorf("failed to marshal public key: %w", err)
	}

	publicKeyPEM := &pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: publicKeyBytes,
	}
	publicKeyString := string(pem.EncodeToMemory(publicKeyPEM))

	// Generate fingerprint
	fingerprint := generateKeyFingerprint(publicKeyBytes)

	// Create encryption key
	encryptionKey := &EncryptionKey{
		ID:          generateKeyID(),
		Fingerprint: fingerprint,
		PrivateKey:  privateKeyString,
		PublicKey:   publicKeyString,
		IsActive:    true,
		CreatedAt:   time.Now(),
	}

	// Save key
	return s.keyRepository.SaveKey(encryptionKey)
}

// encryptWithActiveKey encrypts data using the active key
func (s *RSAEncryptionService) encryptWithActiveKey(data []byte) ([]byte, error) {
	return s.encryptWithKey(data, s.activeKey)
}

// encryptWithKey encrypts data using specified key
func (s *RSAEncryptionService) encryptWithKey(data []byte, key *EncryptionKey) ([]byte, error) {
	// Parse public key
	block, _ := pem.Decode([]byte(key.PublicKey))
	if block == nil {
		return nil, fmt.Errorf("failed to parse public key PEM")
	}

	publicKey, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse public key: %w", err)
	}

	rsaPublicKey, ok := publicKey.(*rsa.PublicKey)
	if !ok {
		return nil, fmt.Errorf("not an RSA public key")
	}

	// Encrypt using RSA-OAEP with SHA-256
	return rsa.EncryptOAEP(sha256.New(), rand.Reader, rsaPublicKey, data, nil)
}

// decryptWithActiveKey decrypts data using the active key
func (s *RSAEncryptionService) decryptWithActiveKey(data []byte) ([]byte, error) {
	return s.decryptWithKey(data, s.activeKey)
}

// decryptWithKey decrypts data using specified key
func (s *RSAEncryptionService) decryptWithKey(data []byte, key *EncryptionKey) ([]byte, error) {
	// Parse private key
	block, _ := pem.Decode([]byte(key.PrivateKey))
	if block == nil {
		return nil, fmt.Errorf("failed to parse private key PEM")
	}

	privateKey, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse private key: %w", err)
	}

	// Decrypt using RSA-OAEP with SHA-256
	return rsa.DecryptOAEP(sha256.New(), rand.Reader, privateKey, data, nil)
}

// generateKeyFingerprint creates a SHA-256 fingerprint for the public key
func generateKeyFingerprint(publicKeyBytes []byte) string {
	hash := sha256.Sum256(publicKeyBytes)
	return base64.StdEncoding.EncodeToString(hash[:])
}

// generateKeyID creates a unique identifier for the key
func generateKeyID() string {
	return fmt.Sprintf("key_%d_%s", time.Now().Unix(), generateNonce())
}

// generateNonce creates a random nonce for token uniqueness
func generateNonce() string {
	bytes := make([]byte, 16)
	rand.Read(bytes)
	return base64.StdEncoding.EncodeToString(bytes)
}