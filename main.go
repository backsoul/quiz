package main

import (
	"encoding/json"
	"log"
	"os"
	"strings"
	"time"

	"github.com/backsoul/quiz/pkg/handlers"
	"github.com/backsoul/quiz/pkg/models"
	"github.com/backsoul/quiz/pkg/redis"
	"github.com/backsoul/quiz/pkg/services"
	hubpkg "github.com/backsoul/quiz/pkg/websocket"
	ws "github.com/fasthttp/websocket"
	"github.com/valyala/fasthttp"
)

// Globals
var sessionService *services.SessionService
var sessionHandler *handlers.SessionHandler
var gameControlHandler *handlers.GameControlHandler
var hub *hubpkg.Hub

func main() {
	// Redis setup
	redisAddr := os.Getenv("REDIS_ADDR")
	if redisAddr == "" {
		redisAddr = "localhost:6379"
	}
	log.Printf("Connecting to Redis %s", redisAddr)
	redisClient := redis.NewRedisClient(redisAddr, "", 0)
	defer redisClient.Close()

	// Load questions from file
	questions, err := loadQuestions("answers.json")
	if err != nil {
		log.Fatalf("Error loading questions: %v", err)
	}
	log.Printf("Loaded %d questions", len(questions))

	// Services
	questionService := services.NewQuestionService(redisClient)
	sessionService = services.NewSessionService(redisClient)
	gameStateService := services.NewGameStateService(redisClient)
	
	// Inyectar dependencia para calcular pregunta actual dinámicamente
	gameStateService.SetSessionService(sessionService)
	
	// Populate Redis
	if err := questionService.LoadQuestionsFromFile("answers.json"); err != nil {
		log.Printf("Warn loading to redis: %v", err)
	}

	// WebSocket hub & handlers
	hub = hubpkg.NewHub()
	go hub.Run()
	sessionHandler = handlers.NewSessionHandler(sessionService, questionService, hub)
	gameControlHandler = handlers.NewGameControlHandler(gameStateService, sessionService, hub)

	// Broadcaster
	go func() {
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()
		for range ticker.C {
			sessions, err := sessionService.GetActiveSessions()
			if err != nil {
				continue
			}
			hub.BroadcastMessage("sessions", sessions)
		}
	}()

	// Server
	server := &fasthttp.Server{Handler: requestRouter}
	log.Fatal(server.ListenAndServe(":8080"))
}

func requestRouter(ctx *fasthttp.RequestCtx) {
	path := string(ctx.Path())
	method := string(ctx.Method())

	// Game API: crear sesión
	if method == "POST" && path == "/api/sessions" {
		sessionHandler.CreateSession(ctx)
		return
	}

	// Game API: obtener sesión específica
	if method == "GET" && strings.HasPrefix(path, "/api/sessions/") && !strings.HasSuffix(path, "/answer") && !strings.HasSuffix(path, "/lifeline") {
		parts := strings.Split(path, "/")
		if len(parts) == 4 {
			ctx.SetUserValue("id", parts[3])
			sessionHandler.GetSession(ctx)
			return
		}
	}

	// Game API: answer/lifeline
	if method == "POST" && strings.HasPrefix(path, "/api/sessions/") {
		parts := strings.Split(path, "/")
		if len(parts) == 5 && parts[4] == "answer" {
			ctx.SetUserValue("id", parts[3])
			sessionHandler.SubmitAnswer(ctx)
			return
		}
		if len(parts) == 5 && parts[4] == "lifeline" {
			ctx.SetUserValue("id", parts[3])
			sessionHandler.UseLifeline(ctx)
			return
		}
	}

	// Game Control API (Admin endpoints)
	if method == "POST" && path == "/api/game/start" {
		gameControlHandler.StartGame(ctx)
		return
	}
	if method == "POST" && path == "/api/game/end" {
		gameControlHandler.EndGame(ctx)
		return
	}
	if method == "POST" && path == "/api/game/next-question" {
		gameControlHandler.NextQuestion(ctx)
		return
	}
	if method == "POST" && path == "/api/game/reveal-answer" {
		gameControlHandler.RevealAnswer(ctx)
		return
	}
	if method == "GET" && path == "/api/game/state" {
		gameControlHandler.GetGameState(ctx)
		return
	}
	// WebSocket endpoint
	if method == "GET" && path == "/ws" {
		upgrader := ws.FastHTTPUpgrader{CheckOrigin: func(ctx *fasthttp.RequestCtx) bool { return true }}
		upgrader.Upgrade(ctx, func(conn *ws.Conn) {
			hub.Register(conn)
			defer hub.Unregister(conn)
			for {
				if _, _, err := conn.ReadMessage(); err != nil {
					break
				}
			}
		})
		return
	}
	// Static routes
	switch path {
	case "/":
		serveFile(ctx, "index.html", "text/html")
		return
	case "/shared.css":
		serveFile(ctx, "shared.css", "text/css")
		return
	case "/admin", "/admin.html":
		serveFile(ctx, "admin.html", "text/html")
		return
	case "/test-data-persistence", "/test-data-persistence.html":
		serveFile(ctx, "test-data-persistence.html", "text/html")
		return
	}
	// Questions API
	if method == "GET" && path == "/api/questions" {
		serveQuestionsFromFile(ctx)
		return
	}
	// Admin sessions
	if method == "GET" && path == "/api/admin/sessions" {
		sessions, err := sessionService.GetActiveSessions()
		if err != nil {
			ctx.Error(err.Error(), fasthttp.StatusInternalServerError)
			return
		}
		data, _ := json.Marshal(sessions)
		ctx.SetContentType("application/json")
		ctx.SetBody(data)
		return
	}
	// Health
	if method == "GET" && path == "/api/health" {
		ctx.SetContentType("application/json")
		ctx.SetBody([]byte(`{"status":"ok"}`))
		return
	}
	// Fallback
	ctx.Error("Not found", fasthttp.StatusNotFound)
}

func serveFile(ctx *fasthttp.RequestCtx, filename, contentType string) {
	if _, err := os.Stat(filename); os.IsNotExist(err) {
		ctx.Error("File not found", fasthttp.StatusNotFound)
		return
	}
	ctx.SetContentType(contentType)
	ctx.SendFile(filename)
}

func serveQuestionsFromFile(ctx *fasthttp.RequestCtx) {
	data, err := os.ReadFile("answers.json")
	if err != nil {
		ctx.Error("Error reading questions", fasthttp.StatusInternalServerError)
		return
	}
	ctx.SetContentType("application/json")
	ctx.SetBody(data)
}

func loadQuestions(filename string) ([]models.Question, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	var qd struct {
		Questions []models.Question `json:"questions"`
	}
	if err := json.Unmarshal(data, &qd); err != nil {
		return nil, err
	}
	return qd.Questions, nil
}
