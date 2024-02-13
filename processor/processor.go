package processor

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"net"
	"time"

	"github.com/charmbracelet/log"
	"github.com/pkg/errors"

	streamdal "github.com/streamdal/streamdal/sdks/go"
)

type Processor struct {
	*Config
	logstashConn net.Conn
}

type Config struct {
	LogStashAddr string
	ListenAddr   string
	Streamdal    *streamdal.Streamdal
	ShutdownCtx  context.Context
}

type LogEntry struct {
	Message string `json:"message"`
}

func New(cfg *Config) (*Processor, error) {
	if err := validateConfig(cfg); err != nil {
		return nil, errors.Wrap(err, "unable to validate config")
	}

	return &Processor{
		Config: cfg,
	}, nil
}

const (
	operationName = "logstash-process"
	componentName = "Logstash"
)

func validateConfig(cfg *Config) error {
	if cfg.Streamdal == nil {
		return errors.New("Streamdal cannot be nil")
	}

	if cfg.LogStashAddr == "" {
		return errors.New("LogStashAddr cannot be empty")
	}

	if cfg.ShutdownCtx == nil {
		return errors.New("ShutdownCtx cannot be nil")
	}

	if cfg.ListenAddr == "" {
		return errors.New("ListenAddr cannot be empty")
	}

	return nil
}

func (p *Processor) Process(logLine string) (string, error) {
	if logLine == "" {
		return "", errors.New("log line cannot be empty")
	}

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

	resp := p.Streamdal.Process(context.Background(), &streamdal.ProcessRequest{
		OperationType: streamdal.OperationTypeConsumer,
		OperationName: operationName,
		ComponentName: componentName,
		Data:          data,
	})

	if resp.Metadata["log_drop"] == "true" {
		fmt.Println("Log message was skipped due to log_drop metadata.")
		return "", nil
	}

	return string(resp.Data), nil
}

func (p *Processor) SendToLogstash(logLine string) error {
	// Establish a new connection to Logstash for each log message
	conn, err := net.Dial("tcp", p.LogStashAddr)
	if err != nil {
		return errors.Wrap(err, "failed to establish connection to Logstash")
	}
	defer conn.Close()

	// Send the log message
	_, err = conn.Write([]byte(logLine + "\n"))
	if err != nil {
		return errors.Wrap(err, "failed to send log line to Logstash")
	}

	return nil
}

func (p *Processor) Close() error {
	return p.logstashConn.Close()
}

func (p *Processor) EstablishLogstashConnection() error {
	var err error
	maxRetries := 5               // Maximum number of retries
	retryDelay := 2 * time.Second // Initial delay between retries

	for i := 0; i < maxRetries; i++ {
		p.logstashConn, err = net.Dial("tcp", p.LogStashAddr)
		if err == nil {
			return nil // Connection successful
		}

		log.Errorf("Failed to establish connection to Logstash (attempt %d/%d): %v", i+1, maxRetries, err)

		// Wait before retrying
		time.Sleep(retryDelay)
		// Increase delay for next retry, up to a maximum value
		retryDelay = time.Duration(float64(retryDelay) * 1.5)
		if retryDelay > 30*time.Second {
			retryDelay = 30 * time.Second // Maximum delay of 30 seconds
		}
	}

	return errors.Wrap(err, "failed to establish connection to Logstash after retries")
}

func (p *Processor) ListenForLogs() {
	// Establish connection to Logstash first
	if err := p.EstablishLogstashConnection(); err != nil {
		log.Fatalf("Failed to establish connection to Logstash: %v", err)
	}
	defer p.logstashConn.Close()

	ln, err := net.Listen("tcp", p.ListenAddr)
	if err != nil {
		log.Fatalf("Failed to listen on port %s: %v", p.ListenAddr, err)
	}

	defer ln.Close()

	log.Infof("Listening on port %s", p.ListenAddr)

	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Errorf("Failed to accept connection: %v", err)
			continue
		}

		scanner := bufio.NewScanner(conn)
		for scanner.Scan() {
			select {
			case <-p.ShutdownCtx.Done():
				return
			default:
				// NOOP
			}

			line := scanner.Text()
			if line == "" {
				continue
			}

			processedLog, err := p.Process(line)
			if err != nil {
				log.Errorf("Error processing log: %s", err)
				continue
			}
			fmt.Printf("Logstash connection %s", p.logstashConn)
			if err := p.SendToLogstash(processedLog); err != nil {
				log.Errorf("Error sending to Logstash: %v", err)
			}

			log.Debug("Processed log")
		}

		conn.Close()
	}
}
