package engine

import (
	"math"
	"testing"
)

func TestParseQuantity(t *testing.T) {
	tests := []struct {
		name         string
		input        string
		expectedVal  float64
		expectedDim  UnitDimension
		expectedCanon float64
	}{
		{
			name:         "Distance in km",
			input:        "The runner completed 42.195 km today",
			expectedVal:  42.195,
			expectedDim:  DimensionLength,
			expectedCanon: 42195.0,
		},
		{
			name:         "Mass in kg",
			input:        "Payload mass is 75.5 kg",
			expectedVal:  75.5,
			expectedDim:  DimensionMass,
			expectedCanon: 75.5,
		},
		{
			name:         "Data size in GB",
			input:        "Database size reached 16 gb",
			expectedVal:  16.0,
			expectedDim:  DimensionData,
			expectedCanon: 16.0 * 1073741824.0,
		},
		{
			name:         "Currency in USD prefix",
			input:        "Subscription cost is $150",
			expectedVal:  150.0,
			expectedDim:  DimensionCurrency,
			expectedCanon: 150.0,
		},
		{
			name:         "Dimensionless integer",
			input:        "The committee approved 12 proposals",
			expectedVal:  12.0,
			expectedDim:  DimensionDimensionless,
			expectedCanon: 12.0,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			q, err := ParseQuantity(tc.input)
			if err != nil {
				t.Fatalf("failed parsing quantity: %v", err)
			}
			if math.Abs(q.RawValue-tc.expectedVal) > 1e-4 {
				t.Errorf("expected raw value %f, got %f", tc.expectedVal, q.RawValue)
			}
			if q.Dimension != tc.expectedDim {
				t.Errorf("expected dimension %s, got %s", tc.expectedDim, q.Dimension)
			}
			if math.Abs(q.CanonicalValue-tc.expectedCanon) > 1e-4 {
				t.Errorf("expected canonical value %f, got %f", tc.expectedCanon, q.CanonicalValue)
			}
		})
	}
}

func TestCheckNumericalConsistency_EquivalenceAndConversion(t *testing.T) {
	// 5 km in claim vs 5000 m in premise -> CONSISTENT
	res := CheckNumericalConsistency("Distance was 5 km", "Distance measured exactly 5000 m", 1.0)
	if res.Verdict != VerdictConsistent {
		t.Errorf("expected CONSISTENT, got %s (%s)", res.Verdict, res.Explanation)
	}
	if res.RelativeError > 1e-6 {
		t.Errorf("expected near zero relative error, got %f", res.RelativeError)
	}
}

func TestCheckNumericalConsistency_WithinTolerance(t *testing.T) {
	// 101 m vs 100 m with 2% tolerance -> CONSISTENT (1% error <= 2%)
	res := CheckNumericalConsistency("Height is 101 m", "Building height is 100 m", 2.0)
	if res.Verdict != VerdictConsistent {
		t.Errorf("expected CONSISTENT within tolerance, got %s", res.Verdict)
	}
}

func TestCheckNumericalConsistency_Contradiction(t *testing.T) {
	// 150 km vs 100 km with 5% tolerance -> CONTRADICTION (50% error > 5%)
	res := CheckNumericalConsistency("Traveled 150 km", "Traveled 100 km", 5.0)
	if res.Verdict != VerdictContradiction {
		t.Errorf("expected CONTRADICTION, got %s", res.Verdict)
	}
	if math.Abs(res.RelativeError-0.5) > 1e-4 {
		t.Errorf("expected 0.5 relative error, got %f", res.RelativeError)
	}
}

func TestCheckNumericalConsistency_DimensionMismatch(t *testing.T) {
	// 50 kg (Mass) vs 50 km (Length) -> DIMENSION_MISMATCH
	res := CheckNumericalConsistency("The parcel weighed 50 kg", "The route was 50 km", 5.0)
	if res.Verdict != VerdictDimensionMismatch {
		t.Errorf("expected DIMENSION_MISMATCH, got %s", res.Verdict)
	}
}

func TestCheckNumericalConsistency_Ungrounded(t *testing.T) {
	res := CheckNumericalConsistency("No numbers here", "Still no numbers", 5.0)
	if res.Verdict != VerdictUngrounded {
		t.Errorf("expected UNGROUNDED, got %s", res.Verdict)
	}
}
