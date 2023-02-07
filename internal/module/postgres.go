package module

type PostgresEvent struct {
	Pid       uint64     `json:"pid"`
	Timestamp uint64     `json:"timestamp"`
	Query     [150]uint8 `json:"Query"`
	Comm      [16]uint8  `json:"Comm"`
}
