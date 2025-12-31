package gompkg

import (
	"github.com/mikeschinkel/go-dt"
)

// InputData is the domain representation of editor input.
type InputData struct {
	ModuleDir     dt.DirPath
	GitDiffOutput string
	ExistingPlans []StagingPlan
	AITakes       *PlanTakes
}

// OutputData is the domain representation of editor output.
type OutputData struct {
	Plans []StagingPlan
}
