package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strconv"

	"github.com/redis/go-redis/v9"
)

// RedisClient estructura para manejar conexiones con Redis
type RedisClient struct {
	client *redis.Client
	ctx    context.Context
}

// Question estructura para representar una pregunta
type Question struct {
	ID          int               `json:"id"`
	Question    string            `json:"question"`
	Options     map[string]string `json:"options"`
	Correct     string            `json:"correctAnswer"`
	Explanation string            `json:"explanation"`
	Difficulty  int               `json:"difficulty"`
}

// QuestionsData estructura para el JSON completo
type QuestionsData struct {
	Questions []Question `json:"questions"`
	Metadata  struct {
		Total       int    `json:"totalQuestions"`
		Version     string `json:"version"`
		LastUpdated string `json:"lastUpdated"`
		Description string `json:"description"`
	} `json:"metadata"`
}

// NewRedisClient crea una nueva instancia del cliente Redis
func NewRedisClient(addr, password string, db int) *RedisClient {
	rdb := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       db,
	})

	ctx := context.Background()

	// Verificar conexión
	_, err := rdb.Ping(ctx).Result()
	if err != nil {
		log.Fatalf("❌ Error conectando a Redis: %v", err)
	}

	log.Println("✅ Conexión exitosa a Redis")

	return &RedisClient{
		client: rdb,
		ctx:    ctx,
	}
}

// LoadQuestionsFromJSON carga las preguntas desde un archivo JSON a Redis
func (r *RedisClient) LoadQuestionsFromJSON(jsonData []byte) error {
	var questionsData QuestionsData
	
	if err := json.Unmarshal(jsonData, &questionsData); err != nil {
		return fmt.Errorf("error parsing JSON: %v", err)
	}

	log.Printf("📚 Cargando %d preguntas a Redis...", len(questionsData.Questions))

	// Limpiar preguntas existentes
	if err := r.ClearAllQuestions(); err != nil {
		log.Printf("⚠️ Error limpiando preguntas existentes: %v", err)
	}

	// Cargar cada pregunta individualmente
	for _, question := range questionsData.Questions {
		if err := r.SaveQuestion(question); err != nil {
			log.Printf("❌ Error guardando pregunta %d: %v", question.ID, err)
			continue
		}
	}

	// Guardar metadatos
	metadataJSON, _ := json.Marshal(questionsData.Metadata)
	if err := r.client.Set(r.ctx, "quiz:metadata", metadataJSON, 0).Err(); err != nil {
		log.Printf("⚠️ Error guardando metadatos: %v", err)
	}

	// Guardar lista de IDs de preguntas
	questionIDs := make([]interface{}, len(questionsData.Questions))
	for i, q := range questionsData.Questions {
		questionIDs[i] = q.ID
	}
	
	if err := r.client.Del(r.ctx, "quiz:question_ids").Err(); err != nil {
		log.Printf("⚠️ Error limpiando lista de IDs: %v", err)
	}
	
	if len(questionIDs) > 0 {
		if err := r.client.SAdd(r.ctx, "quiz:question_ids", questionIDs...).Err(); err != nil {
			log.Printf("⚠️ Error guardando lista de IDs: %v", err)
		}
	}

	log.Printf("✅ %d preguntas cargadas exitosamente en Redis", len(questionsData.Questions))
	return nil
}

// SaveQuestion guarda una pregunta individual en Redis
func (r *RedisClient) SaveQuestion(question Question) error {
	questionJSON, err := json.Marshal(question)
	if err != nil {
		return fmt.Errorf("error serializing question: %v", err)
	}

	key := fmt.Sprintf("quiz:question:%d", question.ID)
	return r.client.Set(r.ctx, key, questionJSON, 0).Err()
}

// GetQuestion obtiene una pregunta específica por ID
func (r *RedisClient) GetQuestion(id int) (*Question, error) {
	key := fmt.Sprintf("quiz:question:%d", id)
	
	questionJSON, err := r.client.Get(r.ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, fmt.Errorf("question %d not found", id)
		}
		return nil, fmt.Errorf("error getting question: %v", err)
	}

	var question Question
	if err := json.Unmarshal([]byte(questionJSON), &question); err != nil {
		return nil, fmt.Errorf("error parsing question: %v", err)
	}

	return &question, nil
}

