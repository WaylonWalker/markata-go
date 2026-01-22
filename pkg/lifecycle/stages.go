package lifecycle

// Stage represents a lifecycle stage in the markata build process.
type Stage string

const (
	// StageConfigure loads config and initializes plugins.
	StageConfigure Stage = "configure"

	// StageValidate validates the configuration.
	StageValidate Stage = "validate"

	// StageGlob discovers content files.
	StageGlob Stage = "glob"

	// StageLoad parses files into posts.
	StageLoad Stage = "load"

	// StageTransform performs pre-render processing (jinja-md, etc.).
	StageTransform Stage = "transform"

	// StageRender converts markdown to HTML.
	StageRender Stage = "render"

	// StageCollect builds feeds and navigation.
	StageCollect Stage = "collect"

	// StageWrite writes output files.
	StageWrite Stage = "write"

	// StageCleanup releases resources.
	StageCleanup Stage = "cleanup"
)

// StageOrder defines the execution order of all lifecycle stages.
var StageOrder = []Stage{
	StageConfigure,
	StageValidate,
	StageGlob,
	StageLoad,
	StageTransform,
	StageRender,
	StageCollect,
	StageWrite,
	StageCleanup,
}

// stageIndex maps stages to their position in StageOrder for ordering comparisons.
var stageIndex = func() map[Stage]int {
	m := make(map[Stage]int, len(StageOrder))
	for i, s := range StageOrder {
		m[s] = i
	}
	return m
}()

// StageIndex returns the index of a stage in the execution order.
// Returns -1 if the stage is not found.
func StageIndex(s Stage) int {
	if idx, ok := stageIndex[s]; ok {
		return idx
	}
	return -1
}

// IsValidStage checks if the given stage is a valid lifecycle stage.
func IsValidStage(s Stage) bool {
	_, ok := stageIndex[s]
	return ok
}

// StagesBefore returns all stages that come before the given stage.
func StagesBefore(s Stage) []Stage {
	idx := StageIndex(s)
	if idx <= 0 {
		return nil
	}
	return StageOrder[:idx]
}

// StagesUpTo returns all stages up to and including the given stage.
func StagesUpTo(s Stage) []Stage {
	idx := StageIndex(s)
	if idx < 0 {
		return nil
	}
	return StageOrder[:idx+1]
}
