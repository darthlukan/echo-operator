package controller

import (
	"github.com/darthlukan/echo-operator/pkg/controller/echo"
)

func init() {
	// AddToManagerFuncs is a list of functions to create controllers and add them to a manager.
	AddToManagerFuncs = append(AddToManagerFuncs, echo.Add)
}
