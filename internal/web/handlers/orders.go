package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/alexedwards/scs/v2"
	"github.com/go-chi/chi/v5"
	"github.com/gorilla/csrf"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
	g "maragu.dev/gomponents"
	html "maragu.dev/gomponents/html"

	"shreelance/internal/models"
	"shreelance/internal/ui"
)

type OrdersHandler struct {
	DB      *gorm.DB
	Session *scs.SessionManager
	RDB     *redis.Client
}

func NewOrdersHandler(db *gorm.DB, session *scs.SessionManager) *OrdersHandler {
	// Re-use connection to Valkey if it's stored or setup a client
	// Since main.go connects to Valkey, we can create or inject a client.
	// For simplicity, let's create a client connected to localhost:6379 or use a default one.
	rdb := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})
	return &OrdersHandler{
		DB:      db,
		Session: session,
		RDB:     rdb,
	}
}

func (h *OrdersHandler) List(w http.ResponseWriter, r *http.Request) {
	user, role := GetUserFromSession(h.DB, h.Session, r)

	var orders []models.Order
	query := h.DB.Preload("Customer")
	if user != nil {
		query = query.Where("status = ? OR customer_id = ? OR id IN (SELECT order_id FROM bids WHERE freelancer_id = ?)", "open", user.ID, user.ID)
	} else {
		query = query.Where("status = ?", "open")
	}

	search := r.URL.Query().Get("search")
	if search != "" {
		query = query.Where("title ILIKE ? OR description ILIKE ?", "%"+search+"%", "%"+search+"%")
	}

	minBudgetStr := r.URL.Query().Get("min_budget")
	if minBudgetStr != "" {
		if minBudget, err := strconv.ParseFloat(minBudgetStr, 64); err == nil {
			query = query.Where("budget >= ?", minBudget)
		}
	}

	maxBudgetStr := r.URL.Query().Get("max_budget")
	if maxBudgetStr != "" {
		if maxBudget, err := strconv.ParseFloat(maxBudgetStr, 64); err == nil {
			query = query.Where("budget <= ?", maxBudget)
		}
	}

	sortBy := r.URL.Query().Get("sort")
	switch sortBy {
	case "budget_desc":
		query = query.Order("budget desc")
	case "budget_asc":
		query = query.Order("budget asc")
	case "created_asc":
		query = query.Order("created_at asc")
	default:
		query = query.Order("created_at desc")
	}

	if err := query.Find(&orders).Error; err != nil {
		http.Error(w, "Failed to load orders: "+err.Error(), http.StatusInternalServerError)
		return
	}

	content := ui.OrdersList(orders, user, role, csrf.Token(r), search, minBudgetStr, maxBudgetStr, sortBy)
	layout := ui.Layout(ui.PageParams{
		Title:       "Все заказы",
		Content:     content,
		User:        user,
		CSRFToken:   csrf.Token(r),
		ContextRole: role,
		Theme:       GetThemeFromCookie(r),
	})

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_ = layout.Render(w)
}

func (h *OrdersHandler) CreateForm(w http.ResponseWriter, r *http.Request) {
	user, role := GetUserFromSession(h.DB, h.Session, r)
	if user == nil || role != "customer" {
		http.Error(w, "Доступно только в роли Заказчика", http.StatusForbidden)
		return
	}

	content := ui.OrderCreateForm(csrf.Token(r))
	layout := ui.Layout(ui.PageParams{
		Title:       "Создать заказ",
		Content:     content,
		User:        user,
		CSRFToken:   csrf.Token(r),
		ContextRole: role,
		Theme:       GetThemeFromCookie(r),
	})

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_ = layout.Render(w)
}

