package services

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/pem"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewCertificateService(t *testing.T) {
	svc := NewCertificateService(nil, nil, nil)
	assert.NotNil(t, svc)
	assert.NotNil(t, svc.deviceCerts)
}

func TestCertificateService_IssueCertificate(t *testing.T) {
	svc := NewCertificateService(nil, nil, nil)

	// Generate a test key pair
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)

	pubKeyDER, err := x509.MarshalPKIXPublicKey(&privateKey.PublicKey)
	require.NoError(t, err)

	pubKeyPEM := pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: pubKeyDER})

	req := &CertificateRequest{
		DeviceID:     "device-001",
		DeviceSerial: "SN-001",
		PublicKeyPEM: pubKeyPEM,
	}

	resp, err := svc.IssueCertificate(req)
	require.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, "device-001", resp.DeviceID)
	assert.NotEmpty(t, resp.Certificate)
	assert.NotEmpty(t, resp.CACertificate)
	assert.NotEmpty(t, resp.SerialNumber)
	assert.False(t, resp.IssuedAt.IsZero())
	assert.False(t, resp.ExpiresAt.IsZero())
	assert.Equal(t, 365, resp.ValidityDays)
}

func TestCertificateService_VerifyCertificate(t *testing.T) {
	svc := NewCertificateService(nil, nil, nil)

	// Generate and issue a certificate
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)

	pubKeyDER, err := x509.MarshalPKIXPublicKey(&privateKey.PublicKey)
	require.NoError(t, err)

	pubKeyPEM := pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: pubKeyDER})

	req := &CertificateRequest{
		DeviceID:     "device-002",
		DeviceSerial: "SN-002",
		PublicKeyPEM: pubKeyPEM,
	}

	resp, err := svc.IssueCertificate(req)
	require.NoError(t, err)

	// Verify the certificate
	status, err := svc.VerifyCertificate("device-002", []byte(resp.Certificate))
	require.NoError(t, err)
	assert.True(t, status.IsValid)
	assert.False(t, status.IsExpired)
	assert.False(t, status.IsRevoked)
	assert.Greater(t, status.DaysUntilExpiry, 360)
}

func TestCertificateService_RevokeCertificate(t *testing.T) {
	svc := NewCertificateService(nil, nil, nil)

	// Generate and issue a certificate
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)

	pubKeyDER, err := x509.MarshalPKIXPublicKey(&privateKey.PublicKey)
	require.NoError(t, err)

	pubKeyPEM := pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: pubKeyDER})

	req := &CertificateRequest{
		DeviceID:     "device-003",
		DeviceSerial: "SN-003",
		PublicKeyPEM: pubKeyPEM,
	}

	_, err = svc.IssueCertificate(req)
	require.NoError(t, err)

	// Revoke the certificate
	err = svc.RevokeCertificate("device-003", "test revocation")
	require.NoError(t, err)

	// Verify it's revoked
	status, err := svc.GetCertificateStatus("device-003")
	require.NoError(t, err)
	assert.True(t, status.IsRevoked)
	assert.False(t, status.IsValid)
}

func TestCertificateService_RotateCertificate(t *testing.T) {
	svc := NewCertificateService(nil, nil, nil)

	// Generate and issue initial certificate
	privateKey1, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)

	pubKeyDER1, err := x509.MarshalPKIXPublicKey(&privateKey1.PublicKey)
	require.NoError(t, err)

	pubKeyPEM1 := pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: pubKeyDER1})

	req := &CertificateRequest{
		DeviceID:     "device-004",
		DeviceSerial: "SN-004",
		PublicKeyPEM: pubKeyPEM1,
	}

	resp1, err := svc.IssueCertificate(req)
	require.NoError(t, err)

	// Generate new key for rotation
	privateKey2, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)

	pubKeyDER2, err := x509.MarshalPKIXPublicKey(&privateKey2.PublicKey)
	require.NoError(t, err)

	pubKeyPEM2 := pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: pubKeyDER2})

	// Rotate certificate
	resp2, err := svc.RotateCertificate("device-004", pubKeyPEM2)
	require.NoError(t, err)
	assert.NotEqual(t, resp1.SerialNumber, resp2.SerialNumber)
	assert.NotEqual(t, resp1.Certificate, resp2.Certificate)
}

