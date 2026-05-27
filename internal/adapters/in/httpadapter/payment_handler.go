package httpadapter

import (
	"errors"
	"net/http"

	portin "go-api/internal/application/ports/in"
	"go-api/internal/domain/account"
	"go-api/internal/domain/pixkey"
	"go-api/internal/domain/transaction"
	"go-api/internal/infrastructure/logger"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type initiatePaymentRequest struct {
	PayerAccountID string `json:"payer_account_id" binding:"required,uuid"`
	ReceiverKey    string `json:"receiver_key"     binding:"required"`
	AmountCents    int64  `json:"amount_cents"     binding:"required,min=1"`
	Description    string `json:"description"      binding:"max=140"`
}

type transactionResponse struct {
	ID                string  `json:"id"`
	PayerAccountID    string  `json:"payer_account_id"`
	ReceiverKey       string  `json:"receiver_key"`
	ReceiverAccountID string  `json:"receiver_account_id,omitempty"`
	AmountCents       int64   `json:"amount_cents"`
	Status            string  `json:"status"`
	Description       string  `json:"description,omitempty"`
	InitiatedAt       string  `json:"initiated_at"`
	CompletedAt       *string `json:"completed_at,omitempty"`
	FailureReason     *string `json:"failure_reason,omitempty"`
}

// PaymentHandler handles HTTP requests for PIX payment operations.
type PaymentHandler struct {
	useCase portin.PaymentUseCase
}

func NewPaymentHandler(uc portin.PaymentUseCase) *PaymentHandler {
	return &PaymentHandler{useCase: uc}
}

// Initiate godoc
// @Summary Initiate a PIX payment
// @Description Creates a new PIX payment. The transfer is processed asynchronously via Kafka. Returns 202 Accepted with the transaction in PENDING status.
// @Tags Payments
// @Accept json
// @Produce json
// @Param body body initiatePaymentRequest true "Payment payload"
// @Success 202 {object} transactionResponse
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 422 {object} ErrorResponse
// @Router /api/v1/pix/payments [post]
func (h *PaymentHandler) Initiate(c *gin.Context) {
	log := LoggerFromContext(c).With(zap.String("handler", "payment.initiate"))

	var req initiatePaymentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Message: err.Error()})
		return
	}

	ctx := logger.WithContext(c.Request.Context(), log)
	tx, err := h.useCase.Initiate(ctx, req.PayerAccountID, req.ReceiverKey, req.Description, req.AmountCents)
	if err != nil {
		switch {
		case errors.Is(err, transaction.ErrInvalidAmount), errors.Is(err, transaction.ErrSameAccount):
			c.JSON(http.StatusBadRequest, ErrorResponse{Message: err.Error()})
		case errors.Is(err, account.ErrAccountNotFound), errors.Is(err, pixkey.ErrKeyNotFound):
			c.JSON(http.StatusNotFound, ErrorResponse{Message: err.Error()})
		case errors.Is(err, account.ErrAccountInactive), errors.Is(err, account.ErrInsufficientFunds):
			c.JSON(http.StatusUnprocessableEntity, ErrorResponse{Message: err.Error()})
		default:
			log.Error("unexpected error", zap.Error(err))
			c.JSON(http.StatusInternalServerError, ErrorResponse{Message: "internal server error"})
		}
		return
	}

	c.JSON(http.StatusAccepted, toTransactionResponse(tx))
}

// GetByID godoc
// @Summary Get transaction by ID
// @Description Returns current status and details of a PIX transaction
// @Tags Payments
// @Produce json
// @Param id path string true "Transaction UUID"
// @Success 200 {object} transactionResponse
// @Failure 404 {object} ErrorResponse
// @Router /api/v1/pix/payments/{id} [get]
func (h *PaymentHandler) GetByID(c *gin.Context) {
	log := LoggerFromContext(c).With(zap.String("handler", "payment.get"))

	ctx := logger.WithContext(c.Request.Context(), log)
	tx, err := h.useCase.GetByID(ctx, c.Param("id"))
	if err != nil {
		if errors.Is(err, transaction.ErrTransactionNotFound) {
			c.JSON(http.StatusNotFound, ErrorResponse{Message: err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, ErrorResponse{Message: "internal server error"})
		return
	}

	c.JSON(http.StatusOK, toTransactionResponse(tx))
}

// ListByAccount godoc
// @Summary List payments made by an account
// @Description Returns all PIX transactions where the account was the payer
// @Tags Payments
// @Produce json
// @Param id path string true "Account UUID"
// @Success 200 {array} transactionResponse
// @Router /api/v1/accounts/{id}/pix/payments [get]
func (h *PaymentHandler) ListByAccount(c *gin.Context) {
	log := LoggerFromContext(c).With(zap.String("handler", "payment.list"))

	ctx := logger.WithContext(c.Request.Context(), log)
	txs, err := h.useCase.ListByAccount(ctx, c.Param("id"))
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Message: "internal server error"})
		return
	}

	resp := make([]transactionResponse, 0, len(txs))
	for i := range txs {
		resp = append(resp, toTransactionResponse(&txs[i]))
	}

	c.JSON(http.StatusOK, resp)
}

func toTransactionResponse(tx *transaction.PixTransaction) transactionResponse {
	resp := transactionResponse{
		ID:             tx.ID,
		PayerAccountID: tx.PayerAccountID,
		ReceiverKey:    tx.ReceiverKey,
		AmountCents:    tx.Amount.Cents(),
		Status:         string(tx.Status),
		Description:    tx.Description,
		InitiatedAt:    formatTime(tx.InitiatedAt),
		FailureReason:  tx.FailureReason,
	}
	if tx.ReceiverAccountID != "" {
		resp.ReceiverAccountID = tx.ReceiverAccountID
	}
	if tx.CompletedAt != nil {
		t := formatTime(*tx.CompletedAt)
		resp.CompletedAt = &t
	}
	return resp
}
