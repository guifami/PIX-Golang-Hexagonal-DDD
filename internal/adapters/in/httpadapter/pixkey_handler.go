package httpadapter

import (
	"errors"
	"net/http"

	portin "go-api/internal/application/ports/in"
	"go-api/internal/domain/account"
	"go-api/internal/domain/pixkey"
	"go-api/internal/infrastructure/logger"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type registerPixKeyRequest struct {
	AccountID string `json:"account_id" binding:"required,uuid"`
	KeyType   string `json:"key_type"   binding:"required"`
	KeyValue  string `json:"key_value"  binding:"required"`
}

type pixKeyResponse struct {
	ID        string `json:"id"`
	AccountID string `json:"account_id"`
	KeyType   string `json:"key_type"`
	KeyValue  string `json:"key_value"`
	Status    string `json:"status"`
	CreatedAt string `json:"created_at"`
}

// PixKeyHandler handles HTTP requests for PIX key operations.
type PixKeyHandler struct {
	useCase portin.PixKeyUseCase
}

func NewPixKeyHandler(uc portin.PixKeyUseCase) *PixKeyHandler {
	return &PixKeyHandler{useCase: uc}
}

// Register godoc
// @Summary Register a PIX key
// @Description Registers a new PIX key (CPF, CNPJ, PHONE, EMAIL or EVP) for an account
// @Tags PIX Keys
// @Accept json
// @Produce json
// @Param body body registerPixKeyRequest true "PIX key payload"
// @Success 201 {object} pixKeyResponse
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 409 {object} ErrorResponse
// @Failure 422 {object} ErrorResponse
// @Router /api/v1/pix/keys [post]
func (h *PixKeyHandler) Register(c *gin.Context) {
	log := LoggerFromContext(c).With(zap.String("handler", "pixkey.register"))

	var req registerPixKeyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Message: err.Error()})
		return
	}

	ctx := logger.WithContext(c.Request.Context(), log)
	key, err := h.useCase.Register(ctx, req.AccountID, req.KeyType, req.KeyValue)
	if err != nil {
		switch {
		case errors.Is(err, account.ErrAccountNotFound):
			c.JSON(http.StatusNotFound, ErrorResponse{Message: err.Error()})
		case errors.Is(err, account.ErrAccountInactive):
			c.JSON(http.StatusUnprocessableEntity, ErrorResponse{Message: err.Error()})
		case errors.Is(err, pixkey.ErrKeyAlreadyExists):
			c.JSON(http.StatusConflict, ErrorResponse{Message: err.Error()})
		case errors.Is(err, pixkey.ErrKeyLimitReached):
			c.JSON(http.StatusUnprocessableEntity, ErrorResponse{Message: err.Error()})
		case errors.Is(err, pixkey.ErrInvalidKeyType), errors.Is(err, pixkey.ErrInvalidKeyValue):
			c.JSON(http.StatusBadRequest, ErrorResponse{Message: err.Error()})
		default:
			log.Error("unexpected error", zap.Error(err))
			c.JSON(http.StatusInternalServerError, ErrorResponse{Message: "internal server error"})
		}
		return
	}

	c.JSON(http.StatusCreated, toPixKeyResponse(key))
}

// GetByValue godoc
// @Summary Get PIX key info
// @Description Returns the account associated with a PIX key value
// @Tags PIX Keys
// @Produce json
// @Param key path string true "PIX key value"
// @Success 200 {object} pixKeyResponse
// @Failure 404 {object} ErrorResponse
// @Router /api/v1/pix/keys/{key} [get]
func (h *PixKeyHandler) GetByValue(c *gin.Context) {
	log := LoggerFromContext(c).With(zap.String("handler", "pixkey.get"))

	ctx := logger.WithContext(c.Request.Context(), log)
	key, err := h.useCase.GetByValue(ctx, c.Param("key"))
	if err != nil {
		if errors.Is(err, pixkey.ErrKeyNotFound) {
			c.JSON(http.StatusNotFound, ErrorResponse{Message: err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, ErrorResponse{Message: "internal server error"})
		return
	}

	c.JSON(http.StatusOK, toPixKeyResponse(key))
}

// Delete godoc
// @Summary Delete a PIX key
// @Description Removes a PIX key. Requires account_id query param for ownership verification.
// @Tags PIX Keys
// @Produce json
// @Param key path string true "PIX key value"
// @Param account_id query string true "Owner account UUID"
// @Success 204 "No Content"
// @Failure 400 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Router /api/v1/pix/keys/{key} [delete]
func (h *PixKeyHandler) Delete(c *gin.Context) {
	log := LoggerFromContext(c).With(zap.String("handler", "pixkey.delete"))

	accountID := c.Query("account_id")
	if accountID == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{Message: "account_id query param is required"})
		return
	}

	ctx := logger.WithContext(c.Request.Context(), log)
	err := h.useCase.Delete(ctx, c.Param("key"), accountID)
	if err != nil {
		switch {
		case errors.Is(err, pixkey.ErrKeyNotFound):
			c.JSON(http.StatusNotFound, ErrorResponse{Message: err.Error()})
		case errors.Is(err, pixkey.ErrUnauthorized):
			c.JSON(http.StatusForbidden, ErrorResponse{Message: err.Error()})
		default:
			log.Error("unexpected error", zap.Error(err))
			c.JSON(http.StatusInternalServerError, ErrorResponse{Message: "internal server error"})
		}
		return
	}

	c.Status(http.StatusNoContent)
}

// ListByAccount godoc
// @Summary List PIX keys of an account
// @Description Returns all active PIX keys linked to an account
// @Tags PIX Keys
// @Produce json
// @Param id path string true "Account UUID"
// @Success 200 {array} pixKeyResponse
// @Failure 404 {object} ErrorResponse
// @Router /api/v1/accounts/{id}/pix/keys [get]
func (h *PixKeyHandler) ListByAccount(c *gin.Context) {
	log := LoggerFromContext(c).With(zap.String("handler", "pixkey.list"))

	ctx := logger.WithContext(c.Request.Context(), log)
	keys, err := h.useCase.ListByAccount(ctx, c.Param("id"))
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Message: "internal server error"})
		return
	}

	resp := make([]pixKeyResponse, 0, len(keys))
	for i := range keys {
		resp = append(resp, toPixKeyResponse(&keys[i]))
	}

	c.JSON(http.StatusOK, resp)
}

func toPixKeyResponse(k *pixkey.PixKey) pixKeyResponse {
	return pixKeyResponse{
		ID:        k.ID,
		AccountID: k.AccountID,
		KeyType:   string(k.Type),
		KeyValue:  k.Value,
		Status:    string(k.Status),
		CreatedAt: formatTime(k.CreatedAt),
	}
}