func TestCertificateService_SignAndVerifyFirmware(t *testing.T) {
	svc := NewCertificateService(nil, nil, nil)

	firmwareData := []byte("test firmware binary data")
	firmwareVersion := "1.0.0"

	// Sign firmware
	sig, err := svc.SignFirmware(firmwareVersion, firmwareData)
	require.NoError(t, err)
	assert.NotNil(t, sig)
	assert.Equal(t, firmwareVersion, sig.FirmwareVersion)
	assert.NotEmpty(t, sig.FirmwareHash)
	assert.NotEmpty(t, sig.Signature)

	// Verify firmware
	result, err := svc.VerifyFirmwareSignature(firmwareData, sig)
	require.NoError(t, err)
	assert.True(t, result.IsValid)
	assert.Equal(t, sig.FirmwareHash, result.FirmwareHash)
}

func TestCertificateService_VerifyFirmware_TamperedData(t *testing.T) {
	svc := NewCertificateService(nil, nil, nil)

	firmwareData := []byte("test firmware binary data")
	firmwareVersion := "1.0.0"

	// Sign firmware
	sig, err := svc.SignFirmware(firmwareVersion, firmwareData)
	require.NoError(t, err)

	// Tamper with firmware data
	tamperedData := []byte("tampered firmware binary data")

	// Verify should fail
	result, err := svc.VerifyFirmwareSignature(tamperedData, sig)
	require.NoError(t, err)
	assert.False(t, result.IsValid)
	assert.Equal(t, "firmware hash mismatch", result.ErrorMessage)
}

func TestCertificateService_GetExpiringCertificates(t *testing.T) {
	svc := NewCertificateService(nil, nil, nil)

	// Issue multiple certificates
	for i := 0; i < 3; i++ {
		privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
		require.NoError(t, err)

		pubKeyDER, err := x509.MarshalPKIXPublicKey(&privateKey.PublicKey)
		require.NoError(t, err)

		pubKeyPEM := pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: pubKeyDER})

		req := &CertificateRequest{
			DeviceID:     "device-exp-" + string(rune('0'+i)),
			DeviceSerial: "SN-EXP-" + string(rune('0'+i)),
			PublicKeyPEM: pubKeyPEM,
		}

		_, err = svc.IssueCertificate(req)
		require.NoError(t, err)
	}

	// Get expiring certificates (within 400 days should include all)
	expiring := svc.GetExpiringCertificates(400)
	assert.GreaterOrEqual(t, len(expiring), 3)
}

func TestCertificateService_GetCACertificate(t *testing.T) {
	svc := NewCertificateService(nil, nil, nil)

	caCert, err := svc.GetCACertificate()
	require.NoError(t, err)
	assert.NotEmpty(t, caCert)

	// Verify it's valid PEM
	block, _ := pem.Decode(caCert)
	assert.NotNil(t, block)
	assert.Equal(t, "CERTIFICATE", block.Type)
}

