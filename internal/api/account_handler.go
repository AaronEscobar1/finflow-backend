package api

import (
	"encoding/json"
	"net/http"

	"github.com/AaronEscobar1/common-go/middleware"
	"github.com/AaronEscobar1/common-go/response"
	"github.com/aaron/finflow-backend/internal/domain"
	"github.com/aaron/finflow-backend/internal/usecase"
)

type AccountHandler struct {
	uc *usecase.AccountUseCase
}

func NewAccountHandler(uc *usecase.AccountUseCase) *AccountHandler {
	return &AccountHandler{uc: uc}
}

// HandleCreate crea una nueva cuenta/wallet.
// @Summary Crear cuenta
// @Description Crea una nueva cuenta (wallet) asociada al usuario autenticado.
// @Tags Accounts
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param body body domain.CreateAccountRequest true "Datos de la cuenta"
// @Success 200 {object} domain.Account "Cuenta creada correctamente"
// @Router /api/v1/accounts [post]
func (h *AccountHandler) HandleCreate(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok {
		response.Error(w, r.Context(), http.StatusOK, "UNAUTHORIZED", "sesión no válida")
		return
	}

	var req domain.CreateAccountRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, r.Context(), http.StatusOK, "INVALID_JSON", "JSON inválido")
		return
	}

	acc, err := h.uc.Create(r.Context(), userID, req)
	if err != nil {
		mapError(w, r, err)
		return
	}

	response.Success(w, r.Context(), "CREATED", "cuenta creada correctamente", acc)
}

// HandleGet obtiene los detalles de una cuenta específica.
// @Summary Obtener una cuenta
// @Description Obtiene la información y el balance calculado de una cuenta del usuario.
// @Tags Accounts
// @Security BearerAuth
// @Produce json
// @Param id path int true "ID de la cuenta"
// @Success 200 {object} domain.Account "Cuenta obtenida"
// @Router /api/v1/accounts/{id} [get]
func (h *AccountHandler) HandleGet(w http.ResponseWriter, r *http.Request) {
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

	acc, err := h.uc.GetByID(r.Context(), userID, id)
	if err != nil {
		mapError(w, r, err)
		return
	}

	response.Success(w, r.Context(), "SUCCESS", "cuenta obtenida", acc)
}

// HandleList lista las cuentas del usuario.
// @Summary Listar cuentas
// @Description Obtiene la lista de todas las cuentas del usuario con sus balances. Admite incluir eliminadas lógicamente.
// @Tags Accounts
// @Security BearerAuth
// @Produce json
// @Param trashed query bool false "Incluir cuentas eliminadas en papelera"
// @Success 200 {object} []domain.Account "Lista de cuentas obtenida"
// @Router /api/v1/accounts [get]
func (h *AccountHandler) HandleList(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok {
		response.Error(w, r.Context(), http.StatusOK, "UNAUTHORIZED", "sesión no válida")
		return
	}

	trashed := false
	if t := parseQueryBool(r, "trashed"); t != nil {
		trashed = *t
	}

	accounts, err := h.uc.GetAll(r.Context(), userID, trashed)
	if err != nil {
		mapError(w, r, err)
		return
	}

	// Default to empty array instead of nil
	if accounts == nil {
		accounts = []domain.Account{}
	}

	response.Success(w, r.Context(), "SUCCESS", "lista de cuentas obtenida", accounts)
}

// HandleUpdate actualiza los datos de una cuenta.
// @Summary Actualizar cuenta
// @Description Actualiza campos específicos de una cuenta existente.
// @Tags Accounts
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path int true "ID de la cuenta"
// @Param body body domain.UpdateAccountRequest true "Campos a actualizar"
// @Success 200 {object} domain.Account "Cuenta actualizada correctamente"
// @Router /api/v1/accounts/{id} [put]
func (h *AccountHandler) HandleUpdate(w http.ResponseWriter, r *http.Request) {
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

	var req domain.UpdateAccountRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, r.Context(), http.StatusOK, "INVALID_JSON", "JSON inválido")
		return
	}

	acc, err := h.uc.Update(r.Context(), userID, id, req)
	if err != nil {
		mapError(w, r, err)
		return
	}

	response.Success(w, r.Context(), "SUCCESS", "cuenta actualizada correctamente", acc)
}

// HandleDelete realiza una eliminación lógica (soft delete) de la cuenta.
// @Summary Eliminar cuenta (Lógico)
// @Description Mueve la cuenta a la papelera (soft-delete).
// @Tags Accounts
// @Security BearerAuth
// @Produce json
// @Param id path int true "ID de la cuenta"
// @Success 200 {string} string "Cuenta eliminada correctamente"
// @Router /api/v1/accounts/{id} [delete]
func (h *AccountHandler) HandleDelete(w http.ResponseWriter, r *http.Request) {
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

	response.Success(w, r.Context(), "SUCCESS", "cuenta eliminada correctamente", nil)
}

// HandleRestore restaura una cuenta eliminada lógicamente.
// @Summary Restaurar cuenta
// @Description Restaura una cuenta que fue eliminada lógicamente y estaba en la papelera.
// @Tags Accounts
// @Security BearerAuth
// @Produce json
// @Param id path int true "ID de la cuenta"
// @Success 200 {string} string "Cuenta restaurada correctamente"
// @Router /api/v1/accounts/{id}/restore [post]
func (h *AccountHandler) HandleRestore(w http.ResponseWriter, r *http.Request) {
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

	response.Success(w, r.Context(), "SUCCESS", "cuenta restaurada correctamente", nil)
}

// HandlePermanentDelete realiza una eliminación física permanente de la cuenta de la base de datos.
// @Summary Eliminar cuenta (Permanente)
// @Description Elimina de forma física y permanente una cuenta del usuario.
// @Tags Accounts
// @Security BearerAuth
// @Produce json
// @Param id path int true "ID de la cuenta"
// @Success 200 {string} string "Cuenta eliminada permanentemente"
// @Router /api/v1/accounts/{id}/permanent [delete]
func (h *AccountHandler) HandlePermanentDelete(w http.ResponseWriter, r *http.Request) {
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

	response.Success(w, r.Context(), "SUCCESS", "cuenta eliminada permanentemente", nil)
}
