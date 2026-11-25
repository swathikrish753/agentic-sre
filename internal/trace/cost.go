package trace

// pricePerMillion is ₹ per million tokens: {in, out}. Local models are free.
// Cloud models get a row on Day 21, from the provider's published price.
var pricePerMillion = map[string][2]float64{
	"llama3.2:3b": {0, 0},
	"qwen2.5:7b":  {0, 0},
}

// Cost returns the ₹ cost of one model call. known is false when the model has no price yet.
func Cost(model string, tokensIn, tokensOut int) (inr float64, known bool) {
	p, known := pricePerMillion[model]
	return (float64(tokensIn)*p[0] + float64(tokensOut)*p[1]) / 1e6, known
}
