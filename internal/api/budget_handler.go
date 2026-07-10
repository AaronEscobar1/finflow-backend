package api

import (
	"encoding/json"
	"net/http"

	"github.com/AaronEscobar1/common/middleware"
	"github.com/AaronEscobar1/common/response"
	"github.com/aaron/finflow-backend/internal/domain"
	"github.com/aaron/finflow-backend/internal/usecase"
)

type BudgetHandler struct {
	uc *usecase.BudgetUseCase
}

func NewBudgetHandler(uc *usecase.BudgetUseCase) *BudgetHandler {
	return &BudgetHandler{uc: uc}
}

// HandleCreate crea un nuevo presupuesto.
// @Summary Crear presupuesto
// @Description Crea un nuevo presupuesto (semanal/mensual/anual) para una categoría o global.
// @Tags Budgets
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param body body domain.CreateBudgetRequest true "Datos del presupuesto"
// @Success 200 {object} domain.Budget "Presupuesto creado correctamente"
// @Router /api/v1/budgets [post]
func (h *BudgetHandler) HandleCreate(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok {
		response.Error(w, r.Context(), http.StatusOK, "UNAUTHORIZED", "sesión no válida")
		return
	}

	var req domain.CreateBudgetRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, r.Context(), http.StatusOK, "INVALID_JSON", "JSON inválido")
		return
	}

	b, err := h.uc.Create(r.Context(), userID, req)
	if err != nil {
		mapError(w, r, err)
		return
	}

	response.Success(w, r.Context(), "CREATED", "presupuesto creado correctamente", b)
}

// HandleList obtiene la lista de presupuestos con el cálculo de consumo en tiempo real.
// @Summary Listar presupuestos
// @Description Obtiene todos los presupuestos con los totales gastados (spent) y restantes (remaining) en el periodo.
// @Tags Budgets
// @Security BearerAuth
// @Produce json
// @Param trashed query bool false "Incluir presupuestos eliminados"
// @Success 200 {object} []domain.BudgetWithSpent "Lista de presupuestos obtenida"
// @Router /api/v1/budgets [get]
func (h *BudgetHandler) HandleList(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok {
		response.Error(w, r.Context(), http.StatusOK, "UNAUTHORIZED", "sesión no válida")
		return
	}

	trashed := false
	if t := parseQueryBool(r, "trashed"); t != nil {
		trashed = *t
	}

	budgets, err := h.uc.GetAll(r.Context(), userID, trashed)
	if err != nil {
		mapError(w, r, err)
		return
	}

	if budgets == nil {
		budgets = []domain.BudgetWithSpent{}
	}

	response.Success(w, r.Context(), "SUCCESS", "lista de presupuestos obtenida", budgets)
}

// HandleUpdate actualiza un presupuesto existente.
// @Summary Actualizar presupuesto
// @Description Actualiza el monto, periodo o categoría de un presupuesto.
// @Tags Budgets
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path int true "ID del presupuesto"
// @Param body body domain.UpdateBudgetRequest true "Datos a actualizar"
// @Success 200 {object} domain.Budget "Presupuesto actualizado correctamente"
// @Router /api/v1/budgets/{id} [put]
func (h *BudgetHandler) HandleUpdate(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok {
		response.Error(w, r.Context(), http.StatusOK, "UNAUTHORIZED", "sesión no válida")
		return
	}

	id, err := extractPathID(r, "id")
	if err != nil {
		response.Error(w, r.Context(), http.StatusOK, "BAD_REQUEST", err.Error())
		return
	}

	var req domain.UpdateBudgetRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, r.Context(), http.StatusOK, "INVALID_JSON", "JSON inválido")
		return
	}

	b, err := h.uc.Update(r.Context(), userID, id, req)
	if err != nil {
		mapError(w, r, err)
		return
	}

	response.Success(w, r.Context(), "SUCCESS", "presupuesto actualizado correctamente", b)
}

// HandleDelete mueve un presupuesto a la papelera.
// @Summary Eliminar presupuesto (Lógico)
// @Description Elimina de forma lógica un presupuesto (soft-delete).
// @Tags Budgets
// @Security BearerAuth
// @Produce json
// @Param id path int true "ID del presupuesto"
// @Success 200 {string} string "Presupuesto eliminado correctamente"
// @Router /api/v1/budgets/{id} [delete]
func (h *BudgetHandler) HandleDelete(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok {
		response.Error(w, r.Context(), http.StatusOK, "UNAUTHORIZED", "sesión no válida")
		return
	}

	id, err := extractPathID(r, "id")
	if err != nil {
		response.Error(w, r.Context(), http.StatusOK, "BAD_REQUEST", err.Error())
		return
	}

	err = h.uc.Delete(r.Context(), userID, id)
	if err != nil {
		mapError(w, r, err)
		return
	}

	response.Success(w, r.Context(), "SUCCESS", "presupuesto eliminado correctamente", nil)
}

// HandleRestore recupera un presupuesto de la papelera.
// @Summary Restaurar presupuesto
// @Description Restaura un presupuesto eliminado lógicamente.
// @Tags Budgets
// @Security BearerAuth
// @Produce json
// @Param id path int true "ID del presupuesto"
// @Success 200 {string} string "Presupuesto restaurado correctamente"
// @Router /api/v1/budgets/{id}/restore [post]
func (h *BudgetHandler) HandleRestore(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok {
		response.Error(w, r.Context(), http.StatusOK, "UNAUTHORIZED", "sesión no válida")
		return
	}

	id, err := extractPathID(r, "id")
	if err != nil {
		response.Error(w, r.Context(), http.StatusOK, "BAD_REQUEST", err.Error())
		return
	}

	err = h.uc.Restore(r.Context(), userID, id)
	if err != nil {
		mapError(w, r, err)
		return
	}

	response.Success(w, r.Context(), "SUCCESS", "presupuesto restaurado correctamente", nil)
}

// HandlePermanentDelete elimina físicamente un presupuesto de la base de datos.
// @Summary Eliminar presupuesto (Permanente)
// @Description Elimina de forma física y permanente un presupuesto.
// @Tags Budgets
// @Security BearerAuth
// @Produce json
// @Param id path int true "ID del presupuesto"
// @Success 200 {string} string "Presupuesto eliminado permanentemente"
// @Router /api/v1/budgets/{id}/permanent [delete]
func (h *BudgetHandler) HandlePermanentDelete(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok {
		response.Error(w, r.Context(), http.StatusOK, "UNAUTHORIZED", "sesión no válida")
		return
	}

	id, err := extractPathID(r, "id")
	if err != nil {
		response.Error(w, r.Context(), http.StatusOK, "BAD_REQUEST", err.Error())
		return
	}

	err = h.uc.PermanentDelete(r.Context(), userID, id)
	if err != nil {
		mapError(w, r, err)
		return
	}

	response.Success(w, r.Context(), "SUCCESS", "presupuesto eliminado permanentemente", nil)
}
