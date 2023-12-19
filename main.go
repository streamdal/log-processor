package main

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"log"
	"net"
	"os"

	streamdal "github.com/streamdal/go-sdk" // Import Streamdal SDK
)

type LogEntry struct {
	Message string `json:"message"`
}

func main() {
	listenPort := getEnv("LISTEN_PORT", ":6000")
	logstashOutputPort := getEnv("LOGSTASH_OUTPUT_PORT", "logstash-server:7002")
	streamdalToken := getEnv("STREAMDAL_TOKEN", "1234")
	streamdalServer := os.Getenv("SERVER")

	streamdalClient, err := streamdal.New(&streamdal.Config{
		ServerURL:   streamdalServer,
		ServerToken: streamdalToken,
		ServiceName: "logstash",
		ShutdownCtx: context.Background(),
	})
	if err != nil {
		log.Fatalf("Failed to initialize Streamdal client: %v", err)
	}

	ln, err := net.Listen("tcp", listenPort)
	if err != nil {
		log.Fatalf("Failed to listen on port %s: %v", listenPort, err)
	}
	defer ln.Close()
	log.Printf("Listening on port %s", listenPort)

	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Printf("Failed to accept connection: %v", err)
			continue
		}

		scanner := bufio.NewScanner(conn)
		for scanner.Scan() {
			logLine := scanner.Text()
			processedLog, err := processLog(logLine, streamdalClient)
			if err != nil {
				log.Printf("Error processing log: %v", err)
				continue
			}
			if err := sendToLogstash(processedLog, logstashOutputPort); err != nil {
				log.Printf("Error sending to Logstash: %v", err)
			}
		}
		conn.Close()
	}
}

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}

func processLog(logLine string, streamdalClient *streamdal.Streamdal) (string, error) {
	var jsonData map[string]interface{}
	err := json.Unmarshal([]byte(logLine), &jsonData)

	var data []byte
	if err != nil {
		logEntry := LogEntry{Message: logLine}
		data, err = json.Marshal(logEntry)
		if err != nil {
			return "", err
		}
	} else {
		data = []byte(logLine)
	}

	if streamdalClient == nil {
		return "", errors.New("streamdal client is nil")
	}

	resp, err := streamdalClient.Process(context.Background(), &streamdal.ProcessRequest{
		OperationType: streamdal.OperationTypeConsumer,
		OperationName: "logstash-process",
		ComponentName: "Logstash",
		Data:          data,
	})
	if err != nil {
		return "", err
	}

	return string(resp.Data), nil
}

func sendToLogstash(logLine string, logstashOutputPort string) error {
	conn, err := net.Dial("tcp", logstashOutputPort)
	if err != nil {
		return err
	}
	defer conn.Close()

	_, err = conn.Write([]byte(logLine + "\n"))
	return err
}
