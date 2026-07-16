package roadmap

import (
	"encoding/json"
	"net/http"

	"passo/backend/internal/config"
)

var cfg = config.LoadConfig()

// GenerateRequest represents the request body for roadmap generation.
type GenerateRequest struct {
	CurrentSkills     []string `json:"current_skills"`
	TargetRole        string   `json:"target_role"`
	HoursPerWeek      int      `json:"hours_per_week"`
	WeeklyTimePattern string   `json:"weekly_time_pattern,omitempty"`
	Timeframe         string   `json:"timeframe,omitempty"` // e.g., "6 months"
	// Optional personalization vectors (see spec)
	CurrentCompany   string `json:"current_company,omitempty"`
	CurrentPackage   string `json:"current_package,omitempty"`
	ExpectedCompany  string `json:"expected_company,omitempty"`
	ExpectedPackage  string `json:"expected_package,omitempty"`
	Education        string `json:"education_background,omitempty"`
	Certifications   []string `json:"certifications,omitempty"`
	PortfolioURL     string `json:"portfolio_url,omitempty"`
	InterviewExperience string `json:"interview_experience,omitempty"`
	LearningStyle    string `json:"learning_style,omitempty"`
	TargetDomain     string `json:"target_domain,omitempty"`
	NetworkAccess    string `json:"network_access,omitempty"`
	AdditionalNotes  string `json:"additional_notes,omitempty"`
}

// GenerateResponse is the roadmap JSON per spec.
type GenerateResponse struct {
	RoadmapID      string `json:"roadmap_id"`
	UserID         string `json:"user_id"`
	Status         string `json:"status"`
	OnboardingSource string `json:"onboarding_source"`
	CurrentSkills  []string `json:"current_skills"`
	TargetRole     string `json:"target_role"`
	HoursPerWeek   int    `json:"hours_per_week"`
	WeeklyTimePattern string `json:"weekly_time_pattern,omitempty"`
	StartDate      string `json:"start_date"`
	TargetEndDate  string `json:"target_end_date"`
	// plus other fields as per spec; we will fill a minimal example.
	Phases []struct {
		PhaseID   string `json:"phase_id"`
		Title     string `json:"title"`
		Order     int    `json:"order"`
		Weeks []struct {
			WeekID   string `json:"week_id"`
			Tag      string `json:"tag"`
			Title    string `json:"title"`
			StartDate string `json:"start_date"`
			EndDate   string `json:"end_date"`
			Tasks []struct {
				TaskID   string `json:"task_id"`
				Title    string `json:"title"`
				Description string `json:"description"`
				Resources []struct {
					Label string `json:"label"`
					URL   string `json:"url"`
				} `json:"resources,omitempty"`
				Checked   bool   `json:"checked"`
				Note      string `json:"note,omitempty"`
				Subtasks []struct {
					SubtaskID string `json:"subtask_id"`
					Text      string `json:"text"`
					Checked   bool   `json:"checked"`
				} `json:"subtasks,omitempty"`
			} `json:"tasks"`
		} `json:"weeks"`
	} `json:"phases"`
	CreatedAt string `json:"created_at"`
	LastRepacedAt *string `json:"last_repaced_at,omitempty"`
}

