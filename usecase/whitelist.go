package usecase

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/AndreeJait/go-utility/v2/logw"
	"github.com/AndreeJait/zora-core/config"
	domainError "github.com/AndreeJait/zora-core/domain/error"
	"github.com/AndreeJait/zora-core/domain/entity"
	"github.com/AndreeJait/zora-core/port/inbound/whitelist"
	"github.com/AndreeJait/zora-core/port/outbound"
)

type whitelistUseCase struct {
	repo          outbound.WhitelistRepository
	adminRepo     outbound.AdminRepository
	wahaClient    outbound.WahaClient
	defaultTokens int
}

var _ whitelist.UseCase = (*whitelistUseCase)(nil)

func NewWhitelistUseCase(repo outbound.WhitelistRepository, adminRepo outbound.AdminRepository, wahaClient outbound.WahaClient, cfg *config.AppConfig) whitelist.UseCase {
	return &whitelistUseCase{
		repo:          repo,
		adminRepo:     adminRepo,
		wahaClient:    wahaClient,
		defaultTokens: cfg.Whitelist.DefaultTokensPerHour,
	}
}

// resolvePhone attempts to determine the sender's phone number.
// Priority: (1) explicit SenderPhone, (2) WAHA LID resolution, (3) empty string.
func (uc *whitelistUseCase) resolvePhone(ctx context.Context, sender whitelist.SenderContext) string {
	if sender.SenderPhone != "" {
		return normalizePhone(sender.SenderPhone)
	}
	if sender.SenderLID != "" {
		phone, err := uc.wahaClient.ResolveLID(ctx, sender.SenderLID)
		if err != nil {
			logw.CtxErrorf(ctx, "whitelist: LID resolution failed for %s: %v", sender.SenderLID, err)
		}
		if phone != "" {
			return normalizePhone(phone)
		}
	}
	return ""
}

// scopeMatches checks if an entry's scope is compatible with the current chat context.
func scopeMatches(scope string, chatIDs string, sender whitelist.SenderContext) bool {
	switch scope {
	case "personal":
		return !sender.IsGroup
	case "group":
		if !sender.IsGroup {
			return false
		}
		// If specific chat_ids are set, only allow in those groups
		var ids []string
		if chatIDs != "" && chatIDs != "[]" {
			if err := json.Unmarshal([]byte(chatIDs), &ids); err != nil {
				return true // malformed chat_ids, allow
			}
			if len(ids) == 0 {
				return true
			}
			for _, id := range ids {
				if id == sender.ChatID {
					return true
				}
			}
			return false
		}
		return true
	case "both", "":
		var ids []string
		if chatIDs != "" && chatIDs != "[]" {
			if err := json.Unmarshal([]byte(chatIDs), &ids); err != nil {
				return true // malformed chat_ids, allow
			}
			if len(ids) == 0 {
				return true
			}
		} else {
			return true
		}
		for _, id := range ids {
			if id == sender.ChatID {
				return true
			}
		}
		return false
	default:
		return true
	}
}

func (uc *whitelistUseCase) IsAdmin(sender whitelist.SenderContext) bool {
	ctx := context.Background()
	phone := uc.resolvePhone(ctx, sender)

	if phone != "" {
		entry, err := uc.adminRepo.GetByPhone(ctx, phone)
		if err != nil {
			logw.Errorf("whitelist: IsAdmin DB check failed for %s: %v", phone, err)
		}
		if entry != nil && scopeMatches(entry.Scope, entry.ChatIDs, sender) {
			return true
		}
	}

	// Fallback: try LID lookup
	if sender.SenderLID != "" {
		entry, err := uc.adminRepo.GetByLID(ctx, sender.SenderLID)
		if err != nil {
			logw.Errorf("whitelist: IsAdmin LID check failed for %s: %v", sender.SenderLID, err)
		}
		if entry != nil && scopeMatches(entry.Scope, entry.ChatIDs, sender) {
			return true
		}
	}

	return false
}

