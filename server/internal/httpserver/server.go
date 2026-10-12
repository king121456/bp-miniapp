package httpserver

import (
	"log/slog"
	"net/http"
	"strings"
	"time"

	"bp-server/internal/service"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/v2/bson"
)

type Server struct {
	App *service.App
}

func New(app *service.App) *Server {
	return &Server{App: app}
}

// Engine builds the Gin engine with middleware and all API routes.
func (s *Server) Engine() *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(slogAccess())
	r.Use(corsMiddleware(s.App.Cfg.AppEnv, s.App.Cfg.CORSOrigin))

	r.GET("/healthz", adapt(func(w http.ResponseWriter, req *http.Request) {
		service.OK(w, "ok")
	}))
	r.POST("/api/auth/login", adapt(s.App.Login))
	r.GET("/api/me", s.auth(s.App.Me))
	r.PUT("/api/me", s.auth(s.App.UpdateMe))
	r.POST("/api/me/avatar", s.auth(s.App.UploadAvatar))
	r.Static("/uploads", service.UploadDir)

	r.GET("/api/bp", s.auth(s.App.ListBP))
	r.POST("/api/bp", s.auth(s.App.CreateBP))
	r.PUT("/api/bp/:id", s.auth(s.App.UpdateBP))
	r.DELETE("/api/bp/:id", s.auth(s.App.DeleteBP))
	r.GET("/api/bp/stats", s.auth(s.App.StatsBP))

	r.GET("/api/meds", s.auth(s.App.ListMeds))
	r.POST("/api/meds", s.auth(s.App.CreateMed))
	r.PUT("/api/meds/:id", s.auth(s.App.UpdateMed))
	r.DELETE("/api/meds/:id", s.auth(s.App.DeleteMed))
	r.POST("/api/meds/:id/checkin", s.auth(s.App.Checkin))
	r.GET("/api/meds/checkins", s.auth(s.App.ListCheckins))

	r.GET("/api/reminders/measure", s.auth(s.App.GetMeasure))
	r.PUT("/api/reminders/measure", s.auth(s.App.PutMeasure))
	r.POST("/api/subscribe", s.auth(s.App.AddGrant))
	r.GET("/api/todos", s.auth(s.App.ListTodos))
	r.POST("/api/todos/:id/done", s.auth(s.App.DoneTodo))

	r.POST("/api/family/invite", s.auth(s.App.Invite))
	r.POST("/api/family/bind", s.auth(s.App.Bind))
	r.GET("/api/family", s.auth(s.App.ListFamily))
	r.DELETE("/api/family/:id", s.auth(s.App.Unbind))
	r.GET("/api/family/patients/:pid/bp", s.auth(s.App.FamilyBP))
	r.GET("/api/family/patients/:pid/bp/stats", s.auth(s.App.FamilyStats))

	r.GET("/api/report", s.auth(s.App.Report))
	r.GET("/api/meds/history", s.auth(s.App.MedHistory))
	r.GET("/api/meds/makeup", s.auth(s.App.MakeupCandidates))
	r.POST("/api/meds/:id/makeup", s.auth(s.App.Makeup))
	r.DELETE("/api/meds/:id/makeup", s.auth(s.App.RevokeMakeup))
	r.POST("/api/meds/:id/skip", s.auth(s.App.Skip))
	r.DELETE("/api/meds/:id/skip", s.auth(s.App.Unskip))
	r.GET("/api/visits", s.auth(s.App.ListVisits))
	r.POST("/api/visits", s.auth(s.App.CreateVisit))
	r.PUT("/api/visits/:id", s.auth(s.App.UpdateVisit))
	r.POST("/api/visits/:id/cancel", s.auth(s.App.CancelVisit))
	r.DELETE("/api/visits/:id", s.auth(s.App.DeleteVisit))
	r.GET("/api/family/patients/:pid/meds/history", s.auth(s.App.FamilyMedHistory))
	r.GET("/api/family/patients/:pid/visit", s.auth(s.App.FamilyVisit))
	r.GET("/api/family/messages", s.auth(s.App.FamilyMessages))
	r.GET("/api/account/export", s.auth(s.App.Export))
	r.DELETE("/api/account", s.auth(s.App.DeleteAccount))

	return r
}

// Routes returns an http.Handler for tests and ListenAndServe.
func (s *Server) Routes() http.Handler { return s.Engine() }

func adapt(h http.HandlerFunc) gin.HandlerFunc {
	return func(c *gin.Context) {
		for _, p := range c.Params {
			c.Request.SetPathValue(p.Key, p.Value)
		}
		h(c.Writer, c.Request)
	}
}

func (s *Server) auth(h func(http.ResponseWriter, *http.Request, bson.ObjectID)) gin.HandlerFunc {
	return adapt(s.App.Auth(h))
}

func slogAccess() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		lat := time.Since(start)
		status := c.Writer.Status()
		attrs := []any{
			"method", c.Request.Method,
			"path", c.FullPath(),
			"raw", c.Request.URL.Path,
			"status", status,
			"latency_ms", lat.Milliseconds(),
		}
		if status >= 500 {
			slog.Error("http", attrs...)
		} else if status >= 400 {
			slog.Warn("http", attrs...)
		} else {
			slog.Info("http", attrs...)
		}
	}
}

// corsMiddleware only when APP_ENV=dev and CORS_ORIGIN is set.
func corsMiddleware(appEnv, origins string) gin.HandlerFunc {
	if appEnv != "dev" || origins == "" {
		return func(c *gin.Context) { c.Next() }
	}
	allow := map[string]bool{}
	for _, o := range strings.Split(origins, ",") {
		allow[strings.TrimSpace(o)] = true
	}
	return func(c *gin.Context) {
		o := c.GetHeader("Origin")
		if o != "" && (allow["*"] || allow[o]) {
			c.Header("Access-Control-Allow-Origin", o)
			c.Header("Vary", "Origin")
			c.Header("Access-Control-Allow-Headers", "Authorization, Content-Type")
			c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			if c.Request.Method == http.MethodOptions {
				c.Status(http.StatusNoContent)
				c.Abort()
				return
			}
		}
		c.Next()
	}
}
