package daemonapi

import (
	"net/http"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"

	"github.com/opensvc/om3/core/clusternode"
	"github.com/opensvc/om3/core/naming"
	"github.com/opensvc/om3/daemon/api"
	"github.com/opensvc/om3/daemon/rbac"
)

func (a *DaemonAPI) PostNodeActionScanCapabilities(ctx echo.Context, nodename string, params api.PostNodeActionScanCapabilitiesParams) error {
	if nodename == a.localhost {
		return a.localNodeActionScanCapabilities(ctx, params)
	} else if !clusternode.Has(nodename) {
		return JSONProblemf(ctx, http.StatusBadRequest, "Invalid nodename", "field 'nodename' with value '%s' is not a cluster node", nodename)
	}
	return a.remoteNodeActionScanCapabilities(ctx, nodename, params)
}

func (a *DaemonAPI) remoteNodeActionScanCapabilities(ctx echo.Context, nodename string, params api.PostNodeActionScanCapabilitiesParams) error {
	c, err := newProxyClient(ctx, nodename)
	if err != nil {
		return JSONProblemf(ctx, http.StatusInternalServerError, "New client", "%s: %s", nodename, err)
	}
	resp, err := c.PostNodeActionScanCapabilitiesWithResponse(ctx.Request().Context(), nodename, &params)
	if err != nil {
		return JSONProblemf(ctx, http.StatusInternalServerError, "Request peer", "%s: %s", nodename, err)
	} else if len(resp.Body) > 0 {
		return ctx.JSONBlob(resp.StatusCode(), resp.Body)
	}
	return nil
}

func (a *DaemonAPI) localNodeActionScanCapabilities(ctx echo.Context, params api.PostNodeActionScanCapabilitiesParams) error {
	if v, err := assertGrant(ctx, rbac.GrantRoot); !v {
		return err
	}
	log := LogHandler(ctx, "PostNodeActionScanCapabilities")
	var requesterSid uuid.UUID
	args := []string{"node", "scan", "capabilities", "--local"}
	if params.RequesterSid != nil {
		requesterSid = *params.RequesterSid
	}
	if sid, err := a.apiExec(ctx, naming.Path{}, requesterSid, args, log); err != nil {
		return JSONProblemf(ctx, http.StatusInternalServerError, "", "%s", err)
	} else {
		return ctx.JSON(http.StatusOK, api.NodeActionAccepted{SessionID: sid})
	}
}
