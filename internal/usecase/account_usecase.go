package usecase

import (
	"context"
	"fmt"
	"strings"

	"github.com/aaron/finflow-backend/internal/domain"
)

type AccountUseCase struct {
	repo domain.AccountRepository
}

func NewAccountUseCase(repo domain.AccountRepository) *AccountUseCase {
	return &AccountUseCase{repo: repo}
}

func (uc *AccountUseCase) Create(ctx context.Context, userID int64, req domain.CreateAccountRequest) (*domain.Account, error) {
	req.Name = strings.TrimSpace(req.Name)
	if req.Name == "" {
		return nil, fmt.Errorf("%w: el nombre de la cuenta es requerido", domain.ErrValidation)
	}

	req.Type = strings.TrimSpace(strings.ToLower(req.Type))
	if req.Type == "" {
		req.Type = "cash"
	}
	validTypes := map[string]bool{"cash": true, "bank": true, "card": true, "investment": true, "other": true}
	if !validTypes[req.Type] {
		return nil, fmt.Errorf("%w: tipo de cuenta inválido", domain.ErrValidation)
	}

	req.Currency = strings.TrimSpace(strings.ToUpper(req.Currency))
	if req.Currency == "" {
		req.Currency = "USD"
	}

	return uc.repo.Create(ctx, userID, req)
}

func (uc *AccountUseCase) GetByID(ctx context.Context, userID int64, id int64) (*domain.Account, error) {
	acc, err := uc.repo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if acc.UserID != userID {
		return nil, domain.ErrForbidden
	}
	return acc, nil
}

func (uc *AccountUseCase) GetAll(ctx context.Context, userID int64, includeTrashed bool) ([]domain.Account, error) {
	return uc.repo.FindAllByUser(ctx, userID, includeTrashed)
}

func (uc *AccountUseCase) Update(ctx context.Context, userID int64, id int64, req domain.UpdateAccountRequest) (*domain.Account, error) {
	acc, err := uc.GetByID(ctx, userID, id)
	if err != nil {
		return nil, err
	}

	if req.Name != nil {
		trimmed := strings.TrimSpace(*req.Name)
		if trimmed == "" {
			return nil, fmt.Errorf("%w: el nombre de la cuenta no puede estar vacío", domain.ErrValidation)
		}
		req.Name = &trimmed
	}

	if req.Type != nil {
		lower := strings.TrimSpace(strings.ToLower(*req.Type))
		validTypes := map[string]bool{"cash": true, "bank": true, "card": true, "investment": true, "other": true}
		if !validTypes[lower] {
			return nil, fmt.Errorf("%w: tipo de cuenta inválido", domain.ErrValidation)
		}
		req.Type = &lower
	}

	if req.Currency != nil {
		upper := strings.TrimSpace(strings.ToUpper(*req.Currency))
		if upper == "" {
			return nil, fmt.Errorf("%w: la moneda no puede estar vacía", domain.ErrValidation)
		}
		req.Currency = &upper
	}

	// Soft-deleted accounts cannot be updated
	if acc.DeletedAt != nil {
		return nil, fmt.Errorf("%w: no se puede actualizar una cuenta eliminada", domain.ErrConflict)
	}

	return uc.repo.Update(ctx, id, req)
}

func (uc *AccountUseCase) Delete(ctx context.Context, userID int64, id int64) error {
	_, err := uc.GetByID(ctx, userID, id)
	if err != nil {
		return err
	}
	return uc.repo.SoftDelete(ctx, id)
}

func (uc *AccountUseCase) Restore(ctx context.Context, userID int64, id int64) error {
	acc, err := uc.repo.FindByID(ctx, id)
	if err != nil {
		return err
	}
	if acc.UserID != userID {
		return domain.ErrForbidden
	}
	return uc.repo.Restore(ctx, id)
}

func (uc *AccountUseCase) PermanentDelete(ctx context.Context, userID int64, id int64) error {
	acc, err := uc.repo.FindByID(ctx, id)
	if err != nil {
		return err
	}
	if acc.UserID != userID {
		return domain.ErrForbidden
	}
	return uc.repo.PermanentDelete(ctx, id)
}