func (h *OrdersHandler) Create(w http.ResponseWriter, r *http.Request) {
	user, role := GetUserFromSession(h.DB, h.Session, r)
	if user == nil || role != "customer" {
		http.Error(w, "Доступно только в роли Заказчика", http.StatusForbidden)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	budget, _ := strconv.ParseFloat(r.FormValue("budget"), 64)

	category := r.FormValue("category")
	if category == "other" || category == "" {
		category = r.FormValue("category_custom")
	}

	requiredTech := r.FormValue("required_tech")

	order := models.Order{
		Title:        r.FormValue("title"),
		Description:  r.FormValue("description"),
		Budget:       budget,
		Category:     category,
		RequiredTech: requiredTech,
		CustomerID:   user.ID,
		Status:       "open",
	}

	if err := h.DB.Create(&order).Error; err != nil {
		http.Error(w, "Failed to create order: "+err.Error(), http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/orders", http.StatusSeeOther)
}

func (h *OrdersHandler) Detail(w http.ResponseWriter, r *http.Request) {
	user, role := GetUserFromSession(h.DB, h.Session, r)

	idStr := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid Order ID", http.StatusBadRequest)
		return
	}

	var order models.Order
	if err := h.DB.Preload("Customer").Preload("Bids.Freelancer").Preload("Freelancer").First(&order, id).Error; err != nil {
		http.Error(w, "Order not found", http.StatusNotFound)
		return
	}

	content := ui.OrderDetail(order, user, role, csrf.Token(r))
	layout := ui.Layout(ui.PageParams{
		Title:       "Заказ #" + idStr,
		Content:     content,
		User:        user,
		CSRFToken:   csrf.Token(r),
		ContextRole: role,
		Theme:       GetThemeFromCookie(r),
	})

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_ = layout.Render(w)
}

func (h *OrdersHandler) AcceptBid(w http.ResponseWriter, r *http.Request) {
	user, _ := GetUserFromSession(h.DB, h.Session, r)
	if user == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	orderIdStr := chi.URLParam(r, "id")
	orderId, err := strconv.Atoi(orderIdStr)
	if err != nil {
		http.Error(w, "Invalid Order ID", http.StatusBadRequest)
		return
	}

	bidIdStr := chi.URLParam(r, "bidId")
	bidId, err := strconv.Atoi(bidIdStr)
	if err != nil {
		http.Error(w, "Invalid Bid ID", http.StatusBadRequest)
		return
	}

	var order models.Order
	if err := h.DB.First(&order, orderId).Error; err != nil {
		http.Error(w, "Order not found", http.StatusNotFound)
		return
	}

	if order.CustomerID != user.ID {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	var bid models.Bid
	if err := h.DB.First(&bid, bidId).Error; err != nil {
		http.Error(w, "Bid not found", http.StatusNotFound)
		return
	}

	if bid.OrderID != order.ID {
		http.Error(w, "Bid does not belong to this order", http.StatusBadRequest)
		return
	}

	// Update order & bid status
	tx := h.DB.Begin()
	order.Status = "in_progress"
	order.FreelancerID = &bid.FreelancerID
	if err := tx.Save(&order).Error; err != nil {
		tx.Rollback()
		http.Error(w, "Failed to update order", http.StatusInternalServerError)
		return
	}

	bid.Status = "accepted"
	if err := tx.Save(&bid).Error; err != nil {
		tx.Rollback()
		http.Error(w, "Failed to update bid", http.StatusInternalServerError)
		return
	}

	// Reject all other bids
	if err := tx.Model(&models.Bid{}).Where("order_id = ? AND id != ?", order.ID, bid.ID).Update("status", "rejected").Error; err != nil {
		tx.Rollback()
		http.Error(w, "Failed to reject other bids", http.StatusInternalServerError)
		return
	}

	tx.Commit()

	http.Redirect(w, r, fmt.Sprintf("/orders/%d", order.ID), http.StatusSeeOther)
}

func (h *OrdersHandler) CancelOrder(w http.ResponseWriter, r *http.Request) {
	user, _ := GetUserFromSession(h.DB, h.Session, r)
	if user == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	idStr := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	var order models.Order
	if err := h.DB.First(&order, id).Error; err != nil {
		http.Error(w, "Order not found", http.StatusNotFound)
		return
	}

	// Only customer or the hired freelancer can cancel/resign
	if order.CustomerID != user.ID && (order.FreelancerID == nil || *order.FreelancerID != user.ID) {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	tx := h.DB.Begin()

	if user.ID == order.CustomerID {
		// Customer cancels the order completely
		order.Status = "cancelled"
		order.FreelancerID = nil
	} else {
		// Freelancer resigns, order goes back to 'open' status so other freelancers can bid
		order.Status = "open"
		order.FreelancerID = nil
		// Reset bid status of this freelancer to pending or rejected
		if err := tx.Model(&models.Bid{}).Where("order_id = ? AND freelancer_id = ?", order.ID, user.ID).Update("status", "pending").Error; err != nil {
			tx.Rollback()
			http.Error(w, "Failed to reset bid status", http.StatusInternalServerError)
			return
		}
	}

	if err := tx.Save(&order).Error; err != nil {
		tx.Rollback()
		http.Error(w, "Failed to update order", http.StatusInternalServerError)
		return
	}

	tx.Commit()

	http.Redirect(w, r, fmt.Sprintf("/orders/%d", order.ID), http.StatusSeeOther)
}

type ChatMessage struct {
	SenderID   uint      `json:"sender_id"`
	SenderName string    `json:"sender_name"`
	Text       string    `json:"text"`
	CreatedAt  time.Time `json:"created_at"`
}

func (h *OrdersHandler) SendChatMessage(w http.ResponseWriter, r *http.Request) {
	user, _ := GetUserFromSession(h.DB, h.Session, r)
	if user == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	idStr := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	var order models.Order
	if err := h.DB.First(&order, id).Error; err != nil {
		http.Error(w, "Order not found", http.StatusNotFound)
		return
	}

	if order.CustomerID != user.ID && (order.FreelancerID == nil || *order.FreelancerID != user.ID) {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	text := r.FormValue("message")
	if text == "" {
		http.Error(w, "Empty message", http.StatusBadRequest)
		return
	}

	msg := ChatMessage{
		SenderID:   user.ID,
		SenderName: user.Username,
		Text:       text,
		CreatedAt:  time.Now(),
	}

	data, err := json.Marshal(msg)
	if err != nil {
		http.Error(w, "Failed to marshal message", http.StatusInternalServerError)
		return
	}

	streamKey := fmt.Sprintf("chat:order:%d", order.ID)
	err = h.RDB.XAdd(context.Background(), &redis.XAddArgs{
		Stream: streamKey,
		Values: map[string]interface{}{
			"message": string(data),
		},
	}).Err()

	if err != nil {
		http.Error(w, "Failed to stream message", http.StatusInternalServerError)
		return
	}

	// Render the single new message template back to HTMX
	htmlMsg := html.Div(
		html.Class("p-2 rounded bg-indigo-50 border border-indigo-100"),
		html.P(html.Class("text-xs font-bold text-indigo-700"), g.Text(msg.SenderName)),
		html.P(html.Class("text-sm text-gray-700"), g.Text(msg.Text)),
		html.P(html.Class("text-right text-[10px] text-gray-400"), g.Text(msg.CreatedAt.Format("15:04:05"))),
	)
	_ = htmlMsg.Render(w)
}

func (h *OrdersHandler) GetChatMessages(w http.ResponseWriter, r *http.Request) {
	user, _ := GetUserFromSession(h.DB, h.Session, r)
	if user == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	idStr := chi.URLParam(r, "id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	var order models.Order
	if err := h.DB.First(&order, id).Error; err != nil {
		http.Error(w, "Order not found", http.StatusNotFound)
		return
	}

	if order.CustomerID != user.ID && (order.FreelancerID == nil || *order.FreelancerID != user.ID) {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	streamKey := fmt.Sprintf("chat:order:%d", order.ID)
	streams, err := h.RDB.XRange(context.Background(), streamKey, "-", "+").Result()
	if err != nil && err != redis.Nil {
		http.Error(w, "Failed to load stream messages", http.StatusInternalServerError)
		return
	}

	var renderedMessages []g.Node
	for _, stream := range streams {
		msgStr, ok := stream.Values["message"].(string)
		if !ok {
			continue
		}

		var msg ChatMessage
		if err := json.Unmarshal([]byte(msgStr), &msg); err != nil {
			continue
		}

		bgColor := "bg-gray-100 border-gray-200"
		nameColor := "text-gray-700"
		if msg.SenderID == user.ID {
			bgColor = "bg-indigo-50 border-indigo-100"
			nameColor = "text-indigo-700"
		}

		renderedMessages = append(renderedMessages, html.Div(
			html.Class("p-2 rounded border "+bgColor),
			html.P(html.Class("text-xs font-bold "+nameColor), g.Text(msg.SenderName)),
			html.P(html.Class("text-sm text-gray-700"), g.Text(msg.Text)),
			html.P(html.Class("text-right text-[10px] text-gray-400"), g.Text(msg.CreatedAt.Format("15:04:05"))),
		))
	}

	if len(renderedMessages) == 0 {
		_ = html.P(html.Class("text-center text-gray-400 text-sm py-4"), g.Text("Сообщений нет. Начните диалог!")).Render(w)
		return
	}

	_ = g.Group(renderedMessages).Render(w)
}

func (h *OrdersHandler) CreateBid(w http.ResponseWriter, r *http.Request) {
	user, role := GetUserFromSession(h.DB, h.Session, r)
	if user == nil || role != "freelancer" {
		http.Error(w, "Доступно только в роли Исполнителя", http.StatusForbidden)
		return
	}

	idStr := chi.URLParam(r, "id")
	orderID, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid Order ID", http.StatusBadRequest)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	price, _ := strconv.ParseFloat(r.FormValue("price"), 64)

	bid := models.Bid{
		OrderID:      uint(orderID),
		FreelancerID: user.ID,
		Price:        price,
		Comment:      r.FormValue("comment"),
		Status:       "pending",
	}

	if err := h.DB.Create(&bid).Error; err != nil {
		http.Error(w, "Failed to create bid: "+err.Error(), http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, fmt.Sprintf("/orders/%d", orderID), http.StatusSeeOther)
}

