FROM golang:1.25.1-bookworm

RUN apt-get update && apt-get install -y \
    ffmpeg \
    build-essential \
    git \
    wget \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /opt
RUN git clone https://github.com/ggerganov/whisper.cpp.git
WORKDIR /opt/whisper.cpp
RUN make

RUN bash ./models/download-ggml-model.sh base

WORKDIR /app

RUN go install github.com/air-verse/air@latest

COPY go.mod* go.sum* ./
RUN go mod download

COPY . .

ENV WHISPER_MODEL=/opt/whisper.cpp/models/ggml-base.bin
ENV WHISPER_BIN=/opt/whisper.cpp/main

CMD ["air"]
