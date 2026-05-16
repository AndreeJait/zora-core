package usecase

import (
	"context"

	"github.com/AndreeJait/go-utility/v2/logw"
	"github.com/AndreeJait/zora-core/config"
	"github.com/AndreeJait/zora-core/domain/entity"
	"github.com/AndreeJait/zora-core/port/outbound"
)

// SeedDefaults inserts default settings into the database if they don't already exist.
func SeedDefaults(ctx context.Context, repo outbound.SettingRepository) {
	defaults := map[string]struct{ value, desc string }{
		"task.max_retry":              {"3", "Maximum number of retries for a failed task"},
		"task.retry_delay_seconds":    {"30", "Base delay in seconds between retry attempts (exponential backoff)"},
		"task.worker_count":           {"5", "Number of background worker goroutines"},
		"task.worker_timeout_seconds": {"300", "Timeout in seconds for task execution"},
		"notification.admin_chat_id":  {"", "WAHA chat ID for admin failure notifications"},
	}

	for key, val := range defaults {
		existing, err := repo.Get(ctx, key)
		if err != nil {
			logw.Errorf("seed: failed to check setting %s: %v", key, err)
			continue
		}
		if existing != nil {
			continue
		}
		desc := val.desc
		if err := repo.Upsert(ctx, &entity.Setting{Key: key, Value: val.value, Description: &desc}); err != nil {
			logw.Errorf("seed: failed to insert setting %s: %v", key, err)
		}
	}
}

// SeedAdmins migrates config-based admin phone numbers into the admin_entries DB table.
// Existing DB entries are not overwritten.
func SeedAdmins(ctx context.Context, adminRepo outbound.AdminRepository, cfg *config.AppConfig) {
	for _, phone := range cfg.Whitelist.Admins {
		normalized := normalizePhone(phone)
		existing, err := adminRepo.GetByPhone(ctx, normalized)
		if err != nil {
			logw.Errorf("seed: failed to check admin %s: %v", normalized, err)
			continue
		}
		if existing != nil {
			continue
		}
		entry := &entity.AdminEntry{
			Phone: normalized,
			Name:  normalized,
		}
		if err := adminRepo.Add(ctx, entry); err != nil {
			logw.Errorf("seed: failed to insert admin %s: %v", normalized, err)
		} else {
			logw.Infof("seed: migrated config admin %s to database", normalized)
		}
	}
}