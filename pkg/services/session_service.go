package services

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/backsoul/quiz/pkg/models"
	"github.com/backsoul/quiz/pkg/redis"
	"github.com/google/uuid"
)

// SessionService maneja las sesiones de los jugadores
type SessionService struct {
	redisClient *redis.RedisClient
}

// NewSessionService crea una nueva instancia del servicio de sesiones
func NewSessionService(redisClient *redis.RedisClient) *SessionService {
	return &SessionService{
		redisClient: redisClient,
	}
}

// CreateSession crea una nueva sesión para un jugador
func (s *SessionService) CreateSession(playerName string) (*models.GameSession, error) {
	// Verificar si ya existe una sesión activa para este jugador
	existingSession, err := s.GetActiveSessionByPlayer(playerName)
	if err == nil && existingSession != nil {
		log.Printf("🔄 Jugador %s ya tiene una sesión activa, continuando...", playerName)
		return existingSession, nil
	}

	// Crear nueva sesión
	sessionID := uuid.New().String()
	session := &models.GameSession{
		ID:                sessionID,
		PlayerName:        playerName,
		CurrentQuestion:   1,
		TotalPrize:        0,
		LifelinesUsed:     models.LifelinesState{},
		AnswersGiven:      []models.PlayerAnswer{},
		GameStatus:        "active",
		StartTime:         time.Now(),
		LastActivity:      time.Now(),
		CurrentQuestionID: 0,
	}

	// Guardar en Redis
	if err := s.saveSession(session); err != nil {
		return nil, fmt.Errorf("error guardando sesión: %v", err)
	}

	// Agregar a la lista de sesiones activas
	if err := s.addToActiveSessions(sessionID); err != nil {
		log.Printf("⚠️ Error agregando a sesiones activas: %v", err)
	}

	// Agregar a las sesiones del jugador
	if err := s.addToPlayerSessions(playerName, sessionID); err != nil {
		log.Printf("⚠️ Error agregando a sesiones del jugador: %v", err)
	}

	log.Printf("✅ Nueva sesión creada para %s (ID: %s)", playerName, sessionID)
	return session, nil
}

// GetSession obtiene una sesión por ID
func (s *SessionService) GetSession(sessionID string) (*models.GameSession, error) {
	sessionJSON, err := s.redisClient.Get(fmt.Sprintf("quiz:session:%s", sessionID))
	if err != nil {
		return nil, fmt.Errorf("sesión no encontrada: %v", err)
	}

	var session models.GameSession
	if err := json.Unmarshal([]byte(sessionJSON), &session); err != nil {
		return nil, fmt.Errorf("error parsing sesión: %v", err)
	}

	return &session, nil
}

// GetActiveSessionByPlayer obtiene la sesión activa de un jugador
func (s *SessionService) GetActiveSessionByPlayer(playerName string) (*models.GameSession, error) {
	// Obtener las sesiones del jugador
	sessionIDs, err := s.getPlayerSessions(playerName)
	if err != nil {
		return nil, err
	}

	// Buscar una sesión activa
	for _, sessionID := range sessionIDs {
		session, err := s.GetSession(sessionID)
		if err != nil {
			continue
		}
		if session.GameStatus == "active" {
			return session, nil
		}
	}

	return nil, fmt.Errorf("no se encontró sesión activa para %s", playerName)
}

// UpdateSession actualiza una sesión existente
func (s *SessionService) UpdateSession(session *models.GameSession) error {
	session.LastActivity = time.Now()
	return s.saveSession(session)
}

// AddAnswer agrega una respuesta a la sesión
func (s *SessionService) AddAnswer(sessionID string, answer models.PlayerAnswer) error {
	session, err := s.GetSession(sessionID)
	if err != nil {
		return err
	}

	// Agregar la respuesta
	session.AnswersGiven = append(session.AnswersGiven, answer)

	// Actualizar pregunta actual si es correcta
	if answer.IsCorrect {
		session.CurrentQuestion++
		session.TotalPrize = answer.PrizeWon
	} else {
		// Juego terminado si es incorrecta
		session.GameStatus = "finished"
	}

	// Verificar si ganó el juego
	if session.CurrentQuestion > 15 {
		session.GameStatus = "finished"
	}

	return s.UpdateSession(session)
}

