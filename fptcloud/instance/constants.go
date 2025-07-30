package fptcloud_instance

const (
	InstanceStatusCreating   = "CREATING"
	InstanceStatusPoweredOn  = "POWERED_ON"
	InstanceStatusPoweredOff = "POWERED_OFF"
	InstanceStatusReboot     = "REBOOT"
)

const (
	INSTANCE_VM_ACTION       = "vm_action"
	INSTANCE_SNAPSHOT_ACTION = "snapshot_action"
	INSTANCE_TEMPLATE_ACTION = "template_action"
)

// state wait
var (
	InstanceStateVMActionPending       = []string{"REBOOTING"}
	InstanceStateVMActionTarget        = []string{"POWERED_ON", "POWERED_OFF"}
	InstanceStateSnapshotActionPending = []string{"SNAPSHOTTING", "CREATING"}
	InstanceStateSnapshotActionTarget  = []string{"POWERED_ON", "POWERED_OFF"}
	InstanceStateTemplateActionPending = []string{"CAPTURING", "CREATING"}
	InstanceStateTemplateActionTarget  = []string{"POWERED_ON", "POWERED_OFF"}
	InstanceStateResizeActionPending   = []string{"RESIZING", "UPDATING"}
	InstanceStateResizeActionTarget    = []string{"POWERED_ON", "POWERED_OFF"}
)

// VmActionReboot block VM Action
const (
	VmActionReboot = "REBOOT"
)

// SnapshotAction block snapshot Action
const (
	SnapshotActionCreate = "CREATE"
	SnapshotActionDelete = "DELETE"
	SnapshotActionUpdate = "UPDATE"
)

// TemplateActionCreate block template Action
const (
	TemplateActionCreate = "CREATE"
)
