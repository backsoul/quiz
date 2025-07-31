package models

import "time"

// GameSession representa la sesión de un jugador
type GameSession struct {
	ID                string         `json:"id"`
	PlayerName        string         `json:"playerName"`
	CurrentQuestion   int            `json:"currentQuestion"`
	TotalPrize        int            `json:"totalPrize"`
	LifelinesUsed     LifelinesState `json:"lifelinesUsed"`
	AnswersGiven      []PlayerAnswer `json:"answersGiven"`
	GameStatus        string         `json:"gameStatus"` // "active", "finished", "paused"
	StartTime         time.Time      `json:"startTime"`
	LastActivity      time.Time      `json:"lastActivity"`
	CurrentQuestionID int            `json:"currentQuestionId"`
}

// LifelinesState estado de los comodines
type LifelinesState struct {
	FiftyFifty bool `json:"fiftyFifty"`
	Audience   bool `json:"audience"`
	Phone      bool `json:"phone"`
}

// PlayerAnswer respuesta dada por el jugador
type PlayerAnswer struct {
	QuestionID       int       `json:"questionId"`
	QuestionNumber   int       `json:"questionNumber"`
	SelectedOption   string    `json:"selectedOption"`
	CorrectOption    string    `json:"correctOption"`
	IsCorrect        bool      `json:"isCorrect"`
	TimeToAnswer     int       `json:"timeToAnswer"`     // en segundos
	LifelinesUsedFor []string  `json:"lifelinesUsedFor"` // comodines usados para esta pregunta
	Timestamp        time.Time `json:"timestamp"`
	PrizeWon         int       `json:"prizeWon"`
}

// SessionCreateRequest request para crear sesión
type SessionCreateRequest struct {
	PlayerName string `json:"playerName"`
}

// SessionResponse respuesta de sesión
type SessionResponse struct {
	Session  *GameSession  `json:"session,omitempty"`
	Sessions []GameSession `json:"sessions,omitempty"`
	Message  string        `json:"message,omitempty"`
}

// PrizeLevel niveles de premios
var PrizeLevels = []int{
	100, 200, 300, 500, 1000, 2000, 4000, 8000, 16000, 32000,
	64000, 125000, 250000, 500000, 1000000,
}

// LeaderboardEntry entrada en la tabla de posiciones
type LeaderboardEntry struct {
	Position     int    `json:"position"`
	PlayerName   string `json:"playerName"`
	CurrentPrize int    `json:"currentPrize"`
	Status       string `json:"status"` // "playing", "eliminated", "finished"
	Avatar       string `json:"avatar"`
	Question     int    `json:"question"`
}

// LeaderboardResponse respuesta de la tabla de posiciones
type LeaderboardResponse struct {
	Leaderboard   []LeaderboardEntry `json:"leaderboard"`
	TotalPlayers  int                `json:"totalPlayers"`
	ActivePlayers int                `json:"activePlayers"`
}
