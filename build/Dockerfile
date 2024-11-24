# Etapa 1: Construcción
FROM golang:1.23-alpine AS builder

# Establece el directorio de trabajo dentro del contenedor
WORKDIR /app

# Copia los archivos de Go y el archivo go.mod y go.sum
COPY go.mod go.sum ./
# Descarga las dependencias (esto se almacenará en la cache si go.mod y go.sum no cambian)
RUN go mod download

# Copia el resto del código fuente
COPY . .

# Compila la aplicación con opciones para reducir el tamaño del binario
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o app ./cmd

# Etapa 2: Imagen final ligera
FROM scratch

# Copia solo el binario desde el build anterior
COPY --from=builder /app/app /app

# Especifica el puerto en el que la aplicación escucha
EXPOSE 8080

# Comando para ejecutar la aplicación
ENTRYPOINT ["/app"]
