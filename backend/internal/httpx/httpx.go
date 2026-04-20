package httpx

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
)

type ErrorResponse struct {
	Error string `json:"error"`
}

func BadRequest(c *gin.Context, msg string) {
	c.AbortWithStatusJSON(http.StatusBadRequest, ErrorResponse{Error: msg})
}

func Unauthorized(c *gin.Context, msg string) {
	if msg == "" {
		msg = "unauthorized"
	}
	c.AbortWithStatusJSON(http.StatusUnauthorized, ErrorResponse{Error: msg})
}

func Forbidden(c *gin.Context, msg string) {
	if msg == "" {
		msg = "forbidden"
	}
	c.AbortWithStatusJSON(http.StatusForbidden, ErrorResponse{Error: msg})
}

func NotFound(c *gin.Context, msg string) {
	if msg == "" {
		msg = "not found"
	}
	c.AbortWithStatusJSON(http.StatusNotFound, ErrorResponse{Error: msg})
}

func Internal(c *gin.Context, err error) {
	_ = c.Error(err)
	c.AbortWithStatusJSON(http.StatusInternalServerError, ErrorResponse{Error: "internal server error"})
}

func HandleDBError(c *gin.Context, err error) {
	if errors.Is(err, pgx.ErrNoRows) {
		NotFound(c, "")
		return
	}
	Internal(c, err)
}
