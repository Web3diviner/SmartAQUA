package services

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/hex"
	"encoding/pem"
	"errors"
	"fmt"
	"math/big"
	"sync"
	"time"

	"smart-fish-feeder/internal/config"
	"smart-fish-feeder/internal/redis"
	"smart-fish-feeder/internal/repository"
)

// CertificateService handles X.509 certificate management for mTLS device authentication
type CertificateService struct {
	repo   *repository.Repository
	redis  *redis.Client
	config *config.Config
	mu     sync.RWMutex

	// CA certificate and key (in production, use HSM)
	caCert   *x509.Certificate
	caKey    *ecdsa.PrivateKey
	caPEM    []byte
	caKeyPEM []byte

	// Certificate store (in production, use database)
	deviceCerts map[string]*DeviceCertificate
}

// DeviceCertificate represents a device's certificate information
type DeviceCertificate struct {
	DeviceID      string     `json:"device_id"`
	SerialNumber  string     `json:"serial_number"`
	Certificate   []byte     `json:"certificate"`
	PublicKey     []byte     `json:"public_key"`
	IssuedAt      time.Time  `json:"issued_at"`
	ExpiresAt     time.Time  `json:"expires_at"`
	Revoked       bool       `json:"revoked"`
	RevokedAt     *time.Time `json:"revoked_at,omitempty"`
	RevokeReason  string     `json:"revoke_reason,omitempty"`
	LastRotation  *time.Time `json:"last_rotation,omitempty"`
	RotationCount int        `json:"rotation_count"`
}

// CertificateRequest represents a certificate signing request
type CertificateRequest struct {
	DeviceID     string `json:"device_id" validate:"required"`
	DeviceSerial string `json:"device_serial" validate:"required"`
	PublicKeyPEM []byte `json:"public_key_pem" validate:"required"`
}

// CertificateResponse represents certificate issuance response
type CertificateResponse struct {
	DeviceID      string    `json:"device_id"`
	Certificate   string    `json:"certificate"`
	CACertificate string    `json:"ca_certificate"`
	SerialNumber  string    `json:"serial_number"`
	IssuedAt      time.Time `json:"issued_at"`
	ExpiresAt     time.Time `json:"expires_at"`
	ValidityDays  int       `json:"validity_days"`
}

// CertificateStatus represents certificate status information
type CertificateStatus struct {
	DeviceID        string    `json:"device_id"`
	SerialNumber    string    `json:"serial_number"`
	IsValid         bool      `json:"is_valid"`
	IsExpired       bool      `json:"is_expired"`
	IsRevoked       bool      `json:"is_revoked"`
	DaysUntilExpiry int       `json:"days_until_expiry"`
	NeedsRotation   bool      `json:"needs_rotation"`
	IssuedAt        time.Time `json:"issued_at"`
	ExpiresAt       time.Time `json:"expires_at"`
}

// FirmwareSignature represents firmware signature verification data
type FirmwareSignature struct {
	FirmwareVersion string    `json:"firmware_version"`
	FirmwareHash    string    `json:"firmware_hash"`
	Signature       []byte    `json:"signature"`
	SignedAt        time.Time `json:"signed_at"`
	SignedBy        string    `json:"signed_by"`
}

// FirmwareVerificationResult represents firmware verification result
type FirmwareVerificationResult struct {
	IsValid         bool      `json:"is_valid"`
	FirmwareVersion string    `json:"firmware_version"`
	FirmwareHash    string    `json:"firmware_hash"`
	VerifiedAt      time.Time `json:"verified_at"`
	ErrorMessage    string    `json:"error_message,omitempty"`
}

// NewCertificateService creates a new certificate service
func NewCertificateService(repo *repository.Repository, redisClient *redis.Client, cfg *config.Config) *CertificateService {
	svc := &CertificateService{
		repo:        repo,
		redis:       redisClient,
		config:      cfg,
		deviceCerts: make(map[string]*DeviceCertificate),
	}

	// Initialize CA certificate
	if err := svc.initializeCA(); err != nil {
		// Log error but don't fail - CA can be initialized later
		fmt.Printf("Warning: Failed to initialize CA: %v\n", err)
	}

	return svc
}

// initializeCA creates or loads the Certificate Authority
func (s *CertificateService) initializeCA() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Generate CA private key
	caKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return fmt.Errorf("failed to generate CA key: %w", err)
	}

	// Create CA certificate template
	serialNumber, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		return fmt.Errorf("failed to generate serial number: %w", err)
	}

	caTemplate := &x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			Organization: []string{"Smart Fish Feeder"},
			Country:      []string{"US"},
			Province:     []string{""},
			Locality:     []string{""},
			CommonName:   "Smart Fish Feeder Root CA",
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().AddDate(10, 0, 0), // 10 years validity
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth},
		BasicConstraintsValid: true,
		IsCA:                  true,
		MaxPathLen:            1,
	}

	// Self-sign the CA certificate
	caCertDER, err := x509.CreateCertificate(rand.Reader, caTemplate, caTemplate, &caKey.PublicKey, caKey)
	if err != nil {
		return fmt.Errorf("failed to create CA certificate: %w", err)
	}

	// Parse the certificate
	caCert, err := x509.ParseCertificate(caCertDER)
	if err != nil {
		return fmt.Errorf("failed to parse CA certificate: %w", err)
	}

	// Encode to PEM
	s.caPEM = pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: caCertDER})

	caKeyDER, err := x509.MarshalECPrivateKey(caKey)
	if err != nil {
		return fmt.Errorf("failed to marshal CA key: %w", err)
	}
	s.caKeyPEM = pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: caKeyDER})

	s.caCert = caCert
	s.caKey = caKey

	return nil
}

