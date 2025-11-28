package utils

import (
	"bytes"
	"testing"
)

func benchmarkMarshall(b *testing.B) {
	var buf bytes.Buffer
	simpleMap := make(map[string]any)
	simpleMap["a"] = 1
	simpleMap["b"] = "bee"
	simpleMap["c"] = [3]float64{1, 2, 3}
	simpleMap["d"] = [3]string{"one", "two", "three"}

	for b.Loop() {
		marshalValue(simpleMap, &buf, 0)
	}
}

func BenchmarkMarshall(b *testing.B) { benchmarkMarshall(b) }
