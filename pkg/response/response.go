package response

import (
	"github.com/gin-gonic/gin"
	"github.com/Zyx-98/auth-service/pkg/apperror"
	"net/http"
)

type Response struct {
	Data interface{} `json:"data,omitempty"`
}

type ErrorResponse struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type ValidationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

type ValidationErrorResponse struct {
	Code   string              `json:"code"`
	Errors []ValidationError `json:"errors"`
}

func Ok(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, Response{Data: data})
}

func Created(c *gin.Context, data interface{}) {
	c.JSON(http.StatusCreated, Response{Data: data})
}

func NoContent(c *gin.Context) {
	c.Status(http.StatusNoContent)
}

func Error(c *gin.Context, err error) {
	if appErr, ok := err.(*apperror.AppError); ok {
		c.JSON(appErr.Status, ErrorResponse{
			Code:    appErr.Code,
			Message: appErr.Message,
		})
		return
	}

	c.JSON(http.StatusInternalServerError, ErrorResponse{
		Code:    "internal_server_error",
		Message: "An unexpected error occurred",
	})
}

func ValidationErrors(c *gin.Context, errors []ValidationError) {
	c.JSON(http.StatusBadRequest, ValidationErrorResponse{
		Code:   "validation_error",
		Errors: errors,
	})
}
