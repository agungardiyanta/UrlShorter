package httpapi

import (
	"context"
	"errors"
	"math/rand"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/agungardiyanta/UrlShorter/apps/api/internal/cache"
	"github.com/agungardiyanta/UrlShorter/apps/api/internal/config"
	"github.com/agungardiyanta/UrlShorter/apps/api/internal/domain"
	"github.com/agungardiyanta/UrlShorter/apps/api/internal/events"
	"github.com/agungardiyanta/UrlShorter/apps/api/internal/store"
	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/cors"
	"github.com/gofiber/fiber/v3/middleware/logger"
	"github.com/gofiber/fiber/v3/middleware/recover"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
)

type Server struct {
	cfg       config.Config
	store     *store.PostgresStore
	cache     *cache.RedisCache
	publisher *events.KafkaPublisher
	app       *fiber.App
}

type createLinkRequest struct {
	TargetURL  string `json:"targetUrl"`
	CustomCode string `json:"customCode"`
}

func NewServer(cfg config.Config, st *store.PostgresStore, rc *cache.RedisCache, publisher *events.KafkaPublisher) *Server {
	server := &Server{
		cfg:       cfg,
		store:     st,
		cache:     rc,
		publisher: publisher,
	}
	server.app = fiber.New(fiber.Config{
		AppName: "urlshorter-api",
		ErrorHandler: func(c fiber.Ctx, err error) error {
			code := fiber.StatusInternalServerError
			message := "internal server error"
			var fiberErr *fiber.Error
			if errors.As(err, &fiberErr) {
				code = fiberErr.Code
				message = fiberErr.Message
			}
			return c.Status(code).JSON(fiber.Map{"error": message})
		},
	})
	server.routes()
	return server
}

func (s *Server) Listen() error {
	return s.app.Listen(s.cfg.HTTPAddr)
}

func (s *Server) Shutdown(ctx context.Context) error {
	return s.app.ShutdownWithContext(ctx)
}

func (s *Server) routes() {
	s.app.Use(recover.New())
	s.app.Use(logger.New())
	s.app.Use(cors.New(cors.Config{
		AllowOrigins: []string{s.cfg.FrontendOrigin},
		AllowHeaders: []string{"Origin", "Content-Type", "Accept"},
		AllowMethods: []string{fiber.MethodGet, fiber.MethodPost, fiber.MethodDelete, fiber.MethodOptions},
	}))
	s.app.Use(otelMiddleware())

	s.app.Get("/healthz", func(c fiber.Ctx) error {
		return c.JSON(fiber.Map{"status": "ok"})
	})

	api := s.app.Group("/api")
	api.Post("/links", s.createLink)
	api.Get("/links", s.listLinks)
	api.Get("/links/:code", s.getLink)
	api.Delete("/links/:code", s.deleteLink)
	s.app.Get("/:code", s.redirect)
}

func (s *Server) createLink(c fiber.Ctx) error {
	ctx := c.Context()
	var req createLinkRequest
	if err := c.Bind().Body(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid json body")
	}
	targetURL := strings.TrimSpace(req.TargetURL)
	if !isHTTPURL(targetURL) {
		return fiber.NewError(fiber.StatusBadRequest, "targetUrl must be a valid http or https URL")
	}

	code := normalizeCode(req.CustomCode)
	if code == "" {
		code = randomCode(s.cfg.ShortCodeLength)
	}

	expiresAt := time.Now().UTC().Add(s.cfg.DefaultLinkTTL)
	link, err := s.store.CreateLink(ctx, code, targetURL, expiresAt)
	if err != nil {
		if strings.Contains(err.Error(), "duplicate key") {
			return fiber.NewError(fiber.StatusConflict, "short code already exists")
		}
		return err
	}
	link.ShortURL = s.shortURL(link.Code)
	_ = s.cache.SetLink(ctx, link)
	return c.Status(fiber.StatusCreated).JSON(link)
}

func (s *Server) listLinks(c fiber.Ctx) error {
	limit := 25
	if raw := c.Query("limit"); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed < 1 || parsed > 100 {
			return fiber.NewError(fiber.StatusBadRequest, "limit must be between 1 and 100")
		}
		limit = parsed
	}
	links, err := s.store.ListLinks(c.Context(), limit)
	if err != nil {
		return err
	}
	for index := range links {
		links[index].ShortURL = s.shortURL(links[index].Code)
	}
	return c.JSON(fiber.Map{"links": links})
}

