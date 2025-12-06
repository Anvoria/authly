package service

import "errors"

var (
	// ErrServiceNotFound is returned when a service is not found
	ErrServiceNotFound = errors.New("service not found")

	// ErrServiceCodeExists is returned when trying to create a service with an existing code
	ErrServiceCodeExists = errors.New("service code already exists")

	// ErrCannotDeleteSystemService is returned when trying to delete a system service
	ErrCannotDeleteSystemService = errors.New("cannot delete system service")

	// ErrCannotUpdateSystemService is returned when trying to update critical fields of a system service
	ErrCannotUpdateSystemService = errors.New("cannot update system service")
)
