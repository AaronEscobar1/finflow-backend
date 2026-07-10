package api

import (
	"encoding/json"
	"net/http"

	"github.com/AaronEscobar1/common-go/middleware"
	"github.com/AaronEscobar1/common-go/response"
	"github.com/aaron/finflow-backend/internal/domain"
	"github.com/aaron/finflow-backend/internal/usecase"
)

type DebtHandler struct {
	uc *usecase.DebtUseCase
}

func NewDebtHandler(uc *usecase.DebtUseCase) *DebtHandler {
	return &DebtHandler{uc: uc}
}

// HandleCreate registra una nueva deuda.
// @Summary Registrar deuda
// @Description Registra una nueva deuda (por pagar o por cobrar) con una persona.
// @Tags Debts
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param body body domain.CreateDebtRequest true "Datos de la deuda"
// @Success 200 {object} domain.Debt "Deuda registrada correctamente"
// @Router /api/v1/debts [post]
func (h *DebtHandler) HandleCreate(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok {
		response.Error(w, r.Context(), http.StatusOK, "UNAUTHORIZED", "sesión no válida")
		return
	}

	var req domain.CreateDebtRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, r.Context(), http.StatusOK, "INVALID_JSON", "JSON inválido")
		return
	}

	debt, err := h.uc.Create(r.Context(), userID, req)
	if err != nil {
		mapError(w, r, err)
		return
	}

	response.Success(w, r.Context(), "CREATED", "deuda registrada correctamente", debt)
}

// HandleGet obtiene una deuda específica por su ID.
// @Summary Obtener una deuda
// @Description Obtiene los detalles de una deuda del usuario.
// @Tags Debts
// @Security BearerAuth
// @Produce json
// @Param id path int true "ID de la deuda"
// @Success 200 {object} domain.Debt "Deuda obtenida"
// @Router /api/v1/debts/{id} [get]
func (h *DebtHandler) HandleGet(w http.ResponseWriter, r *http.Request) {
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

	debt, err := h.uc.GetByID(r.Context(), userID, id)
	if err != nil {
		mapError(w, r, err)
		return
	}

	response.Success(w, r.Context(), "SUCCESS", "deuda obtenida", debt)
}

// HandleList obtiene el listado de deudas del usuario.
// @Summary Listar deudas
// @Description Obtiene todas las deudas del usuario. Permite filtrar por dirección (receivable/payable) y estado de pago.
// @Tags Debts
// @Security BearerAuth
// @Produce json
// @Param direction query string false "Dirección de la deuda (receivable|payable)"
// @Param paid query bool false "Filtrar por pagada (true|false)"
// @Param trashed query bool false "Incluir deudas eliminadas"
// @Success 200 {object} []domain.Debt "Lista de deudas obtenida"
// @Router /api/v1/debts [get]
func (h *DebtHandler) HandleList(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok {
		response.Error(w, r.Context(), http.StatusOK, "UNAUTHORIZED", "sesión no válida")
		return
	}

	direction := parseQueryString(r, "direction")
	paid := parseQueryBool(r, "paid")
	trashed := false
	if t := parseQueryBool(r, "trashed"); t != nil {
		trashed = *t
	}

	debts, err := h.uc.GetAll(r.Context(), userID, direction, paid, trashed)
	if err != nil {
		mapError(w, r, err)
		return
	}

	if debts == nil {
		debts = []domain.Debt{}
	}

	response.Success(w, r.Context(), "SUCCESS", "lista de deudas obtenida", debts)
}

// HandleUpdate actualiza los datos de una deuda existente.
// @Summary Actualizar deuda
// @Description Actualiza campos de una deuda específica.
// @Tags Debts
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path int true "ID de la deuda"
// @Param body body domain.UpdateDebtRequest true "Datos a actualizar"
// @Success 200 {object} domain.Debt "Deuda actualizada correctamente"
// @Router /api/v1/debts/{id} [put]
func (h *DebtHandler) HandleUpdate(w http.ResponseWriter, r *http.Request) {
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

	var req domain.UpdateDebtRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, r.Context(), http.StatusOK, "INVALID_JSON", "JSON inválido")
		return
	}

	debt, err := h.uc.Update(r.Context(), userID, id, req)
	if err != nil {
		mapError(w, r, err)
		return
	}

	response.Success(w, r.Context(), "SUCCESS", "deuda actualizada correctamente", debt)
}

// HandleMarkPaid marca una deuda como pagada.
// @Summary Pagar deuda
// @Description Marca una deuda como liquidada (is_paid: true, paid_at: ahora).
// @Tags Debts
// @Security BearerAuth
// @Produce json
// @Param id path int true "ID de la deuda"
// @Success 200 {object} domain.Debt "Deuda marcada como pagada correctamente"
// @Router /api/v1/debts/{id}/pay [post]
func (h *DebtHandler) HandleMarkPaid(w http.ResponseWriter, r *http.Request) {
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

	debt, err := h.uc.MarkPaid(r.Context(), userID, id)
	if err != nil {
		mapError(w, r, err)
		return
	}

	response.Success(w, r.Context(), "SUCCESS", "deuda marcada como pagada correctamente", debt)
}

// HandleDelete mueve una deuda a la papelera (soft delete).
// @Summary Eliminar deuda (Lógico)
// @Description Elimina de forma lógica una deuda (soft-delete).
// @Tags Debts
// @Security BearerAuth
// @Produce json
// @Param id path int true "ID de la deuda"
// @Success 200 {string} string "Deuda eliminada correctamente"
// @Router /api/v1/debts/{id} [delete]
func (h *DebtHandler) HandleDelete(w http.ResponseWriter, r *http.Request) {
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

	response.Success(w, r.Context(), "SUCCESS", "deuda eliminada correctamente", nil)
}

// HandleRestore recupera una deuda de la papelera.
// @Summary Restaurar deuda
// @Description Restaura una deuda eliminada lógicamente.
// @Tags Debts
// @Security BearerAuth
// @Produce json
// @Param id path int true "ID de la deuda"
// @Success 200 {string} string "Deuda restaurada correctamente"
// @Router /api/v1/debts/{id}/restore [post]
func (h *DebtHandler) HandleRestore(w http.ResponseWriter, r *http.Request) {
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

	response.Success(w, r.Context(), "SUCCESS", "deuda restaurada correctamente", nil)
}

// HandlePermanentDelete elimina físicamente una deuda.
// @Summary Eliminar deuda (Permanente)
// @Description Elimina de forma física y permanente una deuda de la base de datos.
// @Tags Debts
// @Security BearerAuth
// @Produce json
// @Param id path int true "ID de la deuda"
// @Success 200 {string} string "Deuda eliminada permanentemente"
// @Router /api/v1/debts/{id}/permanent [delete]
func (h *DebtHandler) HandlePermanentDelete(w http.ResponseWriter, r *http.Request) {
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

	response.Success(w, r.Context(), "SUCCESS", "deuda eliminada permanentemente", nil)
}
