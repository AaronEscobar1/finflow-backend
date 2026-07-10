package usecase

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/aaron/finflow-backend/internal/domain"
)

type TransactionUseCase struct {
	repo     domain.TransactionRepository
	accRepo  domain.AccountRepository
	catRepo  domain.CategoryRepository
}

func NewTransactionUseCase(repo domain.TransactionRepository, accRepo domain.AccountRepository, catRepo domain.CategoryRepository) *TransactionUseCase {
	return &TransactionUseCase{
		repo:     repo,
		accRepo:  accRepo,
		catRepo:  catRepo,
	}
}

func (uc *TransactionUseCase) Create(ctx context.Context, userID int64, req domain.CreateTransactionRequest) (*domain.Transaction, error) {
	if req.Amount <= 0 {
		return nil, fmt.Errorf("%w: el monto debe ser mayor a 0", domain.ErrValidation)
	}

	req.Type = strings.TrimSpace(strings.ToLower(req.Type))
	if req.Type != "income" && req.Type != "expense" {
		return nil, fmt.Errorf("%w: el tipo debe ser 'income' o 'expense'", domain.ErrValidation)
	}

	req.Currency = strings.TrimSpace(strings.ToUpper(req.Currency))
	if req.Currency == "" {
		req.Currency = "USD"
	}

	// Verify account belongs to user
	acc, err := uc.accRepo.FindByID(ctx, req.AccountID)
	if err != nil {
		return nil, fmt.Errorf("%w: cuenta no encontrada o inválida", domain.ErrValidation)
	}
	if acc.UserID != userID || acc.DeletedAt != nil {
		return nil, domain.ErrForbidden
	}

	// Verify category belongs to user
	cat, err := uc.catRepo.FindByID(ctx, req.CategoryID)
	if err != nil {
		return nil, fmt.Errorf("%w: categoría no encontrada o inválida", domain.ErrValidation)
	}
	if cat.UserID != userID || cat.DeletedAt != nil {
		return nil, domain.ErrForbidden
	}

	// Category type must match transaction type
	if cat.Type != req.Type {
		return nil, fmt.Errorf("%w: el tipo de categoría (%s) no coincide con el tipo de transacción (%s)", domain.ErrValidation, cat.Type, req.Type)
	}

	var parsedDate time.Time
	if strings.TrimSpace(req.Date) == "" {
		parsedDate = time.Now()
	} else {
		var err error
		parsedDate, err = time.Parse(time.RFC3339, req.Date)
		if err != nil {
			parsedDate, err = time.Parse("2006-01-02", req.Date)
			if err != nil {
				return nil, fmt.Errorf("%w: formato de fecha inválido (debe ser RFC3339 o YYYY-MM-DD)", domain.ErrValidation)
			}
		}
	}

	return uc.repo.Create(ctx, userID, req, parsedDate)
}

func (uc *TransactionUseCase) GetByID(ctx context.Context, userID int64, id int64) (*domain.Transaction, error) {
	tx, err := uc.repo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if tx.UserID != userID {
		return nil, domain.ErrForbidden
	}
	return tx, nil
}

func (uc *TransactionUseCase) GetAll(ctx context.Context, userID int64, filter domain.TransactionFilter) ([]domain.Transaction, error) {
	return uc.repo.FindByUser(ctx, userID, filter)
}