// Property 28: Certificate authentication reliability
// Validates: Requirements 11, security and authentication
func TestProperty28_CertificateAuthenticationReliability(t *testing.T) {
	svc := NewCertificateService(nil, nil, nil)

	t.Run("valid certificates are always verified successfully", func(t *testing.T) {
		for i := 0; i < 10; i++ {
			privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
			require.NoError(t, err)

			pubKeyDER, err := x509.MarshalPKIXPublicKey(&privateKey.PublicKey)
			require.NoError(t, err)

			pubKeyPEM := pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: pubKeyDER})

			deviceID := "prop28-device-" + string(rune('a'+i))
			req := &CertificateRequest{
				DeviceID:     deviceID,
				DeviceSerial: "SN-" + deviceID,
				PublicKeyPEM: pubKeyPEM,
			}

			resp, err := svc.IssueCertificate(req)
			require.NoError(t, err)

			status, err := svc.VerifyCertificate(deviceID, []byte(resp.Certificate))
			require.NoError(t, err)
			assert.True(t, status.IsValid, "Valid certificate should verify successfully")
			assert.False(t, status.IsRevoked, "New certificate should not be revoked")
			assert.False(t, status.IsExpired, "New certificate should not be expired")
		}
	})

	t.Run("revoked certificates are always rejected", func(t *testing.T) {
		for i := 0; i < 5; i++ {
			privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
			require.NoError(t, err)

			pubKeyDER, err := x509.MarshalPKIXPublicKey(&privateKey.PublicKey)
			require.NoError(t, err)

			pubKeyPEM := pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: pubKeyDER})

			deviceID := "prop28-revoke-" + string(rune('a'+i))
			req := &CertificateRequest{
				DeviceID:     deviceID,
				DeviceSerial: "SN-" + deviceID,
				PublicKeyPEM: pubKeyPEM,
			}

			_, err = svc.IssueCertificate(req)
			require.NoError(t, err)

			err = svc.RevokeCertificate(deviceID, "test revocation")
			require.NoError(t, err)

			status, err := svc.GetCertificateStatus(deviceID)
			require.NoError(t, err)
			assert.True(t, status.IsRevoked, "Revoked certificate should be marked as revoked")
			assert.False(t, status.IsValid, "Revoked certificate should not be valid")
		}
	})

	t.Run("certificate rotation maintains security", func(t *testing.T) {
		privateKey1, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
		require.NoError(t, err)

		pubKeyDER1, err := x509.MarshalPKIXPublicKey(&privateKey1.PublicKey)
		require.NoError(t, err)

		pubKeyPEM1 := pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: pubKeyDER1})

		deviceID := "prop28-rotate"
		req := &CertificateRequest{
			DeviceID:     deviceID,
			DeviceSerial: "SN-" + deviceID,
			PublicKeyPEM: pubKeyPEM1,
		}

		resp1, err := svc.IssueCertificate(req)
		require.NoError(t, err)

		// Rotate with new key
		privateKey2, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
		require.NoError(t, err)

		pubKeyDER2, err := x509.MarshalPKIXPublicKey(&privateKey2.PublicKey)
		require.NoError(t, err)

		pubKeyPEM2 := pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: pubKeyDER2})

		resp2, err := svc.RotateCertificate(deviceID, pubKeyPEM2)
		require.NoError(t, err)

		// New certificate should be valid
		status, err := svc.VerifyCertificate(deviceID, []byte(resp2.Certificate))
		require.NoError(t, err)
		assert.True(t, status.IsValid, "Rotated certificate should be valid")

		// Serial numbers should be different
		assert.NotEqual(t, resp1.SerialNumber, resp2.SerialNumber, "Rotated certificate should have new serial")
	})

	t.Run("firmware signatures are cryptographically secure", func(t *testing.T) {
		firmwareVersions := []string{"1.0.0", "1.1.0", "2.0.0-beta"}
		for _, version := range firmwareVersions {
			firmwareData := []byte("firmware binary for version " + version)

			sig, err := svc.SignFirmware(version, firmwareData)
			require.NoError(t, err)

			// Valid firmware verifies
			result, err := svc.VerifyFirmwareSignature(firmwareData, sig)
			require.NoError(t, err)
			assert.True(t, result.IsValid, "Valid firmware should verify")

			// Tampered firmware fails
			tamperedData := append(firmwareData, byte(0xFF))
			result, err = svc.VerifyFirmwareSignature(tamperedData, sig)
			require.NoError(t, err)
			assert.False(t, result.IsValid, "Tampered firmware should fail verification")
		}
	})

	t.Run("unique serial numbers for all certificates", func(t *testing.T) {
		serialNumbers := make(map[string]bool)

		for i := 0; i < 20; i++ {
			privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
			require.NoError(t, err)

			pubKeyDER, err := x509.MarshalPKIXPublicKey(&privateKey.PublicKey)
			require.NoError(t, err)

			pubKeyPEM := pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: pubKeyDER})

			deviceID := "prop28-unique-" + string(rune('a'+i))
			req := &CertificateRequest{
				DeviceID:     deviceID,
				DeviceSerial: "SN-" + deviceID,
				PublicKeyPEM: pubKeyPEM,
			}

			resp, err := svc.IssueCertificate(req)
			require.NoError(t, err)

			assert.False(t, serialNumbers[resp.SerialNumber], "Serial number should be unique")
			serialNumbers[resp.SerialNumber] = true
		}
	})

	t.Run("certificate expiry is correctly calculated", func(t *testing.T) {
		privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
		require.NoError(t, err)

		pubKeyDER, err := x509.MarshalPKIXPublicKey(&privateKey.PublicKey)
		require.NoError(t, err)

		pubKeyPEM := pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: pubKeyDER})

		req := &CertificateRequest{
			DeviceID:     "prop28-expiry",
			DeviceSerial: "SN-prop28-expiry",
			PublicKeyPEM: pubKeyPEM,
		}

		resp, err := svc.IssueCertificate(req)
		require.NoError(t, err)

		// Verify expiry is approximately 365 days from now
		expectedExpiry := time.Now().AddDate(0, 0, 365)
		diff := resp.ExpiresAt.Sub(expectedExpiry)
		assert.Less(t, diff.Abs(), time.Hour, "Expiry should be approximately 365 days from now")
	})
}
