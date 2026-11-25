package trace

import (
	"bytes"
	"testing"
)

func TestWrite(t *testing.T) {
	var diary bytes.Buffer
	Write(&diary, Step{Step: 1, Action: "get_pods", TokensIn: 180, TokensOut: 25})
	Write(&diary, Summary{Outcome: "final_answer", Steps: 1})

	want := `{"step":1,"action":"get_pods","tokens_in":180,"tokens_out":25,"ms":0,"cost_inr":0}` + "\n" +
		`{"run_id":"","model":"","outcome":"final_answer","steps":1,"tokens_in":0,"tokens_out":0,"ms":0,"cost_inr":0}` + "\n"
	if diary.String() != want {
		t.Fatalf("got:\n%s\nwant:\n%s", diary.String(), want)
	}

	Write(nil, Step{}) // no diary: must not panic
}

func TestCost(t *testing.T) {
	if inr, known := Cost("llama3.2:3b", 2_000_000, 500_000); inr != 0 || !known {
		t.Fatalf("local model: %v %v", inr, known)
	}

	pricePerMillion["pretend-cloud"] = [2]float64{10, 40}
	defer delete(pricePerMillion, "pretend-cloud")
	if inr, _ := Cost("pretend-cloud", 2_000_000, 500_000); inr != 40 { // 2×₹10 + 0.5×₹40
		t.Fatalf("cloud model: %v", inr)
	}

	if _, known := Cost("unknown", 1, 1); known {
		t.Fatal("unknown model must say known=false")
	}
}
