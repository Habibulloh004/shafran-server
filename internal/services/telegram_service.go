package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
)

// TelegramService handles sending notifications to Telegram.
type TelegramService struct {
	botToken    string
	adminChatID string
}

// NewTelegramService creates a new TelegramService.
func NewTelegramService(botToken, adminChatID string) *TelegramService {
	return &TelegramService{
		botToken:    botToken,
		adminChatID: adminChatID,
	}
}

type telegramMessage struct {
	ChatID    string `json:"chat_id"`
	Text      string `json:"text"`
	ParseMode string `json:"parse_mode"`
}

// SendMessage sends a message to specified chat.
func (s *TelegramService) SendMessage(chatID, text string) error {
	if s.botToken == "" {
		log.Println("[Telegram] Bot token not configured")
		return nil
	}

	url := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", s.botToken)

	msg := telegramMessage{
		ChatID:    chatID,
		Text:      text,
		ParseMode: "HTML",
	}

	body, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	resp, err := http.Post(url, "application/json", bytes.NewReader(body))
	if err != nil {
		log.Printf("[Telegram] Failed to send message: %v", err)
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("[Telegram] Unexpected status: %d", resp.StatusCode)
		return fmt.Errorf("telegram returned status %d", resp.StatusCode)
	}

	return nil
}

// SendToAdmin sends a message to the admin chat.
func (s *TelegramService) SendToAdmin(text string) error {
	if s.adminChatID == "" {
		log.Println("[Telegram] Admin chat ID not configured")
		return nil
	}
	return s.SendMessage(s.adminChatID, text)
}

// OrderNotification contains order data for Telegram notification.
type OrderNotification struct {
	OrderID       string
	OrderNumber   string
	Items         []OrderItemNotification
	TotalAmount   float64
	Currency      string
	UserName      string
	UserPhone     string
	PaymentMethod string
	Status        string
}

// OrderItemNotification contains order item data.
type OrderItemNotification struct {
	Name     string
	Quantity int
	Price    float64
	Currency string
}

// FormatPrice formats price with currency and thousand separators.
func FormatPrice(amount float64, currency string) string {
	if currency == "" {
		currency = "UZS"
	}
	// Format with thousand separators
	intAmount := int64(amount)
	str := fmt.Sprintf("%d", intAmount)

	// Add thousand separators
	var result strings.Builder
	length := len(str)
	for i, digit := range str {
		if i > 0 && (length-i)%3 == 0 {
			result.WriteString(",")
		}
		result.WriteRune(digit)
	}

	return result.String() + " " + currency
}

// NotifyNewOrder sends notification about new order to admin chat.
func (s *TelegramService) NotifyNewOrder(order OrderNotification) error {
	if s.adminChatID == "" {
		return nil
	}

	var itemsList strings.Builder
	for i, item := range order.Items {
		itemTotal := item.Price * float64(item.Quantity)
		currency := item.Currency
		if currency == "" {
			currency = order.Currency
		}
		itemsList.WriteString(fmt.Sprintf("%d. <b>%s</b>\n   %d x %s = %s\n",
			i+1,
			item.Name,
			item.Quantity,
			FormatPrice(item.Price, currency),
			FormatPrice(itemTotal, currency),
		))
	}

	paymentMethodText := "Наличными"
	if order.PaymentMethod == "payme" {
		paymentMethodText = "Payme"
	}

	statusText := "⏳ Kutilmoqda"
	if order.Status == "paid" {
		statusText = "✅ To'langan"
	}

	message := fmt.Sprintf(`<b>🛒 YANGI BUYURTMA!</b>
<b>📋 Buyurtma:</b> %s
<b>👤 Mijoz:</b> %s
<b>📞 Telefon:</b> %s
<b>📦 Mahsulotlar:</b>
%s
<b>💰 Jami:</b> %s
<b>💳 To'lov:</b> %s
<b>📍 Status:</b> %s
━━━━━━━━━━━━━━━━━━`,
		order.OrderNumber,
		order.UserName,
		order.UserPhone,
		itemsList.String(),
		FormatPrice(order.TotalAmount, order.Currency),
		paymentMethodText,
		statusText,
	)

	return s.SendToAdmin(strings.TrimSpace(message))
}

// PaymentSuccessNotification contains payment success data.
type PaymentSuccessNotification struct {
	OrderID      string
	OrderNumber  string
	BillzOrderID string
	Amount       float64
	Currency     string
}

// NotifyPaymentSuccess sends notification about successful payment.
func (s *TelegramService) NotifyPaymentSuccess(payment PaymentSuccessNotification) error {
	if s.adminChatID == "" {
		return nil
	}

	message := fmt.Sprintf(`<b>✅ TO'LOV QABUL QILINDI!</b>
<b>📋 Buyurtma:</b> %s
<b>🏪 Billz Order:</b> %s
<b>💰 Summa:</b> %s
<b>💳 Usul:</b> Payme
━━━━━━━━━━━━━━━━━━
<i>Shafran Parfumery</i>`,
		payment.OrderNumber,
		payment.BillzOrderID,
		FormatPrice(payment.Amount, payment.Currency),
	)

	return s.SendToAdmin(strings.TrimSpace(message))
}
