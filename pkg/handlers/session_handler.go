package handlers

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/backsoul/quiz/pkg/models"
	"github.com/backsoul/quiz/pkg/services"
	"github.com/valyala/fasthttp"
)

// SessionHandler maneja las peticiones HTTP para sesiones
type SessionHandler struct {
	sessionService  *services.SessionService
	questionService *services.QuestionService
}

// NewSessionHandler crea una nueva instancia del handler de sesiones
func NewSessionHandler(sessionService *services.SessionService, questionService *services.QuestionService) *SessionHandler {
	return &SessionHandler{
		sessionService:  sessionService,
		questionService: questionService,
	}
}

// CreateSession maneja POST /api/sessions
func (h *SessionHandler) CreateSession(ctx *fasthttp.RequestCtx) {
	var request models.SessionCreateRequest
	if err := json.Unmarshal(ctx.PostBody(), &request); err != nil {
		h.respondWithError(ctx, fasthttp.StatusBadRequest, "JSON inválido")
		return
	}

	if request.PlayerName == "" {
		h.respondWithError(ctx, fasthttp.StatusBadRequest, "Nombre del jugador es requerido")
		return
	}

	session, err := h.sessionService.CreateSession(request.PlayerName)
	if err != nil {
		h.respondWithError(ctx, fasthttp.StatusInternalServerError, fmt.Sprintf("Error creando sesión: %v", err))
		return
	}

	responseData := models.SessionResponse{
		Session: session,
		Message: "Sesión creada exitosamente",
	}

	h.respondWithSuccess(ctx, responseData, "Sesión creada exitosamente")
}

// GetSession maneja GET /api/sessions/{id}
func (h *SessionHandler) GetSession(ctx *fasthttp.RequestCtx) {
	sessionID := ctx.UserValue("id").(string)

	session, err := h.sessionService.GetSession(sessionID)
	if err != nil {
		h.respondWithError(ctx, fasthttp.StatusNotFound, fmt.Sprintf("Sesión no encontrada: %v", err))
		return
	}

	responseData := models.SessionResponse{
		Session: session,
	}

	h.respondWithSuccess(ctx, responseData, "Sesión obtenida exitosamente")
}

// GetPlayerSession maneja GET /api/sessions/player/{playerName}
func (h *SessionHandler) GetPlayerSession(ctx *fasthttp.RequestCtx) {
	playerName := ctx.UserValue("playerName").(string)

	session, err := h.sessionService.GetActiveSessionByPlayer(playerName)
	if err != nil {
		h.respondWithError(ctx, fasthttp.StatusNotFound, fmt.Sprintf("No se encontró sesión activa para %s", playerName))
		return
	}

	responseData := models.SessionResponse{
		Session: session,
	}

	h.respondWithSuccess(ctx, responseData, "Sesión del jugador obtenida exitosamente")
}

// GetPlayerHistory maneja GET /api/sessions/player/{playerName}/history
func (h *SessionHandler) GetPlayerHistory(ctx *fasthttp.RequestCtx) {
	playerName := ctx.UserValue("playerName").(string)

	sessions, err := h.sessionService.GetPlayerHistory(playerName)
	if err != nil {
		h.respondWithError(ctx, fasthttp.StatusInternalServerError, fmt.Sprintf("Error obteniendo historial: %v", err))
		return
	}

	responseData := models.SessionResponse{
		Sessions: sessions,
	}

	h.respondWithSuccess(ctx, responseData, "Historial del jugador obtenido exitosamente")
}

// GetActiveSessions maneja GET /api/sessions/active
func (h *SessionHandler) GetActiveSessions(ctx *fasthttp.RequestCtx) {
	sessions, err := h.sessionService.GetActiveSessions()
	if err != nil {
		h.respondWithError(ctx, fasthttp.StatusInternalServerError, fmt.Sprintf("Error obteniendo sesiones activas: %v", err))
		return
	}

	responseData := models.SessionResponse{
		Sessions: sessions,
	}

	h.respondWithSuccess(ctx, responseData, fmt.Sprintf("%d sesiones activas obtenidas", len(sessions)))
}

