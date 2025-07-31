package main

import (
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/backsoul/quiz/pkg/handlers"
	"github.com/backsoul/quiz/pkg/redis"
	"github.com/backsoul/quiz/pkg/services"
	"github.com/backsoul/quiz/pkg/websocket"
	"github.com/valyala/fasthttp"
)

var (
	redisClient        *redis.RedisClient
	questionService    *services.QuestionService
	sessionService     *services.SessionService
	gameStateService   *services.GameStateService
	questionHandler    *handlers.QuestionHandler
	sessionHandler     *handlers.SessionHandler
	gameControlHandler *handlers.GameControlHandler
	hub                *websocket.Hub
)

func main() {
	// Inicializar Redis
	log.Println("🚀 Iniciando servidor ¿Quién Quiere Ser Millonario?")
	initRedis()

	// Inicializar servicios
	initServices()

	// Cargar preguntas al inicio
	loadInitialQuestions()

	// Configurar el servidor
	server := &fasthttp.Server{
		Handler: requestHandler,
		Name:    "Quiz Server",
	}

	log.Println("🎮 Servidor ¿Quién Quiere Ser Millonario? iniciado")
	log.Println("📱 Juego principal: http://localhost:8080")
	log.Println("🎛️  Panel Admin: http://localhost:8080/admin")
	log.Println("🔧 API Health: http://localhost:8080/api/health")
	log.Println("📊 API Preguntas: http://localhost:8080/api/questions")
	log.Println("🔄 Presiona Ctrl+C para detener el servidor")

	// Iniciar el servidor en el puerto 8080
	if err := server.ListenAndServe(":8080"); err != nil {
		log.Fatalf("Error al iniciar el servidor: %v", err)
	}
}

func initRedis() {
	// Configuración de Redis (puedes usar variables de entorno)
	redisAddr := getEnv("REDIS_ADDR", "localhost:6379")
	redisPassword := getEnv("REDIS_PASSWORD", "")
	redisDB := 0

	log.Printf("🔌 Conectando a Redis en %s...", redisAddr)
	redisClient = redis.NewRedisClient(redisAddr, redisPassword, redisDB)
}

func initServices() {
	log.Println("⚙️  Inicializando servicios...")
	questionService = services.NewQuestionService(redisClient)
	sessionService = services.NewSessionService(redisClient)
	gameStateService = services.NewGameStateService(redisClient)

	// Inicializar WebSocket Hub
	hub = websocket.NewHub()
	go hub.Run()

	// Inicializar handlers
	questionHandler = handlers.NewQuestionHandler(questionService)
	sessionHandler = handlers.NewSessionHandler(sessionService, questionService, gameStateService, hub)
	gameControlHandler = handlers.NewGameControlHandler(gameStateService, hub)
}

func loadInitialQuestions() {
	log.Println("📚 Cargando preguntas iniciales...")

	// Verificar si ya hay preguntas en Redis
	count, err := questionService.GetQuestionCount()
	if err == nil && count > 0 {
		log.Printf("✅ Ya hay %d preguntas en Redis", count)
		return
	}

	// Cargar preguntas desde el archivo JSON
	if err := questionService.LoadQuestionsFromFile("answers.json"); err != nil {
		log.Printf("⚠️ Error cargando preguntas iniciales: %v", err)
		log.Println("💡 El servidor continuará funcionando. Puedes cargar preguntas usando POST /api/questions/reload")
	} else {
		newCount, _ := questionService.GetQuestionCount()
		log.Printf("✅ %d preguntas cargadas exitosamente", newCount)
	}
}

