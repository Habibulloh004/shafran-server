package services

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net/http"
	"strings"

	"github.com/example/shafran/internal/models"
)

const (
	billzShopID          = "29ce1934-120f-459a-8046-8bfa89529a3c"
	billzCashboxID       = "83cdf361-cb50-48ce-a56f-01c8068bf63b"
	billzPaymentTypeID   = "6042429f-0d4c-40b7-9ee8-55c115865146"
	billzResponseChannel = "HTTP"
	paymePaymentComment  = "Payment completed via Payme"
)

// BillzOrderResult represents the essential identifiers returned by Billz.
type BillzOrderResult struct {
	OrderID     string
	OrderNumber string
	OrderType   string
}

type billzCreateOrderResponse struct {
	ID   string `json:"id"`
	Data struct {
		OrderNumber string `json:"order_number"`
		OrderType   string `json:"order_type"`
	} `json:"data"`
}

type paymeOrderDetails struct {
	Items    []paymeOrderItem `json:"items"`
	Checkout paymeCheckout    `json:"checkout"`
	Totals   paymeTotals      `json:"totals"`
	User     paymeUser        `json:"user"`
}

type paymeOrderItem struct {
	ProductID      string  `json:"productId"`
	ProductIDSnake string  `json:"product_id"`
	Quantity       float64 `json:"quantity"`
	Qty            float64 `json:"qty"`
}

func (i paymeOrderItem) normalizedProductID() string {
	if id := strings.TrimSpace(i.ProductID); id != "" {
		return id
	}
	return strings.TrimSpace(i.ProductIDSnake)
}

func (i paymeOrderItem) normalizedQuantity() float64 {
	if i.Quantity > 0 {
		return i.Quantity
	}
	return i.Qty
}

type paymeCheckout struct {
	PaymentMethod      string `json:"paymentMethod"`
	PaymentMethodSnake string `json:"payment_method"`
	Comment            string `json:"comment"`
	Notes              string `json:"notes"`
}

func (c paymeCheckout) normalizedPaymentMethod() string {
	if method := strings.TrimSpace(c.PaymentMethod); method != "" {
		return method
	}
	return strings.TrimSpace(c.PaymentMethodSnake)
}

func (c paymeCheckout) normalizedComment() string {
	if comment := strings.TrimSpace(c.Comment); comment != "" {
		return comment
	}
	return strings.TrimSpace(c.Notes)
}

type paymeTotals struct {
	Amount      float64 `json:"amount"`
	Total       float64 `json:"total"`
	TotalAmount float64 `json:"total_amount"`
}

func (t paymeTotals) totalAmount() float64 {
	if t.Amount > 0 {
		return t.Amount
	}
	if t.Total > 0 {
		return t.Total
	}
	return t.TotalAmount
}

type paymeUser struct {
	ID     string `json:"id"`
	UserID string `json:"user_id"`
}

func (u paymeUser) normalizedID() string {
	if id := strings.TrimSpace(u.ID); id != "" {
		return id
	}
	return strings.TrimSpace(u.UserID)
}

