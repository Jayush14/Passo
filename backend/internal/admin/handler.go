package admin

import (
	"encoding/json"
	"net/http"

	"passo/backend/internal/config"
)

var cfg = config.LoadConfig()

// Handler functions return dummy data for now.

func HandleGrowth(w http.ResponseWriter, r *http.Request) {
	data := map[string]interface{}{
		"signups_total":           0,
		"signups_last_30_days":    0,
		"onboarding_path_split":   map[string]int{"manual": 0, "resume": 0},
		"onboarding_completion_pct": 0,
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}

func HandleEngagement(w http.ResponseWriter, r *http.Request) {
	data := map[string]interface{}{
		"dau": 0,
		"wau": 0,
		"average_streak": 0,
		"retention_week4": 0,
		"retention_week8": 0,
		"push_open_rate": 0,
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}

func HandleRoadmapHealth(w http.ResponseWriter, r *http.Request) {
	data := map[string]interface{}{
		"roadmaps_generated": 0,
		"avg_tasks_per_roadmap": 0,
		"replan_rate": 0,
		"continue_vs_fresh": map[string]int{"continue": 0, "fresh": 0},
		"deadline_choice": map[string]int{"keep": 0, "extend": 0},
		"most_common_target_roles": []string{},
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}

func HandleAIReliability(w http.ResponseWriter, r *http.Request) {
	data := map[string]interface{}{
		"malformed_json_retry_rate": 0,
		"gemini_error_rate": 0,
		"byok_vs_housekey_calls": map[string]int{"byok": 0, "housekey": 0},
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}

func HandleMonetization(w http.ResponseWriter, r *http.Request) {
	data := map[string]interface{}{
		"active_subscribers": 0,
		"conversion_rate": 0,
		"churn_rate": 0,
		"arr": 0,
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}

func HandleSupport(w http.ResponseWriter, r *http.Request) {
	data := map[string]interface{}{
		"zero_activity_after_signup": 0,
		"top_chat_topics": []string{},
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}