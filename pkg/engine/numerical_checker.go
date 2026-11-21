package engine

import (
	"errors"
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"
)

// UnitDimension categorizes physical and computational quantities.
type UnitDimension string

const (
	DimensionLength        UnitDimension = "LENGTH"
	DimensionMass          UnitDimension = "MASS"
	DimensionTime          UnitDimension = "TIME"
	DimensionData          UnitDimension = "DATA"
	DimensionCurrency      UnitDimension = "CURRENCY"
	DimensionDimensionless UnitDimension = "DIMENSIONLESS"
)

// NumericalVerdict classifies the consistency of numerical assertions.
type NumericalVerdict string

const (
	VerdictConsistent        NumericalVerdict = "CONSISTENT"
	VerdictContradiction     NumericalVerdict = "CONTRADICTION"
	VerdictDimensionMismatch NumericalVerdict = "DIMENSION_MISMATCH"
	VerdictUngrounded        NumericalVerdict = "UNGROUNDED"
)

var (
	ErrInvalidQuantity = errors.New("cannot parse valid quantity expression")
)

type unitDef struct {
	dimension UnitDimension
	scale     float64 // Multiplier to reach base SI unit
}

var unitRegistry = map[string]unitDef{
	// Length (base: meter)
	"m":  {dimension: DimensionLength, scale: 1.0},
	"km": {dimension: DimensionLength, scale: 1000.0},
	"cm": {dimension: DimensionLength, scale: 0.01},
	"mm": {dimension: DimensionLength, scale: 0.001},

	// Mass (base: kg)
	"kg": {dimension: DimensionMass, scale: 1.0},
	"g":  {dimension: DimensionMass, scale: 0.001},
	"mg": {dimension: DimensionMass, scale: 1e-6},

	// Time (base: seconds)
	"s":    {dimension: DimensionTime, scale: 1.0},
	"sec":  {dimension: DimensionTime, scale: 1.0},
	"min":  {dimension: DimensionTime, scale: 60.0},
	"h":    {dimension: DimensionTime, scale: 3600.0},
	"hr":   {dimension: DimensionTime, scale: 3600.0},
	"day":  {dimension: DimensionTime, scale: 86400.0},
	"days": {dimension: DimensionTime, scale: 86400.0},

	// Data (base: bytes)
	"b":  {dimension: DimensionData, scale: 1.0},
	"kb": {dimension: DimensionData, scale: 1024.0},
	"mb": {dimension: DimensionData, scale: 1048576.0},
	"gb": {dimension: DimensionData, scale: 1073741824.0},
	"tb": {dimension: DimensionData, scale: 1099511627776.0},

	// Currency (base: USD)
	"$":   {dimension: DimensionCurrency, scale: 1.0},
	"usd": {dimension: DimensionCurrency, scale: 1.0},
	"eur": {dimension: DimensionCurrency, scale: 1.08},
	"gbp": {dimension: DimensionCurrency, scale: 1.27},
}

// Quantity represents a parsed numerical assertion.
type Quantity struct {
	RawValue       float64       `json:"raw_value"`
	Unit           string        `json:"unit"`
	Dimension      UnitDimension `json:"dimension"`
	CanonicalValue float64       `json:"canonical_value"`
}

var quantityRegex = regexp.MustCompile(`(?i)([$€£])?\s*([-+]?\d+(?:\.\d+)?)\s*([a-zA-Z]+)?`)

// ParseQuantity parses a numeric string and optional unit into a canonical Quantity.
func ParseQuantity(text string) (*Quantity, error) {
	trimmed := strings.TrimSpace(text)
	match := quantityRegex.FindStringSubmatch(trimmed)
	if match == nil || len(match) < 3 {
		return nil, ErrInvalidQuantity
	}

	prefixCurrency := match[1]
	numberStr := match[2]
	suffixUnit := strings.ToLower(strings.TrimSpace(match[3]))

	val, err := strconv.ParseFloat(numberStr, 64)
	if err != nil {
		return nil, err
	}

	var dim UnitDimension = DimensionDimensionless
	var scale float64 = 1.0
	var finalUnit string = ""

	if prefixCurrency != "" {
		dim = DimensionCurrency
		finalUnit = "usd"
		if prefixCurrency == "€" {
			finalUnit = "eur"
			scale = 1.08
		} else if prefixCurrency == "£" {
			finalUnit = "gbp"
			scale = 1.27
		}
	} else if suffixUnit != "" {
		if uDef, exists := unitRegistry[suffixUnit]; exists {
			dim = uDef.dimension
			scale = uDef.scale
			finalUnit = suffixUnit
		} else {
			finalUnit = suffixUnit
		}
	}

	return &Quantity{
		RawValue:       val,
		Unit:           finalUnit,
		Dimension:      dim,
		CanonicalValue: val * scale,
	}, nil
}

// NumericalEvaluation captures the outcome of verifying claim vs premise quantities.
type NumericalEvaluation struct {
	Verdict        NumericalVerdict `json:"verdict"`
	RelativeError  float64          `json:"relative_error"`
	ClaimQuantity  *Quantity        `json:"claim_quantity,omitempty"`
	PremiseQuantity *Quantity       `json:"premise_quantity,omitempty"`
	Explanation    string           `json:"explanation"`
}

// CheckNumericalConsistency verifies whether claim quantity aligns with premise truth.
func CheckNumericalConsistency(claimText, premiseText string, tolerancePct float64) NumericalEvaluation {
	qClaim, errClaim := ParseQuantity(claimText)
	qPremise, errPremise := ParseQuantity(premiseText)

	if errClaim != nil || errPremise != nil {
		return NumericalEvaluation{
			Verdict:     VerdictUngrounded,
			Explanation: "Could not extract comparable numerical quantities from both claim and premise",
		}
	}

	// Check Dimensionality
	if qClaim.Dimension != qPremise.Dimension {
		return NumericalEvaluation{
			Verdict:         VerdictDimensionMismatch,
			ClaimQuantity:   qClaim,
			PremiseQuantity: qPremise,
			Explanation: fmt.Sprintf("Dimensional mismatch: claim has dimension %s while premise has dimension %s",
				qClaim.Dimension, qPremise.Dimension),
		}
	}

	// Compute Relative Error
	denom := math.Abs(qPremise.CanonicalValue)
	var relErr float64
	if denom < 1e-9 {
		relErr = math.Abs(qClaim.CanonicalValue - qPremise.CanonicalValue)
	} else {
		relErr = math.Abs(qClaim.CanonicalValue-qPremise.CanonicalValue) / denom
	}

	tolRatio := tolerancePct / 100.0
	if relErr <= tolRatio {
		return NumericalEvaluation{
			Verdict:         VerdictConsistent,
			RelativeError:   relErr,
			ClaimQuantity:   qClaim,
			PremiseQuantity: qPremise,
			Explanation: fmt.Sprintf("Claim value %.4f %s matches premise %.4f %s within %.2f%% tolerance",
				qClaim.RawValue, qClaim.Unit, qPremise.RawValue, qPremise.Unit, tolerancePct),
		}
	}

	return NumericalEvaluation{
		Verdict:         VerdictContradiction,
		RelativeError:   relErr,
		ClaimQuantity:   qClaim,
		PremiseQuantity: qPremise,
		Explanation: fmt.Sprintf("Numerical contradiction: claim (%.4f canonical) deviates from premise (%.4f canonical) by %.2f%% (threshold %.2f%%)",
			qClaim.CanonicalValue, qPremise.CanonicalValue, relErr*100.0, tolerancePct),
	}
}
