package grupkg

import (
	"github.com/mikeschinkel/go-dt"
	"github.com/mikeschinkel/squire/squirepkg/squirecfg"
	"github.com/mikeschinkel/squire/squirepkg/squiresvc"
)

// InputData is the domain representation of editor input.
type InputData struct {
	ModuleDir     dt.DirPath
	GitDiffOutput string
	ExistingPlans []squiresvc.StagingPlan
	AITakes       *squirecfg.StagingPlanTakes
}

// OutputData is the domain representation of editor output.
type OutputData struct {
	Plans []squiresvc.StagingPlan
}