// UseLifeline marca un comodín como usado
func (s *SessionService) UseLifeline(sessionID string, lifelineType string) error {
	session, err := s.GetSession(sessionID)
	if err != nil {
		return err
	}

	switch lifelineType {
	case "fiftyFifty":
		if session.LifelinesUsed.FiftyFifty {
			return fmt.Errorf("comodín 50:50 ya fue usado")
		}
		session.LifelinesUsed.FiftyFifty = true
	case "audience":
		if session.LifelinesUsed.Audience {
			return fmt.Errorf("comodín pregunta al público ya fue usado")
		}
		session.LifelinesUsed.Audience = true
	case "phone":
		if session.LifelinesUsed.Phone {
			return fmt.Errorf("comodín llamada telefónica ya fue usado")
		}
		session.LifelinesUsed.Phone = true
	default:
		return fmt.Errorf("tipo de comodín desconocido: %s", lifelineType)
	}

	return s.UpdateSession(session)
}

// GetPlayerHistory obtiene el historial de un jugador
func (s *SessionService) GetPlayerHistory(playerName string) ([]models.GameSession, error) {
	sessionIDs, err := s.getPlayerSessions(playerName)
	if err != nil {
		return nil, err
	}

	var sessions []models.GameSession
	for _, sessionID := range sessionIDs {
		session, err := s.GetSession(sessionID)
		if err != nil {
			log.Printf("⚠️ Error obteniendo sesión %s: %v", sessionID, err)
			continue
		}
		sessions = append(sessions, *session)
	}

	return sessions, nil
}

// GetActiveSessions obtiene todas las sesiones activas
func (s *SessionService) GetActiveSessions() ([]models.GameSession, error) {
	sessionIDs, err := s.redisClient.GetSetMembers("quiz:active_sessions")
	if err != nil {
		return nil, fmt.Errorf("error obteniendo sesiones activas: %v", err)
	}

	var sessions []models.GameSession
	for _, sessionID := range sessionIDs {
		session, err := s.GetSession(sessionID)
		if err != nil {
			log.Printf("⚠️ Error obteniendo sesión activa %s: %v", sessionID, err)
			continue
		}

		// Verificar si realmente está activa
		if session.GameStatus == "active" {
			sessions = append(sessions, *session)
		} else {
			// Remover de sesiones activas si ya no está activa
			s.removeFromActiveSessions(sessionID)
		}
	}

	return sessions, nil
}

// FinishSession termina una sesión
func (s *SessionService) FinishSession(sessionID string) error {
	session, err := s.GetSession(sessionID)
	if err != nil {
		return err
	}

	session.GameStatus = "finished"
	session.LastActivity = time.Now()

	// Actualizar sesión
	if err := s.saveSession(session); err != nil {
		return err
	}

	// Remover de sesiones activas
	return s.removeFromActiveSessions(sessionID)
}

// Métodos privados auxiliares

func (s *SessionService) saveSession(session *models.GameSession) error {
	sessionJSON, err := json.Marshal(session)
	if err != nil {
		return fmt.Errorf("error serializando sesión: %v", err)
	}

	key := fmt.Sprintf("quiz:session:%s", session.ID)
	return s.redisClient.Set(key, string(sessionJSON), 24*time.Hour) // TTL de 24 horas
}

func (s *SessionService) addToActiveSessions(sessionID string) error {
	return s.redisClient.AddToSet("quiz:active_sessions", sessionID)
}

func (s *SessionService) removeFromActiveSessions(sessionID string) error {
	return s.redisClient.RemoveFromSet("quiz:active_sessions", sessionID)
}

func (s *SessionService) addToPlayerSessions(playerName, sessionID string) error {
	key := fmt.Sprintf("quiz:player_sessions:%s", playerName)
	return s.redisClient.AddToSet(key, sessionID)
}

func (s *SessionService) getPlayerSessions(playerName string) ([]string, error) {
	key := fmt.Sprintf("quiz:player_sessions:%s", playerName)
	return s.redisClient.GetSetMembers(key)
}

