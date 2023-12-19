package processor

import (
	"bufio"
	"context"
	"encoding/json"
	"net"

	"github.com/charmbracelet/log"
	"github.com/pkg/errors"

	streamdal "github.com/streamdal/go-sdk"
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
	if resp.Error {
		return "", err
	}

	return string(resp.Data), nil
}

func (p *Processor) SendToLogstash(logLine string) error {
	if _, err := p.logstashConn.Write([]byte(logLine + "\n")); err != nil {
		return err
	}

	return nil
}

func (p *Processor) Close() error {
	return p.logstashConn.Close()
}

func (p *Processor) ListenForLogs() {
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
			if err := p.SendToLogstash(processedLog); err != nil {
				log.Errorf("Error sending to Logstash: %v", err)
			}

			log.Debug("Processed log")
		}

		conn.Close()
	}
}