func (uc *TransactionUseCase) Update(ctx context.Context, userID int64, id int64, req domain.UpdateTransactionRequest) (*domain.Transaction, error) {
	tx, err := uc.GetByID(ctx, userID, id)
	if err != nil {
		return nil, err
	}

	if tx.DeletedAt != nil {
		return nil, fmt.Errorf("%w: no se puede actualizar una transacción eliminada", domain.ErrConflict)
	}

	if req.Amount != nil && *req.Amount <= 0 {
		return nil, fmt.Errorf("%w: el monto debe ser mayor a 0", domain.ErrValidation)
	}

	txType := tx.Type
	if req.Type != nil {
		lower := strings.TrimSpace(strings.ToLower(*req.Type))
		if lower != "income" && lower != "expense" {
			return nil, fmt.Errorf("%w: el tipo debe ser 'income' o 'expense'", domain.ErrValidation)
		}
		txType = lower
		req.Type = &lower
	}

	if req.AccountID != nil {
		acc, err := uc.accRepo.FindByID(ctx, *req.AccountID)
		if err != nil {
			return nil, fmt.Errorf("%w: cuenta no encontrada", domain.ErrValidation)
		}
		if acc.UserID != userID || acc.DeletedAt != nil {
			return nil, domain.ErrForbidden
		}
	}

	catID := tx.CategoryID
	if req.CategoryID != nil {
		catID = *req.CategoryID
	}

	// Validate category matches transaction type if category or type is changed
	if req.CategoryID != nil || req.Type != nil {
		cat, err := uc.catRepo.FindByID(ctx, catID)
		if err != nil {
			return nil, fmt.Errorf("%w: categoría no encontrada", domain.ErrValidation)
		}
		if cat.UserID != userID || cat.DeletedAt != nil {
			return nil, domain.ErrForbidden
		}
		if cat.Type != txType {
			return nil, fmt.Errorf("%w: el tipo de categoría (%s) no coincide con el tipo de transacción (%s)", domain.ErrValidation, cat.Type, txType)
		}
	}

	if req.Currency != nil {
		upper := strings.TrimSpace(strings.ToUpper(*req.Currency))
		if upper == "" {
			return nil, fmt.Errorf("%w: la moneda no puede estar vacía", domain.ErrValidation)
		}
		req.Currency = &upper
	}

	var parsedDate *time.Time
	if req.Date != nil {
		t, err := time.Parse(time.RFC3339, *req.Date)
		if err != nil {
			t, err = time.Parse("2006-01-02", *req.Date)
			if err != nil {
				return nil, fmt.Errorf("%w: formato de fecha inválido", domain.ErrValidation)
			}
		}
		parsedDate = &t
	}

	return uc.repo.Update(ctx, id, req, parsedDate)
}

func (uc *TransactionUseCase) Delete(ctx context.Context, userID int64, id int64) error {
	_, err := uc.GetByID(ctx, userID, id)
	if err != nil {
		return err
	}
	return uc.repo.SoftDelete(ctx, id)
}

func (uc *TransactionUseCase) Restore(ctx context.Context, userID int64, id int64) error {
	tx, err := uc.repo.FindByID(ctx, id)
	if err != nil {
		return err
	}
	if tx.UserID != userID {
		return domain.ErrForbidden
	}
	return uc.repo.Restore(ctx, id)
}

func (uc *TransactionUseCase) PermanentDelete(ctx context.Context, userID int64, id int64) error {
	tx, err := uc.repo.FindByID(ctx, id)
	if err != nil {
		return err
	}
	if tx.UserID != userID {
		return domain.ErrForbidden
	}
	return uc.repo.PermanentDelete(ctx, id)
}

func (uc *TransactionUseCase) GetSummary(ctx context.Context, userID int64, from, to *time.Time) (*domain.TransactionSummary, error) {
	return uc.repo.GetSummary(ctx, userID, from, to)
}

func (uc *TransactionUseCase) GetByCategory(ctx context.Context, userID int64, from, to *time.Time, txType *string) ([]domain.CategoryAnalytics, error) {
	if txType != nil {
		trimmed := strings.TrimSpace(strings.ToLower(*txType))
		if trimmed != "" {
			if trimmed != "income" && trimmed != "expense" {
				return nil, fmt.Errorf("%w: tipo inválido en analíticas", domain.ErrValidation)
			}
			txType = &trimmed
		} else {
			txType = nil
		}
	}
	return uc.repo.GetByCategory(ctx, userID, from, to, txType)
}
