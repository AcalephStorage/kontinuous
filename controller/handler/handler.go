package handler

type ResourceAction string

const (
	Get    ResourceAction = "get"
	Set    ResourceAction = "set"
	Delete ResourceAction = "delete"
	Update ResourceAction = "update"
	Create ResourceAction = "create"
	CAS    ResourceAction = "compareAndSwap"
	CAD    ResourceAction = "compareAndDelete"
)

type Handler interface {
	OnResourceChange(action ResourceAction, key string, value []byte) error
}
