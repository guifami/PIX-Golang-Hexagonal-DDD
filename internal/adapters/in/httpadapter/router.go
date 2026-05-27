package httpadapter

import (
	"database/sql"
	"time"

	portin "go-api/internal/application/ports/in"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

// RouterConfig holds security parameters for the HTTP router.
type RouterConfig struct {
	APIKeys        []string
	AllowedOrigins []string
	RateLimitRPS   int
}

func SetupRouter(
	accountUC portin.AccountUseCase,
	pixKeyUC portin.PixKeyUseCase,
	paymentUC portin.PaymentUseCase,
	db *sql.DB,
	cfg RouterConfig,
) *gin.Engine {
	router := gin.New()

	// Global middleware (applied to all routes including health/swagger).
	router.Use(recoverWithLogger())
	router.Use(RequestLogger())
	router.Use(SecurityHeaders())
	router.Use(CORS(cfg.AllowedOrigins))

	if cfg.RateLimitRPS > 0 {
		rl := NewRateLimiter(cfg.RateLimitRPS, cfg.RateLimitRPS*2)
		router.Use(rl.Middleware())
	}

	// Public routes — no API key required.
	router.GET("/health", HealthHandler(db))
	router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	// Authenticated API routes.
	idempotencyStore := NewIdempotencyStore(24 * time.Hour)

	v1 := router.Group("/api/v1")
	v1.Use(APIKeyAuth(cfg.APIKeys))
	v1.Use(RequestTimeout(30 * time.Second))
	{
		accounts := v1.Group("/accounts")
		{
			accounts.POST("", accountHandler(accountUC).Create)
			accounts.GET("/:id", accountHandler(accountUC).GetByID)
			accounts.GET("/:id/balance", accountHandler(accountUC).GetBalance)
			accounts.POST("/:id/deposit",
				IdempotencyMiddleware(idempotencyStore),
				accountHandler(accountUC).Deposit,
			)
			accounts.GET("/:id/pix/keys", pixKeyHandler(pixKeyUC).ListByAccount)
			accounts.GET("/:id/pix/payments", paymentHandler(paymentUC).ListByAccount)
		}

		pix := v1.Group("/pix")
		{
			keys := pix.Group("/keys")
			{
				keys.POST("", pixKeyHandler(pixKeyUC).Register)
				keys.GET("/:key", pixKeyHandler(pixKeyUC).GetByValue)
				keys.DELETE("/:key", pixKeyHandler(pixKeyUC).Delete)
			}

			payments := pix.Group("/payments")
			{
				payments.POST("",
					IdempotencyMiddleware(idempotencyStore),
					paymentHandler(paymentUC).Initiate,
				)
				payments.GET("/:id", paymentHandler(paymentUC).GetByID)
			}
		}
	}

	return router
}

// helpers to avoid repeating NewXxxHandler calls.
func accountHandler(uc portin.AccountUseCase) *AccountHandler   { return NewAccountHandler(uc) }
func pixKeyHandler(uc portin.PixKeyUseCase) *PixKeyHandler       { return NewPixKeyHandler(uc) }
func paymentHandler(uc portin.PaymentUseCase) *PaymentHandler    { return NewPaymentHandler(uc) }
