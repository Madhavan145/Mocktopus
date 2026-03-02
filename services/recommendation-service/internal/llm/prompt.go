package llm

import (
	"encoding/json"
	"fmt"
	"strings"
)

const SystemPrompt = `You are an expert distributed systems architect analyzing simulation results from a microservice topology stress test.

You will receive:
1. Topology information (nodes, types, connections)
2. Simulation metrics (latency, throughput, error rates, queue depths, CPU estimates)

Your job is to provide actionable recommendations to improve the system's performance, reliability, and cost efficiency.

Return your analysis as a JSON array of recommendation objects. Each object must have:
- "problem": A short description of the identified issue
- "reason": Why this is a problem in this specific topology context
- "fix": The specific action to take
- "expectedLatencyDeltaPct": Expected latency improvement as a negative percentage (e.g. -25 means 25% reduction)
- "expectedCostDeltaPct": Expected cost change as a percentage (positive = more expensive)
- "riskScore": How confident you are this is a real problem (0.0 to 1.0, higher = more risky if unaddressed)
- "priority": 1 (critical), 2 (important), 3 (nice-to-have)

Consider:
- Cascading failure risks
- Queue saturation and backpressure
- Connection pool exhaustion
- Cold start penalties
- Cache invalidation storms
- Circuit breaker thresholds
- Autoscaling lag
- Data consistency under load

Return ONLY the JSON array, no other text.`

// BuildUserMessage constructs the user message from topology and metrics data.
func BuildUserMessage(topology any, metrics any) string {
	var sb strings.Builder
	sb.WriteString("## Topology\n```json\n")
	t, _ := json.MarshalIndent(topology, "", "  ")
	sb.Write(t)
	sb.WriteString("\n```\n\n## Simulation Metrics\n```json\n")
	m, _ := json.MarshalIndent(metrics, "", "  ")
	sb.Write(m)
	sb.WriteString("\n```\n\nAnalyze and provide recommendations as a JSON array.")
	return sb.String()
}

// LLMRecommendation is the expected shape from the LLM response.
type LLMRecommendation struct {
	Problem                 string  `json:"problem"`
	Reason                  string  `json:"reason"`
	Fix                     string  `json:"fix"`
	ExpectedLatencyDeltaPct float64 `json:"expectedLatencyDeltaPct"`
	ExpectedCostDeltaPct    float64 `json:"expectedCostDeltaPct"`
	RiskScore               float64 `json:"riskScore"`
	Priority                int     `json:"priority"`
}

// ParseLLMRecommendations tries to extract recommendations from raw LLM output.
func ParseLLMRecommendations(content string) ([]LLMRecommendation, error) {
	content = strings.TrimSpace(content)
	// Handle markdown code blocks
	if strings.HasPrefix(content, "```") {
		lines := strings.Split(content, "\n")
		start := 1
		end := len(lines) - 1
		if end > start {
			content = strings.Join(lines[start:end], "\n")
		}
	}

	var recs []LLMRecommendation
	if err := json.Unmarshal([]byte(content), &recs); err != nil {
		return nil, fmt.Errorf("failed to parse LLM recommendations: %w", err)
	}
	return recs, nil
}