// GetAllQuestions obtiene todas las preguntas
func (r *RedisClient) GetAllQuestions() ([]Question, error) {
	// Obtener todos los IDs de preguntas
	questionIDs, err := r.client.SMembers(r.ctx, "quiz:question_ids").Result()
	if err != nil {
		return nil, fmt.Errorf("error getting question IDs: %v", err)
	}

	var questions []Question
	for _, idStr := range questionIDs {
		id, err := strconv.Atoi(idStr)
		if err != nil {
			log.Printf("⚠️ ID de pregunta inválido: %s", idStr)
			continue
		}

		question, err := r.GetQuestion(id)
		if err != nil {
			log.Printf("⚠️ Error obteniendo pregunta %d: %v", id, err)
			continue
		}

		questions = append(questions, *question)
	}

	return questions, nil
}

// GetQuestionsByDifficulty obtiene preguntas filtradas por dificultad
func (r *RedisClient) GetQuestionsByDifficulty(minDifficulty, maxDifficulty int) ([]Question, error) {
	allQuestions, err := r.GetAllQuestions()
	if err != nil {
		return nil, err
	}

	var filteredQuestions []Question
	for _, question := range allQuestions {
		if question.Difficulty >= minDifficulty && question.Difficulty <= maxDifficulty {
			filteredQuestions = append(filteredQuestions, question)
		}
	}

	return filteredQuestions, nil
}

// GetRandomQuestion obtiene una pregunta aleatoria
func (r *RedisClient) GetRandomQuestion() (*Question, error) {
	// Obtener un ID aleatorio de la lista
	idStr, err := r.client.SRandMember(r.ctx, "quiz:question_ids").Result()
	if err != nil {
		return nil, fmt.Errorf("error getting random question ID: %v", err)
	}

	id, err := strconv.Atoi(idStr)
	if err != nil {
		return nil, fmt.Errorf("invalid question ID: %s", idStr)
	}

	return r.GetQuestion(id)
}

// GetMetadata obtiene los metadatos del quiz
func (r *RedisClient) GetMetadata() (map[string]interface{}, error) {
	metadataJSON, err := r.client.Get(r.ctx, "quiz:metadata").Result()
	if err != nil {
		if err == redis.Nil {
			return nil, fmt.Errorf("metadata not found")
		}
		return nil, fmt.Errorf("error getting metadata: %v", err)
	}

	var metadata map[string]interface{}
	if err := json.Unmarshal([]byte(metadataJSON), &metadata); err != nil {
		return nil, fmt.Errorf("error parsing metadata: %v", err)
	}

	return metadata, nil
}

// GetQuestionCount obtiene el número total de preguntas en Redis
func (r *RedisClient) GetQuestionCount() (int, error) {
	count, err := r.client.SCard(r.ctx, "quiz:question_ids").Result()
	if err != nil {
		return 0, fmt.Errorf("error getting question count: %v", err)
	}
	return int(count), nil
}

// ClearAllQuestions elimina todas las preguntas de Redis
func (r *RedisClient) ClearAllQuestions() error {
	// Obtener todos los IDs para eliminar las preguntas individuales
	questionIDs, err := r.client.SMembers(r.ctx, "quiz:question_ids").Result()
	if err == nil {
		for _, idStr := range questionIDs {
			key := fmt.Sprintf("quiz:question:%s", idStr)
			r.client.Del(r.ctx, key)
		}
	}

	// Limpiar la lista de IDs
	return r.client.Del(r.ctx, "quiz:question_ids").Err()
}

// Close cierra la conexión con Redis
func (r *RedisClient) Close() error {
	return r.client.Close()
}

// HealthCheck verifica que Redis esté funcionando
func (r *RedisClient) HealthCheck() error {
	_, err := r.client.Ping(r.ctx).Result()
	if err != nil {
		return fmt.Errorf("redis health check failed: %v", err)
	}
	return nil
}
