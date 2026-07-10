package api

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/AaronEscobar1/common/middleware"
	"github.com/AaronEscobar1/common/response"
	"github.com/aaron/finflow-backend/internal/domain"
	"github.com/aaron/finflow-backend/internal/usecase"
)

type NotificationHandler struct {
	uc *usecase.NotificationUseCase
}

func NewNotificationHandler(uc *usecase.NotificationUseCase) *NotificationHandler {
	return &NotificationHandler{uc: uc}
}

// HandleRegisterDevice registra un nuevo token de dispositivo push.
// @Summary Registrar dispositivo
// @Description Registra un dispositivo (android/ios/web) y su token push para notificaciones.
// @Tags Notifications
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param body body domain.RegisterDeviceRequest true "Datos del dispositivo"
// @Success 200 {object} domain.Device "Dispositivo registrado correctamente"
// @Router /api/v1/devices [post]
func (h *NotificationHandler) HandleRegisterDevice(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok {
		response.Error(w, r.Context(), http.StatusOK, "UNAUTHORIZED", "sesión no válida")
		return
	}

	var req domain.RegisterDeviceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, r.Context(), http.StatusOK, "INVALID_JSON", "JSON inválido")
		return
	}

	dev, err := h.uc.RegisterDevice(r.Context(), userID, req)
	if err != nil {
		mapError(w, r, err)
		return
	}

	response.Success(w, r.Context(), "SUCCESS", "dispositivo registrado correctamente", dev)
}

// HandleRemoveDevice elimina un token de dispositivo push.
// @Summary Eliminar dispositivo
// @Description Elimina el registro del dispositivo para dejar de enviarle notificaciones.
// @Tags Notifications
// @Security BearerAuth
// @Produce json
// @Param token path string true "Token del dispositivo"
// @Success 200 {string} string "Dispositivo eliminado correctamente"
// @Router /api/v1/devices/{token} [delete]
func (h *NotificationHandler) HandleRemoveDevice(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok {
		response.Error(w, r.Context(), http.StatusOK, "UNAUTHORIZED", "sesión no válida")
		return
	}

	// Try extracting from path first, then query param
	token := r.PathValue("token")
	if token == "" {
		token = r.URL.Query().Get("token")
	}
	token = strings.TrimSpace(token)

	if token == "" {
		response.Error(w, r.Context(), http.StatusOK, "BAD_REQUEST", "el token del dispositivo es requerido")
		return
	}

	err := h.uc.RemoveDevice(r.Context(), userID, token)
	if err != nil {
		mapError(w, r, err)
		return
	}

	response.Success(w, r.Context(), "SUCCESS", "dispositivo eliminado correctamente", nil)
}

// HandleListNotifications obtiene el listado de notificaciones de la bandeja del usuario.
// @Summary Listar notificaciones
// @Description Obtiene las notificaciones del usuario con soporte de paginación.
// @Tags Notifications
// @Security BearerAuth
// @Produce json
// @Param limit query int false "Límite de paginación (defecto 20)"
// @Param offset query int false "Offset de paginación (defecto 0)"
// @Success 200 {object} []domain.Notification "Lista de notificaciones obtenida"
// @Router /api/v1/notifications [get]
func (h *NotificationHandler) HandleListNotifications(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok {
		response.Error(w, r.Context(), http.StatusOK, "UNAUTHORIZED", "sesión no válida")
		return
	}

	limit := parseQueryInt(r, "limit", 20)
	offset := parseQueryInt(r, "offset", 0)

	notifications, err := h.uc.GetNotifications(r.Context(), userID, limit, offset)
	if err != nil {
		mapError(w, r, err)
		return
	}

	if notifications == nil {
		notifications = []domain.Notification{}
	}

	response.Success(w, r.Context(), "SUCCESS", "lista de notificaciones obtenida", notifications)
}

// HandleMarkRead marca una notificación como leída.
// @Summary Marcar notificación como leída
// @Description Marca una notificación específica de la bandeja del usuario como leída.
// @Tags Notifications
// @Security BearerAuth
// @Produce json
// @Param id path int true "ID de la notificación"
// @Success 200 {string} string "Notificación leída"
// @Router /api/v1/notifications/{id}/read [put]
func (h *NotificationHandler) HandleMarkRead(w http.ResponseWriter, r *http.Request) {
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

	err = h.uc.MarkRead(r.Context(), userID, id)
	if err != nil {
		mapError(w, r, err)
		return
	}

	response.Success(w, r.Context(), "SUCCESS", "notificación marcada como leída", nil)
}

// HandleMarkAllRead marca todas las notificaciones como leídas.
// @Summary Marcar todas las notificaciones como leídas
// @Description Marca todas las notificaciones de la bandeja del usuario como leídas.
// @Tags Notifications
// @Security BearerAuth
// @Produce json
// @Success 200 {string} string "Todas leídas"
// @Router /api/v1/notifications/read-all [put]
func (h *NotificationHandler) HandleMarkAllRead(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok {
		response.Error(w, r.Context(), http.StatusOK, "UNAUTHORIZED", "sesión no válida")
		return
	}

	err := h.uc.MarkAllRead(r.Context(), userID)
	if err != nil {
		mapError(w, r, err)
		return
	}

	response.Success(w, r.Context(), "SUCCESS", "todas las notificaciones marcadas como leídas", nil)
}

// HandleDeleteNotification elimina una notificación de la bandeja.
// @Summary Eliminar notificación
// @Description Elimina de forma física una notificación de la bandeja del usuario.
// @Tags Notifications
// @Security BearerAuth
// @Produce json
// @Param id path int true "ID de la notificación"
// @Success 200 {string} string "Notificación eliminada correctamente"
// @Router /api/v1/notifications/{id} [delete]
func (h *NotificationHandler) HandleDeleteNotification(w http.ResponseWriter, r *http.Request) {
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

	err = h.uc.DeleteNotification(r.Context(), userID, id)
	if err != nil {
		mapError(w, r, err)
		return
	}

	response.Success(w, r.Context(), "SUCCESS", "notificación eliminada correctamente", nil)
}

type TestPushRequest struct {
	Title string `json:"title"`
	Body  string `json:"body"`
}

// HandleTestPush envía una notificación de prueba (FCM) a todos los dispositivos del usuario.
// @Summary Enviar push de prueba
// @Description Envía una notificación push de prueba a todos los dispositivos del usuario actual.
// @Tags Notifications
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param body body TestPushRequest true "Datos de la notificación de prueba"
// @Success 200 {object} domain.Notification "Notificación creada y enviada"
// @Router /api/v1/notifications/test-push [post]
func (h *NotificationHandler) HandleTestPush(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok {
		response.Error(w, r.Context(), http.StatusOK, "UNAUTHORIZED", "sesión no válida")
		return
	}

	var req TestPushRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, r.Context(), http.StatusOK, "INVALID_JSON", "JSON inválido")
		return
	}

	req.Title = strings.TrimSpace(req.Title)
	req.Body = strings.TrimSpace(req.Body)
	if req.Title == "" || req.Body == "" {
		response.Error(w, r.Context(), http.StatusOK, "BAD_REQUEST", "el título y cuerpo son requeridos")
		return
	}

	payload := map[string]any{
		"click_action": "FLUTTER_NOTIFICATION_CLICK",
		"timestamp":    time.Now().Format(time.RFC3339),
	}

	notif, err := h.uc.CreateAndSend(r.Context(), userID, req.Title, req.Body, "system", payload)
	if err != nil {
		mapError(w, r, err)
		return
	}

	response.Success(w, r.Context(), "SUCCESS", "notificación push de prueba enviada", notif)
}

