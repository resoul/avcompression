FROM golang:1.25.1-bookworm

RUN apt-get update && apt-get install -y ffmpeg && rm -rf /var/lib/apt/lists/*

WORKDIR /app

RUN go install github.com/air-verse/air@latest

COPY go.mod* go.sum* ./
RUN go mod download

COPY . .

CMD ["air"]