// IssueCertificate issues a new certificate for a device
func (s *CertificateService) IssueCertificate(req *CertificateRequest) (*CertificateResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.caCert == nil || s.caKey == nil {
		return nil, errors.New("CA not initialized")
	}

	// Parse the public key from PEM
	block, _ := pem.Decode(req.PublicKeyPEM)
	if block == nil {
		return nil, errors.New("failed to decode public key PEM")
	}

	pubKey, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse public key: %w", err)
	}

	// Generate serial number
	serialNumber, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		return nil, fmt.Errorf("failed to generate serial number: %w", err)
	}

	// Certificate validity (1 year for devices)
	validityDays := 365
	notBefore := time.Now()
	notAfter := notBefore.AddDate(0, 0, validityDays)

	// Create certificate template
	certTemplate := &x509.Certificate{
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			Organization: []string{"Smart Fish Feeder"},
			CommonName:   fmt.Sprintf("device-%s", req.DeviceID),
		},
		NotBefore:             notBefore,
		NotAfter:              notAfter,
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
		BasicConstraintsValid: true,
		IsCA:                  false,
	}

	// Sign the certificate with CA
	certDER, err := x509.CreateCertificate(rand.Reader, certTemplate, s.caCert, pubKey, s.caKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create certificate: %w", err)
	}

	// Encode to PEM
	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER})

	// Store certificate info
	deviceCert := &DeviceCertificate{
		DeviceID:      req.DeviceID,
		SerialNumber:  serialNumber.String(),
		Certificate:   certPEM,
		PublicKey:     req.PublicKeyPEM,
		IssuedAt:      notBefore,
		ExpiresAt:     notAfter,
		Revoked:       false,
		RotationCount: 0,
	}

	// Check if this is a rotation
	if existing, exists := s.deviceCerts[req.DeviceID]; exists {
		deviceCert.RotationCount = existing.RotationCount + 1
		now := time.Now()
		deviceCert.LastRotation = &now
	}

	s.deviceCerts[req.DeviceID] = deviceCert

	return &CertificateResponse{
		DeviceID:      req.DeviceID,
		Certificate:   string(certPEM),
		CACertificate: string(s.caPEM),
		SerialNumber:  serialNumber.String(),
		IssuedAt:      notBefore,
		ExpiresAt:     notAfter,
		ValidityDays:  validityDays,
	}, nil
}

// VerifyCertificate verifies a device certificate
func (s *CertificateService) VerifyCertificate(deviceID string, certPEM []byte) (*CertificateStatus, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.caCert == nil {
		return nil, errors.New("CA not initialized")
	}

	// Parse the certificate
	block, _ := pem.Decode(certPEM)
	if block == nil {
		return nil, errors.New("failed to decode certificate PEM")
	}

	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse certificate: %w", err)
	}

	// Create certificate pool with CA
	roots := x509.NewCertPool()
	roots.AddCert(s.caCert)

	// Verify certificate chain
	opts := x509.VerifyOptions{
		Roots:       roots,
		CurrentTime: time.Now(),
		KeyUsages:   []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
	}

	_, err = cert.Verify(opts)
	isValid := err == nil

	// Check expiration
	now := time.Now()
	isExpired := now.After(cert.NotAfter)
	daysUntilExpiry := int(cert.NotAfter.Sub(now).Hours() / 24)

	// Check revocation status
	isRevoked := false
	if stored, exists := s.deviceCerts[deviceID]; exists {
		isRevoked = stored.Revoked
	}

	// Determine if rotation is needed (30 days before expiry)
	needsRotation := daysUntilExpiry <= 30

	return &CertificateStatus{
		DeviceID:        deviceID,
		SerialNumber:    cert.SerialNumber.String(),
		IsValid:         isValid && !isRevoked,
		IsExpired:       isExpired,
		IsRevoked:       isRevoked,
		DaysUntilExpiry: daysUntilExpiry,
		NeedsRotation:   needsRotation,
		IssuedAt:        cert.NotBefore,
		ExpiresAt:       cert.NotAfter,
	}, nil
}

// RevokeCertificate revokes a device certificate
func (s *CertificateService) RevokeCertificate(deviceID string, reason string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	cert, exists := s.deviceCerts[deviceID]
	if !exists {
		return errors.New("certificate not found")
	}

	if cert.Revoked {
		return errors.New("certificate already revoked")
	}

	now := time.Now()
	cert.Revoked = true
	cert.RevokedAt = &now
	cert.RevokeReason = reason

	return nil
}

