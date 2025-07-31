package handlers

import (
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/backsoul/quiz/pkg/models"
	"github.com/backsoul/quiz/pkg/services"
	"github.com/valyala/fasthttp"
)

// QuestionHandler maneja las peticiones HTTP para preguntas
type QuestionHandler struct {
	questionService *services.QuestionService
}

// NewQuestionHandler crea una nueva instancia del handler
func NewQuestionHandler(questionService *services.QuestionService) *QuestionHandler {
	return &QuestionHandler{
		questionService: questionService,
	}
}

// respondWithJSON envía una respuesta JSON
func (h *QuestionHandler) respondWithJSON(ctx *fasthttp.RequestCtx, statusCode int, response interface{}) {
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

// respondWithError envía una respuesta de error
func (h *QuestionHandler) respondWithError(ctx *fasthttp.RequestCtx, statusCode int, message string) {
	response := models.APIResponse{
		Success: false,
		Error:   message,
	}
	h.respondWithJSON(ctx, statusCode, response)
}

// respondWithSuccess envía una respuesta exitosa
func (h *QuestionHandler) respondWithSuccess(ctx *fasthttp.RequestCtx, data interface{}, message string) {
	response := models.APIResponse{
		Success: true,
		Message: message,
		Data:    data,
	}
	h.respondWithJSON(ctx, fasthttp.StatusOK, response)
}

// GetAllQuestions maneja GET /api/questions
func (h *QuestionHandler) GetAllQuestions(ctx *fasthttp.RequestCtx) {
	questions, err := h.questionService.GetAllQuestions()
	if err != nil {
		h.respondWithError(ctx, fasthttp.StatusInternalServerError, fmt.Sprintf("Error obteniendo preguntas: %v", err))
		return
	}

	responseData := models.QuestionResponse{
		Questions: questions,
		Count:     len(questions),
	}

	h.respondWithSuccess(ctx, responseData, "Preguntas obtenidas exitosamente")
}

// GetQuestion maneja GET /api/questions/{id}
func (h *QuestionHandler) GetQuestion(ctx *fasthttp.RequestCtx) {
	// Obtener el ID de los parámetros de la URL
	idStr := ctx.UserValue("id").(string)
	id, err := strconv.Atoi(idStr)
	if err != nil {
		h.respondWithError(ctx, fasthttp.StatusBadRequest, "ID de pregunta inválido")
		return
	}

	question, err := h.questionService.GetQuestion(id)
	if err != nil {
		h.respondWithError(ctx, fasthttp.StatusNotFound, fmt.Sprintf("Pregunta no encontrada: %v", err))
		return
	}

	responseData := models.QuestionResponse{
		Question: question,
	}

	h.respondWithSuccess(ctx, responseData, "Pregunta obtenida exitosamente")
}

// GetRandomQuestion maneja GET /api/questions/random
func (h *QuestionHandler) GetRandomQuestion(ctx *fasthttp.RequestCtx) {
	question, err := h.questionService.GetRandomQuestion()
	if err != nil {
		h.respondWithError(ctx, fasthttp.StatusInternalServerError, fmt.Sprintf("Error obteniendo pregunta aleatoria: %v", err))
		return
	}

	responseData := models.QuestionResponse{
		Question: question,
	}

	h.respondWithSuccess(ctx, responseData, "Pregunta aleatoria obtenida exitosamente")
}

// GetQuestionsByDifficulty maneja GET /api/questions/difficulty?min=1&max=5
func (h *QuestionHandler) GetQuestionsByDifficulty(ctx *fasthttp.RequestCtx) {
	// Obtener parámetros de query
	minStr := string(ctx.QueryArgs().Peek("min"))
	maxStr := string(ctx.QueryArgs().Peek("max"))

	if minStr == "" || maxStr == "" {
		h.respondWithError(ctx, fasthttp.StatusBadRequest, "Parámetros 'min' y 'max' son requeridos")
		return
	}

	min, err := strconv.Atoi(minStr)
	if err != nil {
		h.respondWithError(ctx, fasthttp.StatusBadRequest, "Parámetro 'min' debe ser un número")
		return
	}

	max, err := strconv.Atoi(maxStr)
	if err != nil {
		h.respondWithError(ctx, fasthttp.StatusBadRequest, "Parámetro 'max' debe ser un número")
		return
	}

	questions, err := h.questionService.GetQuestionsByDifficulty(min, max)
	if err != nil {
		h.respondWithError(ctx, fasthttp.StatusInternalServerError, fmt.Sprintf("Error obteniendo preguntas por dificultad: %v", err))
		return
	}

	responseData := models.QuestionResponse{
		Questions: questions,
		Count:     len(questions),
	}

	h.respondWithSuccess(ctx, responseData, fmt.Sprintf("Preguntas de dificultad %d-%d obtenidas exitosamente", min, max))
}

// GetRandomQuestionByDifficulty maneja GET /api/questions/random/difficulty?min=1&max=5
func (h *QuestionHandler) GetRandomQuestionByDifficulty(ctx *fasthttp.RequestCtx) {
	// Obtener parámetros de query
	minStr := string(ctx.QueryArgs().Peek("min"))
	maxStr := string(ctx.QueryArgs().Peek("max"))

	if minStr == "" || maxStr == "" {
		h.respondWithError(ctx, fasthttp.StatusBadRequest, "Parámetros 'min' y 'max' son requeridos")
		return
	}

	min, err := strconv.Atoi(minStr)
	if err != nil {
		h.respondWithError(ctx, fasthttp.StatusBadRequest, "Parámetro 'min' debe ser un número")
		return
	}

	max, err := strconv.Atoi(maxStr)
	if err != nil {
		h.respondWithError(ctx, fasthttp.StatusBadRequest, "Parámetro 'max' debe ser un número")
		return
	}

	question, err := h.questionService.GetRandomQuestionByDifficulty(min, max)
	if err != nil {
		h.respondWithError(ctx, fasthttp.StatusNotFound, fmt.Sprintf("Error obteniendo pregunta aleatoria por dificultad: %v", err))
		return
	}

	responseData := models.QuestionResponse{
		Question: question,
	}

	h.respondWithSuccess(ctx, responseData, fmt.Sprintf("Pregunta aleatoria de dificultad %d-%d obtenida exitosamente", min, max))
}

// GetQuestionMetadata maneja GET /api/questions/metadata
func (h *QuestionHandler) GetQuestionMetadata(ctx *fasthttp.RequestCtx) {
	metadata, err := h.questionService.GetQuestionMetadata()
	if err != nil {
		h.respondWithError(ctx, fasthttp.StatusInternalServerError, fmt.Sprintf("Error obteniendo metadatos: %v", err))
		return
	}

	count, err := h.questionService.GetQuestionCount()
	if err != nil {
		h.respondWithError(ctx, fasthttp.StatusInternalServerError, fmt.Sprintf("Error obteniendo conteo: %v", err))
		return
	}

	responseData := models.QuestionResponse{
		Metadata: metadata,
		Count:    count,
	}

	h.respondWithSuccess(ctx, responseData, "Metadatos obtenidos exitosamente")
}

// ReloadQuestions maneja POST /api/questions/reload
func (h *QuestionHandler) ReloadQuestions(ctx *fasthttp.RequestCtx) {
	err := h.questionService.ReloadQuestions("answers.json")
	if err != nil {
		h.respondWithError(ctx, fasthttp.StatusInternalServerError, fmt.Sprintf("Error recargando preguntas: %v", err))
		return
	}

	h.respondWithSuccess(ctx, nil, "Preguntas recargadas exitosamente")
}

// HealthCheck maneja GET /api/health
func (h *QuestionHandler) HealthCheck(ctx *fasthttp.RequestCtx) {
	err := h.questionService.HealthCheck()
	if err != nil {
		h.respondWithError(ctx, fasthttp.StatusServiceUnavailable, fmt.Sprintf("Servicio no disponible: %v", err))
		return
	}

	h.respondWithSuccess(ctx, map[string]interface{}{
		"status": "healthy",
		"redis":  "connected",
	}, "Servicio funcionando correctamente")
}
