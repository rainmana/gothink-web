package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gorilla/mux"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/rainmana/gothink/internal/config"
	"github.com/rainmana/gothink/internal/middleware"
	"github.com/rainmana/gothink/internal/models"
	"github.com/rainmana/gothink/internal/storage"
	"github.com/rainmana/gothink/internal/types"
	"github.com/sirupsen/logrus"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Override port from environment if set (required for some cloud platforms)
	if port := os.Getenv("PORT"); port != "" {
		cfg.Port = port
	}

	// Create storage
	store, err := storage.New(cfg)
	if err != nil {
		log.Fatalf("Failed to create storage: %v", err)
	}

	// Create mental models loader
	logger := logrus.New()
	logger.SetOutput(os.Stderr)
	if cfg.LogLevel == "debug" {
		logger.SetLevel(logrus.DebugLevel)
	}
	modelsLoader := models.NewLoader(logger)

	// Create MCP server
	s := server.NewMCPServer(
		"GoThink MCP Server",
		"1.0.0",
		server.WithToolCapabilities(true),
		server.WithResourceCapabilities(false, false),
		server.WithPromptCapabilities(false),
	)

	// Add all the thinking tools
	addThinkingTools(s, store, modelsLoader, cfg)
	addSessionTools(s, store)

	// Create HTTP router
	router := mux.NewRouter()

	// Apply middleware
	router.Use(middleware.CORS())
	router.Use(middleware.Logging(logger))

	// Health check endpoint
	router.HandleFunc("/health", healthCheckHandler).Methods("GET")

	// Root endpoint with server info
	router.HandleFunc("/", rootHandler).Methods("GET")

	// Create SSE server for MCP
	sseServer := server.NewSSEServer(s)

	// Mount SSE handler
	router.PathPrefix("/sse").Handler(sseServer)

	// Create HTTP server
	addr := fmt.Sprintf("%s:%s", cfg.Host, cfg.Port)
	if cfg.Host == "localhost" {
		// For cloud deployment, bind to all interfaces
		addr = fmt.Sprintf(":%s", cfg.Port)
	}

	httpServer := &http.Server{
		Addr:         addr,
		Handler:      router,
		ReadTimeout:  cfg.ReadTimeout,
		WriteTimeout: cfg.WriteTimeout,
	}

	// Start server in a goroutine
	go func() {
		logger.Infof("Starting GoThink HTTP MCP Server on %s", addr)
		logger.Info("Endpoints:")
		logger.Infof("  - Health Check: http://%s/health", addr)
		logger.Infof("  - SSE Endpoint: http://%s/sse", addr)
		logger.Infof("  - Root Info:    http://%s/", addr)

		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatalf("Server failed to start: %v", err)
		}
	}()

	// Wait for interrupt signal to gracefully shutdown the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down server...")

	// Graceful shutdown with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := httpServer.Shutdown(ctx); err != nil {
		logger.Errorf("Server forced to shutdown: %v", err)
	}

	logger.Info("Server exited")
}

func healthCheckHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  "healthy",
		"service": "gothink-mcp-server",
		"version": "1.0.0",
		"time":    time.Now().UTC().Format(time.RFC3339),
	})
}

func rootHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"name":        "GoThink MCP Server",
		"version":     "1.0.0",
		"description": "Advanced MCP server combining systematic thinking, mental models, and debugging approaches",
		"endpoints": map[string]string{
			"health": "/health",
			"sse":    "/sse",
		},
		"transport": "HTTP with Server-Sent Events (SSE)",
		"protocol":  "Model Context Protocol (MCP)",
	})
}

