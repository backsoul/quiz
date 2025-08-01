package services

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/backsoul/quiz/pkg/models"
	"github.com/backsoul/quiz/pkg/redis"
)

type GameStateService struct {
	redisClient    *redis.RedisClient
	sessionService *SessionService
}

func NewGameStateService(redisClient *redis.RedisClient) *GameStateService {
	return &GameStateService{
		redisClient: redisClient,
	}
}

// SetSessionService permite inyectar el servicio de sesiones para calcular la pregunta actual
func (gs *GameStateService) SetSessionService(sessionService *SessionService) {
	gs.sessionService = sessionService
}

const gameStateKey = "quiz:game_state"

func (gs *GameStateService) GetGameState() (*models.GameState, error) {
	data, err := gs.redisClient.Get(gameStateKey)
	if err != nil && err.Error() == "redis: nil" {
		// Estado inicial del juego
		return &models.GameState{
			IsActive:        false,
			Message:         "Partida detenida - Los jugadores no pueden ingresar",
			CurrentQuestion: 1,
			MaxQuestions:    8, // Número total de preguntas del quiz
		}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("error obteniendo estado del juego: %w", err)
	}

	var gameState models.GameState
	if err := json.Unmarshal([]byte(data), &gameState); err != nil {
		return nil, fmt.Errorf("error deserializando estado del juego: %w", err)
	}

	// Calcular la pregunta actual dinámicamente basándose en el progreso de los jugadores
	if gs.sessionService != nil && gameState.IsActive {
		currentQuestion := gs.calculateCurrentQuestion()
		gameState.CurrentQuestion = currentQuestion
	}

	// Asegurar que MaxQuestions esté establecido
	if gameState.MaxQuestions == 0 {
		gameState.MaxQuestions = 8
	}

	return &gameState, nil
}

// calculateCurrentQuestion calcula la pregunta actual basándose en el progreso de todos los jugadores
func (gs *GameStateService) calculateCurrentQuestion() int {
	if gs.sessionService == nil {
		return 1
	}

	// Obtener todas las sesiones activas
	activeSessions, err := gs.sessionService.GetActiveSessions()
	if err != nil {
		return 1
	}

	if len(activeSessions) == 0 {
		return 1
	}

	// Encontrar la pregunta más alta alcanzada por cualquier jugador
	maxQuestion := 1
	for _, session := range activeSessions {
		if session.CurrentQuestion > maxQuestion {
			maxQuestion = session.CurrentQuestion
		}
	}

	return maxQuestion
}

func (gs *GameStateService) StartGame() error {
	now := time.Now()
	gameState := &models.GameState{
		IsActive:        true,
		StartTime:       &now,
		EndTime:         nil,
		Message:         "Partida activa - Los jugadores pueden ingresar",
		CurrentQuestion: 1,
		MaxQuestions:    8,
	}

	data, err := json.Marshal(gameState)
	if err != nil {
		return fmt.Errorf("error serializando estado del juego: %w", err)
	}

	return gs.redisClient.Set(gameStateKey, string(data), 0)
}

func (gs *GameStateService) EndGame() error {
	currentState, err := gs.GetGameState()
	if err != nil {
		return err
	}

	now := time.Now()
	currentState.IsActive = false
	currentState.EndTime = &now
	currentState.Message = "Partida terminada - Los jugadores no pueden ingresar"
	currentState.CurrentQuestion = 1 // Reset pregunta al terminar
	currentState.MaxQuestions = 8

	data, err := json.Marshal(currentState)
	if err != nil {
		return fmt.Errorf("error serializando estado del juego: %w", err)
	}

	return gs.redisClient.Set(gameStateKey, string(data), 0)
}

func (gs *GameStateService) IsGameActive() (bool, error) {
	gameState, err := gs.GetGameState()
	if err != nil {
		return false, err
	}
	return gameState.IsActive, nil
}
