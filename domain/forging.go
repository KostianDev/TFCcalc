package domain

// ForgeAction represents a single forging action that affects the anvil arrow.
type ForgeAction string

const (
	// Hit variants (counted as "Hit" for final-three matching)
	WeakHit   ForgeAction = "weak_hit"
	MediumHit ForgeAction = "medium_hit"
	StrongHit ForgeAction = "strong_hit"
	Draw      ForgeAction = "draw"

	// Positive adjustments
	Stamp  ForgeAction = "stamp"
	Bend   ForgeAction = "bend"
	Upset  ForgeAction = "upset"
	Shrink ForgeAction = "shrink"
)

// ForgeActionDelta contains the arrow delta (integer) applied by each action.
var ForgeActionDelta = map[ForgeAction]int{
	WeakHit:   -3,
	MediumHit: -6,
	StrongHit: -9,
	Draw:      -15,

	Stamp:  +2,
	Bend:   +7,
	Upset:  +13,
	Shrink: +16,
}

// IsHit returns true for any action that counts as a generic "Hit".
func IsHit(a ForgeAction) bool {
	return a == WeakHit || a == MediumHit || a == StrongHit
}

// FinalReqKind describes the requirement kind for a final-slot in a recipe.
type FinalReqKind int

const (
	FinalAny FinalReqKind = iota
	FinalHit
	FinalExact
)

// FinalRequirement describes one of the required final three actions.
type FinalRequirement struct {
	Kind   FinalReqKind `json:"kind"`
	Action ForgeAction  `json:"action,omitempty"`
}

// ForgingRecipe defines a recipe used for anvil forging.
// RequiredFinalThree must contain exactly three entries; use FinalAny for wildcards.
type ForgingRecipe struct {
	ID                 string             `json:"id"`
	Name               string             `json:"name"`
	DefaultTargetValue int                `json:"default_target_value"`
	RequiredFinalThree []FinalRequirement `json:"required_final_three"`
}

// ForgingResult is a lightweight structure that will carry a solved sequence.
type ForgingResult struct {
	Sequence []ForgeAction `json:"sequence"`
}
