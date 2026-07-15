package usecase

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/aaron/finflow-backend/internal/domain"
)

type DebtUseCase struct {
	repo    domain.DebtRepository
	txRepo  domain.TransactionRepository
	catRepo domain.CategoryRepository
	accRepo domain.AccountRepository
}

func NewDebtUseCase(repo domain.DebtRepository, txRepo domain.TransactionRepository, catRepo domain.CategoryRepository, accRepo domain.AccountRepository) *DebtUseCase {
	return &DebtUseCase{
		repo:    repo,
		txRepo:  txRepo,
		catRepo: catRepo,
		accRepo: accRepo,
	}
}

func (uc *DebtUseCase) Create(ctx context.Context, userID int64, req domain.CreateDebtRequest) (*domain.Debt, error) {
	if req.Amount <= 0 {
		return nil, fmt.Errorf("%w: el monto de la deuda debe ser mayor a 0", domain.ErrValidation)
	}

	req.Direction = strings.TrimSpace(strings.ToLower(req.Direction))
	if req.Direction != "receivable" && req.Direction != "payable" {
		return nil, fmt.Errorf("%w: dirección de deuda inválida (debe ser 'receivable' o 'payable')", domain.ErrValidation)
	}

	req.PersonName = strings.TrimSpace(req.PersonName)
	if req.PersonName == "" {
		return nil, fmt.Errorf("%w: el nombre de la persona es requerido", domain.ErrValidation)
	}

	req.Currency = strings.TrimSpace(strings.ToUpper(req.Currency))
	if req.Currency == "" {
		req.Currency = "USD"
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
				return nil, fmt.Errorf("%w: formato de fecha inválido", domain.ErrValidation)
			}
		}
	}

	var parsedDueDate *time.Time
	if req.DueDate != nil && strings.TrimSpace(*req.DueDate) != "" {
		t, err := time.Parse(time.RFC3339, *req.DueDate)
		if err != nil {
			t, err = time.Parse("2006-01-02", *req.DueDate)
			if err != nil {
				return nil, fmt.Errorf("%w: formato de fecha de vencimiento inválido", domain.ErrValidation)
			}
		}
		parsedDueDate = &t
	}

	return uc.repo.Create(ctx, userID, req, parsedDate, parsedDueDate)
}

func (uc *DebtUseCase) GetByID(ctx context.Context, userID int64, id int64) (*domain.Debt, error) {
	d, err := uc.repo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if d.UserID != userID {
		return nil, domain.ErrForbidden
	}
	return d, nil
}

func (uc *DebtUseCase) GetAll(ctx context.Context, userID int64, direction *string, paid *bool, includeTrashed bool) ([]domain.Debt, error) {
	if direction != nil {
		trimmed := strings.TrimSpace(strings.ToLower(*direction))
		if trimmed != "" {
			if trimmed != "receivable" && trimmed != "payable" {
				return nil, fmt.Errorf("%w: dirección inválida", domain.ErrValidation)
			}
			direction = &trimmed
		} else {
			direction = nil
		}
	}
	return uc.repo.FindAllByUser(ctx, userID, direction, paid, includeTrashed)
}