// CreateBillzOrderFromPaymeTransaction builds a Billz order using the Payme payload saved with the transaction.
func CreateBillzOrderFromPaymeTransaction(txn models.PaymeTransaction) (*BillzOrderResult, error) {
	fmt.Printf("[Billz/Payme] CreateBillzOrderFromPaymeTransaction called for txn %s\n", txn.ID)
	fmt.Printf("[Billz/Payme] OrderDetails raw (first 200 chars): %.200s\n", string(txn.OrderDetails))

	if len(txn.OrderDetails) == 0 {
		fmt.Println("[Billz/Payme] OrderDetails is empty, returning nil")
		return nil, nil
	}

	// Handle double-encoded JSON: the web app sends JSON.stringify(payload) so we receive a string
	orderData := txn.OrderDetails
	if len(orderData) > 0 && orderData[0] == '"' {
		// It's a JSON string, need to unmarshal first to get the actual JSON
		var jsonStr string
		if err := json.Unmarshal(orderData, &jsonStr); err != nil {
			fmt.Printf("[Billz/Payme] Failed to unwrap string-encoded JSON: %v\n", err)
			return nil, fmt.Errorf("unwrap order details string: %w", err)
		}
		orderData = []byte(jsonStr)
		fmt.Printf("[Billz/Payme] Unwrapped string-encoded JSON (first 200 chars): %.200s\n", string(orderData))
	}

	var details paymeOrderDetails
	if err := json.Unmarshal(orderData, &details); err != nil {
		fmt.Printf("[Billz/Payme] Failed to parse order details: %v\n", err)
		return nil, fmt.Errorf("parse order details: %w", err)
	}

	fmt.Printf("[Billz/Payme] Parsed details: items=%d, user=%s, totalAmount=%.2f\n",
		len(details.Items), details.User.normalizedID(), details.Totals.totalAmount())

	if len(details.Items) == 0 {
		fmt.Println("[Billz/Payme] No items found in order details")
		return nil, errors.New("order details missing items")
	}

	customerID := details.User.normalizedID()
	if customerID == "" && txn.UserID != nil {
		customerID = txn.UserID.String()
	}
	if customerID == "" {
		return nil, errors.New("customer id missing")
	}

	draft, err := createBillzDraftOrder()
	if err != nil {
		return nil, err
	}

	addedProduct := false
	for i, item := range details.Items {
		productID := item.normalizedProductID()
		fmt.Printf("[Billz/Payme] Item %d: productID=%s, qty=%.2f\n", i, productID, item.normalizedQuantity())
		if productID == "" {
			fmt.Printf("[Billz/Payme] Skipping item %d: empty product ID\n", i)
			continue
		}
		qty := item.normalizedQuantity()
		if qty <= 0 {
			fmt.Printf("[Billz/Payme] Skipping item %d: invalid quantity\n", i)
			continue
		}
		if err := addBillzOrderProduct(draft.ID, item); err != nil {
			fmt.Printf("[Billz/Payme] Failed to add product %s: %v\n", productID, err)
			return nil, err
		}
		fmt.Printf("[Billz/Payme] Product %s added successfully\n", productID)
		addedProduct = true
	}
	if !addedProduct {
		fmt.Println("[Billz/Payme] No valid products were added")
		return nil, errors.New("no valid products in order details")
	}

	if err := attachBillzOrderCustomer(draft.ID, customerID); err != nil {
		return nil, err
	}

	paymentAmount := details.Totals.totalAmount()
	if paymentAmount <= 0 {
		paymentAmount = float64(txn.Amount)
	}
	if paymentAmount <= 0 {
		return nil, errors.New("payment amount missing")
	}

	comment := paymeOrderPaymentComment(details.Checkout.normalizedComment())
	fmt.Printf("[Billz/Payme] Registering payment: amount=%.2f, method=%s\n", paymentAmount, details.Checkout.normalizedPaymentMethod())
	if err := registerBillzOrderPayment(draft.ID, paymentAmount, details.Checkout.normalizedPaymentMethod(), comment); err != nil {
		fmt.Printf("[Billz/Payme] Failed to register payment: %v\n", err)
		return nil, err
	}

	fmt.Printf("[Billz/Payme] Order completed: ID=%s, Number=%s, Type=%s\n", draft.ID, draft.Data.OrderNumber, draft.Data.OrderType)
	return &BillzOrderResult{
		OrderID:     draft.ID,
		OrderNumber: draft.Data.OrderNumber,
		OrderType:   draft.Data.OrderType,
	}, nil
}

func createBillzDraftOrder() (*billzCreateOrderResponse, error) {
	payload := map[string]any{
		"shop_id":    billzShopID,
		"cashbox_id": billzCashboxID,
	}

	opts := BillzRequestOpts{
		Method:  http.MethodPost,
		Path:    "v2/order",
		Body:    payload,
		Query:   map[string]string{"Billz-Response-Channel": billzResponseChannel},
		Headers: map[string]string{"Billz-Response-Channel": billzResponseChannel},
	}

	resp, err := DoBillzRequest(opts)
	if err != nil {
		return nil, fmt.Errorf("create billz order: %w", err)
	}
	if resp.Status < 200 || resp.Status >= 300 {
		return nil, fmt.Errorf("create billz order: status %d body %s", resp.Status, string(resp.Body))
	}

	var result billzCreateOrderResponse
	if err := json.Unmarshal(resp.Body, &result); err != nil {
		return nil, fmt.Errorf("unmarshal billz order response: %w", err)
	}
	if result.ID == "" {
		return nil, errors.New("billz order response missing id")
	}

	return &result, nil
}

