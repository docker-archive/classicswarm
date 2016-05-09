package states

type EventEnum int

const (
	//EventEnum - Describes type of events for the Validation logic
	NotSupported EventEnum = iota
	ContainerCreate
	ContainersList
	ContainerInspect
	ContainerOthers
	VolumeCreate
	VolumesList
	VolumeInspect
	VolumeRemove
	PassAsIs
	Unauthorized
	StreamOrHijack
)

type ApprovalEnum int

const (
	//ApprovalEnum - Describes Validations verdict
	Approved ApprovalEnum = iota
	NotApproved
	ConditionFilter
	ConditionOverride
	Admin
	QuotaLimit
)
