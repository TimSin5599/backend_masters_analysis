package entity

const (
	RoleAdmin    = "admin"
	RoleManager  = "manager"
	RoleObserver = "observer"
	RoleOperator = "operator"
)

func IsValidRole(role string) bool {
	switch role {
	case RoleAdmin, RoleManager, RoleObserver, RoleOperator:
		return true
	}
	return false
}