func addBillzOrderProduct(orderID string, item paymeOrderItem) error {
	payload := map[string]any{
		"sold_measurement_value": item.normalizedQuantity(),
		"product_id":             item.normalizedProductID(),
		"used_wholesale_price":   false,
		"is_manual":              false,
		"response_type":          "HTTP",
	}

	opts := BillzRequestOpts{
		Method:  http.MethodPost,
		Path:    fmt.Sprintf("v2/order-product/%s", orderID),
		Body:    payload,
		Query:   map[string]string{"Billz-Response-Channel": billzResponseChannel},
		Headers: map[string]string{"Billz-Response-Channel": billzResponseChannel},
	}

	resp, err := DoBillzRequest(opts)
	if err != nil {
		return fmt.Errorf("add product %s: %w", item.normalizedProductID(), err)
	}
	if resp.Status < 200 || resp.Status >= 300 {
		return fmt.Errorf("add product %s: status %d body %s", item.normalizedProductID(), resp.Status, string(resp.Body))
	}
	return nil
}

func attachBillzOrderCustomer(orderID, customerID string) error {
	payload := map[string]any{
		"customer_id":     customerID,
		"check_auth_code": false,
	}

	opts := BillzRequestOpts{
		Method:  http.MethodPut,
		Path:    fmt.Sprintf("v2/order-customer-new/%s", orderID),
		Body:    payload,
		Query:   map[string]string{"Billz-Response-Channel": billzResponseChannel},
		Headers: map[string]string{"Billz-Response-Channel": billzResponseChannel},
	}

	resp, err := DoBillzRequest(opts)
	if err != nil {
		return fmt.Errorf("attach customer: %w", err)
	}
	if resp.Status < 200 || resp.Status >= 300 {
		return fmt.Errorf("attach customer: status %d body %s", resp.Status, string(resp.Body))
	}
	return nil
}

func registerBillzOrderPayment(orderID string, amount float64, method, comment string) error {
	paidAmount := int64(math.Round(amount))
	if paidAmount <= 0 {
		return errors.New("invalid payment amount")
	}

	payload := map[string]any{
		"payments": []map[string]any{
			{
				"company_payment_type_id": billzPaymentTypeID,
				"paid_amount":             paidAmount,
				"company_payment_type": map[string]any{
					"name": billzPaymentTypeName(method),
				},
				"returned_amount": 0,
			},
		},
		"comment":          strings.TrimSpace(comment),
		"with_cashback":    0,
		"without_cashback": false,
		"skip_ofd":         false,
	}

	opts := BillzRequestOpts{
		Method:  http.MethodPost,
		Path:    fmt.Sprintf("v2/order-payment/%s", orderID),
		Body:    payload,
		Query:   map[string]string{"Billz-Response-Channel": billzResponseChannel},
		Headers: map[string]string{"Billz-Response-Channel": billzResponseChannel},
	}

	resp, err := DoBillzRequest(opts)
	if err != nil {
		return fmt.Errorf("register payment: %w", err)
	}
	if resp.Status < 200 || resp.Status >= 300 {
		return fmt.Errorf("register payment: status %d body %s", resp.Status, string(resp.Body))
	}
	return nil
}

func billzPaymentTypeName(method string) string {
	switch strings.ToLower(strings.TrimSpace(method)) {
	case "cash", "nalichniy", "наличные":
		return "Наличные"
	default:
		return "Безналичный расчет"
	}
}

func paymeOrderPaymentComment(existing string) string {
	trimmed := strings.TrimSpace(existing)
	if trimmed == "" {
		return paymePaymentComment
	}
	if strings.Contains(trimmed, paymePaymentComment) {
		return trimmed
	}
	return fmt.Sprintf("%s | %s", trimmed, paymePaymentComment)
}

// BillzOrderItem represents a single item for direct Billz order creation
type BillzOrderItem struct {
	ProductID string
	Quantity  float64
}

// BillzOrderPayload contains data for creating a Billz order directly
type BillzOrderPayload struct {
	Items         []BillzOrderItem
	CustomerID    string
	PaymentMethod string
	TotalAmount   float64
	Comment       string
}

