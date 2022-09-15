package doc_type

type DocType struct {
	ID              int    `json:"id,omitempty"`
	Name            string `json:"name,omitempty"`
	CtrStatusTypeId int    `json:"ctr_status_type_id"`
}

type CreateDocTypeDTO struct {
	Name            string `json:"name,omitempty"`
	CtrStatusTypeId int    `json:"ctr_status_type_id"`
}

type UpdateDocTypeDTO struct {
	ID              int    `json:"id,omitempty"`
	Name            string `json:"name,omitempty"`
	CtrStatusTypeId int    `json:"ctr_status_type_id"`
}