// SubmitAnswer maneja POST /api/sessions/{id}/answer
func (h *SessionHandler) SubmitAnswer(ctx *fasthttp.RequestCtx) {
	sessionID := ctx.UserValue("id").(string)

	// Estructura para recibir la respuesta
	var answerRequest struct {
		QuestionID     int    `json:"questionId"`
		SelectedOption string `json:"selectedOption"`
		TimeToAnswer   int    `json:"timeToAnswer"`
	}

	if err := json.Unmarshal(ctx.PostBody(), &answerRequest); err != nil {
		h.respondWithError(ctx, fasthttp.StatusBadRequest, "JSON inválido")
		return
	}

	// Obtener la sesión
	session, err := h.sessionService.GetSession(sessionID)
	if err != nil {
		h.respondWithError(ctx, fasthttp.StatusNotFound, "Sesión no encontrada")
		return
	}

	// Obtener la pregunta para verificar la respuesta
	question, err := h.questionService.GetQuestion(answerRequest.QuestionID)
	if err != nil {
		h.respondWithError(ctx, fasthttp.StatusNotFound, "Pregunta no encontrada")
		return
	}

	// Crear la respuesta del jugador
	isCorrect := answerRequest.SelectedOption == question.Correct
	prizeWon := 0
	if isCorrect && session.CurrentQuestion <= len(models.PrizeLevels) {
		prizeWon = models.PrizeLevels[session.CurrentQuestion-1]
	}

	answer := models.PlayerAnswer{
		QuestionID:     answerRequest.QuestionID,
		QuestionNumber: session.CurrentQuestion,
		SelectedOption: answerRequest.SelectedOption,
		CorrectOption:  question.Correct,
		IsCorrect:      isCorrect,
		TimeToAnswer:   answerRequest.TimeToAnswer,
		Timestamp:      time.Now(),
		PrizeWon:       prizeWon,
	}

	// Agregar la respuesta a la sesión
	if err := h.sessionService.AddAnswer(sessionID, answer); err != nil {
		h.respondWithError(ctx, fasthttp.StatusInternalServerError, fmt.Sprintf("Error guardando respuesta: %v", err))
		return
	}

	// Obtener la sesión actualizada
	updatedSession, _ := h.sessionService.GetSession(sessionID)

	responseData := models.SessionResponse{
		Session: updatedSession,
	}

	message := "Respuesta guardada"
	if isCorrect {
		message = fmt.Sprintf("¡Correcto! Has ganado $%d", prizeWon)
	} else {
		message = "Respuesta incorrecta. Juego terminado."
	}

	h.respondWithSuccess(ctx, responseData, message)
}

// UseLifeline maneja POST /api/sessions/{id}/lifeline
func (h *SessionHandler) UseLifeline(ctx *fasthttp.RequestCtx) {
	sessionID := ctx.UserValue("id").(string)

	var lifelineRequest struct {
		Type string `json:"type"`
	}

	if err := json.Unmarshal(ctx.PostBody(), &lifelineRequest); err != nil {
		h.respondWithError(ctx, fasthttp.StatusBadRequest, "JSON inválido")
		return
	}

	if err := h.sessionService.UseLifeline(sessionID, lifelineRequest.Type); err != nil {
		h.respondWithError(ctx, fasthttp.StatusBadRequest, fmt.Sprintf("Error usando comodín: %v", err))
		return
	}

	// Obtener la sesión actualizada
	session, _ := h.sessionService.GetSession(sessionID)

	responseData := models.SessionResponse{
		Session: session,
	}

	h.respondWithSuccess(ctx, responseData, fmt.Sprintf("Comodín %s usado exitosamente", lifelineRequest.Type))
}

// FinishSession maneja POST /api/sessions/{id}/finish
func (h *SessionHandler) FinishSession(ctx *fasthttp.RequestCtx) {
	sessionID := ctx.UserValue("id").(string)

	if err := h.sessionService.FinishSession(sessionID); err != nil {
		h.respondWithError(ctx, fasthttp.StatusInternalServerError, fmt.Sprintf("Error terminando sesión: %v", err))
		return
	}

	h.respondWithSuccess(ctx, nil, "Sesión terminada exitosamente")
}

// GetPlayerNames maneja GET /api/sessions/players
func (h *SessionHandler) GetPlayerNames(ctx *fasthttp.RequestCtx) {
	playerNames, err := h.sessionService.GetPlayerNames()
	if err != nil {
		h.respondWithError(ctx, fasthttp.StatusInternalServerError, fmt.Sprintf("Error obteniendo jugadores: %v", err))
		return
	}

	h.respondWithSuccess(ctx, map[string]interface{}{
		"players": playerNames,
		"count":   len(playerNames),
	}, fmt.Sprintf("%d jugadores registrados", len(playerNames)))
}

// GetLeaderboard maneja GET /api/leaderboard
func (h *SessionHandler) GetLeaderboard(ctx *fasthttp.RequestCtx) {
	leaderboard, err := h.sessionService.GetLeaderboard()
	if err != nil {
		h.respondWithError(ctx, fasthttp.StatusInternalServerError, fmt.Sprintf("Error obteniendo tabla de posiciones: %v", err))
		return
	}

	h.respondWithSuccess(ctx, leaderboard, "Tabla de posiciones obtenida exitosamente")
}

// Métodos auxiliares para respuestas HTTP
func (h *SessionHandler) respondWithJSON(ctx *fasthttp.RequestCtx, statusCode int, response interface{}) {
	ctx.Response.Header.Set("Content-Type", "application/json")
	ctx.SetStatusCode(statusCode)

	jsonData, err := json.Marshal(response)
	if err != nil {
		ctx.SetStatusCode(fasthttp.StatusInternalServerError)
		ctx.SetBodyString(`{"success": false, "error": "Error al serializar respuesta"}`)
		return
	}

	ctx.SetBody(jsonData)
}

func (h *SessionHandler) respondWithError(ctx *fasthttp.RequestCtx, statusCode int, message string) {
	response := models.APIResponse{
		Success: false,
		Error:   message,
	}
	h.respondWithJSON(ctx, statusCode, response)
}

func (h *SessionHandler) respondWithSuccess(ctx *fasthttp.RequestCtx, data interface{}, message string) {
	response := models.APIResponse{
		Success: true,
		Message: message,
		Data:    data,
	}
	h.respondWithJSON(ctx, fasthttp.StatusOK, response)
}
