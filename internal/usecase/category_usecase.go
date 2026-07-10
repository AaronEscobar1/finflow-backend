package usecase

import (
	"context"
	"fmt"
	"strings"

	"github.com/aaron/finflow-backend/internal/domain"
)

type CategoryUseCase struct {
	repo domain.CategoryRepository
}

func NewCategoryUseCase(repo domain.CategoryRepository) *CategoryUseCase {
	return &CategoryUseCase{repo: repo}
}

func (uc *CategoryUseCase) Create(ctx context.Context, userID int64, req domain.CreateCategoryRequest) (*domain.Category, error) {
	req.Name = strings.TrimSpace(req.Name)
	if req.Name == "" {
		return nil, fmt.Errorf("%w: el nombre de la categoría es requerido", domain.ErrValidation)
	}

	req.Type = strings.TrimSpace(strings.ToLower(req.Type))
	if req.Type != "income" && req.Type != "expense" {
		return nil, fmt.Errorf("%w: el tipo de categoría debe ser 'income' o 'expense'", domain.ErrValidation)
	}

	return uc.repo.Create(ctx, userID, req)
}

func (uc *CategoryUseCase) GetByID(ctx context.Context, userID int64, id int64) (*domain.Category, error) {
	cat, err := uc.repo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if cat.UserID != userID {
		return nil, domain.ErrForbidden
	}
	return cat, nil
}

func (uc *CategoryUseCase) GetAll(ctx context.Context, userID int64, typeFilter string, includeTrashed bool) ([]domain.Category, error) {
	typeFilter = strings.TrimSpace(strings.ToLower(typeFilter))
	return uc.repo.FindAllByUser(ctx, userID, typeFilter, includeTrashed)
}

func (uc *CategoryUseCase) Update(ctx context.Context, userID int64, id int64, req domain.UpdateCategoryRequest) (*domain.Category, error) {
	cat, err := uc.GetByID(ctx, userID, id)
	if err != nil {
		return nil, err
	}

	if req.Name != nil {
		trimmed := strings.TrimSpace(*req.Name)
		if trimmed == "" {
			return nil, fmt.Errorf("%w: el nombre no puede estar vacío", domain.ErrValidation)
		}
		req.Name = &trimmed
	}

	if req.Type != nil {
		lower := strings.TrimSpace(strings.ToLower(*req.Type))
		if lower != "income" && lower != "expense" {
			return nil, fmt.Errorf("%w: el tipo de categoría debe ser 'income' o 'expense'", domain.ErrValidation)
		}
		req.Type = &lower
	}

	if cat.DeletedAt != nil {
		return nil, fmt.Errorf("%w: no se puede actualizar una categoría eliminada", domain.ErrConflict)
	}

	return uc.repo.Update(ctx, id, req)
}

func (uc *CategoryUseCase) Delete(ctx context.Context, userID int64, id int64) error {
	_, err := uc.GetByID(ctx, userID, id)
	if err != nil {
		return err
	}
	return uc.repo.SoftDelete(ctx, id)
}

func (uc *CategoryUseCase) Restore(ctx context.Context, userID int64, id int64) error {
	cat, err := uc.repo.FindByID(ctx, id)
	if err != nil {
		return err
	}
	if cat.UserID != userID {
		return domain.ErrForbidden
	}
	return uc.repo.Restore(ctx, id)
}

func (uc *CategoryUseCase) PermanentDelete(ctx context.Context, userID int64, id int64) error {
	cat, err := uc.repo.FindByID(ctx, id)
	if err != nil {
		return err
	}
	if cat.UserID != userID {
		return domain.ErrForbidden
	}
	return uc.repo.PermanentDelete(ctx, id)
}
