# Quiz ¿Quién Quiere Ser Millonario?

Un juego de preguntas y respuestas inspirado en el programa "¿Quién Quiere Ser Millonario?" desarrollado en Go con FastHTTP y Redis.

## 🎮 Características

- **8 preguntas** cuidadosamente seleccionadas con temática brasileña y conocimiento general
- **Sistema de comodines**: 50/50, Llamada a un amigo, Pregunta al público
- **Tiempo real** con WebSockets para sincronización entre jugadores
- **Sistema de premios** dinámico basado en el número de preguntas
- **Interfaz responsiva** que funciona en móviles y desktop
- **UUID persistente** para cada jugador
- **Estado del juego** guardado en localStorage

## 🚀 Instalación y Uso

### Prerrequisitos

- Go 1.23+
- Redis
- Docker (opcional)

### Ejecutar localmente

1. **Clonar el repositorio:**

   ```bash
   git clone <repository-url>
   cd quiz
   ```

2. **Instalar dependencias:**

   ```bash
   go mod download
   ```

3. **Iniciar Redis:**

   ```bash
   # Con Docker
   docker run -d -p 6379:6379 redis:7-alpine

   # O instalar localmente
   redis-server
   ```

4. **Ejecutar el servidor:**

   ```bash
   go run main.go
   ```

5. **Abrir en el navegador:**
   ```
   http://localhost:8080
   ```

### Ejecutar con Docker

```bash
# Construir y ejecutar con docker-compose
docker-compose up -d
```

## 📊 API Endpoints

### Preguntas

- `GET /api/questions` - Obtener todas las preguntas
- `GET /api/questions/{id}` - Obtener pregunta específica
- `GET /api/questions/random` - Pregunta aleatoria
- `GET /api/questions/metadata` - Metadatos del quiz

### Sesiones de Juego

- `POST /api/sessions` - Crear nueva sesión de jugador
- `GET /api/sessions/{id}` - Obtener sesión específica
- `POST /api/sessions/{id}/answer` - Enviar respuesta
- `POST /api/sessions/{id}/lifeline` - Usar comodín
- `GET /api/sessions/active` - Sesiones activas
- `GET /api/leaderboard` - Tabla de posiciones

### Control del Juego

- `POST /api/game/start` - Iniciar juego
- `POST /api/game/end` - Terminar juego
- `GET /api/game/state` - Estado actual del juego
- `POST /api/game/next-question` - Avanzar pregunta
- `POST /api/game/reveal-answer` - Revelar respuesta

### WebSocket

- `GET /ws` - Conexión WebSocket para tiempo real

## 🎯 Preguntas Incluidas

El quiz incluye 8 preguntas cuidadosamente seleccionadas:

1. **Arte clásico** - Mona Lisa
2. **Idioma portugués** - Traducción de "jamón"
3. **Cultura brasileña** - Bebida nacional
4. **Ciencia** - Tabla periódica
5. **Gastronomía brasileña** - Palomitas de maíz
6. **Bebidas** - Café descafeinado
7. **Humor** - Chiste de abeja
8. **Jerga brasileña** - Expresión "pindaíba"

## 🏗️ Arquitectura

```
├── main.go                 # Servidor principal
├── pkg/
│   ├── handlers/          # Handlers HTTP
│   ├── models/            # Modelos de datos
│   ├── services/          # Lógica de negocio
│   ├── redis/             # Cliente Redis
│   └── websocket/         # Hub WebSocket
├── index.html             # Interfaz del juego
├── shared.css             # Estilos compartidos
├── answers.json           # Base de datos de preguntas
└── docker-compose.yml     # Configuración Docker
```

## 🔧 Configuración

### Variables de Entorno

```bash
REDIS_ADDR=localhost:6379
REDIS_PASSWORD=
REDIS_DB=0
PORT=8080
```

### Personalizar Preguntas

Edita el archivo `answers.json` para modificar las preguntas:

```json
{
  "questions": [
    {
      "id": 1,
      "question": "Tu pregunta aquí",
      "options": {
        "A": "Opción A",
        "B": "Opción B",
        "C": "Opción C",
        "D": "Opción D"
      },
      "correctAnswer": "D",
      "explanation": "Explicación de la respuesta",
      "difficulty": 1
    }
  ],
  "metadata": {
    "totalQuestions": 8,
    "version": "1.1"
  }
}
```

## 🎮 Cómo Jugar

1. **Ingresa tu nombre** en la pantalla de bienvenida
2. **Espera** a que otros jugadores se unan
3. **Responde las preguntas** seleccionando A, B, C o D
4. **Usa comodines** cuando los necesites:
   - 🔄 **50/50**: Elimina 2 opciones incorrectas
   - 📞 **Llamada**: Sugerencia automática
   - 👥 **Público**: Porcentajes de respuestas
5. **Gana premios** por cada respuesta correcta
6. **¡Intenta llegar hasta la pregunta final!**

## 🏆 Sistema de Premios

Los premios se escalan automáticamente según el número de preguntas:

- Pregunta 1: $1,000
- Pregunta 2: $2,000
- Pregunta 3: $5,000
- ...
- Pregunta 8: $1,000,000

## 🤝 Contribuir

1. Fork del proyecto
2. Crear rama para nueva característica
3. Commit de cambios
4. Push a la rama
5. Crear Pull Request

## 📝 Licencia

Este proyecto está bajo la Licencia MIT.

---

¡Diviértete jugando! 🎯✨