func (s *Server) getLink(c fiber.Ctx) error {
	link, err := s.lookupLink(c.Context(), c.Params("code"))
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return fiber.NewError(fiber.StatusNotFound, "link not found")
		}
		return err
	}
	link.ShortURL = s.shortURL(link.Code)
	return c.JSON(link)
}

func (s *Server) deleteLink(c fiber.Ctx) error {
	code := normalizeCode(c.Params("code"))
	if code == "" {
		return fiber.NewError(fiber.StatusNotFound, "link not found")
	}
	err := s.store.SoftDeleteLink(c.Context(), code)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return fiber.NewError(fiber.StatusNotFound, "link not found")
		}
		return err
	}
	_ = s.cache.DeleteLink(c.Context(), code)
	return c.SendStatus(fiber.StatusNoContent)
}

func (s *Server) redirect(c fiber.Ctx) error {
	code := c.Params("code")
	link, err := s.lookupLink(c.Context(), code)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return fiber.NewError(fiber.StatusNotFound, "link not found")
		}
		return err
	}

	if err := s.store.IncrementClicks(c.Context(), link.Code); err == nil {
		_ = s.cache.DeleteLink(c.Context(), link.Code)
	}

	event := domain.ClickEvent{
		Code:      link.Code,
		TargetURL: link.TargetURL,
		IP:        c.IP(),
		UserAgent: c.Get(fiber.HeaderUserAgent),
		Referer:   c.Get(fiber.HeaderReferer),
		ClickedAt: time.Now().UTC(),
	}
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		_ = s.publisher.PublishClick(ctx, event)
	}()

	return c.Redirect().To(link.TargetURL)
}

func (s *Server) lookupLink(ctx context.Context, code string) (domain.Link, error) {
	code = normalizeCode(code)
	if code == "" {
		return domain.Link{}, store.ErrNotFound
	}
	link, err := s.cache.GetLink(ctx, code)
	if err == nil {
		if link.IsExpired(time.Now().UTC()) {
			_ = s.cache.DeleteLink(ctx, link.Code)
			return domain.Link{}, store.ErrNotFound
		}
		return link, nil
	}
	link, err = s.store.GetLink(ctx, code)
	if err != nil {
		return domain.Link{}, err
	}
	if link.IsExpired(time.Now().UTC()) {
		_ = s.cache.DeleteLink(ctx, link.Code)
		return domain.Link{}, store.ErrNotFound
	}
	_ = s.cache.SetLink(ctx, link)
	return link, nil
}

func (s *Server) shortURL(code string) string {
	return s.cfg.BaseURL + "/" + code
}

func isHTTPURL(value string) bool {
	parsed, err := url.ParseRequestURI(value)
	if err != nil {
		return false
	}
	return parsed.Scheme == "http" || parsed.Scheme == "https"
}

func normalizeCode(value string) string {
	value = strings.TrimSpace(value)
	value = strings.Trim(value, "/")
	value = strings.ToLower(value)
	var builder strings.Builder
	for _, r := range value {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' || r == '_' {
			builder.WriteRune(r)
		}
	}
	return builder.String()
}

func randomCode(length int) string {
	const alphabet = "abcdefghijklmnopqrstuvwxyz0123456789"
	if length < 4 {
		length = 7
	}
	var builder strings.Builder
	builder.Grow(length)
	for range length {
		builder.WriteByte(alphabet[rand.Intn(len(alphabet))])
	}
	return builder.String()
}

func otelMiddleware() fiber.Handler {
	tracer := otel.Tracer("urlshorter-api/http")
	propagator := otel.GetTextMapPropagator()
	return func(c fiber.Ctx) error {
		ctx := propagator.Extract(c.Context(), propagation.HeaderCarrier(http.Header(c.GetReqHeaders())))
		spanName := c.Method() + " " + c.Path()
		ctx, span := tracer.Start(ctx, spanName)
		defer span.End()
		c.SetContext(ctx)

		err := c.Next()
		status := c.Response().StatusCode()
		span.SetAttributes(
			attribute.String("http.request.method", c.Method()),
			attribute.String("url.path", c.Path()),
			attribute.Int("http.response.status_code", status),
		)
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
		} else if status >= 500 {
			span.SetStatus(codes.Error, "server error")
		}
		return err
	}
}
