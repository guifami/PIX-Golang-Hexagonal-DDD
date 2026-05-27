package httpadapter

import (
	"errors"
	"net/http"

	portin "go-api/internal/application/ports/in"
	"go-api/internal/domain/account"
	"go-api/internal/infrastructure/logger"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type createAccountRequest struct {
	OwnerName string `json:"owner_name" binding:"required,min=2,max=255"`
	CPF       string `json:"cpf"        binding:"required"`
}

type depositRequest struct {
	AmountCents int64 `json:"amount_cents" binding:"required,min=1"`
}

type accountResponse struct {
	ID           string `json:"id"`
	OwnerName    string `json:"owner_name"`
	CPF          string `json:"cpf"`
	BalanceCents int64  `json:"balance_cents"`
	Status       string `json:"status"`
	CreatedAt    string `json:"created_at"`
}

type balanceResponse struct {
	AccountID    string `json:"account_id"`
	BalanceCents int64  `json:"balance_cents"`
	Balance      string `json:"balance"`
}

// AccountHandler handles HTTP requests for account operations.
type AccountHandler struct {
	useCase portin.AccountUseCase
}

func NewAccountHandler(uc portin.AccountUseCase) *AccountHandler {
	return &AccountHandler{useCase: uc}
}

// Create godoc
// @Summary Create account
// @Description Creates a new bank account. Initial balance is zero; use the deposit endpoint to add funds.
// @Tags Accounts
// @Accept json
// @Produce json
// @Param account body createAccountRequest true "Account payload"
// @Success 201 {object} accountResponse
// @Failure 400 {object} ErrorResponse
// @Failure 409 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/accounts [post]
func (h *AccountHandler) Create(c *gin.Context) {
	log := LoggerFromContext(c).With(zap.String("handler", "account.create"))

	var req createAccountRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Message: err.Error()})
		return
	}

	ctx := logger.WithContext(c.Request.Context(), log)
	acc, err := h.useCase.Create(ctx, req.OwnerName, req.CPF)
	if err != nil {
		switch {
		case errors.Is(err, account.ErrInvalidCPF), errors.Is(err, account.ErrInvalidName):
			c.JSON(http.StatusBadRequest, ErrorResponse{Message: err.Error()})
		case errors.Is(err, account.ErrCPFAlreadyInUse):
			c.JSON(http.StatusConflict, ErrorResponse{Message: err.Error()})
		default:
			log.Error("unexpected error", zap.Error(err))
			c.JSON(http.StatusInternalServerError, ErrorResponse{Message: "internal server error"})
		}
		return
	}

	c.JSON(http.StatusCreated, toAccountResponse(acc))
}

// GetByID godoc
// @Summary Get account
// @Description Returns account details by ID
// @Tags Accounts
// @Produce json
// @Param id path string true "Account UUID"
// @Success 200 {object} accountResponse
// @Failure 404 {object} ErrorResponse
// @Router /api/v1/accounts/{id} [get]
func (h *AccountHandler) GetByID(c *gin.Context) {
	log := LoggerFromContext(c).With(zap.String("handler", "account.get"))

	ctx := logger.WithContext(c.Request.Context(), log)
	acc, err := h.useCase.GetByID(ctx, c.Param("id"))
	if err != nil {
		if errors.Is(err, account.ErrAccountNotFound) {
			c.JSON(http.StatusNotFound, ErrorResponse{Message: err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, ErrorResponse{Message: "internal server error"})
		return
	}

	c.JSON(http.StatusOK, toAccountResponse(acc))
}

// GetBalance godoc
// @Summary Get account balance
// @Description Returns current balance in cents
// @Tags Accounts
// @Produce json
// @Param id path string true "Account UUID"
// @Success 200 {object} balanceResponse
// @Failure 404 {object} ErrorResponse
// @Router /api/v1/accounts/{id}/balance [get]
func (h *AccountHandler) GetBalance(c *gin.Context) {
	log := LoggerFromContext(c).With(zap.String("handler", "account.balance"))

	id := c.Param("id")
	ctx := logger.WithContext(c.Request.Context(), log)

	balance, err := h.useCase.GetBalance(ctx, id)
	if err != nil {
		if errors.Is(err, account.ErrAccountNotFound) {
			c.JSON(http.StatusNotFound, ErrorResponse{Message: err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, ErrorResponse{Message: "internal server error"})
		return
	}

	c.JSON(http.StatusOK, balanceResponse{
		AccountID:    id,
		BalanceCents: balance.Cents(),
		Balance:      balance.String(),
	})
}

// Deposit godoc
// @Summary Deposit to account
// @Description Adds funds to an account (simulation — no real banking involved)
// @Tags Accounts
// @Accept json
// @Produce json
// @Param id path string true "Account UUID"
// @Param body body depositRequest true "Deposit payload"
// @Success 200 {object} accountResponse
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Router /api/v1/accounts/{id}/deposit [post]
func (h *AccountHandler) Deposit(c *gin.Context) {
	log := LoggerFromContext(c).With(zap.String("handler", "account.deposit"))

	var req depositRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Message: err.Error()})
		return
	}

	ctx := logger.WithContext(c.Request.Context(), log)
	acc, err := h.useCase.Deposit(ctx, c.Param("id"), req.AmountCents)
	if err != nil {
		switch {
		case errors.Is(err, account.ErrAccountNotFound):
			c.JSON(http.StatusNotFound, ErrorResponse{Message: err.Error()})
		case errors.Is(err, account.ErrInvalidAmount), errors.Is(err, account.ErrAccountInactive):
			c.JSON(http.StatusBadRequest, ErrorResponse{Message: err.Error()})
		default:
			log.Error("unexpected error", zap.Error(err))
			c.JSON(http.StatusInternalServerError, ErrorResponse{Message: "internal server error"})
		}
		return
	}

	c.JSON(http.StatusOK, toAccountResponse(acc))
}

func toAccountResponse(acc *account.Account) accountResponse {
	return accountResponse{
		ID:           acc.ID,
		OwnerName:    acc.OwnerName,
		CPF:          acc.CPF.String(),
		BalanceCents: acc.Balance.Cents(),
		Status:       string(acc.Status),
		CreatedAt:    formatTime(acc.CreatedAt),
	}
}
