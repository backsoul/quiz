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
	hub              *websocketHub.Hub
}

func NewGameControlHandler(gameStateService *services.GameStateService, hub *websocketHub.Hub) *GameControlHandler {
	return &GameControlHandler{
		gameStateService: gameStateService,
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
			// Por ahora, solo escuchamos sin procesar mensajes del cliente
		}
	})

	if err != nil {
		log.Printf("Error upgrading to WebSocket: %v", err)
		ctx.Error("Error upgrading to WebSocket", fasthttp.StatusInternalServerError)
	}
}

// StartGame inicia una nueva partida
func (gc *GameControlHandler) StartGame(ctx *fasthttp.RequestCtx) {
	// Verificar si ya hay una partida activa
	gameState, err := gc.gameStateService.GetGameState()
	if err != nil {
		gc.respondWithError(ctx, fasthttp.StatusInternalServerError, "Error obteniendo estado del juego")
		return
	}

	if gameState.IsActive {
		gc.respondWithError(ctx, fasthttp.StatusBadRequest, "Ya hay una partida activa")
		return
	}

	// Iniciar nueva partida
	err = gc.gameStateService.StartGame()
	if err != nil {
		gc.respondWithError(ctx, fasthttp.StatusInternalServerError, "Error iniciando partida")
		return
	}

	// Notificar a todos los clientes conectados
	gc.hub.BroadcastGameState(true, "Partida iniciada - Los jugadores pueden ingresar")

	// Respuesta exitosa
	gc.respondWithSuccess(ctx, map[string]interface{}{
		"timestamp": time.Now().Format(time.RFC3339),
	}, "Partida iniciada exitosamente")

	log.Println("🟢 Partida iniciada desde el panel de administración")
}

// EndGame termina la partida actual
func (gc *GameControlHandler) EndGame(ctx *fasthttp.RequestCtx) {
	// Verificar si hay una partida activa
	gameState, err := gc.gameStateService.GetGameState()
	if err != nil {
		gc.respondWithError(ctx, fasthttp.StatusInternalServerError, "Error obteniendo estado del juego")
		return
	}

	if !gameState.IsActive {
		gc.respondWithError(ctx, fasthttp.StatusBadRequest, "No hay partida activa para terminar")
		return
	}

	// Terminar partida
	err = gc.gameStateService.EndGame()
	if err != nil {
		gc.respondWithError(ctx, fasthttp.StatusInternalServerError, "Error terminando partida")
		return
	}

	// Notificar a todos los clientes conectados
	gc.hub.BroadcastGameState(false, "Partida terminada - Los jugadores no pueden ingresar")

	// Respuesta exitosa
	gc.respondWithSuccess(ctx, map[string]interface{}{
		"timestamp": time.Now().Format(time.RFC3339),
	}, "Partida terminada exitosamente")

	log.Println("🔴 Partida terminada desde el panel de administración")
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

// Métodos de utilidad para respuestas HTTP
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
