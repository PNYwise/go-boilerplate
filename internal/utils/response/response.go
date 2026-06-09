package response

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// BaseResponse represents the standard JSON response format.
type BaseResponse struct {
	ResponseCode string      `json:"responseCode"`
	ResponseDesc string      `json:"responseDesc"`
	ResponseData interface{} `json:"responseData,omitempty"`
}

// TransactionData represents the data object for transaction responses.
type TransactionData struct {
	TransactionID string `json:"transactionId"`
	Files         []File `json:"files,omitempty"`
}

// File represents the file details in the response.
type File struct {
	Name    string `json:"name"`
	Storage string `json:"storage"`
	Path    string `json:"path"`
	Bucket  string `json:"bucket"`
	Date    string `json:"date"`
}

// Response Codes
const (
	CodeSuccess         = "00"
	CodeInternalError   = "01"
	CodeNotFound        = "02"
	CodeRequestNotValid = "03"
)

// Response Descriptions
const (
	DescSuccess         = "Success"
	DescInternalError   = "Internal Error"
	DescNotFound        = "Not Found"
	DescRequestNotValid = "Request Not Valid"
)

// JSON dynamically writes a standard response to the Gin context
// based on the provided HTTP status code.
func JSON(c *gin.Context, httpStatus int, data interface{}) {
	var code string
	var desc string

	switch httpStatus {
	case http.StatusOK, http.StatusCreated, http.StatusAccepted, http.StatusNoContent:
		code = CodeSuccess
		desc = DescSuccess
	case http.StatusNotFound:
		code = CodeNotFound
		desc = DescNotFound
	case http.StatusBadRequest, http.StatusUnprocessableEntity:
		code = CodeRequestNotValid
		desc = DescRequestNotValid
	case http.StatusInternalServerError, http.StatusServiceUnavailable, http.StatusBadGateway:
		code = CodeInternalError
		desc = DescInternalError
	default:
		// Fallback grouping for unexpected status codes
		if httpStatus >= 200 && httpStatus < 300 {
			code = CodeSuccess
			desc = DescSuccess
		} else {
			code = CodeInternalError
			desc = "Unknown Error"
		}
	}

	c.JSON(httpStatus, BaseResponse{
		ResponseCode: code,
		ResponseDesc: desc,
		ResponseData: data,
	})
}
