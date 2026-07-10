package api

import (
	"encoding/json"
	"net/http"

	"github.com/AaronEscobar1/common/middleware"
	"github.com/AaronEscobar1/common/response"
	"github.com/aaron/finflow-backend/internal/domain"
	"github.com/aaron/finflow-backend/internal/usecase"
)

type TransactionHandler struct {
	uc *usecase.TransactionUseCase
}

func NewTransactionHandler(uc *usecase.TransactionUseCase) *TransactionHandler {
	return &TransactionHandler{uc: uc}
}

// HandleCreate registra una nueva transacción.
// @Summary Registrar transacción
// @Description Registra un nuevo ingreso o gasto. Valida que la cuenta y categoría pertenezcan al usuario.
// @Tags Transactions
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param body body domain.CreateTransactionRequest true "Datos de la transacción"
// @Success 200 {object} domain.Transaction "Transacción registrada correctamente"
// @Router /api/v1/transactions [post]
func (h *TransactionHandler) HandleCreate(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok {
		response.Error(w, r.Context(), http.StatusOK, "UNAUTHORIZED", "sesión no válida")
		return
	}

	var req domain.CreateTransactionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, r.Context(), http.StatusOK, "INVALID_JSON", "JSON inválido")
		return
	}

	tx, err := h.uc.Create(r.Context(), userID, req)
	if err != nil {
		mapError(w, r, err)
		return
	}

	response.Success(w, r.Context(), "CREATED", "transacción registrada correctamente", tx)
}

// HandleGet obtiene una transacción específica.
// @Summary Obtener una transacción
// @Description Obtiene los detalles de una transacción del usuario.
// @Tags Transactions
// @Security BearerAuth
// @Produce json
// @Param id path int true "ID de la transacción"
// @Success 200 {object} domain.Transaction "Transacción obtenida"
// @Router /api/v1/transactions/{id} [get]
func (h *TransactionHandler) HandleGet(w http.ResponseWriter, r *http.Request) {
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

	tx, err := h.uc.GetByID(r.Context(), userID, id)
	if err != nil {
		mapError(w, r, err)
		return
	}

	response.Success(w, r.Context(), "SUCCESS", "transacción obtenida", tx)
}

// HandleList obtiene la lista de transacciones del usuario con filtros dinámicos.
// @Summary Listar transacciones
// @Description Obtiene la lista de transacciones aplicando filtros opcionales (fechas, cuentas, categorías, texto).
// @Tags Transactions
// @Security BearerAuth
// @Produce json
// @Param from query string false "Fecha desde (YYYY-MM-DD o RFC3339)"
// @Param to query string false "Fecha hasta (YYYY-MM-DD o RFC3339)"
// @Param type query string false "Filtrar por tipo (income|expense)"
// @Param category_id query int false "ID de la categoría"
// @Param account_id query int false "ID de la cuenta"
// @Param q query string false "Búsqueda por descripción (trigram search)"
// @Param trashed query bool false "Incluir transacciones eliminadas lógicamente"
// @Param limit query int false "Límite de paginación (defecto 20)"
// @Param offset query int false "Offset de paginación (defecto 0)"
// @Success 200 {object} []domain.Transaction "Lista de transacciones obtenida"
// @Router /api/v1/transactions [get]
func (h *TransactionHandler) HandleList(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok {
		response.Error(w, r.Context(), http.StatusOK, "UNAUTHORIZED", "sesión no válida")
		return
	}

	filter := domain.TransactionFilter{
		From:       parseQueryTime(r, "from"),
		To:         parseQueryTime(r, "to"),
		Type:       parseQueryString(r, "type"),
		CategoryID: parseQueryInt64(r, "category_id"),
		AccountID:  parseQueryInt64(r, "account_id"),
		Query:      parseQueryString(r, "q"),
		Limit:      parseQueryInt(r, "limit", 20),
		Offset:     parseQueryInt(r, "offset", 0),
	}

	if t := parseQueryBool(r, "trashed"); t != nil {
		filter.Trashed = *t
	}

	txs, err := h.uc.GetAll(r.Context(), userID, filter)
	if err != nil {
		mapError(w, r, err)
		return
	}

	if txs == nil {
		txs = []domain.Transaction{}
	}

	response.Success(w, r.Context(), "SUCCESS", "lista de transacciones obtenida", txs)
}