func requestHandler(ctx *fasthttp.RequestCtx) {
	// Obtener la ruta solicitada
	path := string(ctx.Path())
	method := string(ctx.Method())

	// Log de la petición
	log.Printf("📡 %s %s", method, path)

	// Configurar headers de respuesta
	ctx.Response.Header.Set("Server", "Quiz-FastHTTP/1.0")
	ctx.Response.Header.Set("Cache-Control", "no-cache")

	// Headers CORS para desarrollo
	ctx.Response.Header.Set("Access-Control-Allow-Origin", "*")
	ctx.Response.Header.Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
	ctx.Response.Header.Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

	// Manejar preflight requests
	if method == "OPTIONS" {
		ctx.SetStatusCode(fasthttp.StatusOK)
		return
	}

	// Enrutamiento
	switch {
	// Páginas principales
	case path == "/":
		serveFile(ctx, "index.html")
	case path == "/admin":
		serveFile(ctx, "admin.html")
	case path == "/favicon.ico":
		ctx.SetStatusCode(fasthttp.StatusNotFound)
		ctx.SetBodyString("🎮")

	// API Routes - Health
	case path == "/api/health":
		questionHandler.HealthCheck(ctx)

	// API Routes - Questions
	case path == "/api/questions" && method == "GET":
		questionHandler.GetAllQuestions(ctx)
	case path == "/api/questions/random" && method == "GET":
		questionHandler.GetRandomQuestion(ctx)
	case path == "/api/questions/difficulty" && method == "GET":
		questionHandler.GetQuestionsByDifficulty(ctx)
	case path == "/api/questions/random/difficulty" && method == "GET":
		questionHandler.GetRandomQuestionByDifficulty(ctx)
	case path == "/api/questions/metadata" && method == "GET":
		questionHandler.GetQuestionMetadata(ctx)
	case path == "/api/questions/reload" && method == "POST":
		questionHandler.ReloadQuestions(ctx)

	// API Routes - Sessions
	case path == "/api/sessions" && method == "POST":
		sessionHandler.CreateSession(ctx)
	case path == "/api/sessions/active" && method == "GET":
		sessionHandler.GetActiveSessions(ctx)
	case path == "/api/sessions/players" && method == "GET":
		sessionHandler.GetPlayerNames(ctx)
	case path == "/api/leaderboard" && method == "GET":
		sessionHandler.GetLeaderboard(ctx)

	// API Routes - Game Control
	case path == "/api/game/start" && method == "POST":
		gameControlHandler.StartGame(ctx)
	case path == "/api/game/end" && method == "POST":
		gameControlHandler.EndGame(ctx)
	case path == "/api/game/state" && method == "GET":
		gameControlHandler.GetGameState(ctx)
	case path == "/api/game/next-question" && method == "POST":
		gameControlHandler.NextQuestion(ctx)
	case path == "/api/game/reveal-answer" && method == "POST":
		gameControlHandler.RevealAnswer(ctx)

	// WebSocket Route
	case path == "/ws":
		gameControlHandler.HandleWebSocket(ctx)

	// API Routes - Individual Questions and Sessions (with parameters)
	case strings.HasPrefix(path, "/api/questions/") && method == "GET":
		// Manejar /api/questions/{id}
		parts := strings.Split(path, "/")
		if len(parts) == 4 && parts[1] == "api" && parts[2] == "questions" {
			ctx.SetUserValue("id", parts[3])
			questionHandler.GetQuestion(ctx)
		} else {
			serve404(ctx)
		}
	case strings.HasPrefix(path, "/api/sessions/") && method == "GET":
		handleSessionGetRoutes(ctx, path)
	case strings.HasPrefix(path, "/api/sessions/") && method == "POST":
		handleSessionPostRoutes(ctx, path)

	default:
		serve404(ctx)
	}
}

func serveFile(ctx *fasthttp.RequestCtx, filename string) {
	// Obtener la ruta absoluta del archivo
	filePath := filepath.Join(".", filename)

	// Verificar si el archivo existe
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		ctx.SetStatusCode(fasthttp.StatusNotFound)
		ctx.SetContentType("text/html; charset=utf-8")
		ctx.SetBodyString(`
			<!DOCTYPE html>
			<html>
			<head>
				<title>Archivo no encontrado</title>
				<style>
					body { 
						font-family: Arial, sans-serif; 
						background: linear-gradient(135deg, #0f0f0f 0%, #1a1a2e 50%, #16213e 100%);
						color: white; 
						text-align: center; 
						padding: 50px;
						margin: 0;
						min-height: 100vh;
						display: flex;
						flex-direction: column;
						justify-content: center;
						align-items: center;
					}
					h1 { 
						font-size: 2.5rem; 
						margin-bottom: 20px;
						color: #f44336;
					}
					p { font-size: 1.1rem; color: #ccc; }
				</style>
			</head>
			<body>
				<h1>⚠️ Archivo no encontrado</h1>
				<p>El archivo <strong>` + filename + `</strong> no existe en el servidor.</p>
				<p>Asegúrate de que el archivo esté en el directorio correcto.</p>
			</body>
			</html>
		`)
		return
	}

	// Configurar el content-type basado en la extensión
	if filepath.Ext(filename) == ".html" {
		ctx.SetContentType("text/html; charset=utf-8")
	}

	// Servir el archivo
	fasthttp.ServeFile(ctx, filePath)

	// Log exitoso
	log.Printf("✅ Archivo servido: %s", filename)
}

