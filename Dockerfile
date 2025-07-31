FROM golang:1.23-alpine AS builder

WORKDIR /app

# Copy go mod and sum files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o main .

# Final stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates tzdata

WORKDIR /root/

# Copy the binary from builder
COPY --from=builder /app/main .

# Copy static files needed by the application
COPY --from=builder /app/index.html .
COPY --from=builder /app/admin.html .
COPY --from=builder /app/answers.json .

# Set timezone
ENV TZ=America/Bogota

# Expose port
EXPOSE 8080

# Command to run
CMD ["./main"]