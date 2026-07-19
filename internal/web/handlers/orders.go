package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
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

	content := h.renderOrdersList(orders, user, role, csrf.Token(r), search, minBudgetStr, maxBudgetStr, sortBy)
	layout := ui.Layout(ui.PageParams{
		Title:       "Все заказы",
		Content:     content,
		User:        user,
		CSRFToken:   csrf.Token(r),
		ContextRole: role,
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

	content := h.renderCreateForm(csrf.Token(r))
	layout := ui.Layout(ui.PageParams{
		Title:       "Создать заказ",
		Content:     content,
		User:        user,
		CSRFToken:   csrf.Token(r),
		ContextRole: role,
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

	content := h.renderOrderDetail(order, user, role, csrf.Token(r))
	layout := ui.Layout(ui.PageParams{
		Title:       order.Title,
		Content:     content,
		User:        user,
		CSRFToken:   csrf.Token(r),
		ContextRole: role,
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

func (h *OrdersHandler) renderOrdersList(orders []models.Order, user *models.User, role string, csrfToken string, search, minBudget, maxBudget, sort string) g.Node {
	var createBtn g.Node
	if user != nil && role == "customer" {
		createBtn = html.A(
			html.Href("/orders/new"),
			html.Class("bg-indigo-600 hover:bg-indigo-700 text-white px-4 py-2 rounded-md font-semibold text-sm"),
			g.Text("Создать заказ"),
		)
	}

	var orderCards []g.Node
	for _, o := range orders {
		orderCards = append(orderCards, html.Div(
			html.Class("bg-white p-6 rounded-lg shadow-sm border border-gray-100 hover:shadow-md transition-shadow flex flex-col justify-between space-y-3"),
			html.Div(
				html.Div(
					html.Class("flex justify-between items-start mb-2"),
					html.H3(
						html.Class("text-xl font-bold text-gray-900 line-clamp-1"),
						html.A(html.Href(fmt.Sprintf("/orders/%d", o.ID)), html.Class("hover:text-indigo-600"), g.Text(o.Title)),
					),
					html.Span(
						html.Class("text-lg font-extrabold text-green-600 ml-2 whitespace-nowrap"),
						g.Text(fmt.Sprintf("%.0f ₽", o.Budget)),
					),
				),
				g.If(o.Category != "", html.Div(
					html.Class("mb-3"),
					html.Span(
						html.Class("inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-semibold bg-purple-50 text-purple-700 border border-purple-100"),
						g.Text(o.Category),
					),
				)),
				html.P(html.Class("text-gray-600 mb-3 line-clamp-3 text-sm leading-relaxed"), g.Text(o.Description)),
				g.If(o.RequiredTech != "", html.Div(
					html.Class("flex flex-wrap gap-1 mb-2"),
					g.Group(func() []g.Node {
						var tags []g.Node
						for _, t := range strings.Split(o.RequiredTech, ",") {
							trimmed := strings.TrimSpace(t)
							if trimmed != "" {
								tags = append(tags, renderTechBadge(trimmed))
							}
						}
						return tags
					}()),
				)),
			),
			html.Div(
				html.Class("flex justify-between items-center text-xs text-gray-400 border-t border-gray-100 pt-3"),
				html.Span(g.Text("Заказчик: "+o.Customer.Username)),
				html.Span(g.Text(o.CreatedAt.Format("02.01.2006 15:04"))),
			),
		))
	}

	if len(orderCards) == 0 {
		orderCards = append(orderCards, html.Div(
			html.Class("col-span-full text-center py-12 text-gray-500 bg-white rounded-lg border border-gray-100"),
			g.Text("Заказы не найдены по заданным критериям."),
		))
	}

	return html.Div(
		html.Class("space-y-6"),
		html.Div(
			html.Class("flex justify-between items-center"),
			html.H1(html.Class("text-3xl font-extrabold text-gray-900"), g.Text("Доступные заказы")),
			createBtn,
		),
		html.Div(
			html.Class("grid grid-cols-1 lg:grid-cols-4 gap-8"),
			// Sidebar Filters
			html.Div(
				html.Class("lg:col-span-1 bg-white p-6 rounded-lg shadow-sm border border-gray-100 self-start"),
				html.H2(html.Class("text-lg font-bold text-gray-900 mb-4"), g.Text("Фильтры")),
				html.Form(
					html.Method("GET"),
					html.Class("space-y-4"),
					html.Div(
						html.Label(html.Class("block text-xs font-semibold text-gray-700 mb-1 uppercase tracking-wider"), g.Text("Поиск")),
						html.Input(html.Type("text"), html.Name("search"), html.Value(search), html.Placeholder("Название или описание..."), html.Class("w-full border border-gray-300 rounded px-3 py-2 text-sm focus:ring-indigo-500 focus:border-indigo-500")),
					),
					html.Div(
						html.Label(html.Class("block text-xs font-semibold text-gray-700 mb-1 uppercase tracking-wider"), g.Text("Минимальный бюджет (₽)")),
						html.Input(html.Type("number"), html.Name("min_budget"), html.Value(minBudget), html.Class("w-full border border-gray-300 rounded px-3 py-2 text-sm focus:ring-indigo-500 focus:border-indigo-500")),
					),
					html.Div(
						html.Label(html.Class("block text-xs font-semibold text-gray-700 mb-1 uppercase tracking-wider"), g.Text("Максимальный бюджет (₽)")),
						html.Input(html.Type("number"), html.Name("max_budget"), html.Value(maxBudget), html.Class("w-full border border-gray-300 rounded px-3 py-2 text-sm focus:ring-indigo-500 focus:border-indigo-500")),
					),
					html.Div(
						html.Label(html.Class("block text-xs font-semibold text-gray-700 mb-1 uppercase tracking-wider"), g.Text("Сортировка")),
						html.Select(
							html.Name("sort"),
							html.Class("w-full border border-gray-300 rounded px-3 py-2 text-sm focus:ring-indigo-500 focus:border-indigo-500"),
							html.Option(g.Attr("value", "created_desc"), g.If(sort == "created_desc" || sort == "", g.Attr("selected", "selected")), g.Text("Сначала новые")),
							html.Option(g.Attr("value", "created_asc"), g.If(sort == "created_asc", g.Attr("selected", "selected")), g.Text("Сначала старые")),
							html.Option(g.Attr("value", "budget_desc"), g.If(sort == "budget_desc", g.Attr("selected", "selected")), g.Text("Бюджет: по убыванию")),
							html.Option(g.Attr("value", "budget_asc"), g.If(sort == "budget_asc", g.Attr("selected", "selected")), g.Text("Бюджет: по возрастанию")),
						),
					),
					html.Button(
						html.Type("submit"),
						html.Class("w-full bg-indigo-600 hover:bg-indigo-700 text-white font-semibold py-2 rounded text-sm transition-colors"),
						g.Text("Применить"),
					),
					html.A(
						html.Href("/orders"),
						html.Class("block text-center text-xs text-gray-500 hover:text-indigo-600 mt-2"),
						g.Text("Сбросить все"),
					),
				),
			),
			// Orders Cards List
			html.Div(
				html.Class("lg:col-span-3 grid grid-cols-1 md:grid-cols-2 gap-6"),
				g.Group(orderCards),
			),
		),
	)
}

func (h *OrdersHandler) renderCreateForm(csrfToken string) g.Node {
	presetTechs := []string{"Go", "Python", "TypeScript", "JavaScript", "Rust", "React", "Vue.js", "Docker", "Kubernetes", "PostgreSQL", "Redis", "TailwindCSS", "HTMX", "GitOps", "ML"}

	return html.Div(
		html.Class("max-w-2xl mx-auto bg-white p-8 rounded-2xl shadow-md border border-gray-100"),
		html.H1(html.Class("text-2xl font-bold mb-6 text-gray-900"), g.Text("Создать новый заказ")),
		html.Form(
			html.Action("/orders"),
			html.Method("POST"),
			html.Class("space-y-6"),
			g.Attr("x-data", `{
				category: 'Фронтенд',
				customCategory: '',
				selectedTechs: [],
				customTechInput: '',
				toggleTech(tech) {
					if (this.selectedTechs.includes(tech)) {
						this.selectedTechs = this.selectedTechs.filter(t => t !== tech);
					} else {
						this.selectedTechs.push(tech);
					}
				},
				addCustomTech() {
					let val = this.customTechInput.trim();
					if (val && !this.selectedTechs.includes(val)) {
						this.selectedTechs.push(val);
						this.customTechInput = '';
					}
				},
				removeTech(tech) {
					this.selectedTechs = this.selectedTechs.filter(t => t !== tech);
				},
				get combinedTechs() {
					return this.selectedTechs.join(', ');
				}
			}`),
			html.Input(html.Type("hidden"), html.Name("csrf_token"), html.Value(csrfToken)),
			html.Input(html.Type("hidden"), html.Name("required_tech"), g.Attr(":value", "combinedTechs")),
			
			// Title
			html.Div(
				html.Label(html.Class("block text-sm font-semibold text-gray-700 mb-1"), g.Text("Название задания")),
				html.Input(html.Type("text"), html.Name("title"), html.Required(), html.Placeholder("Например: Разработка REST API на Go"), html.Class("w-full border border-gray-300 rounded-lg px-3.5 py-2.5 focus:ring-2 focus:ring-indigo-500 focus:border-indigo-500 text-sm")),
			),

			// Category Selection (Interactive Pill Buttons)
			html.Div(
				html.Label(html.Class("block text-sm font-semibold text-gray-700 mb-2"), g.Text("Область / Категория")),
				html.Input(html.Type("hidden"), html.Name("category"), g.Attr(":value", "category === 'Другое' ? customCategory : category")),
				html.Div(
					html.Class("flex flex-wrap gap-2 mb-2"),
					g.Group(func() []g.Node {
						cats := []string{"Фронтенд", "Бэкенд", "Фулстак", "GitOps", "DevOps", "Machine Learning", "Другое"}
						var nodes []g.Node
						for _, c := range cats {
							nodes = append(nodes, html.Button(
								html.Type("button"),
								g.Attr("@click", fmt.Sprintf("category = '%s'", c)),
								g.Attr(":class", fmt.Sprintf("category === '%s' ? 'bg-indigo-600 text-white shadow-sm ring-2 ring-indigo-600' : 'bg-gray-100 text-gray-700 hover:bg-gray-200'", c)),
								html.Class("px-3.5 py-1.5 rounded-full text-xs font-semibold transition-all cursor-pointer"),
								g.Text(c),
							))
						}
						return nodes
					}()),
				),
				html.Div(
					g.Attr("x-show", "category === 'Другое'"),
					html.Input(
						html.Type("text"),
						g.Attr("x-model", "customCategory"),
						html.Placeholder("Укажите свою область (например: QA, Blockchain)"),
						html.Class("w-full border border-gray-300 rounded-lg px-3.5 py-2 focus:ring-2 focus:ring-indigo-500 text-sm mt-2"),
					),
				),
			),

			// Tech Stack Selection (Interactive Preset Chips + Custom Addition)
			html.Div(
				html.Label(html.Class("block text-sm font-semibold text-gray-700 mb-2"), g.Text("Необходимые языки и технологии")),
				
				// Selected technologies preview
				html.Div(
					g.Attr("x-show", "selectedTechs.length > 0"),
					html.Class("mb-3 flex flex-wrap gap-1.5 p-3 bg-indigo-50/50 rounded-xl border border-indigo-100"),
					html.Template(
						g.Attr("x-for", "tech in selectedTechs"),
						g.Attr(":key", "tech"),
						html.Span(
							html.Class("inline-flex items-center space-x-1.5 px-3 py-1 rounded-full text-xs font-bold bg-indigo-600 text-white shadow-sm"),
							html.Span(g.Attr("x-text", "tech")),
							html.Button(
								html.Type("button"),
								g.Attr("@click", "removeTech(tech)"),
								html.Class("hover:text-red-200 font-bold ml-1 cursor-pointer"),
								g.Text("×"),
							),
						),
					),
				),

				// Preset chips
				html.Div(
					html.Class("flex flex-wrap gap-1.5 mb-3"),
					g.Group(func() []g.Node {
						var nodes []g.Node
						for _, t := range presetTechs {
							nodes = append(nodes, html.Button(
								html.Type("button"),
								g.Attr("@click", fmt.Sprintf("toggleTech('%s')", t)),
								g.Attr(":class", fmt.Sprintf("selectedTechs.includes('%s') ? 'opacity-40 ring-2 ring-indigo-500 scale-95' : 'hover:scale-105'", t)),
								html.Class("transition-transform cursor-pointer"),
								renderTechBadge(t),
							))
						}
						return nodes
					}()),
				),

				// Custom tech input field
				html.Div(
					html.Class("flex space-x-2"),
					html.Input(
						html.Type("text"),
						g.Attr("x-model", "customTechInput"),
						g.Attr("@keydown.enter.prevent", "addCustomTech()"),
						html.Placeholder("Добавить свою технологию..."),
						html.Class("flex-grow border border-gray-300 rounded-lg px-3.5 py-2 text-sm focus:ring-2 focus:ring-indigo-500"),
					),
					html.Button(
						html.Type("button"),
						g.Attr("@click", "addCustomTech()"),
						html.Class("bg-gray-800 hover:bg-gray-900 text-white px-4 py-2 rounded-lg text-xs font-semibold transition-colors cursor-pointer"),
						g.Text("+ Добавить"),
					),
				),
			),

			// Description
			html.Div(
				html.Label(html.Class("block text-sm font-semibold text-gray-700 mb-1"), g.Text("Описание задачи")),
				html.Textarea(html.Name("description"), html.Required(), html.Rows("5"), html.Placeholder("Подробно опишите требования к задаче..."), html.Class("w-full border border-gray-300 rounded-lg px-3.5 py-2.5 focus:ring-2 focus:ring-indigo-500 text-sm")),
			),

			// Budget
			html.Div(
				html.Label(html.Class("block text-sm font-semibold text-gray-700 mb-1"), g.Text("Бюджет (₽)")),
				html.Input(html.Type("number"), html.Name("budget"), html.Required(), html.Placeholder("50000"), html.Class("w-full border border-gray-300 rounded-lg px-3.5 py-2.5 focus:ring-2 focus:ring-indigo-500 text-sm")),
			),

			// Submit Button
			html.Button(
				html.Type("submit"),
				html.Class("w-full bg-indigo-600 hover:bg-indigo-700 text-white font-semibold py-3 rounded-xl transition-all text-sm shadow-md hover:shadow-indigo-200"),
				g.Text("Опубликовать заказ"),
			),
		),
	)
}

func renderTechBadge(tech string) g.Node {
	t := strings.TrimSpace(tech)
	if t == "" {
		return nil
	}
	lower := strings.ToLower(t)
	colorClass := "bg-slate-100 text-slate-700 border-slate-200"

	switch {
	case strings.Contains(lower, "go") || strings.Contains(lower, "golang"):
		colorClass = "bg-cyan-100 text-cyan-800 border-cyan-200"
	case strings.Contains(lower, "python"):
		colorClass = "bg-amber-100 text-amber-800 border-amber-200"
	case strings.Contains(lower, "typescript") || strings.EqualFold(lower, "ts"):
		colorClass = "bg-blue-100 text-blue-800 border-blue-200"
	case strings.Contains(lower, "javascript") || strings.EqualFold(lower, "js"):
		colorClass = "bg-yellow-100 text-yellow-800 border-yellow-200"
	case strings.Contains(lower, "react") || strings.Contains(lower, "vue") || strings.Contains(lower, "next") || strings.Contains(lower, "htmx"):
		colorClass = "bg-sky-100 text-sky-800 border-sky-200"
	case strings.Contains(lower, "rust"):
		colorClass = "bg-orange-100 text-orange-800 border-orange-200"
	case strings.Contains(lower, "docker") || strings.Contains(lower, "kubernetes") || strings.Contains(lower, "k8s") || strings.Contains(lower, "devops") || strings.Contains(lower, "gitops"):
		colorClass = "bg-indigo-100 text-indigo-800 border-indigo-200"
	case strings.Contains(lower, "postgres") || strings.Contains(lower, "sql") || strings.Contains(lower, "redis"):
		colorClass = "bg-emerald-100 text-emerald-800 border-emerald-200"
	case strings.Contains(lower, "ml") || strings.Contains(lower, "ai") || strings.Contains(lower, "pytorch"):
		colorClass = "bg-rose-100 text-rose-800 border-rose-200"
	}

	return html.Span(
		html.Class("inline-flex items-center px-3 py-1 rounded-full text-xs font-semibold border "+colorClass),
		g.Text(t),
	)
}

func (h *OrdersHandler) renderOrderDetail(order models.Order, user *models.User, role string, csrfToken string) g.Node {
	var bidForm g.Node
	if user != nil && role == "freelancer" && order.CustomerID != user.ID {
		bidForm = html.Div(
			html.Class("mt-8 bg-gray-50 p-6 rounded-lg border border-gray-200"),
			html.H3(html.Class("text-lg font-bold mb-4"), g.Text("Откликнуться на заказ")),
			html.Form(
				html.Action(fmt.Sprintf("/orders/%d/bids", order.ID)),
				html.Method("POST"),
				html.Class("space-y-4"),
				html.Input(html.Type("hidden"), html.Name("csrf_token"), html.Value(csrfToken)),
				html.Div(
					html.Label(html.Class("block text-sm font-semibold text-gray-700 mb-1"), g.Text("Предлагаемая стоимость (₽)")),
					html.Input(html.Type("number"), html.Name("price"), html.Required(), html.Class("w-full border border-gray-300 rounded px-3 py-2 focus:ring-indigo-500 focus:border-indigo-500")),
				),
				html.Div(
					html.Label(html.Class("block text-sm font-semibold text-gray-700 mb-1"), g.Text("Сопроводительное письмо")),
					html.Textarea(html.Name("comment"), html.Required(), html.Rows("3"), html.Class("w-full border border-gray-300 rounded px-3 py-2 focus:ring-indigo-500 focus:border-indigo-500")),
				),
				html.Button(
					html.Type("submit"),
					html.Class("w-full bg-green-600 hover:bg-green-700 text-white font-semibold py-2 rounded transition-colors"),
					g.Text("Отправить отклик"),
				),
			),
		)
	}

	var bidsList []g.Node
	for _, b := range order.Bids {
		// Only show the bid to the Order Owner (Customer) or the Author of the bid (Freelancer)
		if user == nil || (order.CustomerID != user.ID && b.FreelancerID != user.ID) {
			continue
		}
		
		// If current user is the customer and the order is open, they should see an "Accept" button
		var acceptButton g.Node
		if user.ID == order.CustomerID && order.Status == "open" {
			acceptButton = html.Form(
				html.Action(fmt.Sprintf("/orders/%d/bids/%d/accept", order.ID, b.ID)),
				html.Method("POST"),
				html.Class("inline-block"),
				html.Input(html.Type("hidden"), html.Name("csrf_token"), html.Value(csrfToken)),
				html.Button(
					html.Type("submit"),
					html.Class("bg-indigo-600 hover:bg-indigo-700 text-white text-xs font-semibold py-1 px-3 rounded transition-colors"),
					g.Text("Выбрать исполнителем"),
				),
			)
		}

		bidsList = append(bidsList, html.Div(
			html.Class("p-4 border-b border-gray-100 last:border-0"),
			html.Div(
				html.Class("flex justify-between items-start mb-2"),
				html.Div(
					html.P(html.Class("font-bold text-gray-800"), g.Text(b.Freelancer.Username)),
					html.P(html.Class("text-xs text-gray-400"), g.Text(b.CreatedAt.Format("02.01.2006 15:04"))),
				),
				html.Div(
					html.Class("flex items-center space-x-3"),
					html.Span(html.Class("font-bold text-green-600 mr-2"), g.Text(fmt.Sprintf("%.0f ₽", b.Price))),
					acceptButton,
				),
			),
			html.P(html.Class("text-sm text-gray-600"), g.Text(b.Comment)),
		))
	}

	var bidsContainer g.Node
	if len(bidsList) > 0 {
		bidsContainer = html.Div(
			html.Class("mt-8 bg-white rounded-lg shadow-sm border border-gray-100"),
			html.Div(html.Class("p-4 border-b border-gray-100 bg-gray-50 rounded-t-lg"), html.H3(html.Class("font-bold text-gray-700"), g.Text("Отклики исполнителей"))),
			html.Div(g.Group(bidsList)),
		)
	} else {
		bidsContainer = html.Div(
			html.Class("mt-8 text-center text-gray-400 py-6"),
			g.Text("Откликов на это задание пока нет."),
		)
	}

	var chatContainer g.Node
	if (order.Status == "in_progress" || order.Status == "completed") && user != nil && (order.CustomerID == user.ID || (order.FreelancerID != nil && *order.FreelancerID == user.ID)) {
		chatContainer = html.Div(
			html.Class("mt-8 bg-white rounded-lg shadow-sm border border-gray-100 p-6"),
			html.H3(html.Class("text-lg font-bold text-gray-800 mb-4"), g.Text("Чат по заказу")),
			html.Div(
				html.ID("chat-messages"),
				html.Class("h-64 overflow-y-auto p-4 bg-gray-50 rounded border border-gray-150 mb-4 space-y-2"),
				g.Attr("hx-get", fmt.Sprintf("/orders/%d/chat/messages", order.ID)),
				g.Attr("hx-trigger", "load, every 2s"),
				g.Text("Загрузка сообщений..."),
			),
			html.Form(
				g.Attr("hx-post", fmt.Sprintf("/orders/%d/chat/send", order.ID)),
				g.Attr("hx-target", "#chat-messages"),
				g.Attr("hx-swap", "beforeend"),
				g.Attr("hx-on::after-request", "this.reset()"),
				html.Class("flex space-x-2"),
				html.Input(html.Type("hidden"), html.Name("csrf_token"), html.Value(csrfToken)),
				html.Input(
					html.Type("text"),
					html.Name("message"),
					html.Required(),
					html.Placeholder("Введите сообщение..."),
					html.Class("flex-grow border border-gray-300 rounded px-3 py-2 focus:ring-indigo-500 focus:border-indigo-500"),
				),
				html.Button(
					html.Type("submit"),
					html.Class("bg-indigo-600 hover:bg-indigo-700 text-white font-semibold px-4 py-2 rounded transition-colors"),
					g.Text("Отправить"),
				),
			),
		)
	}

	var actionButtons g.Node
	if user != nil && (order.CustomerID == user.ID || (order.FreelancerID != nil && *order.FreelancerID == user.ID)) {
		if order.Status == "in_progress" {
			actionButtons = html.Div(
				html.Class("flex space-x-4 mt-6"),
				html.Form(
					html.Action(fmt.Sprintf("/orders/%d/cancel", order.ID)),
					html.Method("POST"),
					html.Input(html.Type("hidden"), html.Name("csrf_token"), html.Value(csrfToken)),
					html.Button(
						html.Type("submit"),
						html.Class("bg-red-600 hover:bg-red-700 text-white font-semibold py-2 px-4 rounded transition-colors"),
						g.Text("Отказаться от работы (Вернуть в список / Отменить)"),
					),
				),
			)
		}
	}

	return html.Div(
		html.Class("max-w-3xl mx-auto space-y-6"),
		html.Div(
			html.Class("bg-white p-8 rounded-lg shadow-sm border border-gray-100"),
			html.Div(
				html.Class("flex justify-between items-start mb-6"),
				html.Div(
					html.H1(html.Class("text-3xl font-extrabold text-gray-900"), g.Text(order.Title)),
					html.Div(
						html.Class("flex items-center space-x-2 mt-2"),
						html.Span(
							html.Class("inline-block px-2.5 py-0.5 rounded-full text-xs font-semibold "+
								map[string]string{
									"open":        "bg-green-100 text-green-800",
									"in_progress": "bg-blue-100 text-blue-800",
									"completed":   "bg-gray-100 text-gray-800",
									"cancelled":   "bg-red-100 text-red-800",
								}[order.Status]),
							g.Text(map[string]string{
								"open":        "Открыт",
								"in_progress": "В работе",
								"completed":   "Завершен",
								"cancelled":   "Отменен",
							}[order.Status]),
						),
						g.If(order.Category != "", html.Span(
							html.Class("inline-block px-2.5 py-0.5 rounded-full text-xs font-semibold bg-purple-100 text-purple-800"),
							g.Text(order.Category),
						)),
					),
				),
				html.Span(html.Class("text-2xl font-extrabold text-green-600"), g.Text(fmt.Sprintf("%.0f ₽", order.Budget))),
			),
			g.If(order.RequiredTech != "", html.Div(
				html.Class("mb-6 border-b border-gray-150 pb-4"),
				html.H4(html.Class("text-xs font-semibold text-gray-400 uppercase tracking-wider mb-2"), g.Text("Необходимые технологии")),
				html.Div(
					html.Class("flex flex-wrap gap-2"),
					g.Group(func() []g.Node {
						var tags []g.Node
						for _, t := range strings.Split(order.RequiredTech, ",") {
							trimmed := strings.TrimSpace(t)
							if trimmed != "" {
								tags = append(tags, renderTechBadge(trimmed))
							}
						}
						return tags
					}()),
				),
			)),
			html.Div(
				html.Class("prose max-w-none text-gray-700 mb-6 leading-relaxed"),
				g.Text(order.Description),
			),
			html.Div(
				html.Class("flex justify-between items-center text-xs text-gray-400 border-t border-gray-100 pt-4"),
				html.Span(g.Text("Заказчик: "+order.Customer.Username)),
				html.Span(g.Text("Дата публикации: "+order.CreatedAt.Format("02.01.2006 15:04"))),
			),
			actionButtons,
		),
		bidForm,
		bidsContainer,
		chatContainer,
	)
}
