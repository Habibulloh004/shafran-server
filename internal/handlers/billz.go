package handlers

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/gofiber/fiber/v2"

	"github.com/example/shafran/internal/services"
)

// BillzHandler provides endpoints that proxy requests to the Billz API.
type BillzHandler struct{}

// NewBillzHandler builds a BillzHandler instance.
func NewBillzHandler() *BillzHandler {
	return &BillzHandler{}
}

// Proxy forwards the incoming request to the Billz API, injecting the server-side token.
func (h *BillzHandler) Proxy(c *fiber.Ctx) error {
	method := strings.ToUpper(strings.TrimSpace(c.Method()))
	if method == "" {
		method = http.MethodGet
	}

	path := strings.TrimLeft(strings.TrimSpace(c.Params("*")), "/")
	if path == "" {
		// Support calls to /billz without trailing path by attempting to reuse current route.
		path = strings.TrimLeft(strings.TrimPrefix(c.OriginalURL(), "/api/billz"), "/")
	}
	if path == "" {
		return fiber.NewError(fiber.StatusBadRequest, "missing Billz API path")
	}

	var body any
	if len(c.Body()) > 0 {
		body = json.RawMessage(c.Body())
	}

	queryMap := make(map[string]string, len(c.Queries()))
	for k, v := range c.Queries() {
		queryMap[k] = v
	}

	reqHeaders := c.GetReqHeaders()
	headers := make(map[string]string, len(reqHeaders))
	for k, vals := range reqHeaders {
		if strings.EqualFold(k, "Authorization") {
			continue
		}
		if len(vals) > 0 {
			headers[k] = vals[0]
		}
	}

	opts := services.BillzRequestOpts{
		Method:  method,
		Path:    path,
		Query:   queryMap,
		Body:    body,
		Headers: headers,
	}

	resp, err := services.DoBillzRequest(opts)
	if err != nil {
		return fiber.NewError(fiber.StatusBadGateway, err.Error())
	}

	c.Status(resp.Status)

	if ct := resp.Header.Get("Content-Type"); ct != "" {
		c.Set("Content-Type", ct)
	}

	for k, vals := range resp.Header {
		if len(vals) == 0 {
			continue
		}
		if strings.EqualFold(k, "Content-Type") || strings.EqualFold(k, "Content-Length") {
			continue
		}
		c.Set(k, vals[0])
	}

	return c.Send(resp.Body)
}
