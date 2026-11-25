package trace

import "github.com/swathikrish753/agentic-sre/internal/llm"

// Price is what a model charges, in rupees per million tokens.
type Price struct {
	InputPerMTok  float64
	OutputPerMTok float64
}

// Prices maps a model tag to its price. Local models cost nothing.
// Cloud models are added (from the provider's published price) when Day 21 brings one in.
var Prices = map[string]Price{
	"llama3.2:3b": {},
	"qwen2.5:7b":  {},
}

// Cost returns the rupee cost of one call. ok is false when the model has no known price,
// so a missing price never silently reads as ₹0.
func Cost(model string, u llm.Usage) (inr float64, ok bool) {
	p, ok := Prices[model]
	if !ok {
		return 0, false
	}
	return float64(u.PromptTokens)*p.InputPerMTok/1e6 + float64(u.CompletionTokens)*p.OutputPerMTok/1e6, true
}