func (uc *whitelistUseCase) CheckAccess(ctx context.Context, sender whitelist.SenderContext) error {
	// Admins always have access
	if uc.IsAdmin(sender) {
		return nil
	}

	phone := uc.resolvePhone(ctx, sender)

	// Try phone lookup
	if phone != "" {
		entry, err := uc.repo.GetByPhone(ctx, phone)
		if err != nil {
			logw.CtxErrorf(ctx, "whitelist: check access failed for %s: %v", phone, err)
			return domainError.ErrNotWhitelisted
		}
		if entry != nil && scopeMatches(entry.Scope, entry.ChatIDs, sender) {
			if entry.IsUnlimited() {
				return nil
			}
			windowStart := time.Now().Truncate(time.Hour)
			used, err := uc.repo.GetUsage(ctx, phone, windowStart)
			if err != nil {
				logw.CtxErrorf(ctx, "whitelist: get token usage failed for %s: %v", phone, err)
				return domainError.ErrNotWhitelisted
			}
			if used >= entry.TokensPerHour {
				return domainError.ErrTokenExhausted
			}
			return nil
		}
	}

	// Fallback: try LID lookup
	if sender.SenderLID != "" {
		entry, err := uc.repo.GetByLID(ctx, sender.SenderLID)
		if err != nil {
			logw.CtxErrorf(ctx, "whitelist: LID check access failed for %s: %v", sender.SenderLID, err)
			return domainError.ErrNotWhitelisted
		}
		if entry != nil && scopeMatches(entry.Scope, entry.ChatIDs, sender) {
			if entry.IsUnlimited() {
				return nil
			}
			// Use the entry's phone for token tracking
			windowStart := time.Now().Truncate(time.Hour)
			used, err := uc.repo.GetUsage(ctx, entry.Phone, windowStart)
			if err != nil {
				logw.CtxErrorf(ctx, "whitelist: get token usage failed for %s: %v", entry.Phone, err)
				return domainError.ErrNotWhitelisted
			}
			if used >= entry.TokensPerHour {
				return domainError.ErrTokenExhausted
			}
			return nil
		}
	}

	return domainError.ErrNotWhitelisted
}

func (uc *whitelistUseCase) ConsumeToken(ctx context.Context, sender whitelist.SenderContext) error {
	// Admins don't consume tokens
	if uc.IsAdmin(sender) {
		return nil
	}

	phone := uc.resolvePhone(ctx, sender)

	// Find entry by phone or LID
	var entry *entity.WhitelistEntry
	var err error

	if phone != "" {
		entry, err = uc.repo.GetByPhone(ctx, phone)
		if err != nil || entry == nil {
			return nil // already checked in CheckAccess
		}
	} else if sender.SenderLID != "" {
		entry, err = uc.repo.GetByLID(ctx, sender.SenderLID)
		if err != nil || entry == nil {
			return nil
		}
		phone = entry.Phone // use entry's phone for token tracking
	} else {
		return nil
	}

	// Unlimited = no tracking needed
	if entry.IsUnlimited() {
		return nil
	}

	windowStart := time.Now().Truncate(time.Hour)

	// Cleanup old windows in background
	go func() {
		cutoff := time.Now().Add(-2 * time.Hour)
		_ = uc.repo.CleanupExpiredTokens(context.Background(), cutoff)
	}()

	total, err := uc.repo.IncrementUsage(ctx, phone, windowStart)
	if err != nil {
		return fmt.Errorf("consume token: %w", err)
	}
	if total > entry.TokensPerHour {
		return domainError.ErrTokenExhausted
	}

	return nil
}

func (uc *whitelistUseCase) HandleCommand(ctx context.Context, sender whitelist.SenderContext, args string, mentionedIDs []string) (string, error) {
	if !uc.IsAdmin(sender) {
		return "", domainError.ErrNotAdmin
	}

	args = strings.TrimSpace(args)

	// No args: list whitelist
	if args == "" {
		return uc.handleList(ctx)
	}

	// Remove command
	if strings.HasPrefix(args, "remove") {
		return uc.handleRemove(ctx, strings.TrimSpace(strings.TrimPrefix(args, "remove")), mentionedIDs)
	}

	// Add command (default when no subcommand prefix)
	return uc.handleAdd(ctx, args, mentionedIDs, sender)
}

