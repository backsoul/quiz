package models

import "time"

type GameState struct {
	IsActive        bool       `json:"isActive"`
	StartTime       *time.Time `json:"startTime,omitempty"`
	EndTime         *time.Time `json:"endTime,omitempty"`
	Message         string     `json:"message"`
	PlayerCount     int        `json:"playerCount"`
	CurrentQuestion int        `json:"currentQuestion"` // Pregunta más alta alcanzada por algún jugador
	MaxQuestions    int        `json:"maxQuestions"`    // Total de preguntas disponibles
}

type GameControl struct {
	Action  string `json:"action"` // "start" o "end"
	AdminID string `json:"adminId,omitempty"`
}
