package v1

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/snykk/go-rest-boilerplate/internal/apperror"
	"github.com/snykk/go-rest-boilerplate/internal/constants"
	"github.com/snykk/go-rest-boilerplate/pkg/logger"
	"github.com/snykk/go-rest-boilerplate/pkg/validators"
)

type BaseResponse struct {
	Status  bool        `json:"status"`
	Message string      `json:"message,omitempty"`
	Data    interface{} `json:"data,omitempty"`
}

func NewSuccessResponse(c *gin.Context, statusCode int, message string, data interface{}) {
	c.JSON(statusCode, BaseResponse{
		Status:  true,
		Message: message,
		Data:    data,
	})
}

func NewErrorResponse(c *gin.Context, statusCode int, err string) {
	c.JSON(statusCode, BaseResponse{
		Status:  false,
		Message: err,
	})
}

// NewAbortResponse renders a 401 with the unified BaseResponse envelope
// and aborts the middleware chain. Kept as a thin wrapper so middlewares
// and handlers emit identical JSON shapes.
func NewAbortResponse(c *gin.Context, message string) {
	c.AbortWithStatusJSON(http.StatusUnauthorized, BaseResponse{
		Status:  false,
		Message: message,
	})
}

// mapDomainErrorToHTTP converts a domain error to an HTTP status code.
func mapDomainErrorToHTTP(err error) int {
	var domErr *apperror.DomainError
	if errors.As(err, &domErr) {
		switch domErr.Type {
		case apperror.ErrTypeNotFound:
			return http.StatusNotFound
		case apperror.ErrTypeUnauthorized:
			return http.StatusUnauthorized
		case apperror.ErrTypeForbidden:
			return http.StatusForbidden
		case apperror.ErrTypeConflict:
			return http.StatusConflict
		case apperror.ErrTypeBadRequest:
			return http.StatusBadRequest
		default:
			return http.StatusInternalServerError
		}
	}
	return http.StatusInternalServerError
}

// RespondWithError emits a sanitized error response and logs the
// underlying cause for any internal (5xx) failures. Centralizes the
// "log the gory detail, show the user a clean message" rule so no
// handler accidentally pushes a wrapped library error into the body.
func RespondWithError(c *gin.Context, err error) {
	// Validation failures get their own structured response — clients
	// need to know *which field* is wrong, not just the concatenated
	// summary. Render as 422 with per-field detail in `data`.
	var ve *validators.ValidationErrors
	if errors.As(err, &ve) {
		c.JSON(http.StatusUnprocessableEntity, BaseResponse{
			Status:  false,
			Message: "validation failed",
			Data:    map[string]any{"errors": ve.Errors},
		})
		return
	}

	status := mapDomainErrorToHTTP(err)
	message := err.Error()

	if status >= http.StatusInternalServerError {
		// Log the real cause (or the error itself if no cause is
		// attached) and send back a generic message. Without this,
		// clients see things like "hash password: bcrypt: …".
		fields := logger.Fields{
			constants.LoggerCategory: constants.LoggerCategoryHTTP,
			"path":                   c.FullPath(),
		}
		var domErr *apperror.DomainError
		if errors.As(err, &domErr) && domErr.Cause != nil {
			fields["cause"] = domErr.Cause.Error()
		} else {
			fields["cause"] = err.Error()
		}
		if rid, ok := c.Get("X-Request-ID"); ok {
			if s, ok := rid.(string); ok {
				fields["request_id"] = s
			}
		}
		logger.Error("internal error while handling request", fields)
		message = "internal server error"
	}

	NewErrorResponse(c, status, message)
}
