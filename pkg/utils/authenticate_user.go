package utils

import (
	"context"
	"errors"
	"fmt"
)

func IsAuthorizedUser(ctx context.Context, allowedRoles ...string) error {
	userRole, ok := ctx.Value("role").(string)
	if !ok {
		return fmt.Errorf("Unauthorized access: role not found.")
	}
	for _, allowedRole := range allowedRoles {
		if userRole == allowedRole {
			return nil
		}
	}
	return errors.New("Unauthorized access: No permission to access.")
}