func (uc *DebtUseCase) Update(ctx context.Context, userID int64, id int64, req domain.UpdateDebtRequest) (*domain.Debt, error) {
	debt, err := uc.GetByID(ctx, userID, id)
	if err != nil {
		return nil, err
	}

	if debt.DeletedAt != nil {
		return nil, fmt.Errorf("%w: no se puede actualizar una deuda eliminada", domain.ErrConflict)
	}

	if req.Amount != nil && *req.Amount <= 0 {
		return nil, fmt.Errorf("%w: el monto debe ser mayor a 0", domain.ErrValidation)
	}

	if req.Direction != nil {
		lower := strings.TrimSpace(strings.ToLower(*req.Direction))
		if lower != "receivable" && lower != "payable" {
			return nil, fmt.Errorf("%w: dirección inválida", domain.ErrValidation)
		}
		req.Direction = &lower
	}

	if req.PersonName != nil {
		trimmed := strings.TrimSpace(*req.PersonName)
		if trimmed == "" {
			return nil, fmt.Errorf("%w: el nombre de la persona no puede estar vacío", domain.ErrValidation)
		}
		req.PersonName = &trimmed
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

	var parsedDueDate *time.Time
	if req.DueDate != nil && strings.TrimSpace(*req.DueDate) != "" {
		t, err := time.Parse(time.RFC3339, *req.DueDate)
		if err != nil {
			t, err = time.Parse("2006-01-02", *req.DueDate)
			if err != nil {
				return nil, fmt.Errorf("%w: formato de fecha de vencimiento inválido", domain.ErrValidation)
			}
		}
		parsedDueDate = &t
	}

	return uc.repo.Update(ctx, id, req, parsedDate, parsedDueDate)
}

func (uc *DebtUseCase) MarkPaid(ctx context.Context, userID int64, id int64) (*domain.Debt, error) {
	d, err := uc.GetByID(ctx, userID, id)
	if err != nil {
		return nil, err
	}
	if d.DeletedAt != nil {
		return nil, fmt.Errorf("%w: no se puede pagar una deuda eliminada", domain.ErrConflict)
	}
	if d.IsPaid {
		return d, nil // already paid
	}

	updatedDebt, err := uc.repo.MarkPaid(ctx, id)
	if err != nil {
		return nil, err
	}

	// Determinar el tipo de transacción correspondiente
	targetType := "expense" // Payable -> Pago es un egreso
	if d.Direction == "receivable" {
		targetType = "income" // Receivable -> Cobro es un ingreso
	}

	// Buscar las categorías del usuario para asignar una del tipo correspondiente
	categories, err := uc.catRepo.FindAllByUser(ctx, userID, targetType, false)
	if err != nil {
		return nil, fmt.Errorf("error al obtener categorías del usuario: %w", err)
	}
	if len(categories) == 0 {
		return nil, fmt.Errorf("%w: no se encontró una categoría de tipo %s para registrar el pago", domain.ErrValidation, targetType)
	}
	catID := categories[0].ID

	// Buscar la cuenta del usuario para asignar el pago
	accounts, err := uc.accRepo.FindAllByUser(ctx, userID, false)
	if err != nil {
		return nil, fmt.Errorf("error al obtener cuentas del usuario: %w", err)
	}
	if len(accounts) == 0 {
		return nil, fmt.Errorf("%w: el usuario no tiene ninguna cuenta activa", domain.ErrValidation)
	}
	accID := accounts[0].ID

	// Generar descripción descriptiva
	desc := fmt.Sprintf("Pago de deuda: %s", d.PersonName)
	if d.Direction == "payable" {
		desc = fmt.Sprintf("Pago de deuda a: %s", d.PersonName)
	}
	if strings.TrimSpace(d.Description) != "" {
		desc += fmt.Sprintf(" (%s)", d.Description)
	}

	// Crear el movimiento
	txReq := domain.CreateTransactionRequest{
		AccountID:   accID,
		Amount:      d.Amount,
		Type:        targetType,
		CategoryID:  catID,
		Currency:    d.Currency,
		Description: desc,
		InputInBs:   false,
		DebtID:      &d.ID,
	}

	_, err = uc.txRepo.Create(ctx, userID, txReq, time.Now())
	if err != nil {
		return nil, fmt.Errorf("error al generar transacción de pago de deuda: %w", err)
	}

	return updatedDebt, nil
}

func (uc *DebtUseCase) MarkUnpaid(ctx context.Context, userID int64, id int64) (*domain.Debt, error) {
	d, err := uc.GetByID(ctx, userID, id)
	if err != nil {
		return nil, err
	}
	if d.DeletedAt != nil {
		return nil, fmt.Errorf("%w: no se puede revertir el pago de una deuda eliminada", domain.ErrConflict)
	}
	if !d.IsPaid {
		return d, nil // already unpaid
	}

	updatedDebt, err := uc.repo.MarkUnpaid(ctx, id)
	if err != nil {
		return nil, err
	}

	// Eliminar transacciones vinculadas a esta deuda
	err = uc.txRepo.DeleteByDebtID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("error al eliminar transacciones vinculadas a la deuda: %w", err)
	}

	return updatedDebt, nil
}

func (uc *DebtUseCase) Delete(ctx context.Context, userID int64, id int64) error {
	_, err := uc.GetByID(ctx, userID, id)
	if err != nil {
		return err
	}
	return uc.repo.SoftDelete(ctx, id)
}

func (uc *DebtUseCase) Restore(ctx context.Context, userID int64, id int64) error {
	d, err := uc.repo.FindByID(ctx, id)
	if err != nil {
		return err
	}
	if d.UserID != userID {
		return domain.ErrForbidden
	}
	return uc.repo.Restore(ctx, id)
}

func (uc *DebtUseCase) PermanentDelete(ctx context.Context, userID int64, id int64) error {
	d, err := uc.repo.FindByID(ctx, id)
	if err != nil {
		return err
	}
	if d.UserID != userID {
		return domain.ErrForbidden
	}
	return uc.repo.PermanentDelete(ctx, id)
}
