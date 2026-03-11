package order

type Status string

const (
	StatusCreated    Status = "criado"
	StatusProcessing Status = "em_processamento"
	StatusShipped    Status = "enviado"
	StatusDelivered  Status = "entregue"
)

func (s Status) IsValid() bool {
	switch s {
	case StatusCreated, StatusProcessing, StatusShipped, StatusDelivered:
		return true
	default:
		return false
	}
}
