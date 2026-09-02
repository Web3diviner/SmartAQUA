package handlers

import (
	"net/http"
	"strconv"

	"smart-fish-feeder/internal/models"
	"smart-fish-feeder/internal/services"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// AquaDocHandler handles client requests for AI clinical advice, grounded hybrid RAG, and conversation history
type AquaDocHandler struct {
	aquaDocService *services.AquaDocService
	logger         *logrus.Logger
}

// NewAquaDocHandler creates a new AquaDocHandler instance
func NewAquaDocHandler(aquaDocService *services.AquaDocService, logger *logrus.Logger) *AquaDocHandler {
	return &AquaDocHandler{
		aquaDocService: aquaDocService,
		logger:         logger,
	}
}

func (h *AquaDocHandler) getUserID(c *gin.Context) (uint, bool) {
	val, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return 0, false
	}
	userID, ok := val.(uint)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid user context"})
		return 0, false
	}
	return userID, true
}

// Chat executes a grounded conversation turn with AquaDoc
func (h *AquaDocHandler) Chat(c *gin.Context) {
	userID, ok := h.getUserID(c)
	if !ok {
		return
	}

	var req models.AquaDocChatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.Question == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Question is required"})
		return
	}

	resp, err := h.aquaDocService.Chat(c.Request.Context(), userID, &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, resp)
}

// ListConversations retrieves past chat sessions for the authenticated user
func (h *AquaDocHandler) ListConversations(c *gin.Context) {
	userID, ok := h.getUserID(c)
	if !ok {
		return
	}

	limit := 20
	if limitParam := c.Query("limit"); limitParam != "" {
		if l, err := strconv.Atoi(limitParam); err == nil && l > 0 {
			limit = l
		}
	}

	convs, err := h.aquaDocService.ListConversations(userID, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, convs)
}

// GetConversation returns a single conversation with full message turn history and citations
func (h *AquaDocHandler) GetConversation(c *gin.Context) {
	userID, ok := h.getUserID(c)
	if !ok {
		return
	}

	convID := c.Param("id")
	if convID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Conversation ID is required"})
		return
	}

	conv, err := h.aquaDocService.GetConversationDetails(userID, convID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, conv)
}
