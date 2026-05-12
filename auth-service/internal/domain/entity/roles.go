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

func IsValidRoles(roles []string) bool {
	if len(roles) == 0 {
		return false
	}
	for _, r := range roles {
		if !IsValidRole(r) {
			return false
		}
	}
	return true
}
