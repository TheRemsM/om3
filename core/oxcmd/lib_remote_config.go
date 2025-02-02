package oxcmd

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/opensvc/om3/core/client"
	"github.com/opensvc/om3/core/naming"
	"github.com/opensvc/om3/daemon/api"
)

func createTempRemoteConfig(p naming.Path, c *client.T) (string, error) {
	var (
		err  error
		buff []byte
		f    *os.File
	)
	if c, err = remoteClient(p, c); err != nil {
		return "", err
	}
	if buff, err = fetchConfig(p, c); err != nil {
		return "", err
	}
	if f, err = os.CreateTemp("", ".opensvc.remote.config.*"); err != nil {
		return "", err
	}
	filename := f.Name()
	if _, err = f.Write(buff); err != nil {
		os.Remove(filename)
		return "", err
	}
	return filename, nil
}

func remoteClient(p naming.Path, c *client.T) (*client.T, error) {
	resp, err := c.GetObjectWithResponse(context.Background(), p.Namespace, p.Kind, p.Name)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode() != http.StatusOK {
		return nil, fmt.Errorf("get object %s data from %s: %s", p, c.URL(), resp.Status())
	}
	var nodename string
	for k := range resp.JSON200.Data.Instances {
		nodename = k
		break
	}
	if nodename == "" {
		return nil, fmt.Errorf("%s has no instance", p)
	}
	if c, err = client.New(client.WithURL(nodename)); err != nil {
		return nil, err
	}
	return c, nil
}

func fetchConfig(p naming.Path, c *client.T) ([]byte, error) {
	resp, err := c.GetObjectConfigFileWithResponse(context.Background(), p.Namespace, p.Kind, p.Name)
	if err != nil {
		return nil, err
	} else if resp.StatusCode() != http.StatusOK {
		return nil, fmt.Errorf("get object %s file from %s: %s", p, c.URL(), resp.Status())
	}
	return resp.JSON200.Data, nil
}

func putConfig(p naming.Path, fName string, c *client.T) (err error) {
	body := api.PutObjectConfigFileJSONRequestBody{}
	body.Mtime = time.Now()
	if buff, err := os.ReadFile(fName); err != nil {
		return err
	} else {
		body.Data = buff
	}
	resp, err := c.PutObjectConfigFileWithResponse(context.Background(), p.Namespace, p.Kind, p.Name, body)
	if err != nil {
		return err
	}
	switch resp.StatusCode() {
	case http.StatusNoContent:
		return nil
	default:
		return fmt.Errorf("put object %s file from %s: %s", p, c.URL(), resp.Status()+string(resp.Body))
	}
}
