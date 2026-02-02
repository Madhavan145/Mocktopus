package http

import (
	"encoding/json"
	"math"
	"net/http"
	"strings"

	"moktopus/cost-service/internal/model"
	"moktopus/cost-service/internal/store"
)

// Base pricing tiers (when no specific instance config provided)
var basePricingMonthly = map[string]float64{
	"service":       90,
	"worker":        75,
	"api-gateway":   40,
	"load-balancer": 35,
	"database":      120,
	"cache":         55,
	"queue":         25,
	"cdn":           20,
	"storage":       10,
	"monitoring":    15,
}

// Provider-specific pricing multipliers
var providerMultiplier = map[string]float64{
	"aws":   1.0,
	"gcp":   0.92,
	"azure": 0.96,
}

// Instance-aware pricing for cache nodes
var cachePricing = map[string]float64{
	"cache.t4g.micro":   11.52,
	"cache.t4g.small":   23.04,
	"cache.t4g.medium":  46.80,
	"cache.r7g.large":   181.44,
	"cache.r7g.xlarge":  362.88,
	"cache.r7g.2xlarge": 725.76,
	"cache.m7g.large":   113.76,
	"M1":                35.04,
	"M2":                48.96,
	"M3":                97.92,
	"M4":                244.80,
	"C0 Basic":          16.06,
	"C1 Basic":          40.15,
	"C2 Standard":       81.76,
	"P1 Premium":        426.32,
}

// Instance-aware pricing for database nodes
var dbPricing = map[string]float64{
	"db.t4g.micro":      11.52,
	"db.t4g.small":      23.04,
	"db.t4g.medium":     46.80,
	"db.r6g.large":      187.20,
	"db.r6g.xlarge":     374.40,
	"db.r6g.2xlarge":    748.80,
	"db-f1-micro":       10.95,
	"db-g1-small":       36.50,
	"db-custom-2-8192":  94.54,
	"db-custom-4-16384": 189.07,
	"B1ms":              23.36,
	"B2s":               46.72,
	"D2s_v3":            90.52,
	"D4s_v3":            181.04,
}

// Compute instance pricing
var computePricing = map[string]float64{
	"t3.micro":      7.49,
	"t3.small":      14.98,
	"t3.medium":     29.95,
	"t3.large":      59.90,
	"m6i.large":     69.12,
	"m6i.xlarge":    138.24,
	"m6i.2xlarge":   276.48,
	"c6i.large":     61.20,
	"c6i.xlarge":    122.40,
	"e2-micro":      6.11,
	"e2-small":      12.23,
	"e2-standard-2": 48.92,
	"n2-standard-2": 70.91,
	"n2-standard-4": 141.82,
	"B1s":           7.59,
	"B2s":           30.37,
	"D2s_v5":        70.08,
	"D4s_v5":        140.16,
}

type costHandler struct {
	store store.Store
}

func NewHandler(st store.Store) http.Handler {
	h := &costHandler{store: st}
	mux := http.NewServeMux()
	mux.HandleFunc("/health", health)
	mux.HandleFunc("/api/cost/estimate", h.estimate)
	mux.HandleFunc("/api/v1/cost/estimate", h.estimateV1)
	mux.HandleFunc("/api/v1/cost/history/", h.getHistory)
	return mux
}

func health(w http.ResponseWriter, _ *http.Request) {
	respondJSON(w, http.StatusOK, map[string]string{"service": "cost-service", "status": "ok"})
}

