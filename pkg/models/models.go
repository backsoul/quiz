package models

// Question estructura para representar una pregunta del quiz
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

// APIResponse estructura estándar para respuestas de API
type APIResponse struct {
	Success bool        `json:"success"`
	Message string      `json:"message,omitempty"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
}

// QuestionResponse respuesta específica para preguntas
type QuestionResponse struct {
	Question    *Question `json:"question,omitempty"`
	Questions   []Question `json:"questions,omitempty"`
	Count       int       `json:"count,omitempty"`
	Metadata    interface{} `json:"metadata,omitempty"`
}