func (uc *whitelistUseCase) handleList(ctx context.Context) (string, error) {
	entries, err := uc.repo.List(ctx)
	if err != nil {
		return "", fmt.Errorf("list whitelist: %w", err)
	}
	if len(entries) == 0 {
		return "No users in whitelist.", nil
	}

	var sb strings.Builder
	sb.WriteString("Whitelist:\n")
	for _, e := range entries {
		tokens := "unlimited"
		if !e.IsUnlimited() {
			tokens = fmt.Sprintf("%d tokens/hour", e.TokensPerHour)
		}
		fmt.Fprintf(&sb, "- %s (%s): %s [%s]\n", e.Name, e.Phone, tokens, e.Scope)
	}
	return sb.String(), nil
}

func (uc *whitelistUseCase) handleAdd(ctx context.Context, args string, mentionedIDs []string, sender whitelist.SenderContext) (string, error) {
	phone := ""
	name := ""
	tokens := uc.defaultTokens // use default from config
	scope := "" // will default based on sender context

	// Extract phone from @mention or raw number

	// Extract phone from @mention or raw number
	parts := strings.Fields(args)
	for _, part := range parts {
		if strings.HasPrefix(part, "@") {
			phone = resolveMention(part, mentionedIDs)
			continue
		}
		if strings.HasPrefix(part, "name=") {
			name = strings.TrimPrefix(part, "name=")
			continue
		}
		if strings.HasPrefix(part, "tokens=") {
			val := strings.TrimPrefix(part, "tokens=")
			if strings.EqualFold(val, "unlimited") {
				tokens = 0
			} else {
				fmt.Sscanf(val, "%d", &tokens)
			}
			continue
		}
		if strings.HasPrefix(part, "scope=") {
			scope = strings.TrimPrefix(part, "scope=")
			continue
		}
		// Raw phone number
		if phone == "" && isPhoneNumber(part) {
			phone = normalizePhone(part)
		}
	}

	if phone == "" {
		return "Usage: !zora whitelist <phone> [name=Name] [tokens=N|unlimited] [scope=personal|group|both]", nil
	}
	if name == "" {
		name = phone
	}
	// Validate scope and determine chatIDs
	chatIDsJSON := "[]"
	switch scope {
	case "":
		if !sender.IsGroup {
			return "Usage: !zora whitelist <phone> scope=group|personal|both\nscope is required when adding from personal chat", nil
		}
		scope = "group"
		fallthrough
	case "group":
		if !sender.IsGroup || sender.ChatID == "" {
			return "Cannot add group-scoped whitelist from personal chat. Use scope=personal or scope=both.", nil
		}
		ids, _ := json.Marshal([]string{sender.ChatID})
		chatIDsJSON = string(ids)
	case "personal", "both":
		// No chat ID restriction for personal/both scopes
	default:
		return fmt.Sprintf("Invalid scope: %s. Use: personal, group, or both", scope), nil
	}

	entry := &entity.WhitelistEntry{
		Phone:         phone,
		Name:          name,
		TokensPerHour: tokens,
		Scope:         scope,
		ChatIDs:       chatIDsJSON,
	}

	if err := uc.repo.Add(ctx, entry); err != nil {
		return "", fmt.Errorf("add whitelist entry: %w", err)
	}

	tokenLabel := "unlimited"
	if tokens > 0 {
		tokenLabel = fmt.Sprintf("%d tokens/hour", tokens)
	}
	return fmt.Sprintf("Added %s (%s) to whitelist: %s [%s]", name, phone, tokenLabel, scope), nil
}

func (uc *whitelistUseCase) handleRemove(ctx context.Context, args string, mentionedIDs []string) (string, error) {
	phone := ""

	parts := strings.Fields(args)
	for _, part := range parts {
		if strings.HasPrefix(part, "@") {
			phone = resolveMention(part, mentionedIDs)
			break
		}
		if isPhoneNumber(part) {
			phone = normalizePhone(part)
			break
		}
	}

	if phone == "" {
		return "Usage: !zora whitelist remove @mention", nil
	}

	if err := uc.repo.Remove(ctx, phone); err != nil {
		return "", fmt.Errorf("remove whitelist entry: %w", err)
	}

	return fmt.Sprintf("Removed %s from whitelist", phone), nil
}