// RotateCertificate rotates a device certificate
func (s *CertificateService) RotateCertificate(deviceID string, newPublicKeyPEM []byte) (*CertificateResponse, error) {
	// Check if device has existing certificate
	s.mu.RLock()
	existing, exists := s.deviceCerts[deviceID]
	s.mu.RUnlock()

	if !exists {
		return nil, errors.New("no existing certificate found for device")
	}

	// Revoke old certificate
	if err := s.RevokeCertificate(deviceID, "certificate rotation"); err != nil {
		// Log but continue - old cert might already be revoked
		fmt.Printf("Warning: Failed to revoke old certificate: %v\n", err)
	}

	// Issue new certificate
	return s.IssueCertificate(&CertificateRequest{
		DeviceID:     deviceID,
		DeviceSerial: existing.DeviceID, // Use device ID as serial for simplicity
		PublicKeyPEM: newPublicKeyPEM,
	})
}

// GetCertificateStatus gets the status of a device's certificate
func (s *CertificateService) GetCertificateStatus(deviceID string) (*CertificateStatus, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	cert, exists := s.deviceCerts[deviceID]
	if !exists {
		return nil, errors.New("certificate not found")
	}

	now := time.Now()
	isExpired := now.After(cert.ExpiresAt)
	daysUntilExpiry := int(cert.ExpiresAt.Sub(now).Hours() / 24)
	needsRotation := daysUntilExpiry <= 30

	return &CertificateStatus{
		DeviceID:        deviceID,
		SerialNumber:    cert.SerialNumber,
		IsValid:         !cert.Revoked && !isExpired,
		IsExpired:       isExpired,
		IsRevoked:       cert.Revoked,
		DaysUntilExpiry: daysUntilExpiry,
		NeedsRotation:   needsRotation,
		IssuedAt:        cert.IssuedAt,
		ExpiresAt:       cert.ExpiresAt,
	}, nil
}

// SignFirmware signs firmware for secure OTA updates
func (s *CertificateService) SignFirmware(firmwareVersion string, firmwareData []byte) (*FirmwareSignature, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.caKey == nil {
		return nil, errors.New("CA not initialized")
	}

	// Calculate firmware hash
	hash := sha256.Sum256(firmwareData)
	hashHex := hex.EncodeToString(hash[:])

	// Sign the hash
	signature, err := ecdsa.SignASN1(rand.Reader, s.caKey, hash[:])
	if err != nil {
		return nil, fmt.Errorf("failed to sign firmware: %w", err)
	}

	return &FirmwareSignature{
		FirmwareVersion: firmwareVersion,
		FirmwareHash:    hashHex,
		Signature:       signature,
		SignedAt:        time.Now(),
		SignedBy:        "Smart Fish Feeder CA",
	}, nil
}

// VerifyFirmwareSignature verifies firmware signature
func (s *CertificateService) VerifyFirmwareSignature(firmwareData []byte, sig *FirmwareSignature) (*FirmwareVerificationResult, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.caCert == nil {
		return nil, errors.New("CA not initialized")
	}

	result := &FirmwareVerificationResult{
		FirmwareVersion: sig.FirmwareVersion,
		VerifiedAt:      time.Now(),
	}

	// Calculate firmware hash
	hash := sha256.Sum256(firmwareData)
	hashHex := hex.EncodeToString(hash[:])
	result.FirmwareHash = hashHex

	// Verify hash matches
	if hashHex != sig.FirmwareHash {
		result.IsValid = false
		result.ErrorMessage = "firmware hash mismatch"
		return result, nil
	}

	// Get CA public key
	caPublicKey, ok := s.caCert.PublicKey.(*ecdsa.PublicKey)
	if !ok {
		return nil, errors.New("CA public key is not ECDSA")
	}

	// Verify signature
	valid := ecdsa.VerifyASN1(caPublicKey, hash[:], sig.Signature)
	result.IsValid = valid

	if !valid {
		result.ErrorMessage = "signature verification failed"
	}

	return result, nil
}

// GetCACertificate returns the CA certificate in PEM format
func (s *CertificateService) GetCACertificate() ([]byte, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.caPEM == nil {
		return nil, errors.New("CA not initialized")
	}

	return s.caPEM, nil
}

// ListDeviceCertificates lists all device certificates
func (s *CertificateService) ListDeviceCertificates() []*DeviceCertificate {
	s.mu.RLock()
	defer s.mu.RUnlock()

	certs := make([]*DeviceCertificate, 0, len(s.deviceCerts))
	for _, cert := range s.deviceCerts {
		certs = append(certs, cert)
	}
	return certs
}

// GetExpiringCertificates returns certificates expiring within specified days
func (s *CertificateService) GetExpiringCertificates(withinDays int) []*DeviceCertificate {
	s.mu.RLock()
	defer s.mu.RUnlock()

	threshold := time.Now().AddDate(0, 0, withinDays)
	expiring := make([]*DeviceCertificate, 0)

	for _, cert := range s.deviceCerts {
		if !cert.Revoked && cert.ExpiresAt.Before(threshold) {
			expiring = append(expiring, cert)
		}
	}

	return expiring
}
