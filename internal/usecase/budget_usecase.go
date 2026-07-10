package usecase

import (
	"context"
	"fmt"
	"strings"

	"github.com/aaron/finflow-backend/internal/domain"
)

type BudgetUseCase struct {
	repo    domain.BudgetRepository
	catRepo domain.CategoryRepository
}

func NewBudgetUseCase(repo domain.BudgetRepository, catRepo domain.CategoryRepository) *BudgetUseCase {
	return &BudgetUseCase{
		repo:    repo,
		catRepo: catRepo,
	}
}

func (uc *BudgetUseCase) Create(ctx context.Context, userID int64, req domain.CreateBudgetRequest) (*domain.Budget, error) {
	if req.Amount <= 0 {
		return nil, fmt.Errorf("%w: el monto del presupuesto debe ser mayor a 0", domain.ErrValidation)
	}

	req.Period = strings.TrimSpace(strings.ToLower(req.Period))
	if req.Period != "weekly" && req.Period != "monthly" && req.Period != "yearly" {
		return nil, fmt.Errorf("%w: período inválido (debe ser 'weekly', 'monthly' o 'yearly')", domain.ErrValidation)
	}

	req.Currency = strings.TrimSpace(strings.ToUpper(req.Currency))
	if req.Currency == "" {
		req.Currency = "USD"
	}

	if req.CategoryID != nil && *req.CategoryID > 0 {
		cat, err := uc.catRepo.FindByID(ctx, *req.CategoryID)
		if err != nil {
			return nil, fmt.Errorf("%w: categoría no encontrada", domain.ErrValidation)
		}
		if cat.UserID != userID || cat.DeletedAt != nil {
			return nil, domain.ErrForbidden
		}
	} else {
		req.CategoryID = nil
	}

	return uc.repo.Create(ctx, userID, req)
}

func (uc *BudgetUseCase) GetByID(ctx context.Context, userID int64, id int64) (*domain.Budget, error) {
	b, err := uc.repo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if b.UserID != userID {
		return nil, domain.ErrForbidden
	}
	return b, nil
}

func (uc *BudgetUseCase) GetAll(ctx context.Context, userID int64, includeTrashed bool) ([]domain.BudgetWithSpent, error) {
	return uc.repo.FindAllByUser(ctx, userID, includeTrashed)
}

func (uc *BudgetUseCase) Update(ctx context.Context, userID int64, id int64, req domain.UpdateBudgetRequest) (*domain.Budget, error) {
	b, err := uc.GetByID(ctx, userID, id)
	if err != nil {
		return nil, err
	}

	if b.DeletedAt != nil {
		return nil, fmt.Errorf("%w: no se puede actualizar un presupuesto eliminado", domain.ErrConflict)
	}

	if req.Amount != nil && *req.Amount <= 0 {
		return nil, fmt.Errorf("%w: el monto debe ser mayor a 0", domain.ErrValidation)
	}

	if req.Period != nil {
		lower := strings.TrimSpace(strings.ToLower(*req.Period))
		if lower != "weekly" && lower != "monthly" && lower != "yearly" {
			return nil, fmt.Errorf("%w: período inválido", domain.ErrValidation)
		}
		req.Period = &lower
	}

	if req.Currency != nil {
		upper := strings.TrimSpace(strings.ToUpper(*req.Currency))
		if upper == "" {
			return nil, fmt.Errorf("%w: la moneda no puede estar vacía", domain.ErrValidation)
		}
		req.Currency = &upper
	}

	if req.CategoryID != nil {
		// CategoryID is 0 if we want to remove the category constraint (make it global)
		if *req.CategoryID > 0 {
			cat, err := uc.catRepo.FindByID(ctx, *req.CategoryID)
			if err != nil {
				return nil, fmt.Errorf("%w: categoría no encontrada", domain.ErrValidation)
			}
			if cat.UserID != userID || cat.DeletedAt != nil {
				return nil, domain.ErrForbidden
			}
		}
	}

	return uc.repo.Update(ctx, id, req)
}

func (uc *BudgetUseCase) Delete(ctx context.Context, userID int64, id int64) error {
	_, err := uc.GetByID(ctx, userID, id)
	if err != nil {
		return err
	}
	return uc.repo.SoftDelete(ctx, id)
}

func (uc *BudgetUseCase) Restore(ctx context.Context, userID int64, id int64) error {
	b, err := uc.repo.FindByID(ctx, id)
	if err != nil {
		return err
	}
	if b.UserID != userID {
		return domain.ErrForbidden
	}
	return uc.repo.Restore(ctx, id)
}

func (uc *BudgetUseCase) PermanentDelete(ctx context.Context, userID int64, id int64) error {
	b, err := uc.repo.FindByID(ctx, id)
	if err != nil {
		return err
	}
	if b.UserID != userID {
		return domain.ErrForbidden
	}
	return uc.repo.PermanentDelete(ctx, id)
}