// HandleUpdate actualiza una transacción existente.
// @Summary Actualizar transacción
// @Description Actualiza campos específicos de una transacción.
// @Tags Transactions
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path int true "ID de la transacción"
// @Param body body domain.UpdateTransactionRequest true "Campos a actualizar"
// @Success 200 {object} domain.Transaction "Transacción actualizada correctamente"
// @Router /api/v1/transactions/{id} [put]
func (h *TransactionHandler) HandleUpdate(w http.ResponseWriter, r *http.Request) {
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

	var req domain.UpdateTransactionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, r.Context(), http.StatusOK, "INVALID_JSON", "JSON inválido")
		return
	}

	tx, err := h.uc.Update(r.Context(), userID, id, req)
	if err != nil {
		mapError(w, r, err)
		return
	}

	response.Success(w, r.Context(), "SUCCESS", "transacción actualizada correctamente", tx)
}

// HandleDelete mueve una transacción a la papelera (soft delete).
// @Summary Eliminar transacción (Lógico)
// @Description Elimina de forma lógica una transacción (soft-delete).
// @Tags Transactions
// @Security BearerAuth
// @Produce json
// @Param id path int true "ID de la transacción"
// @Success 200 {string} string "Transacción eliminada correctamente"
// @Router /api/v1/transactions/{id} [delete]
func (h *TransactionHandler) HandleDelete(w http.ResponseWriter, r *http.Request) {
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

	response.Success(w, r.Context(), "SUCCESS", "transacción eliminada correctamente", nil)
}

// HandleRestore recupera una transacción de la papelera.
// @Summary Restaurar transacción
// @Description Restaura una transacción eliminada lógicamente.
// @Tags Transactions
// @Security BearerAuth
// @Produce json
// @Param id path int true "ID de la transacción"
// @Success 200 {string} string "Transacción restaurada correctamente"
// @Router /api/v1/transactions/{id}/restore [post]
func (h *TransactionHandler) HandleRestore(w http.ResponseWriter, r *http.Request) {
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

	response.Success(w, r.Context(), "SUCCESS", "transacción restaurada correctamente", nil)
}

// HandlePermanentDelete elimina físicamente una transacción.
// @Summary Eliminar transacción (Permanente)
// @Description Elimina físicamente una transacción de la base de datos.
// @Tags Transactions
// @Security BearerAuth
// @Produce json
// @Param id path int true "ID de la transacción"
// @Success 200 {string} string "Transacción eliminada permanentemente"
// @Router /api/v1/transactions/{id}/permanent [delete]
func (h *TransactionHandler) HandlePermanentDelete(w http.ResponseWriter, r *http.Request) {
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

	response.Success(w, r.Context(), "SUCCESS", "transacción eliminada permanentemente", nil)
}

// HandleSummary obtiene un resumen financiero consolidado.
// @Summary Resumen financiero
// @Description Obtiene el total de ingresos, gastos, balance y cantidad de transacciones del usuario en un rango de fechas.
// @Tags Transactions
// @Security BearerAuth
// @Produce json
// @Param from query string false "Fecha desde (YYYY-MM-DD o RFC3339)"
// @Param to query string false "Fecha hasta (YYYY-MM-DD o RFC3339)"
// @Success 200 {object} domain.TransactionSummary "Resumen de transacciones obtenido"
// @Router /api/v1/transactions/summary [get]
func (h *TransactionHandler) HandleSummary(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok {
		response.Error(w, r.Context(), http.StatusOK, "UNAUTHORIZED", "sesión no válida")
		return
	}

	from := parseQueryTime(r, "from")
	to := parseQueryTime(r, "to")

	sum, err := h.uc.GetSummary(r.Context(), userID, from, to)
	if err != nil {
		mapError(w, r, err)
		return
	}

	response.Success(w, r.Context(), "SUCCESS", "resumen de transacciones obtenido", sum)
}

// HandleByCategory obtiene las analíticas agrupadas por categoría.
// @Summary Analíticas por categoría
// @Description Obtiene el listado de gastos o ingresos agrupado por categoría, con montos y porcentajes de participación.
// @Tags Transactions
// @Security BearerAuth
// @Produce json
// @Param from query string false "Fecha desde (YYYY-MM-DD o RFC3339)"
// @Param to query string false "Fecha hasta (YYYY-MM-DD o RFC3339)"
// @Param type query string false "Filtrar tipo de transacción (income|expense)"
// @Success 200 {object} []domain.CategoryAnalytics "Analíticas obtenidas"
// @Router /api/v1/analytics/by-category [get]
func (h *TransactionHandler) HandleByCategory(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok {
		response.Error(w, r.Context(), http.StatusOK, "UNAUTHORIZED", "sesión no válida")
		return
	}

	from := parseQueryTime(r, "from")
	to := parseQueryTime(r, "to")
	txType := parseQueryString(r, "type")

	analytics, err := h.uc.GetByCategory(r.Context(), userID, from, to, txType)
	if err != nil {
		mapError(w, r, err)
		return
	}

	if analytics == nil {
		analytics = []domain.CategoryAnalytics{}
	}

	response.Success(w, r.Context(), "SUCCESS", "analítica por categoría obtenida", analytics)
}
