package services

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/backsoul/quiz/pkg/models"
	"github.com/backsoul/quiz/pkg/redis"
)

type GameStateService struct {
	redisClient *redis.RedisClient
}

func NewGameStateService(redisClient *redis.RedisClient) *GameStateService {
	return &GameStateService{
		redisClient: redisClient,
	}
}

const gameStateKey = "quiz:game_state"

func (gs *GameStateService) GetGameState() (*models.GameState, error) {
	data, err := gs.redisClient.Get(gameStateKey)
	if err != nil && err.Error() == "redis: nil" {
		// Estado inicial del juego
		return &models.GameState{
			IsActive: false,
			Message:  "Partida detenida - Los jugadores no pueden ingresar",
		}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("error obteniendo estado del juego: %w", err)
	}

	var gameState models.GameState
	if err := json.Unmarshal([]byte(data), &gameState); err != nil {
		return nil, fmt.Errorf("error deserializando estado del juego: %w", err)
	}

	return &gameState, nil
}

func (gs *GameStateService) StartGame() error {
	now := time.Now()
	gameState := &models.GameState{
		IsActive:  true,
		StartTime: &now,
		EndTime:   nil,
		Message:   "Partida activa - Los jugadores pueden ingresar",
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
