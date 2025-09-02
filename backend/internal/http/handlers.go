package http

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	gin "github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/example/trades-aggregator/internal/cache"
	"github.com/example/trades-aggregator/internal/domain"
	"github.com/example/trades-aggregator/internal/holdings"
	"github.com/example/trades-aggregator/internal/models"
)

type Server struct {
	R               *gin.Engine
	HoldingsService *holdings.Service
	HoldingsCache   *cache.MapCache[cache.HoldingsKey, []models.Holding]
	TradesCache     *cache.MapCache[cache.TradesKey, []models.Trade]
	Logger          *zap.Logger
	TTL             time.Duration
}

type apiError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type tradesResponse struct {
	Rows []models.Trade `json:"rows"`
}

// NewServer wires the router, service, caches, and middleware.
func NewServer(holdingsService *holdings.Service, logger *zap.Logger, corsOrigin string) *Server {
	g := gin.New()

	// Request logging
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

	// CORS
	g.Use(func(cn *gin.Context) {
		origin := cn.GetHeader("Origin")
		cn.Writer.Header().Set("Vary", "Origin")
		cn.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		cn.Writer.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
		cn.Writer.Header().Set("Access-Control-Max-Age", "86400")
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

	// Typed caches (no TTLs)
	hc := cache.NewMapCache[cache.HoldingsKey, []models.Holding]()
	tc := cache.NewMapCache[cache.TradesKey, []models.Trade]()

	s := &Server{
		R:               g,
		HoldingsService: holdingsService,
		HoldingsCache:   hc,
		TradesCache:     tc,
		Logger:          logger,
	}

	g.GET("/health", func(cn *gin.Context) { cn.JSON(http.StatusOK, gin.H{"ok": true}) })
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

// --- Handlers ---

func (s *Server) getAllHoldings(c *gin.Context) {
	// Use the enum value for "all"
	key := cache.HoldingsKey{Entity: domain.EntityAll}

	if rows, ok := s.HoldingsCache.Get(key); ok && rows != nil {
		c.JSON(http.StatusOK, rows)
		return
	}

	rows, err := s.HoldingsService.GetAll(c.Request.Context())
	if err != nil {
		s.internalError(c, "GetAll", err)
		return
	}
	if rows == nil {
		rows = []models.Holding{}
	}

	s.HoldingsCache.Set(key, rows)
	c.JSON(http.StatusOK, rows)
}

func (s *Server) getEntityHoldings(c *gin.Context) {
	// Parse and validate via enum
	entRaw := strings.TrimSpace(c.Param("entity"))
	ent, ok := domain.ParseEntity(entRaw)
	if !ok {
		s.badRequest(c, "invalid entity (use 'zurich' or 'new_york')")
		return
	}

	key := cache.HoldingsKey{Entity: ent}
	if rows, ok := s.HoldingsCache.Get(key); ok && rows != nil {
		c.JSON(http.StatusOK, rows)
		return
	}

	rows, err := s.HoldingsService.GetByEntity(c.Request.Context(), ent.String())
	if err != nil {
		if holdings.IsNotFound(err) {
			rows = []models.Holding{}
			s.HoldingsCache.Set(key, rows)
			c.JSON(http.StatusOK, rows)
			return
		}
		s.internalError(c, "GetByEntity", err)
		return
	}
	if rows == nil {
		rows = []models.Holding{}
	}

	s.HoldingsCache.Set(key, rows)
	c.JSON(http.StatusOK, rows)
}

func (s *Server) getTrades(c *gin.Context) {
	limitInt := parseLimit(c.Query("limit"), 100, 1, 1000)
	limit := uint16(limitInt) // cache.TradesKey.Limit is uint16

	entity := domain.EntityAll // default "all" means no filter

	if raw := strings.TrimSpace(c.Query("entity")); raw != "" {
		if ent, ok := domain.ParseEntity(raw); ok {
			if ent != domain.EntityAll {
				entity = ent
			}
		} else {
			s.badRequest(c, "invalid entity (use 'zurich' or 'new_york')")
			return
		}
	}

	// Cache key uses the enum string ("all" if none)
	tkey := cache.TradesKey{Entity: entity, Limit: limit}
	if rows, ok := s.TradesCache.Get(tkey); ok && rows != nil {
		c.JSON(http.StatusOK, tradesResponse{Rows: rows})
		return
	}

	rows, err := s.HoldingsService.GetTrades(c.Request.Context(), limit, &entity)
	if err != nil {
		s.internalError(c, "GetTrades", err)
		return
	}
	if rows == nil {
		rows = make([]models.Trade, 0)
	}

	s.TradesCache.Set(tkey, rows)
	c.JSON(http.StatusOK, tradesResponse{Rows: rows})
}