// ListWhitelist returns all whitelist entries.
func (uc *whitelistUseCase) ListWhitelist(ctx context.Context) ([]entity.WhitelistEntry, error) {
	return uc.repo.List(ctx)
}

// AddWhitelist creates or updates a whitelist entry.
func (uc *whitelistUseCase) AddWhitelist(ctx context.Context, phone, name string, tokensPerHour int, scope string, chatIDs []string) error {
	chatIDsJSON := "[]"
	if len(chatIDs) > 0 {
		b, err := json.Marshal(chatIDs)
		if err != nil {
			return fmt.Errorf("marshal chat_ids: %w", err)
		}
		chatIDsJSON = string(b)
	}
	if scope == "" {
		scope = "both"
	}
	entry := &entity.WhitelistEntry{
		Phone:         normalizePhone(phone),
		Name:          name,
		TokensPerHour: tokensPerHour,
		Scope:         scope,
		ChatIDs:       chatIDsJSON,
	}
	return uc.repo.Add(ctx, entry)
}

// RemoveWhitelist deletes a whitelist entry by phone.
func (uc *whitelistUseCase) RemoveWhitelist(ctx context.Context, phone string) error {
	return uc.repo.Remove(ctx, normalizePhone(phone))
}

// ListAdmins returns all admin entries from the database.
func (uc *whitelistUseCase) ListAdmins(ctx context.Context) ([]entity.AdminEntry, error) {
	return uc.adminRepo.List(ctx)
}

// AddAdmin creates or updates an admin entry in the database.
func (uc *whitelistUseCase) AddAdmin(ctx context.Context, phone, name string, scope string, chatIDs []string) error {
	chatIDsJSON := "[]"
	if len(chatIDs) > 0 {
		b, err := json.Marshal(chatIDs)
		if err != nil {
			return fmt.Errorf("marshal chat_ids: %w", err)
		}
		chatIDsJSON = string(b)
	}
	if scope == "" {
		scope = "both"
	}
	entry := &entity.AdminEntry{
		Phone: normalizePhone(phone),
		Name:  name,
		Scope: scope,
		ChatIDs: chatIDsJSON,
	}
	return uc.adminRepo.Add(ctx, entry)
}

// RemoveAdmin deletes an admin entry from the database by phone.
func (uc *whitelistUseCase) RemoveAdmin(ctx context.Context, phone string) error {
	return uc.adminRepo.Remove(ctx, normalizePhone(phone))
}

// normalizePhone strips WhatsApp suffixes and converts local Indonesian format.
// It handles: @c.us, @g.us, @lid suffixes, + prefix, and 08xx → 628xx.
func normalizePhone(phone string) string {
	phone = strings.TrimSuffix(phone, "@c.us")
	phone = strings.TrimSuffix(phone, "@g.us")
	phone = strings.TrimSuffix(phone, "@lid")
	phone = strings.TrimPrefix(phone, "+")
	if strings.HasPrefix(phone, "08") {
		phone = "628" + phone[2:]
	}
	return phone
}

// resolveMention converts @N to the Nth mentioned ID, stripping suffixes.
func resolveMention(ref string, mentionedIDs []string) string {
	phone := strings.TrimPrefix(ref, "@")
	if isNumeric(phone) && len(mentionedIDs) > 0 {
		idx := 0
		fmt.Sscanf(phone, "%d", &idx)
		if idx < len(mentionedIDs) {
			return normalizePhone(mentionedIDs[idx])
		}
	}
	return normalizePhone(phone)
}

// isPhoneNumber checks if a string looks like a phone number.
func isPhoneNumber(s string) bool {
	for _, c := range s {
		if c < '0' || c > '9' {
			if c != '+' {
				return false
			}
		}
	}
	return len(s) >= 6
}

// isNumeric checks if a string is all digits.
func isNumeric(s string) bool {
	for _, c := range s {
		if c < '0' || c > '9' {
			return false
		}
	}
	return len(s) > 0
}