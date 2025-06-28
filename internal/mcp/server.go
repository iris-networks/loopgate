package mcp

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
)

type Server struct {
	protocol *Protocol
	input    io.Reader
	output   io.Writer
}

func NewServer() *Server {
	return &Server{
		protocol: NewProtocol(),
		input:    os.Stdin,
		output:   os.Stdout,
	}
}

func NewServerWithStreams(input io.Reader, output io.Writer) *Server {
	return &Server{
		protocol: NewProtocol(),
		input:    input,
		output:   output,
	}
}

func (s *Server) Start() error {
	log.Println("Starting MCP server...")
	
	scanner := bufio.NewScanner(s.input)
	
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}

		response, err := s.protocol.HandleRequest([]byte(line))
		if err != nil {
			log.Printf("Error handling request: %v", err)
			continue
		}

		_, err = s.output.Write(append(response, '\n'))
		if err != nil {
			log.Printf("Error writing response: %v", err)
			return err
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("error reading input: %w", err)
	}

	return nil
}

func (s *Server) HandleHTTPRequest(requestData []byte) ([]byte, error) {
	return s.protocol.HandleRequest(requestData)
}

func (s *Server) GetCapabilities() map[string]interface{} {
	return map[string]interface{}{
		"tools": map[string]interface{}{
			"listChanged": true,
		},
	}
}

func (s *Server) GetServerInfo() map[string]interface{} {
	return map[string]interface{}{
		"name":    ServerName,
		"version": ServerVersion,
	}
}

func (s *Server) CreateToolsListResponse() ([]byte, error) {
	tools := s.protocol.GetAvailableTools()
	
	result := map[string]interface{}{
		"tools": tools,
	}

	response := map[string]interface{}{
		"result": result,
		"id":     "tools-list",
	}

	return json.Marshal(response)
}