func calculateCost(req model.EstimateRequest) model.EstimateResponse {
	provider := strings.ToLower(req.Provider)
	if provider == "" {
		provider = "aws"
	}
	multiplier := providerMultiplier[provider]
	if multiplier == 0 {
		multiplier = 1.0
	}

	items := make([]model.CostLineItem, 0, len(req.Topology.Nodes))
	total := 0.0
	breakdown := model.CostBreakdown{}

	for _, node := range req.Topology.Nodes {
		t := strings.TrimSpace(strings.ToLower(node.Type))
		label := getStringConfig(node.Config, "label", node.ID)
		instanceType := getStringConfig(node.Config, "instanceType", "")
		nodeType := getStringConfig(node.Config, "nodeType", "")
		instanceClass := getStringConfig(node.Config, "instanceClass", "")
		replicas := getIntConfig(node.Config, "replicas", 1)

		unitCost := 0.0
		tier := "standard"

		switch t {
		case "service", "worker":
			if price, ok := computePricing[instanceType]; ok {
				unitCost = price
				tier = instanceType
			} else {
				unitCost = basePricingMonthly[t]
			}
			breakdown.Compute += unitCost * float64(replicas) * multiplier

		case "database":
			ic := instanceClass
			if ic == "" {
				ic = instanceType
			}
			if price, ok := dbPricing[ic]; ok {
				unitCost = price
				tier = ic
			} else {
				unitCost = basePricingMonthly["database"]
			}
			breakdown.Database += unitCost * float64(replicas) * multiplier

		case "cache":
			nt := nodeType
			if nt == "" {
				nt = instanceType
			}
			if price, ok := cachePricing[nt]; ok {
				unitCost = price
				tier = nt
			} else {
				unitCost = basePricingMonthly["cache"]
			}
			breakdown.Cache += unitCost * float64(replicas) * multiplier

		case "queue":
			unitCost = basePricingMonthly["queue"]
			breakdown.Queue += unitCost * multiplier

		case "load-balancer", "api-gateway":
			unitCost = basePricingMonthly[t]
			breakdown.Network += unitCost * multiplier

		case "cdn":
			unitCost = basePricingMonthly["cdn"]
			breakdown.Network += unitCost * multiplier

		case "storage":
			unitCost = basePricingMonthly["storage"]
			breakdown.Storage += unitCost * multiplier

		default:
			unitCost = basePricingMonthly["service"]
			if unitCost == 0 {
				unitCost = 50
			}
			breakdown.Other += unitCost * multiplier
		}

		subtotal := round(unitCost * float64(replicas) * multiplier)
		total += subtotal

		items = append(items, model.CostLineItem{
			ResourceType: t,
			NodeID:       node.ID,
			Label:        label,
			InstanceType: tier,
			Count:        replicas,
			UnitMonthly:  round(unitCost * multiplier),
			Subtotal:     subtotal,
			Provider:     provider,
			Tier:         tier,
		})
	}

	// Generate multi-cloud comparison
	comparison := make([]model.ProviderCost, 0, 3)
	maxCost := 0.0
	for p, mult := range providerMultiplier {
		pCost := round(total / multiplier * mult) // normalize then apply
		if pCost > maxCost {
			maxCost = pCost
		}
		comparison = append(comparison, model.ProviderCost{
			Provider: p,
			Total:    pCost,
		})
	}
	for i := range comparison {
		if maxCost > 0 {
			comparison[i].Savings = round((1 - comparison[i].Total/maxCost) * 100)
		}
	}

	return model.EstimateResponse{
		ProjectID:      req.ProjectID,
		Currency:       "USD",
		Total:          round(total),
		MonthlyTotal:   round(total),
		AnnualEstimate: round(total * 12),
		Items:          items,
		Breakdown:      breakdown,
		Comparison:     comparison,
	}
}

func (h *costHandler) estimate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req model.EstimateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid payload", http.StatusBadRequest)
		return
	}

	resp := calculateCost(req)
	h.persistEstimate(req, resp)
	respondJSON(w, http.StatusOK, resp)
}

func (h *costHandler) estimateV1(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		respondV1Error(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "method not allowed")
		return
	}

	var req model.EstimateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondV1Error(w, http.StatusBadRequest, "INVALID_PAYLOAD", "invalid payload")
		return
	}

	resp := calculateCost(req)
	h.persistEstimate(req, resp)
	respondV1Data(w, http.StatusOK, resp)
}

func (h *costHandler) getHistory(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		respondV1Error(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "method not allowed")
		return
	}

	projectID := strings.TrimPrefix(r.URL.Path, "/api/v1/cost/history/")
	projectID = strings.TrimSpace(projectID)
	if projectID == "" {
		respondV1Error(w, http.StatusBadRequest, "INVALID_PROJECT_ID", "project id required")
		return
	}

	if h.store == nil {
		respondV1Data(w, http.StatusOK, []any{})
		return
	}

	records, err := h.store.GetEstimates(projectID, 20)
	if err != nil {
		respondV1Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", "failed to get estimates")
		return
	}
	respondV1Data(w, http.StatusOK, records)
}

func (h *costHandler) persistEstimate(req model.EstimateRequest, resp model.EstimateResponse) {
	if h.store == nil {
		return
	}
	region := req.Region
	if region == "" {
		region = "us-east-1"
	}
	provider := req.Provider
	if provider == "" {
		provider = "aws"
	}
	_ = h.store.SaveEstimate(&store.CostRecord{
		ProjectID:      req.ProjectID,
		Provider:       provider,
		Region:         region,
		MonthlyTotal:   resp.MonthlyTotal,
		AnnualEstimate: resp.AnnualEstimate,
		Breakdown:      resp.Breakdown,
		LineItems:      resp.Items,
		Comparison:     resp.Comparison,
	})
}

func getStringConfig(config map[string]any, key, fallback string) string {
	if v, ok := config[key]; ok {
		if s, ok := v.(string); ok && s != "" {
			return s
		}
	}
	return fallback
}

func getIntConfig(config map[string]any, key string, fallback int) int {
	if v, ok := config[key]; ok {
		switch n := v.(type) {
		case float64:
			return int(n)
		case int:
			return n
		}
	}
	return fallback
}

func respondJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

type apiEnvelope struct {
	Success bool      `json:"success"`
	Data    any       `json:"data"`
	Error   *apiError `json:"error"`
}

type apiError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func respondV1Data(w http.ResponseWriter, status int, payload any) {
	respondJSON(w, status, apiEnvelope{Success: true, Data: payload, Error: nil})
}

func respondV1Error(w http.ResponseWriter, status int, code, message string) {
	respondJSON(w, status, apiEnvelope{Success: false, Data: nil, Error: &apiError{Code: code, Message: message}})
}

func round(value float64) float64 {
	return math.Round(value*100) / 100
}
