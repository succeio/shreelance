package worker

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"regexp"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"

	"shreelance/internal/config"
	"shreelance/internal/models"
)

type DonationAlertResponse struct {
	Data []DonationItem `json:"data"`
}

type DonationItem struct {
	ID        int64   `json:"id"`
	Name      string  `json:"name"`
	Username  string  `json:"username"`
	Message   string  `json:"message"`
	Amount    float64 `json:"amount"`
	Currency  string  `json:"currency"`
	CreatedAt string  `json:"created_at"`
}

func StartDonationWorker(db *gorm.DB, valkeyClient *redis.Client, cfg *config.Config) {
	ticker := time.NewTicker(1 * time.Hour)
	go func() {
		// Run once immediately on start
		pollDonations(db, valkeyClient, cfg)

		for range ticker.C {
			pollDonations(db, valkeyClient, cfg)
		}
	}()
}

func pollDonations(db *gorm.DB, valkeyClient *redis.Client, cfg *config.Config) {
	ctx := context.Background()

	// Retrieve stored access token from Redis or config
	token, err := valkeyClient.Get(ctx, "donationalerts_access_token").Result()
	if err != nil || token == "" {
		// If token isn't in Redis yet, fallback to API Key if configured
		token = cfg.DonationAlertsAPIKey
	}

	if token == "" {
		log.Println("DonationAlerts Worker: No access token or API key configured, skipping check.")
		return
	}

	reqURL := "https://www.donationalerts.com/api/v1/alerts/donations"
	req, err := http.NewRequest("GET", reqURL, nil)
	if err != nil {
		log.Printf("DonationAlerts Worker: Error creating request: %v\n", err)
		return
	}

	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Printf("DonationAlerts Worker: Request failed: %v\n", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("DonationAlerts Worker: API returned status %d\n", resp.StatusCode)
		return
	}

	var alertResp DonationAlertResponse
	if err := json.NewDecoder(resp.Body).Decode(&alertResp); err != nil {
		log.Printf("DonationAlerts Worker: Failed to decode response: %v\n", err)
		return
	}

	re := regexp.MustCompile(`PRO_ID_(\d+)`)

	for _, item := range alertResp.Data {
		// Check if already processed
		var existing models.ProcessedDonation
		if err := db.Where("donation_id = ?", item.ID).First(&existing).Error; err == nil {
			continue // Already processed
		}

		matches := re.FindStringSubmatch(item.Message)
		if len(matches) < 2 {
			continue // No PRO_ID found in comment
		}

		userID, err := strconv.ParseUint(matches[1], 10, 64)
		if err != nil {
			continue
		}

		var user models.User
		if err := db.First(&user, uint(userID)).Error; err != nil {
			log.Printf("DonationAlerts Worker: User ID %d not found for donation %d\n", userID, item.ID)
			continue
		}

		// Calculate PRO days (100 RUB = 10 days => 1 RUB = 0.1 day)
		daysAdded := int((item.Amount / 100.0) * 10)
		if daysAdded < 1 {
			daysAdded = 1 // Minimum 1 day if any small donation is made
		}

		now := time.Now()
		var proBase time.Time
		if user.ProUntil != nil && user.ProUntil.After(now) {
			proBase = *user.ProUntil
		} else {
			proBase = now
		}

		newProUntil := proBase.Add(time.Duration(daysAdded) * 24 * time.Hour)
		user.ProUntil = &newProUntil
		db.Save(&user)

		// Record processed donation
		processed := models.ProcessedDonation{
			DonationID: item.ID,
			UserID:     user.ID,
			Amount:     item.Amount,
			DaysAdded:  daysAdded,
			CreatedAt:  time.Now(),
		}
		db.Create(&processed)

		log.Printf("DonationAlerts Worker: Processed donation %d (%.2f %s). Added %d PRO days to user ID %d (%s).\n",
			item.ID, item.Amount, item.Currency, daysAdded, user.ID, user.Username)
	}
}