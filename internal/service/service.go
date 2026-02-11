package service

import (
	"context"

	"github.com/alonsoF100/reporting-service/internal/models"
	"github.com/alonsoF100/reporting-service/internal/repository/postgres"
)

type DeviceService struct {
	repo *postgres.Repository
}

func NewDeviceService(repo *postgres.Repository) *DeviceService {
	return &DeviceService{
		repo: repo,
	}
}

func (s *DeviceService) GetDeviceMessages(ctx context.Context, unitGUID string, page, limit int) ([]models.DeviceMessage, int, error) {
	return s.repo.GetMessagesByUnitGUIDWithPagination(ctx, unitGUID, page, limit)
}
