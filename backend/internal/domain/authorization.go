package domain

type Action string

const (
	ActionCatalogWrite         Action = "catalog_write"
	ActionSubmitExecution      Action = "submit_execution_request"
	ActionManageExecution      Action = "manage_execution_request"
	ActionResolveApproval      Action = "resolve_approval_task"
	ActionRecordReceipt        Action = "record_receipts"
	ActionReviewPolicyIncident Action = "review_policy_incident"
	ActionReadGovernance       Action = "read_governance"
	ActionReadAudit            Action = "read_audit"
)

var roleActions = map[Role]map[Action]bool{
	RoleAgentDeveloper:    {ActionCatalogWrite: true, ActionSubmitExecution: true, ActionManageExecution: true, ActionResolveApproval: true, ActionRecordReceipt: true, ActionReadGovernance: true, ActionReadAudit: true},
	RoleToolOperator:      {ActionManageExecution: true, ActionResolveApproval: true, ActionRecordReceipt: true, ActionReadGovernance: true},
	RoleSecurityReviewer:  {ActionReviewPolicyIncident: true, ActionReadGovernance: true},
	RoleComplianceAuditor: {ActionReadGovernance: true, ActionReadAudit: true},
}

func (p Principal) CanAction(action Action) bool {
	return roleActions[p.Role][action]
}