// GetPlayerNames obtiene todos los nombres de jugadores registrados
func (s *SessionService) GetPlayerNames() ([]string, error) {
	pattern := "quiz:player_sessions:*"
	keys, err := s.redisClient.GetKeysByPattern(pattern)
	if err != nil {
		return nil, fmt.Errorf("error obteniendo nombres de jugadores: %v", err)
	}

	var playerNames []string
	for _, key := range keys {
		// Extraer el nombre del jugador de la clave
		playerName := key[len("quiz:player_sessions:"):]
		playerNames = append(playerNames, playerName)
	}

	return playerNames, nil
}

// GetLeaderboard obtiene la tabla de posiciones
func (s *SessionService) GetLeaderboard() (*models.LeaderboardResponse, error) {
	// Obtener todas las sesiones activas y terminadas recientes
	allSessions, err := s.getAllRecentSessions()
	if err != nil {
		return nil, fmt.Errorf("error obteniendo sesiones: %v", err)
	}

	// Crear entradas de la tabla de posiciones
	var leaderboard []models.LeaderboardEntry
	avatars := []string{"🎯", "⭐", "🔥", "💎", "🌟", "🎪", "🚀", "👤", "🎨", "🎵", "🌊", "⚡", "🎭", "🦄", "🔮"}

	activePlayers := 0
	for i, session := range allSessions {
		if session.GameStatus == "active" {
			activePlayers++
		}

		// Asignar avatar basado en el índice
		avatar := avatars[i%len(avatars)]

		entry := models.LeaderboardEntry{
			Position:     i + 1,
			PlayerName:   session.PlayerName,
			CurrentPrize: session.TotalPrize,
			Status:       session.GameStatus,
			Avatar:       avatar,
			Question:     session.CurrentQuestion,
		}

		leaderboard = append(leaderboard, entry)
	}

	response := &models.LeaderboardResponse{
		Leaderboard:   leaderboard,
		TotalPlayers:  len(allSessions),
		ActivePlayers: activePlayers,
	}

	return response, nil
}

// getAllRecentSessions obtiene todas las sesiones ordenadas por premio
func (s *SessionService) getAllRecentSessions() ([]models.GameSession, error) {
	// Obtener todas las sesiones activas
	activeSessions, err := s.GetActiveSessions()
	if err != nil {
		return nil, err
	}

	// Obtener también algunas sesiones terminadas recientes
	finishedSessions, err := s.getRecentFinishedSessions()
	if err != nil {
		log.Printf("⚠️ Error obteniendo sesiones terminadas: %v", err)
		finishedSessions = []models.GameSession{}
	}

	// Combinar y ordenar por premio (mayor a menor)
	allSessions := append(activeSessions, finishedSessions...)

	// Ordenar por premio total (descendente)
	for i := 0; i < len(allSessions)-1; i++ {
		for j := i + 1; j < len(allSessions); j++ {
			if allSessions[i].TotalPrize < allSessions[j].TotalPrize {
				allSessions[i], allSessions[j] = allSessions[j], allSessions[i]
			}
		}
	}

	// Limitar a los primeros 20 para no sobrecargar
	if len(allSessions) > 20 {
		allSessions = allSessions[:20]
	}

	return allSessions, nil
}

// getRecentFinishedSessions obtiene sesiones terminadas recientes
func (s *SessionService) getRecentFinishedSessions() ([]models.GameSession, error) {
	// Obtener todas las claves de sesiones
	pattern := "quiz:session:*"
	keys, err := s.redisClient.GetKeysByPattern(pattern)
	if err != nil {
		return nil, err
	}

	var finishedSessions []models.GameSession

	// Revisar cada sesión para encontrar las terminadas
	for _, key := range keys {
		sessionID := key[len("quiz:session:"):]
		session, err := s.GetSession(sessionID)
		if err != nil {
			continue
		}

		if session.GameStatus == "finished" {
			finishedSessions = append(finishedSessions, *session)
		}
	}

	// Limitar a las 10 más recientes
	if len(finishedSessions) > 10 {
		finishedSessions = finishedSessions[:10]
	}

	return finishedSessions, nil
}
