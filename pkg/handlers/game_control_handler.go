package handlers

import (
	"encoding/json"
	"log"
	"time"

	"github.com/backsoul/quiz/pkg/models"
	"github.com/backsoul/quiz/pkg/services"
	websocketHub "github.com/backsoul/quiz/pkg/websocket"
	"github.com/fasthttp/websocket"
	"github.com/valyala/fasthttp"
)

type GameControlHandler struct {
	gameStateService *services.GameStateService
	sessionService   *services.SessionService
	hub              *websocketHub.Hub
}

func NewGameControlHandler(gameStateService *services.GameStateService, sessionService *services.SessionService, hub *websocketHub.Hub) *GameControlHandler {
	return &GameControlHandler{
		gameStateService: gameStateService,
		sessionService:   sessionService,
		hub:              hub,
	}
}

var upgrader = websocket.FastHTTPUpgrader{
	CheckOrigin: func(ctx *fasthttp.RequestCtx) bool {
		return true // Permitir conexiones desde cualquier origen en desarrollo
	},
}

// HandleWebSocket maneja las conexiones WebSocket
func (gc *GameControlHandler) HandleWebSocket(ctx *fasthttp.RequestCtx) {
	err := upgrader.Upgrade(ctx, func(ws *websocket.Conn) {
		defer ws.Close()

		gc.hub.Register(ws)
		defer gc.hub.Unregister(ws)

		// Enviar estado actual del juego al conectarse
		gameState, err := gc.gameStateService.GetGameState()
		if err == nil {
			message := websocketHub.Message{
				Type: "gameState",
				Data: gameState,
			}
			data, _ := json.Marshal(message)
			ws.WriteMessage(websocket.TextMessage, data)
		}

		// Escuchar mensajes del cliente
		for {
			_, _, err := ws.ReadMessage()
			if err != nil {
				log.Printf("Error leyendo mensaje WebSocket: %v", err)
				break
			}
		}
	})

	if err != nil {
		log.Printf("Error upgrading to WebSocket: %v", err)
		ctx.Error("Error upgrading to WebSocket", fasthttp.StatusInternalServerError)
	}
}

// StartGame inicia una nueva partida
func (gc *GameControlHandler) StartGame(ctx *fasthttp.RequestCtx) {
	gameState, err := gc.gameStateService.GetGameState()
	if err != nil {
		gc.respondWithError(ctx, fasthttp.StatusInternalServerError, "Error obteniendo estado del juego")
		return
	}

	if gameState.IsActive {
		gc.respondWithError(ctx, fasthttp.StatusBadRequest, "Ya hay una partida activa")
		return
	}

	err = gc.gameStateService.StartGame()
	if err != nil {
		gc.respondWithError(ctx, fasthttp.StatusInternalServerError, "Error iniciando partida")
		return
	}

	gc.hub.BroadcastGameState(true, "Partida iniciada - Los jugadores pueden ingresar")

	gc.respondWithSuccess(ctx, map[string]interface{}{
		"timestamp": time.Now().Format(time.RFC3339),
	}, "Partida iniciada exitosamente")

	log.Println("🟢 Partida iniciada desde el panel de administración")
}

// EndGame termina la partida actual
func (gc *GameControlHandler) EndGame(ctx *fasthttp.RequestCtx) {
	gameState, err := gc.gameStateService.GetGameState()
	if err != nil {
		gc.respondWithError(ctx, fasthttp.StatusInternalServerError, "Error obteniendo estado del juego")
		return
	}

	if !gameState.IsActive {
		gc.respondWithError(ctx, fasthttp.StatusBadRequest, "No hay partida activa para terminar")
		return
	}

	// Obtener estadísticas antes de limpiar para el reporte final
	activeSessions, _ := gc.sessionService.GetActiveSessions()
	totalPlayers := len(activeSessions)
	
	// Terminar el juego
	err = gc.gameStateService.EndGame()
	if err != nil {
		gc.respondWithError(ctx, fasthttp.StatusInternalServerError, "Error terminando partida")
		return
	}

	// Notificar a todos los jugadores que la partida ha terminado ANTES de limpiar datos
	gc.hub.BroadcastMessage("gameEnded", map[string]interface{}{
		"timestamp":    time.Now().Format(time.RFC3339),
		"message":      "La partida ha terminado. Todos los datos serán limpiados.",
		"totalPlayers": totalPlayers,
	})

	// Esperar un momento para que el mensaje llegue a todos los clientes
	time.Sleep(1 * time.Second)

	// Limpiar todas las sesiones y datos de la partida
	err = gc.sessionService.ClearAllSessions()
	if err != nil {
		log.Printf("⚠️ Error limpiando sesiones: %v", err)
		gc.respondWithError(ctx, fasthttp.StatusInternalServerError, "Error limpiando datos de la partida")
		return
	}

	// Notificar estado final después de la limpieza
	gc.hub.BroadcastGameState(false, "Partida terminada - Todos los datos han sido limpiados")

	gc.respondWithSuccess(ctx, map[string]interface{}{
		"timestamp":    time.Now().Format(time.RFC3339),
		"totalPlayers": totalPlayers,
		"dataCleared":  true,
	}, "Partida terminada exitosamente y datos limpiados")

	log.Printf("🔴 Partida terminada y datos de %d jugadores limpiados desde el panel de administración", totalPlayers)
}

