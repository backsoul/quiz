package main

import (
	"encoding/json"
	"log"
	"os"

	"github.com/backsoul/quiz/pkg/models"
	"github.com/backsoul/quiz/pkg/redis"
	"github.com/backsoul/quiz/pkg/services"
	"github.com/backsoul/quiz/pkg/websocket"
	"github.com/valyala/fasthttp"
)

func main() {
	// Configurar Redis (usar variable de entorno para Docker)
	redisAddr := os.Getenv("REDIS_ADDR")
	if redisAddr == "" {
		redisAddr = "localhost:6379" // Para desarrollo local
	}

	log.Printf("🔌 Conectando a Redis en %s...", redisAddr)
	redisClient := redis.NewRedisClient(redisAddr, "", 0)
	defer redisClient.Close()

	// Cargar preguntas desde archivo JSON
	questions, err := loadQuestions("answers.json")
	if err != nil {
		log.Fatalf("Error cargando preguntas: %v", err)
	}
	log.Printf("✅ Cargadas %d preguntas", len(questions))

	// Crear servicios
	questionService := services.NewQuestionService(redisClient)

	// Cargar preguntas en Redis
	err = questionService.LoadQuestionsFromFile("answers.json")
	if err != nil {
		log.Printf("⚠️ Error cargando preguntas en Redis: %v", err)
	}

	// Crear WebSocket Hub
	hub := websocket.NewHub()
	go hub.Run()

	// Configurar servidor
	server := &fasthttp.Server{
		Handler: requestRouter,
		Name:    "Quiz Server",
	}

	log.Printf("🚀 Servidor iniciado en puerto 8080")
	log.Printf("🎮 Juego disponible en: http://localhost:8080")
	log.Fatal(server.ListenAndServe(":8080"))
}

func requestRouter(ctx *fasthttp.RequestCtx) {
	path := string(ctx.Path())
	method := string(ctx.Method())

	// Rutas estáticas
	switch path {
	case "/":
		serveFile(ctx, "index.html", "text/html")
		return
	case "/shared.css":
		serveFile(ctx, "shared.css", "text/css")
		return
	}

	// API routes
	if method == "GET" && path == "/api/questions" {
		// Servir preguntas directamente desde el archivo
		serveQuestionsFromFile(ctx)
		return
	}

	if method == "GET" && path == "/api/health" {
		ctx.SetContentType("application/json")
		ctx.SetBody([]byte(`{"status":"ok","message":"Servidor funcionando correctamente"}`))
		return
	}

	// 404 para rutas no encontradas
	ctx.SetStatusCode(fasthttp.StatusNotFound)
	ctx.SetBody([]byte("404 - Página no encontrada"))
}

func serveFile(ctx *fasthttp.RequestCtx, filename, contentType string) {
	if _, err := os.Stat(filename); os.IsNotExist(err) {
		ctx.SetStatusCode(fasthttp.StatusNotFound)
		ctx.SetBody([]byte("⚠️ Archivo no encontrado\n\nEl archivo " + filename + " no existe en el servidor.\n\nAsegúrate de que el archivo esté en el directorio correcto."))
		return
	}

	ctx.SetContentType(contentType)
	ctx.SendFile(filename)
}

func serveQuestionsFromFile(ctx *fasthttp.RequestCtx) {
	data, err := os.ReadFile("answers.json")
	if err != nil {
		ctx.SetStatusCode(fasthttp.StatusInternalServerError)
		ctx.SetBody([]byte(`{"success": false, "error": "Error leyendo preguntas"}`))
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

	var questionsData struct {
		Questions []models.Question `json:"questions"`
	}

	if err := json.Unmarshal(data, &questionsData); err != nil {
		return nil, err
	}

	return questionsData.Questions, nil
}
