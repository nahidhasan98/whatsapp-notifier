package validation

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/nahidhasan98/whatsapp-notifier/internal/errors"
	"github.com/nahidhasan98/whatsapp-notifier/internal/models"
)

// WhatsApp JID patterns
var (
	// Individual JID pattern: number@s.whatsapp.net
	individualJIDPattern = regexp.MustCompile(`^\d{10,15}@s\.whatsapp\.net$`)

	// Group JID pattern: groupid@g.us
	groupJIDPattern = regexp.MustCompile(`^\d+@g\.us$`)

	// Business JID pattern: number@c.us
	businessJIDPattern = regexp.MustCompile(`^\d{10,15}@c\.us$`)
)

// Validator provides validation methods
type Validator struct{}

// New creates a new validator instance
func New() *Validator {
	return &Validator{}
}

// ValidateSendMessageRequest validates a send message request
func (v *Validator) ValidateSendMessageRequest(req *models.SendMessageRequest) *errors.AppError {
	if req == nil {
		return errors.InvalidRequest("Request body is required")
	}

	// Validate 'to' field
	if strings.TrimSpace(req.To) == "" {
		return errors.ValidationError("'to' field is required")
	}

	if !v.IsValidJID(req.To) {
		return errors.InvalidJID(req.To)
	}

	// Validate 'message' field
	if strings.TrimSpace(req.Message) == "" {
		return errors.ValidationError("'message' field is required")
	}

	if len(req.Message) > 4096 {
		return errors.ValidationError("Message too long (maximum 4096 characters)")
	}

	return nil
}

// IsValidJID checks if a JID is valid WhatsApp format
func (v *Validator) IsValidJID(jid string) bool {
	jid = strings.TrimSpace(jid)

	// Check individual user JID
	if individualJIDPattern.MatchString(jid) {
		return true
	}

	// Check group JID
	if groupJIDPattern.MatchString(jid) {
		return true
	}

	// Check business JID
	if businessJIDPattern.MatchString(jid) {
		return true
	}

	return false
}

// NormalizeJID normalizes a JID to proper WhatsApp format
func (v *Validator) NormalizeJID(jid string) (string, *errors.AppError) {
	jid = strings.TrimSpace(jid)

	// If already in proper format, return as is
	if v.IsValidJID(jid) {
		return jid, nil
	}

	// Try to normalize phone number to individual JID
	if phoneNumber := v.extractPhoneNumber(jid); phoneNumber != "" {
		normalizedJID := phoneNumber + "@s.whatsapp.net"
		if v.IsValidJID(normalizedJID) {
			return normalizedJID, nil
		}
	}

	return "", errors.InvalidJID(jid)
}

// extractPhoneNumber extracts a phone number from various formats
func (v *Validator) extractPhoneNumber(input string) string {
	// Remove all non-digit characters
	phoneRegex := regexp.MustCompile(`\D`)
	phone := phoneRegex.ReplaceAllString(input, "")

	// Check if it's a valid phone number length (10-15 digits)
	if len(phone) >= 10 && len(phone) <= 15 {
		return phone
	}

	return ""
}

// SanitizeMessage sanitizes a message by removing potential harmful content
func (v *Validator) SanitizeMessage(message string) string {
	// Trim whitespace
	message = strings.TrimSpace(message)

	// Remove null bytes
	message = strings.ReplaceAll(message, "\x00", "")

	// Limit consecutive newlines
	newlineRegex := regexp.MustCompile(`\n{3,}`)
	message = newlineRegex.ReplaceAllString(message, "\n\n")

	return message
}

// ValidateQueryParams validates common query parameters
func (v *Validator) ValidateQueryParams(params map[string]string) *errors.AppError {
	for key, value := range params {
		switch key {
		case "limit":
			if err := v.validateLimit(value); err != nil {
				return err
			}
		case "offset":
			if err := v.validateOffset(value); err != nil {
				return err
			}
		}
	}
	return nil
}

// validateLimit validates the limit parameter
func (v *Validator) validateLimit(limit string) *errors.AppError {
	if limit == "" {
		return nil
	}

	// Check if it's a number
	var limitInt int
	if _, err := fmt.Sscanf(limit, "%d", &limitInt); err != nil {
		return errors.ValidationError("Invalid limit parameter: must be a number")
	}

	// Check range
	if limitInt < 1 || limitInt > 1000 {
		return errors.ValidationError("Invalid limit parameter: must be between 1 and 1000")
	}

	return nil
}

// validateOffset validates the offset parameter
func (v *Validator) validateOffset(offset string) *errors.AppError {
	if offset == "" {
		return nil
	}

	// Check if it's a number
	var offsetInt int
	if _, err := fmt.Sscanf(offset, "%d", &offsetInt); err != nil {
		return errors.ValidationError("Invalid offset parameter: must be a number")
	}

	// Check range
	if offsetInt < 0 {
		return errors.ValidationError("Invalid offset parameter: must be non-negative")
	}

	return nil
}