func serve404(ctx *fasthttp.RequestCtx) {
	ctx.SetStatusCode(fasthttp.StatusNotFound)
	ctx.SetContentType("text/html; charset=utf-8")
	ctx.SetBodyString(`
		<!DOCTYPE html>
		<html>
		<head>
			<title>404 - Página no encontrada</title>
			<style>
				body { 
					font-family: Arial, sans-serif; 
					background: linear-gradient(135deg, #0f0f0f 0%, #1a1a2e 50%, #16213e 100%);
					color: white; 
					text-align: center; 
					padding: 50px;
					margin: 0;
					min-height: 100vh;
					display: flex;
					flex-direction: column;
					justify-content: center;
					align-items: center;
				}
				h1 { 
					font-size: 3rem; 
					margin-bottom: 20px;
					background: linear-gradient(45deg, #ffd700, #ffed4e);
					-webkit-background-clip: text;
					background-clip: text;
					-webkit-text-fill-color: transparent;
				}
				p { font-size: 1.2rem; margin-bottom: 30px; color: #ccc; }
				a { 
					color: #ffd700; 
					text-decoration: none;
					background: rgba(255, 215, 0, 0.1);
					padding: 10px 20px;
					border-radius: 25px;
					border: 2px solid #ffd700;
					transition: all 0.3s ease;
					display: inline-block;
					margin: 0 10px;
				}
				a:hover { 
					background: #ffd700;
					color: #000;
					transform: scale(1.05);
				}
				.api-info {
					background: rgba(255, 255, 255, 0.1);
					border-radius: 10px;
					padding: 20px;
					margin-top: 20px;
					text-align: left;
				}
				.endpoint {
					background: rgba(0, 0, 0, 0.3);
					padding: 5px 10px;
					border-radius: 5px;
					margin: 5px 0;
					font-family: monospace;
				}
			</style>
		</head>
		<body>
			<h1>🎮 404 - Página no encontrada</h1>
			<p>La página que buscas no existe en este servidor.</p>
			<div>
				<a href="/">🏠 Ir al Juego</a>
				<a href="/admin">🎛️ Panel Admin</a>
			</div>
			<div class="api-info">
				<h3>🔧 Endpoints API disponibles:</h3>
				<h4>📊 Preguntas:</h4>
				<div class="endpoint">GET /api/health</div>
				<div class="endpoint">GET /api/questions</div>
				<div class="endpoint">GET /api/questions/{id}</div>
				<div class="endpoint">GET /api/questions/random</div>
				<div class="endpoint">GET /api/questions/difficulty?min=1&max=5</div>
				<div class="endpoint">GET /api/questions/random/difficulty?min=1&max=5</div>
				<div class="endpoint">GET /api/questions/metadata</div>
				<div class="endpoint">POST /api/questions/reload</div>
				<h4>👤 Sesiones:</h4>
				<div class="endpoint">POST /api/sessions</div>
				<div class="endpoint">GET /api/sessions/{id}</div>
				<div class="endpoint">GET /api/sessions/active</div>
				<div class="endpoint">GET /api/sessions/players</div>
				<div class="endpoint">GET /api/sessions/player/{playerName}</div>
				<div class="endpoint">GET /api/sessions/player/{playerName}/history</div>
				<div class="endpoint">POST /api/sessions/{id}/answer</div>
				<div class="endpoint">POST /api/sessions/{id}/lifeline</div>
				<div class="endpoint">POST /api/sessions/{id}/finish</div>
				<div class="endpoint">GET /api/leaderboard</div>
			</div>
		</body>
		</html>
	`)
}

func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}

func handleSessionGetRoutes(ctx *fasthttp.RequestCtx, path string) {
	parts := strings.Split(path, "/")

	// /api/sessions/player/{playerName}
	if len(parts) == 5 && parts[1] == "api" && parts[2] == "sessions" && parts[3] == "player" {
		ctx.SetUserValue("playerName", parts[4])
		sessionHandler.GetPlayerSession(ctx)
		return
	}

	// /api/sessions/player/{playerName}/history
	if len(parts) == 6 && parts[1] == "api" && parts[2] == "sessions" && parts[3] == "player" && parts[5] == "history" {
		ctx.SetUserValue("playerName", parts[4])
		sessionHandler.GetPlayerHistory(ctx)
		return
	}

	// /api/sessions/{id}
	if len(parts) == 4 && parts[1] == "api" && parts[2] == "sessions" {
		ctx.SetUserValue("id", parts[3])
		sessionHandler.GetSession(ctx)
		return
	}

	serve404(ctx)
}

func handleSessionPostRoutes(ctx *fasthttp.RequestCtx, path string) {
	parts := strings.Split(path, "/")

	// /api/sessions/{id}/answer
	if len(parts) == 5 && parts[1] == "api" && parts[2] == "sessions" && parts[4] == "answer" {
		ctx.SetUserValue("id", parts[3])
		sessionHandler.SubmitAnswer(ctx)
		return
	}

	// /api/sessions/{id}/lifeline
	if len(parts) == 5 && parts[1] == "api" && parts[2] == "sessions" && parts[4] == "lifeline" {
		ctx.SetUserValue("id", parts[3])
		sessionHandler.UseLifeline(ctx)
		return
	}

	// /api/sessions/{id}/finish
	if len(parts) == 5 && parts[1] == "api" && parts[2] == "sessions" && parts[4] == "finish" {
		ctx.SetUserValue("id", parts[3])
		sessionHandler.FinishSession(ctx)
		return
	}

	serve404(ctx)
}
