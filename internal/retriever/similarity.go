package retriever

import "math"

func Cosine(a, b []float32) float64 {
	var dot, na, nb float64
	for i := range a {
		ai, bi := float64(a[i]), float64(b[i])
		dot += ai * bi
		na += ai * ai
		nb += bi * bi
	}
	return dot / (math.Sqrt(na) * math.Sqrt(nb))
}