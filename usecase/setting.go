package usecase

import (
	"context"
	"fmt"

	"github.com/AndreeJait/zora-core/domain/entity"
	"github.com/AndreeJait/zora-core/port/inbound/setting"
	"github.com/AndreeJait/zora-core/port/outbound"
)

type settingUseCase struct {
	repo outbound.SettingRepository
}

var _ setting.UseCase = (*settingUseCase)(nil)

func NewSettingUseCase(repo outbound.SettingRepository) setting.UseCase {
	return &settingUseCase{repo: repo}
}

func (uc *settingUseCase) Get(ctx context.Context, key string) (string, error) {
	s, err := uc.repo.Get(ctx, key)
	if err != nil {
		return "", fmt.Errorf("get setting: %w", err)
	}
	if s == nil {
		return "", nil
	}
	return s.Value, nil
}

func (uc *settingUseCase) GetAll(ctx context.Context) (map[string]string, error) {
	settings, err := uc.repo.GetAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("get all settings: %w", err)
	}
	result := make(map[string]string, len(settings))
	for _, s := range settings {
		result[s.Key] = s.Value
	}
	return result, nil
}

func (uc *settingUseCase) Set(ctx context.Context, key, value, description string) error {
	s := &entity.Setting{
		Key:   key,
		Value: value,
	}
	if description != "" {
		s.Description = &description
	}
	if err := uc.repo.Upsert(ctx, s); err != nil {
		return fmt.Errorf("set setting: %w", err)
	}
	return nil
}