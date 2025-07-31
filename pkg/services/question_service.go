package services

import (
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"time"

	"github.com/backsoul/quiz/pkg/models"
	"github.com/backsoul/quiz/pkg/redis"
)

// QuestionService maneja la lógica de negocio para las preguntas
type QuestionService struct {
	redisClient *redis.RedisClient
}

// NewQuestionService crea una nueva instancia del servicio
func NewQuestionService(redisClient *redis.RedisClient) *QuestionService {
	return &QuestionService{
		redisClient: redisClient,
	}
}

// LoadQuestionsFromFile carga las preguntas desde el archivo JSON a Redis
func (s *QuestionService) LoadQuestionsFromFile(filePath string) error {
	log.Printf("📂 Cargando preguntas desde: %s", filePath)
	
	// Leer el archivo JSON
	jsonData, err := ioutil.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("error leyendo archivo JSON: %v", err)
	}

	// Cargar a Redis usando el cliente
	if err := s.redisClient.LoadQuestionsFromJSON(jsonData); err != nil {
		return fmt.Errorf("error cargando preguntas a Redis: %v", err)
	}

	log.Println("✅ Preguntas cargadas exitosamente desde archivo")
	return nil
}

// GetAllQuestions obtiene todas las preguntas
func (s *QuestionService) GetAllQuestions() ([]models.Question, error) {
	redisQuestions, err := s.redisClient.GetAllQuestions()
	if err != nil {
		return nil, fmt.Errorf("error obteniendo preguntas de Redis: %v", err)
	}

	// Convertir de redis.Question a models.Question
	questions := make([]models.Question, len(redisQuestions))
	for i, rq := range redisQuestions {
		questions[i] = models.Question{
			ID:          rq.ID,
			Question:    rq.Question,
			Options:     rq.Options,
			Correct:     rq.Correct,
			Explanation: rq.Explanation,
			Difficulty:  rq.Difficulty,
		}
	}

	return questions, nil
}

// GetQuestion obtiene una pregunta específica por ID
func (s *QuestionService) GetQuestion(id int) (*models.Question, error) {
	redisQuestion, err := s.redisClient.GetQuestion(id)
	if err != nil {
		return nil, fmt.Errorf("error obteniendo pregunta %d: %v", id, err)
	}

	question := &models.Question{
		ID:          redisQuestion.ID,
		Question:    redisQuestion.Question,
		Options:     redisQuestion.Options,
		Correct:     redisQuestion.Correct,
		Explanation: redisQuestion.Explanation,
		Difficulty:  redisQuestion.Difficulty,
	}

	return question, nil
}

// GetRandomQuestion obtiene una pregunta aleatoria
func (s *QuestionService) GetRandomQuestion() (*models.Question, error) {
	redisQuestion, err := s.redisClient.GetRandomQuestion()
	if err != nil {
		return nil, fmt.Errorf("error obteniendo pregunta aleatoria: %v", err)
	}

	question := &models.Question{
		ID:          redisQuestion.ID,
		Question:    redisQuestion.Question,
		Options:     redisQuestion.Options,
		Correct:     redisQuestion.Correct,
		Explanation: redisQuestion.Explanation,
		Difficulty:  redisQuestion.Difficulty,
	}

	return question, nil
}

// GetQuestionsByDifficulty obtiene preguntas filtradas por dificultad
func (s *QuestionService) GetQuestionsByDifficulty(minDifficulty, maxDifficulty int) ([]models.Question, error) {
	redisQuestions, err := s.redisClient.GetQuestionsByDifficulty(minDifficulty, maxDifficulty)
	if err != nil {
		return nil, fmt.Errorf("error obteniendo preguntas por dificultad: %v", err)
	}

	questions := make([]models.Question, len(redisQuestions))
	for i, rq := range redisQuestions {
		questions[i] = models.Question{
			ID:          rq.ID,
			Question:    rq.Question,
			Options:     rq.Options,
			Correct:     rq.Correct,
			Explanation: rq.Explanation,
			Difficulty:  rq.Difficulty,
		}
	}

	return questions, nil
}

// GetRandomQuestionByDifficulty obtiene una pregunta aleatoria de cierta dificultad
func (s *QuestionService) GetRandomQuestionByDifficulty(minDifficulty, maxDifficulty int) (*models.Question, error) {
	questions, err := s.GetQuestionsByDifficulty(minDifficulty, maxDifficulty)
	if err != nil {
		return nil, err
	}

	if len(questions) == 0 {
		return nil, fmt.Errorf("no hay preguntas disponibles en el rango de dificultad %d-%d", minDifficulty, maxDifficulty)
	}

	// Semilla aleatoria
	rand.Seed(time.Now().UnixNano())
	randomIndex := rand.Intn(len(questions))
	
	return &questions[randomIndex], nil
}

// GetQuestionMetadata obtiene los metadatos del quiz
func (s *QuestionService) GetQuestionMetadata() (interface{}, error) {
	metadata, err := s.redisClient.GetMetadata()
	if err != nil {
		return nil, fmt.Errorf("error obteniendo metadatos: %v", err)
	}

	return metadata, nil
}

// GetQuestionCount obtiene el número total de preguntas
func (s *QuestionService) GetQuestionCount() (int, error) {
	count, err := s.redisClient.GetQuestionCount()
	if err != nil {
		return 0, fmt.Errorf("error obteniendo conteo de preguntas: %v", err)
	}

	return count, nil
}

// HealthCheck verifica que el servicio esté funcionando
func (s *QuestionService) HealthCheck() error {
	if err := s.redisClient.HealthCheck(); err != nil {
		return fmt.Errorf("error en health check de Redis: %v", err)
	}

	return nil
}

// ReloadQuestions recarga las preguntas desde el archivo JSON
func (s *QuestionService) ReloadQuestions(filePath string) error {
	log.Println("🔄 Recargando preguntas...")
	
	if err := s.LoadQuestionsFromFile(filePath); err != nil {
		return fmt.Errorf("error recargando preguntas: %v", err)
	}

	log.Println("✅ Preguntas recargadas exitosamente")
	return nil
}
