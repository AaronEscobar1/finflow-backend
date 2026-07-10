package api

import (
	"encoding/json"
	"net/http"

	"github.com/AaronEscobar1/common/middleware"
	"github.com/AaronEscobar1/common/response"
	"github.com/aaron/finflow-backend/internal/domain"
	"github.com/aaron/finflow-backend/internal/usecase"
)

type CategoryHandler struct {
	uc *usecase.CategoryUseCase
}

func NewCategoryHandler(uc *usecase.CategoryUseCase) *CategoryHandler {
	return &CategoryHandler{uc: uc}
}

// HandleCreate crea una nueva categoría.
// @Summary Crear categoría
// @Description Crea una nueva categoría (ingreso/gasto) para el usuario autenticado.
// @Tags Categories
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param body body domain.CreateCategoryRequest true "Datos de la categoría"
// @Success 200 {object} domain.Category "Categoría creada correctamente"
// @Router /api/v1/categories [post]
func (h *CategoryHandler) HandleCreate(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok {
		response.Error(w, r.Context(), http.StatusOK, "UNAUTHORIZED", "sesión no válida")
		return
	}

	var req domain.CreateCategoryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, r.Context(), http.StatusOK, "INVALID_JSON", "JSON inválido")
		return
	}

	cat, err := h.uc.Create(r.Context(), userID, req)
	if err != nil {
		mapError(w, r, err)
		return
	}

	response.Success(w, r.Context(), "CREATED", "categoría creada correctamente", cat)
}

// HandleList obtiene la lista de categorías del usuario.
// @Summary Listar categorías
// @Description Obtiene las categorías de ingresos/gastos del usuario. Permite filtrar por tipo e incluir eliminadas.
// @Tags Categories
// @Security BearerAuth
// @Produce json
// @Param type query string false "Filtrar por tipo (income|expense)"
// @Param trashed query bool false "Incluir categorías en papelera"
// @Success 200 {object} []domain.Category "Lista de categorías obtenida"
// @Router /api/v1/categories [get]
func (h *CategoryHandler) HandleList(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok {
		response.Error(w, r.Context(), http.StatusOK, "UNAUTHORIZED", "sesión no válida")
		return
	}

	typeFilter := ""
	if t := parseQueryString(r, "type"); t != nil {
		typeFilter = *t
	}

	trashed := false
	if t := parseQueryBool(r, "trashed"); t != nil {
		trashed = *t
	}

	categories, err := h.uc.GetAll(r.Context(), userID, typeFilter, trashed)
	if err != nil {
		mapError(w, r, err)
		return
	}

	if categories == nil {
		categories = []domain.Category{}
	}

	response.Success(w, r.Context(), "SUCCESS", "lista de categorías obtenida", categories)
}

// HandleUpdate actualiza una categoría existente.
// @Summary Actualizar categoría
// @Description Actualiza campos de una categoría (nombre, tipo, color, icono).
// @Tags Categories
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path int true "ID de la categoría"
// @Param body body domain.UpdateCategoryRequest true "Datos a actualizar"
// @Success 200 {object} domain.Category "Categoría actualizada correctamente"
// @Router /api/v1/categories/{id} [put]
func (h *CategoryHandler) HandleUpdate(w http.ResponseWriter, r *http.Request) {
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

	var req domain.UpdateCategoryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, r.Context(), http.StatusOK, "INVALID_JSON", "JSON inválido")
		return
	}

	cat, err := h.uc.Update(r.Context(), userID, id, req)
	if err != nil {
		mapError(w, r, err)
		return
	}

	response.Success(w, r.Context(), "SUCCESS", "categoría actualizada correctamente", cat)
}

// HandleDelete mueve una categoría a la papelera (soft delete).
// @Summary Eliminar categoría (Lógico)
// @Description Mueve la categoría a la papelera (soft-delete).
// @Tags Categories
// @Security BearerAuth
// @Produce json
// @Param id path int true "ID de la categoría"
// @Success 200 {string} string "Categoría eliminada correctamente"
// @Router /api/v1/categories/{id} [delete]
func (h *CategoryHandler) HandleDelete(w http.ResponseWriter, r *http.Request) {
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

	response.Success(w, r.Context(), "SUCCESS", "categoría eliminada correctamente", nil)
}

// HandleRestore recupera una categoría de la papelera.
// @Summary Restaurar categoría
// @Description Restaura una categoría eliminada lógicamente.
// @Tags Categories
// @Security BearerAuth
// @Produce json
// @Param id path int true "ID de la categoría"
// @Success 200 {string} string "Categoría restaurada correctamente"
// @Router /api/v1/categories/{id}/restore [post]
func (h *CategoryHandler) HandleRestore(w http.ResponseWriter, r *http.Request) {
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

	response.Success(w, r.Context(), "SUCCESS", "categoría restaurada correctamente", nil)
}

// HandlePermanentDelete elimina físicamente una categoría de la base de datos.
// @Summary Eliminar categoría (Permanente)
// @Description Elimina de forma física y permanente una categoría del usuario.
// @Tags Categories
// @Security BearerAuth
// @Produce json
// @Param id path int true "ID de la categoría"
// @Success 200 {string} string "Categoría eliminada permanentemente"
// @Router /api/v1/categories/{id}/permanent [delete]
func (h *CategoryHandler) HandlePermanentDelete(w http.ResponseWriter, r *http.Request) {
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

	response.Success(w, r.Context(), "SUCCESS", "categoría eliminada permanentemente", nil)
}
