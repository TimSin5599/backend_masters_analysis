package entity

const (
	RoleAdmin    = "admin"
	RoleManager  = "manager"
	RoleExpert   = "expert"
	RoleOperator = "operator"
)

func IsValidRole(role string) bool {
	switch role {
	case RoleAdmin, RoleManager, RoleExpert, RoleOperator:
		return true
	}
	return false
}