// GetGameState devuelve el estado actual del juego
func (gc *GameControlHandler) GetGameState(ctx *fasthttp.RequestCtx) {
	gameState, err := gc.gameStateService.GetGameState()
	if err != nil {
		gc.respondWithError(ctx, fasthttp.StatusInternalServerError, "Error obteniendo estado del juego")
		return
	}

	gc.respondWithSuccess(ctx, map[string]interface{}{
		"gameState": gameState,
	}, "Estado del juego obtenido exitosamente")
}

// NextQuestion avanza a la siguiente pregunta para todos los jugadores
func (gc *GameControlHandler) NextQuestion(ctx *fasthttp.RequestCtx) {
	gameState, err := gc.gameStateService.GetGameState()
	if err != nil {
		gc.respondWithError(ctx, fasthttp.StatusInternalServerError, "Error obteniendo estado del juego")
		return
	}

	if !gameState.IsActive {
		gc.respondWithError(ctx, fasthttp.StatusBadRequest, "No hay partida activa")
		return
	}

	// Enviar comando via WebSocket para que todos los jugadores avancen
	gc.hub.BroadcastMessage("nextQuestion", map[string]interface{}{
		"timestamp": time.Now().Format(time.RFC3339),
		"message":   "El administrador ha avanzado a la siguiente pregunta",
	})

	gc.respondWithSuccess(ctx, map[string]interface{}{
		"timestamp": time.Now().Format(time.RFC3339),
	}, "Comando enviado para avanzar a la siguiente pregunta")

	log.Println("➡️ Administrador ha forzado el avance a la siguiente pregunta")
}

// RevealAnswer revela la respuesta correcta a todos los jugadores
func (gc *GameControlHandler) RevealAnswer(ctx *fasthttp.RequestCtx) {
	gameState, err := gc.gameStateService.GetGameState()
	if err != nil {
		gc.respondWithError(ctx, fasthttp.StatusInternalServerError, "Error obteniendo estado del juego")
		return
	}

	if !gameState.IsActive {
		gc.respondWithError(ctx, fasthttp.StatusBadRequest, "No hay partida activa")
		return
	}

	// Enviar comando via WebSocket para revelar la respuesta
	gc.hub.BroadcastMessage("revealAnswer", map[string]interface{}{
		"timestamp": time.Now().Format(time.RFC3339),
		"message":   "El administrador ha revelado la respuesta correcta",
	})

	gc.respondWithSuccess(ctx, map[string]interface{}{
		"timestamp": time.Now().Format(time.RFC3339),
	}, "Comando enviado para revelar la respuesta correcta")

	log.Println("💡 Administrador ha revelado la respuesta correcta")
}

func (gc *GameControlHandler) respondWithError(ctx *fasthttp.RequestCtx, statusCode int, message string) {
	response := models.APIResponse{
		Success: false,
		Error:   message,
	}
	gc.respondWithJSON(ctx, statusCode, response)
}

func (gc *GameControlHandler) respondWithSuccess(ctx *fasthttp.RequestCtx, data interface{}, message string) {
	response := models.APIResponse{
		Success: true,
		Message: message,
		Data:    data,
	}
	gc.respondWithJSON(ctx, fasthttp.StatusOK, response)
}

func (gc *GameControlHandler) respondWithJSON(ctx *fasthttp.RequestCtx, statusCode int, data interface{}) {
	ctx.SetStatusCode(statusCode)
	ctx.SetContentType("application/json")
	json.NewEncoder(ctx).Encode(data)
}