// HandleGenerate returns a dummy roadmap.
func HandleGenerate(w http.ResponseWriter, r *http.Request) {
	var req GenerateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}
	// Validate required fields
	if len(req.CurrentSkills) == 0 || req.TargetRole == "" || req.HoursPerWeek <= 0 {
		http.Error(w, "missing required fields", http.StatusBadRequest)
		return
	}
	// Build a simple hardcoded response matching schema (simplified)
	resp := GenerateResponse{
		RoadmapID:      "dummy-roadmap-id",
		UserID:         "dummy-user-id",
		Status:         "active",
		OnboardingSource: "manual",
		CurrentSkills:  req.CurrentSkills,
		TargetRole:     req.TargetRole,
		HoursPerWeek:   req.HoursPerWeek,
		WeeklyTimePattern: req.WeeklyTimePattern,
		StartDate:      "2026-07-13",
		TargetEndDate:  "2026-12-07",
		CreatedAt:      "2026-07-13T09:00:00Z",
		Phases: []struct {
			PhaseID   string `json:"phase_id"`
			Title     string `json:"title"`
			Order     int    `json:"order"`
			Weeks []struct {
				WeekID   string `json:"week_id"`
				Tag      string `json:"tag"`
				Title    string `json:"title"`
				StartDate string `json:"start_date"`
				EndDate   string `json:"end_date"`
				Tasks []struct {
					TaskID   string `json:"task_id"`
					Title    string `json:"title"`
					Description string `json:"description"`
					Resources []struct {
						Label string `json:"label"`
						URL   string `json:"url"`
					} `json:"resources,omitempty"`
					Checked   bool   `json:"checked"`
					Note      string `json:"note,omitempty"`
					Subtasks []struct {
						SubtaskID string `json:"subtask_id"`
						Text      string `json:"text"`
						Checked   bool   `json:"checked"`
					} `json:"subtasks,omitempty"`
				} `json:"tasks"`
			} `json:"weeks"`
		}{
			{
				PhaseID:   "p1",
				Title:     "Phase 1 — Foundation",
				Order:     1,
				Weeks: []struct {
					WeekID   string `json:"week_id"`
					Tag      string `json:"tag"`
					Title    string `json:"title"`
					StartDate string `json:"start_date"`
					EndDate   string `json:"end_date"`
					Tasks []struct {
						TaskID   string `json:"task_id"`
						Title    string `json:"title"`
						Description string `json:"description"`
						Resources []struct {
							Label string `json:"label"`
							URL   string `json:"url"`
						} `json:"resources,omitempty"`
						Checked   bool   `json:"checked"`
						Note      string `json:"note,omitempty"`
						Subtasks []struct {
							SubtaskID string `json:"subtask_id"`
							Text      string `json:"text"`
							Checked   bool   `json:"checked"`
						} `json:"subtasks,omitempty"`
					} `json:"tasks"`
				}{
					{
						WeekID:   "w1",
						Tag:      "W1-2",
						Title:    "Go syntax & core language",
						StartDate: "2026-07-13",
						EndDate:   "2026-07-26",
						Tasks: []struct {
							TaskID   string `json:"task_id"`
							Title    string `json:"title"`
							Description string `json:"description"`
							Resources []struct {
								Label string `json:"label"`
								URL   string `json:"url"`
							} `json:"resources,omitempty"`
							Checked   bool   `json:"checked"`
							Note      string `json:"note,omitempty"`
							Subtasks []struct {
								SubtaskID string `json:"subtask_id"`
								Text      string `json:"text"`
								Checked   bool   `json:"checked"`
							} `json:"subtasks,omitempty"`
						}{
							{
								TaskID:   "w1_0",
								Title:    "Finish 'A Tour of Go' end to end",
								Description: "Work through Go's official interactive tour, covering basic syntax, structs, methods, and interfaces. Type out every exercise rather than skimming — muscle memory pays off once you hit concurrency in week 3. Budget roughly 4-5 hours across the week.",
								Resources: []struct {
									Label string `json:"label"`
									URL   string `json:"url"`
								}{
									{Label: "A Tour of Go (official)", URL: "https://go.dev/tour/"},
								},
								Checked: false,
								Note:    "",
								Subtasks: []struct {
									SubtaskID string `json:"subtask_id"`
									Text      string `json:"text"`
									Checked   bool   `json:"checked"`
								}{
									{SubtaskID: "w1_0_a", Text: "Complete Basics section", Checked: false},
									{SubtaskID: "w1_0_b", Text: "Complete Methods and Interfaces section", Checked: false},
									{SubtaskID: "w1_0_c", Text: "Complete Concurrency intro section", Checked: false},
								},
							},
						},
					},
				},
			},
		},
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// HandleReplan handles replanning requests (continue vs fresh, deadline choice).
func HandleReplan(w http.ResponseWriter, r *http.Request) {
	// TODO: implement
	_ = r
	w.Header().Set("Content-Type", "application/json")
	// Return a dummy warning and new dates
	response := map[string]interface{}{
		"warning": nil,
		"new_target_end_date": "2026-12-07",
		"repaced_weeks": []interface{}{},
	}
	json.NewEncoder(w).Encode(response)
}