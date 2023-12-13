package daemonapi

import (
	"github.com/labstack/echo/v4"
	"github.com/opensvc/om3/core/instance"
	"github.com/opensvc/om3/core/naming"
)

func (a *DaemonApi) PostObjectActionDelete(ctx echo.Context, namespace string, kind naming.Kind, name string) error {
	return a.postObjectAction(ctx, namespace, kind, name, instance.MonitorGlobalExpectDeleted)
}