func addThinkingTools(s *server.MCPServer, store *storage.Storage, modelsLoader *models.Loader, cfg *config.Config) {
	// Sequential Thinking Tool
	s.AddTool(
		mcp.NewTool("sequential_thinking",
			mcp.WithDescription("Perform sequential thinking operations with structured thought progression"),
			mcp.WithString("session_id", mcp.Required(), mcp.Description("Session identifier")),
			mcp.WithString("thought", mcp.Required(), mcp.Description("Current thought content")),
			mcp.WithNumber("thought_number", mcp.Required(), mcp.Description("Current thought number in sequence")),
			mcp.WithNumber("total_thoughts", mcp.Required(), mcp.Description("Total number of thoughts planned")),
			mcp.WithBoolean("next_thought_needed", mcp.Required(), mcp.Description("Whether another thought is needed")),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			sessionID, _ := req.RequireString("session_id")
			thought, _ := req.RequireString("thought")
			thoughtNumber, _ := req.RequireInt("thought_number")
			totalThoughts, _ := req.RequireInt("total_thoughts")
			nextThoughtNeeded, _ := req.RequireBool("next_thought_needed")

			result, err := handleSequentialThinking(store, sessionID, thought, thoughtNumber, totalThoughts, nextThoughtNeeded)
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}

			return mcp.NewToolResultText(result), nil
		},
	)

	// Mental Model Tool
	s.AddTool(
		mcp.NewTool("mental_model",
			mcp.WithDescription("Apply mental models to solve problems using structured thinking frameworks"),
			mcp.WithString("session_id", mcp.Required(), mcp.Description("Session identifier")),
			mcp.WithString("model_name", mcp.Required(), mcp.Description("Name of the mental model to apply")),
			mcp.WithString("problem", mcp.Required(), mcp.Description("Problem statement to analyze")),
			mcp.WithArray("steps", mcp.Description("Steps to follow for the mental model")),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			sessionID, _ := req.RequireString("session_id")
			modelName, _ := req.RequireString("model_name")
			problem, _ := req.RequireString("problem")
			steps := req.GetStringSlice("steps", []string{})

			// Load available mental models
			availableModels, err := modelsLoader.LoadMentalModels(cfg.MentalModelsPath)
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("Failed to load mental models: %v", err)), nil
			}

			// Check if the requested model exists
			model, exists := availableModels[modelName]
			if !exists {
				// Return available models for reference
				available := modelsLoader.GetAvailableModels(availableModels)
				return mcp.NewToolResultError(fmt.Sprintf("Mental model '%s' not found. Available models: %v", modelName, available)), nil
			}

			// Use model steps if no custom steps provided
			if len(steps) == 0 {
				steps = model.Steps
			}

			// Create mental model data
			modelData := &types.MentalModelData{
				ID:        fmt.Sprintf("%d-%d", time.Now().UnixNano(), len(steps)),
				ModelName: modelName,
				Problem:   problem,
				Steps:     steps,
				CreatedAt: time.Now(),
			}

			// Store the mental model
			store.AddMentalModel(sessionID, modelData)

			// Get session stats
			stats, _ := store.GetSessionStats(sessionID)

			// Create response
			response := map[string]interface{}{
				"status":   "success",
				"model_id": modelData.ID,
				"model_info": map[string]interface{}{
					"name":        model.Name,
					"description": model.Description,
					"category":    model.Category,
					"priority":    model.Priority,
				},
				"steps_used":     steps,
				"has_steps":      len(steps) > 0,
				"has_conclusion": false,
				"session_context": map[string]interface{}{
					"session_id":          sessionID,
					"total_mental_models": stats.Stores["mental_models"].(map[string]int)["count"],
				},
			}

			result, _ := json.Marshal(response)
			return mcp.NewToolResultText(string(result)), nil
		},
	)

	// Debugging Approach Tool
	s.AddTool(
		mcp.NewTool("debugging_approach",
			mcp.WithDescription("Apply systematic debugging approaches to identify and resolve issues"),
			mcp.WithString("session_id", mcp.Required(), mcp.Description("Session identifier")),
			mcp.WithString("approach_name", mcp.Required(), mcp.Description("Name of the debugging approach")),
			mcp.WithString("issue", mcp.Required(), mcp.Description("Issue description to debug")),
			mcp.WithArray("steps", mcp.Description("Debugging steps to follow")),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			sessionID, _ := req.RequireString("session_id")
			_, _ = req.RequireString("approach_name")
			_, _ = req.RequireString("issue")
			steps := req.GetStringSlice("steps", []string{})

			// Create response
			response := map[string]interface{}{
				"status":         "success",
				"approach_id":    fmt.Sprintf("%d-%d", time.Now().UnixNano(), len(steps)),
				"has_steps":      len(steps) > 0,
				"has_findings":   false,
				"has_resolution": false,
				"session_context": map[string]interface{}{
					"session_id": sessionID,
				},
			}

			result, _ := json.Marshal(response)
			return mcp.NewToolResultText(string(result)), nil
		},
	)

	// List Available Mental Models Tool
	s.AddTool(
		mcp.NewTool("list_mental_models",
			mcp.WithDescription("List all available mental models with their details"),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			// Load available mental models
			availableModels, err := modelsLoader.LoadMentalModels(cfg.MentalModelsPath)
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("Failed to load mental models: %v", err)), nil
			}

			// Get models sorted by priority
			modelsByPriority := modelsLoader.GetModelsByPriority(availableModels)
			modelsByCategory := modelsLoader.GetModelsByCategory(availableModels)

			// Create response
			response := map[string]interface{}{
				"status":             "success",
				"total_models":       len(availableModels),
				"models_by_priority": modelsByPriority,
				"models_by_category": modelsByCategory,
				"available_models":   modelsLoader.GetAvailableModels(availableModels),
			}

			result, _ := json.Marshal(response)
			return mcp.NewToolResultText(string(result)), nil
		},
	)
}

