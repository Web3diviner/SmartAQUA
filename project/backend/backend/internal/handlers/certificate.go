package handlers

import (
	"fmt"
	"net/http"

	"smart-fish-feeder/internal/services"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// CertificateHandler handles certificate management endpoints
type CertificateHandler struct {
	services *services.Services
	logger   *logrus.Logger
}

// NewCertificateHandler creates a new certificate handler
func NewCertificateHandler(services *services.Services, logger *logrus.Logger) *CertificateHandler {
	return &CertificateHandler{
		services: services,
		logger:   logger,
	}
}

// IssueCertificate issues a new certificate for a device
// POST /api/v1/certificates/issue
func (h *CertificateHandler) IssueCertificate(c *gin.Context) {
	var req services.CertificateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	response, err := h.services.Certificate.IssueCertificate(&req)
	if err != nil {
		h.logger.WithError(err).Error("Failed to issue certificate")
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, response)
}

// VerifyCertificate verifies a device certificate
// POST /api/v1/certificates/verify
func (h *CertificateHandler) VerifyCertificate(c *gin.Context) {
	var req struct {
		DeviceID    string `json:"device_id" binding:"required"`
		Certificate string `json:"certificate" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	status, err := h.services.Certificate.VerifyCertificate(req.DeviceID, []byte(req.Certificate))
	if err != nil {
		h.logger.WithError(err).Error("Failed to verify certificate")
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, status)
}

// RevokeCertificate revokes a device certificate
// POST /api/v1/certificates/revoke
func (h *CertificateHandler) RevokeCertificate(c *gin.Context) {
	var req struct {
		DeviceID string `json:"device_id" binding:"required"`
		Reason   string `json:"reason" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.services.Certificate.RevokeCertificate(req.DeviceID, req.Reason); err != nil {
		h.logger.WithError(err).Error("Failed to revoke certificate")
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "certificate revoked successfully"})
}

// RotateCertificate rotates a device certificate
// POST /api/v1/certificates/rotate
func (h *CertificateHandler) RotateCertificate(c *gin.Context) {
	var req struct {
		DeviceID     string `json:"device_id" binding:"required"`
		PublicKeyPEM string `json:"public_key_pem" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	response, err := h.services.Certificate.RotateCertificate(req.DeviceID, []byte(req.PublicKeyPEM))
	if err != nil {
		h.logger.WithError(err).Error("Failed to rotate certificate")
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, response)
}

// GetCertificateStatus gets the status of a device's certificate
// GET /api/v1/certificates/:device_id/status
func (h *CertificateHandler) GetCertificateStatus(c *gin.Context) {
	deviceID := c.Param("device_id")
	if deviceID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "device_id is required"})
		return
	}

	status, err := h.services.Certificate.GetCertificateStatus(deviceID)
	if err != nil {
		h.logger.WithError(err).Error("Failed to get certificate status")
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, status)
}

// GetCACertificate returns the CA certificate
// GET /api/v1/certificates/ca
func (h *CertificateHandler) GetCACertificate(c *gin.Context) {
	caCert, err := h.services.Certificate.GetCACertificate()
	if err != nil {
		h.logger.WithError(err).Error("Failed to get CA certificate")
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"ca_certificate": string(caCert)})
}

// ListCertificates lists all device certificates
// GET /api/v1/certificates
func (h *CertificateHandler) ListCertificates(c *gin.Context) {
	certs := h.services.Certificate.ListDeviceCertificates()
	c.JSON(http.StatusOK, gin.H{"certificates": certs})
}

// GetExpiringCertificates returns certificates expiring soon
// GET /api/v1/certificates/expiring
func (h *CertificateHandler) GetExpiringCertificates(c *gin.Context) {
	days := 30 // Default to 30 days
	if d := c.Query("days"); d != "" {
		if _, err := fmt.Sscanf(d, "%d", &days); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid days parameter"})
			return
		}
	}

	certs := h.services.Certificate.GetExpiringCertificates(days)
	c.JSON(http.StatusOK, gin.H{"expiring_certificates": certs, "within_days": days})
}

// SignFirmware signs firmware for secure OTA updates
// POST /api/v1/certificates/firmware/sign
func (h *CertificateHandler) SignFirmware(c *gin.Context) {
	var req struct {
		FirmwareVersion string `json:"firmware_version" binding:"required"`
		FirmwareData    []byte `json:"firmware_data" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	signature, err := h.services.Certificate.SignFirmware(req.FirmwareVersion, req.FirmwareData)
	if err != nil {
		h.logger.WithError(err).Error("Failed to sign firmware")
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, signature)
}

// VerifyFirmware verifies firmware signature
// POST /api/v1/certificates/firmware/verify
func (h *CertificateHandler) VerifyFirmware(c *gin.Context) {
	var req struct {
		FirmwareData []byte                      `json:"firmware_data" binding:"required"`
		Signature    *services.FirmwareSignature `json:"signature" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	result, err := h.services.Certificate.VerifyFirmwareSignature(req.FirmwareData, req.Signature)
	if err != nil {
		h.logger.WithError(err).Error("Failed to verify firmware")
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, result)
}