// CreateBillzOrderDirect creates a Billz order from a direct payload (for cash orders)
func CreateBillzOrderDirect(payload BillzOrderPayload) (*BillzOrderResult, error) {
	fmt.Printf("[Billz] CreateBillzOrderDirect called with %d items, total: %.2f\n", len(payload.Items), payload.TotalAmount)

	if len(payload.Items) == 0 {
		return nil, errors.New("no items provided")
	}

	// 1. Create draft order
	fmt.Println("[Billz] Step 1: Creating draft order...")
	draft, err := createBillzDraftOrder()
	if err != nil {
		fmt.Printf("[Billz] Failed to create draft order: %v\n", err)
		return nil, err
	}
	fmt.Printf("[Billz] Draft order created: ID=%s, Number=%s\n", draft.ID, draft.Data.OrderNumber)

	// 2. Add products
	fmt.Println("[Billz] Step 2: Adding products...")
	addedProduct := false
	for i, item := range payload.Items {
		productID := strings.TrimSpace(item.ProductID)
		if productID == "" {
			fmt.Printf("[Billz] Skipping item %d: empty product ID\n", i)
			continue
		}
		qty := item.Quantity
		if qty <= 0 {
			qty = 1
		}

		fmt.Printf("[Billz] Adding product %d: ID=%s, qty=%.2f\n", i, productID, qty)
		if err := addBillzOrderProductDirect(draft.ID, productID, qty); err != nil {
			fmt.Printf("[Billz] Failed to add product %s: %v\n", productID, err)
			return nil, err
		}
		addedProduct = true
		fmt.Printf("[Billz] Product %s added successfully\n", productID)
	}

	if !addedProduct {
		return nil, errors.New("no valid products added to order")
	}

	// 3. Attach customer (optional - skip if no valid Billz customer ID)
	if payload.CustomerID != "" {
		fmt.Printf("[Billz] Step 3: Attaching customer %s...\n", payload.CustomerID)
		if err := attachBillzOrderCustomer(draft.ID, payload.CustomerID); err != nil {
			fmt.Printf("[Billz] Warning: failed to attach customer %s to order %s: %v\n", payload.CustomerID, draft.ID, err)
		}
	} else {
		fmt.Println("[Billz] Step 3: Skipping customer attachment (no customer ID)")
	}

	// 4. Register payment
	fmt.Printf("[Billz] Step 4: Registering payment %.2f (%s)...\n", payload.TotalAmount, payload.PaymentMethod)
	if payload.TotalAmount <= 0 {
		return nil, errors.New("invalid payment amount")
	}

	if err := registerBillzOrderPayment(draft.ID, payload.TotalAmount, payload.PaymentMethod, payload.Comment); err != nil {
		fmt.Printf("[Billz] Failed to register payment: %v\n", err)
		return nil, err
	}
	fmt.Println("[Billz] Payment registered successfully")

	fmt.Printf("[Billz] Order completed: ID=%s, Number=%s, Type=%s\n", draft.ID, draft.Data.OrderNumber, draft.Data.OrderType)
	return &BillzOrderResult{
		OrderID:     draft.ID,
		OrderNumber: draft.Data.OrderNumber,
		OrderType:   draft.Data.OrderType,
	}, nil
}

func addBillzOrderProductDirect(orderID, productID string, quantity float64) error {
	payload := map[string]any{
		"sold_measurement_value": quantity,
		"product_id":             productID,
		"used_wholesale_price":   false,
		"is_manual":              false,
		"response_type":          "HTTP",
	}

	opts := BillzRequestOpts{
		Method:  http.MethodPost,
		Path:    fmt.Sprintf("v2/order-product/%s", orderID),
		Body:    payload,
		Query:   map[string]string{"Billz-Response-Channel": billzResponseChannel},
		Headers: map[string]string{"Billz-Response-Channel": billzResponseChannel},
	}

	resp, err := DoBillzRequest(opts)
	if err != nil {
		return fmt.Errorf("add product %s: %w", productID, err)
	}
	if resp.Status < 200 || resp.Status >= 300 {
		return fmt.Errorf("add product %s: status %d body %s", productID, resp.Status, string(resp.Body))
	}
	return nil
}
