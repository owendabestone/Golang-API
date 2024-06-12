package main

// Use this to parse login responses
type LoginResponse struct {
	Response struct {
		SessionContextId string `json:"SessionContextId"`
		InitialBranch    string `json:"InitialBranch"`
		ReturnCode       int    `json:"ReturnCode"`
		MessageText      string `json:"MessageText"`
	} `json:"response"`
}
type PriceAuditInner struct {
	AuditSequence int    `json:"AuditSequence"`
	AuditType     string `json:"AuditType"`
	AuditText     string `json:"AuditText"`
}

// Use this to parse pricing group update responses
type PricingGroupUpdateResponse struct {
	Response struct {
		AuditResults struct {
			DsAuditResults struct {
				DtAuditResults []PriceAuditInner `json:"dtAuditResults"`
			} `json:"dsAuditResults"`
		} `json:"AuditResults"`
		ReturnCode  int    `json:"ReturnCode"`
		MessageText string `json:"MessageText"`
	} `json:"response"`
}

// Use this to parse list of branches user has access to update responses
type BranchListInner struct {
	BranchID    string `json:"BranchID"`
	CompanyName string `json:"CompanyName"`
	ProfName    string `json:"ProfName"`
}

type BranchListResponse struct {
	Response struct {
		BranchListResponse struct {
			DsBranchListResponse struct {
				DtBranchListResponse []BranchListInner `json:"dtBranchListResponse"`
			} `json:"dsBranchListResponse"`
		} `json:"BranchListResponse"`
		ReturnCode  int    `json:"ReturnCode"`
		MessageText string `json:"MessageText"`
	} `json:"response"`
}
