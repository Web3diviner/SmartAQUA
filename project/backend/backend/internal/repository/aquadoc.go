package repository

import (
	"errors"

	"smart-fish-feeder/internal/models"

	"gorm.io/gorm"
)

// AquaDocRepository handles storage of AquaDoc conversations, grounded turns, and source evidence
type AquaDocRepository struct {
	db *gorm.DB
}

// NewAquaDocRepository creates a new AquaDocRepository instance
func NewAquaDocRepository(db *gorm.DB) *AquaDocRepository {
	return &AquaDocRepository{db: db}
}

// CreateConversation initializes a new AquaDoc conversation record
func (r *AquaDocRepository) CreateConversation(conv *models.AquaDocConversationRecord) error {
	return r.db.Create(conv).Error
}

// GetConversation retrieves a conversation with its message history and citations
func (r *AquaDocRepository) GetConversation(id string) (*models.AquaDocConversationRecord, error) {
	var conv models.AquaDocConversationRecord
	if err := r.db.Preload("Messages.Evidence").Where("id = ?", id).First(&conv).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("conversation not found")
		}
		return nil, err
	}
	return &conv, nil
}

// ListConversations retrieves conversations for a user
func (r *AquaDocRepository) ListConversations(userID uint, limit int) ([]models.AquaDocConversationRecord, error) {
	var convs []models.AquaDocConversationRecord
	query := r.db.Where("user_id = ?", userID).Order("updated_at DESC")
	if limit > 0 {
		query = query.Limit(limit)
	}
	if err := query.Find(&convs).Error; err != nil {
		return nil, err
	}
	return convs, nil
}

// SaveMessageTurn logs a question/answer turn and associated evidence citations in a transaction
func (r *AquaDocRepository) SaveMessageTurn(userMsg *models.AquaDocMessageRecord, asstMsg *models.AquaDocMessageRecord, evidence []models.AquaDocEvidenceRecord) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		if userMsg != nil {
			if err := tx.Create(userMsg).Error; err != nil {
				return err
			}
		}
		if asstMsg != nil {
			if err := tx.Create(asstMsg).Error; err != nil {
				return err
			}
		}
		if len(evidence) > 0 {
			if err := tx.Create(&evidence).Error; err != nil {
				return err
			}
		}
		return nil
	})
}