func addSessionTools(s *server.MCPServer, store *storage.Storage) {
	// Session Stats Tool
	s.AddTool(
		mcp.NewTool("session_stats",
			mcp.WithDescription("Get statistics for a session"),
			mcp.WithString("session_id", mcp.Required(), mcp.Description("Session identifier")),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			sessionID, _ := req.RequireString("session_id")

			// Get session stats
			stats, err := store.GetSessionStats(sessionID)
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("Failed to get session stats: %v", err)), nil
			}

			// Create response
			response := map[string]interface{}{
				"session_id":         sessionID,
				"created_at":         stats.CreatedAt.Format(time.RFC3339),
				"last_accessed_at":   stats.LastAccessedAt.Format(time.RFC3339),
				"thought_count":      stats.ThoughtCount,
				"tools_used":         stats.ToolsUsed,
				"total_operations":   stats.TotalOperations,
				"is_active":          stats.IsActive,
				"remaining_thoughts": stats.RemainingThoughts,
				"stores":             stats.Stores,
			}

			result, _ := json.Marshal(response)
			return mcp.NewToolResultText(string(result)), nil
		},
	)

	// Session Export Tool
	s.AddTool(
		mcp.NewTool("session_export",
			mcp.WithDescription("Export all data for a session"),
			mcp.WithString("session_id", mcp.Required(), mcp.Description("Session identifier")),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			sessionID, _ := req.RequireString("session_id")

			// Export session data
			exportData, err := store.ExportSession(sessionID)
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("Failed to export session: %v", err)), nil
			}

			// Create response
			response := map[string]interface{}{
				"version":      "1.0.0",
				"timestamp":    time.Now().Format(time.RFC3339),
				"session_id":   sessionID,
				"session_type": "hybrid",
				"data":         exportData,
				"metadata": map[string]interface{}{
					"exported_at": time.Now().Format(time.RFC3339),
					"version":     "0.1.0",
				},
			}

			result, _ := json.Marshal(response)
			return mcp.NewToolResultText(string(result)), nil
		},
	)
}

// handleSequentialThinking processes sequential thinking requests
func handleSequentialThinking(store *storage.Storage, sessionID, thought string, thoughtNumber, totalThoughts int, nextThoughtNeeded bool) (string, error) {
	// Create thought data
	thoughtData := &types.ThoughtData{
		ID:                fmt.Sprintf("%d-%d", time.Now().UnixNano(), thoughtNumber),
		Thought:           thought,
		ThoughtNumber:     thoughtNumber,
		TotalThoughts:     totalThoughts,
		NextThoughtNeeded: nextThoughtNeeded,
		CreatedAt:         time.Now(),
	}

	// Store the thought
	if err := store.AddThought(sessionID, thoughtData); err != nil {
		return "", err
	}

	// Get session stats
	stats, err := store.GetSessionStats(sessionID)
	if err != nil {
		return "", err
	}

	// Create response
	response := map[string]interface{}{
		"status":     "success",
		"thought_id": thoughtData.ID,
		"session_context": map[string]interface{}{
			"session_id":         sessionID,
			"total_thoughts":     stats.ThoughtCount,
			"remaining_thoughts": 100 - stats.ThoughtCount,
		},
	}

	result, err := json.Marshal(response)
	if err != nil {
		return "", err
	}

	return string(result), nil
}
