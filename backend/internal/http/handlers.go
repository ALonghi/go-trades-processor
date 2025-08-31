package http

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	gin "github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/example/trades-aggregator/internal/cache"
	"github.com/example/trades-aggregator/internal/holdings"
	"github.com/example/trades-aggregator/internal/models"
)

type Server struct {
	R               *gin.Engine
	HoldingsService *holdings.Service
	Cache           *cache.Cache
	Logger          *zap.Logger
	TTL             time.Duration
}

type apiError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type tradesResponse struct {
	Rows [][]any `json:"rows"`
}

var validEntities = map[string]struct{}{
	"zurich":   {},
	"new_york": {},
}

func NewServer(holdingsService *holdings.Service, c *cache.Cache, logger *zap.Logger, corsOrigin string) *Server {
	g := gin.New()

	// Logging middleware (basic and safe)
	g.Use(func(cn *gin.Context) {
		start := time.Now()
		cn.Next()
		logger.Info("http_request",
			zap.String("method", cn.Request.Method),
			zap.String("path", cn.Request.URL.Path),
			zap.Int("status", cn.Writer.Status()),
			zap.String("ip", cn.ClientIP()),
			zap.Duration("latency", time.Since(start)),
		)
	})

	g.Use(gin.Recovery())

	// CORS (reflect configured origin; add Vary and Max-Age)
	g.Use(func(cn *gin.Context) {
		origin := cn.GetHeader("Origin")
		cn.Writer.Header().Set("Vary", "Origin")
		cn.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		cn.Writer.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
		cn.Writer.Header().Set("Access-Control-Max-Age", "86400")
		// Only echo the configured origin (or * if explicitly configured)
		if corsOrigin == "*" {
			cn.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		} else if origin != "" && origin == corsOrigin {
			cn.Writer.Header().Set("Access-Control-Allow-Origin", corsOrigin)
		}
		if cn.Request.Method == http.MethodOptions {
			cn.AbortWithStatus(http.StatusNoContent)
			return
		}
		cn.Next()
	})

	s := &Server{R: g, HoldingsService: holdingsService, Cache: c, Logger: logger}

	g.GET("/health", func(cn *gin.Context) {
		cn.JSON(http.StatusOK, gin.H{"ok": true})
	})

	g.GET("/api/holdings", s.getAllHoldings)
	g.GET("/api/holdings/:entity", s.getEntityHoldings)
	g.GET("/api/trades", s.getTrades)

	return s
}

// --- Helpers ---

func (s *Server) badRequest(c *gin.Context, msg string) {
	c.JSON(http.StatusBadRequest, apiError{Code: "bad_request", Message: msg})
}

func (s *Server) internalError(c *gin.Context, where string, err error) {
	s.Logger.Error("internal_error", zap.String("where", where), zap.Error(err))
	c.JSON(http.StatusInternalServerError, apiError{Code: "internal_server_error", Message: "internal server error"})
}

func parseLimit(v string, def, min, max int) int {
	if v == "" {
		return def
	}
	n, err := strconv.Atoi(v)
	if err != nil || n < min || n > max {
		return def
	}
	return n
}

func normalizeEntity(v string) string {
	return strings.ToLower(strings.TrimSpace(v))
}

// --- Handlers ---

func (s *Server) getAllHoldings(c *gin.Context) {
	const key = "holdings:all"

	if val, ok := s.Cache.Get(key); ok {
		if arr, ok2 := val.([]models.Holding); ok2 && arr != nil {
			// Always non-nil array to avoid null JSON
			if arr == nil {
				arr = []models.Holding{}
			}
			c.JSON(http.StatusOK, arr)
			return
		}
	}

	rows, err := s.HoldingsService.GetAll(c.Request.Context())
	if err != nil {
		s.internalError(c, "GetAll", err)
		return
	}
	if rows == nil {
		rows = []models.Holding{}
	}

	s.Cache.Set(key, rows)
	c.JSON(http.StatusOK, rows)
}

func (s *Server) getEntityHoldings(c *gin.Context) {
	ent := normalizeEntity(c.Param("entity"))
	if _, ok := validEntities[ent]; !ok {
		s.badRequest(c, "invalid entity (use 'zurich' or 'new_york')")
		return
	}

	key := "holdings:" + ent
	if val, ok := s.Cache.Get(key); ok {
		if arr, ok2 := val.([]models.Holding); ok2 && arr != nil {
			if arr == nil {
				arr = []models.Holding{}
			}
			c.JSON(http.StatusOK, arr)
			return
		}
	}

	rows, err := s.HoldingsService.GetByEntity(c.Request.Context(), ent)
	if err != nil {
		// Treat "not found" as empty array with 200 for a friendlier API
		if holdings.IsNotFound(err) {
			rows = []models.Holding{}
			s.Cache.Set(key, rows)
			c.JSON(http.StatusOK, rows)
			return
		}
		s.internalError(c, "GetByEntity", err)
		return
	}
	if rows == nil {
		rows = []models.Holding{}
	}

	s.Cache.Set(key, rows)
	c.JSON(http.StatusOK, rows)
}

func (s *Server) getTrades(c *gin.Context) {
	limit := parseLimit(c.Query("limit"), 100, 1, 1000)

	var entPtr *string
	if entQ := strings.TrimSpace(c.Query("entity")); entQ != "" {
		ent := normalizeEntity(entQ)
		if _, ok := validEntities[ent]; !ok {
			s.badRequest(c, "invalid entity (use 'zurich' or 'new_york')")
			return
		}
		entPtr = &ent
	}

	rows, err := s.HoldingsService.GetTrades(c.Request.Context(), limit, entPtr)
	if err != nil {
		s.internalError(c, "GetTrades", err)
		return
	}

	resp := tradesResponse{Rows: rows}
	if resp.Rows == nil {
		resp.Rows = make([][]any, 0)
	}
	c.JSON(http.StatusOK, resp)
}
