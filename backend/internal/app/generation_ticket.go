package app

import (
	"errors"
	"strings"
	"time"

	"infinite-canvas/backend/internal/kernel"
	"infinite-canvas/backend/internal/model"
	"infinite-canvas/backend/internal/repository"
)

func (s *Service) ReportTaskProviderRequest(userID, taskID, providerRequestID string) (*model.Task, error) {
	providerRequestID = strings.TrimSpace(providerRequestID)
	if providerRequestID == "" {
		return nil, BadAuthRequest("缺少厂商任务 ID")
	}
	task, err := s.repo.ConsumeTaskTicket(userID, strings.TrimSpace(taskID), providerRequestID, time.Now())
	if errors.Is(err, repository.ErrTicketConsumed) {
		return nil, kernel.FailedPrecondition("同一票根不能提交两次生成")
	}
	if errors.Is(err, repository.ErrTicketExpired) {
		return nil, BadAuthRequest("任务票已过期，请重新提交")
	}
	if errors.Is(err, repository.ErrTicketNotFound) {
		return nil, kernel.NotFound("任务不存在")
	}
	if err != nil {
		return nil, err
	}
	_ = s.log(userID, task.ID, "info", "客户端已提交厂商任务", providerRequestID)
	return taskForOutput(*task), nil
}

func generationTicketExpired(task *model.Task, now time.Time) bool {
	if task == nil || task.TicketExpiresAt == nil {
		return false
	}
	if task.TicketConsumedAt != nil || strings.TrimSpace(task.ProviderRequestID) != "" {
		return false
	}
	return !now.Before(*task.TicketExpiresAt)
}
