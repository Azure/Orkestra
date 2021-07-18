package workflow

const (
	// TODO: we might need to make it unique (this might solve parallel tests issue)
	EntrypointTemplateName = "entry"

	HelmReleaseArg = "helmrelease"
	TimeoutArg     = "timeout"

	ValuesKeyGlobal = "global"
	ChartLabelKey   = "chart"

	DefaultTimeout = "5m"

	ChartMuseumName = "chartmuseum"

	Project        = "orkestra"
	OwnershipLabel = "owner"
	HeritageLabel  = "heritage"
)
