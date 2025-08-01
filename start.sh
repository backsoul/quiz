#!/bin/bash

# Script para testing rápido del servidor Quiz
echo "🎮 Iniciando pruebas del servidor Quiz..."

# Verificar si Go está instalado
if ! command -v go &> /dev/null; then
    echo "❌ Go no está instalado. Por favor instala Go 1.23+"
    exit 1
fi

# Verificar si Redis está corriendo
if ! command -v redis-cli &> /dev/null; then
    echo "⚠️  Redis CLI no encontrado. Asegúrate de que Redis esté instalado."
else
    if redis-cli ping | grep -q "PONG"; then
        echo "✅ Redis está funcionando"
    else
        echo "❌ Redis no está respondiendo. Inicia Redis con: redis-server"
        exit 1
    fi
fi

# Construir el proyecto
echo "🔨 Construyendo el proyecto..."
go build -o quiz-server main.go

if [ $? -eq 0 ]; then
    echo "✅ Compilación exitosa"
else
    echo "❌ Error en la compilación"
    exit 1
fi

# Ejecutar el servidor
echo "🚀 Iniciando servidor..."
echo "📱 Abre http://localhost:8080 en tu navegador"
echo "🔄 Presiona Ctrl+C para detener"
./quiz-server
