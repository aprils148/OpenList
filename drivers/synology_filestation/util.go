package synology_filestation

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/OpenListTeam/OpenList/v4/drivers/base"
	"github.com/go-resty/resty/v2"
)

const (
	authEndpoint  = "/webapi/auth.cgi"
	entryEndpoint = "/webapi/entry.cgi"
	sessionName   = "FileStation"
)

var errorText = map[int]string{
	100: "Unknown error",
	101: "Invalid parameter",
	102: "API does not exist",
	103: "Method does not exist",
	104: "Version does not support the functionality",
	105: "Insufficient user privilege",
	106: "Session timeout",
	107: "Session interrupted by duplicate login",
	119: "Invalid session",
	400: "Invalid parameter of file operation",
	401: "Unknown error of file operation",
	402: "System is too busy",
	403: "Operation forbidden",
	404: "Path not found",
	405: "Insufficient storage space",
	406: "Operation not supported",
	407: "Operation failed",
	408: "File or folder already exists",
	409: "File does not exist",
}

func (d *Driver) ensureClient() {
	if d.client != nil {
		return
	}
	d.client = base.NewRestyClient()
	d.client.SetBaseURL(d.baseURL)
}

func (d *Driver) login(ctx context.Context) error {
	d.ensureClient()
	var resp loginResponse
	_, err := d.client.R().
		SetContext(ctx).
		SetResult(&resp).
		SetQueryParams(map[string]string{
			"api":     "SYNO.API.Auth",
			"method":  "login",
			"version": "6",
			"account": d.Username,
			"passwd":  d.Password,
			"session": sessionName,
			"format":  "cookie",
		}).
		Get(authEndpoint)
	if err != nil {
		return err
	}
	if !resp.Success {
		return d.apiError(resp.Error)
	}
	d.sid = resp.Data.Sid
	d.token = resp.Data.SynoToken
	return nil
}

func (d *Driver) request(ctx context.Context, method, endpoint string, params map[string]string, result apiResponse, allowRetry bool) (*resty.Response, error) {
	d.ensureClient()
	req := d.client.R().SetContext(ctx)
	if result != nil {
		req.SetResult(result)
	}
	if d.token != "" {
		req.SetHeader("X-SYNO-TOKEN", d.token)
	}
	qp := make(map[string]string, len(params)+1)
	for k, v := range params {
		qp[k] = v
	}
	if d.sid != "" {
		qp["_sid"] = d.sid
	}
	req.SetQueryParams(qp)
	res, err := req.Execute(method, endpoint)
	if err != nil {
		return nil, err
	}
	if result != nil {
		base := result.base()
		if base != nil && !base.Success {
			if allowRetry && base.Error != nil && base.Error.Code == 119 {
				if err = d.login(ctx); err != nil {
					return nil, err
				}
				return d.request(ctx, method, endpoint, params, result, false)
			}
			return nil, d.apiError(base.Error)
		}
	}
	return res, nil
}

func (d *Driver) listDirectory(ctx context.Context, folderPath string) ([]entry, error) {
	var resp listResponse
	_, err := d.request(ctx, http.MethodGet, entryEndpoint, map[string]string{
		"api":         "SYNO.FileStation.List",
		"method":      "list",
		"version":     "2",
		"folder_path": folderPath,
		"additional":  "real_path,size,time",
	}, &resp, true)
	if err != nil {
		return nil, err
	}
	combined := make([]entry, 0, len(resp.Data.Files)+len(resp.Data.Folders))
	combined = append(combined, resp.Data.Files...)
	combined = append(combined, resp.Data.Folders...)
	return combined, nil
}

func (d *Driver) buildDownloadURL(ctx context.Context, path string) (string, error) {
	if err := d.ensureLoggedIn(ctx); err != nil {
		return "", err
	}
	paths, err := json.Marshal([]string{path})
	if err != nil {
		return "", err
	}
	values := url.Values{}
	values.Set("api", "SYNO.FileStation.Download")
	values.Set("method", "download")
	values.Set("version", "2")
	values.Set("path", string(paths))
	values.Set("mode", d.Mode)
	if d.sid != "" {
		values.Set("_sid", d.sid)
	}
	return fmt.Sprintf("%s%s?%s", d.baseURL, entryEndpoint, values.Encode()), nil
}

func (d *Driver) ensureLoggedIn(ctx context.Context) error {
	if d.sid != "" {
		return nil
	}
	return d.login(ctx)
}

func (d *Driver) logout(ctx context.Context) error {
	if d.sid == "" {
		return nil
	}
	var resp baseOnlyResponse
	_, err := d.request(ctx, http.MethodGet, authEndpoint, map[string]string{
		"api":     "SYNO.API.Auth",
		"method":  "logout",
		"version": "6",
		"session": sessionName,
	}, &resp, false)
	d.sid = ""
	d.token = ""
	return err
}

func (d *Driver) apiError(errBlock *errorBlock) error {
	if errBlock == nil {
		return fmt.Errorf("synology file station: unknown error")
	}
	if msg, ok := errorText[errBlock.Code]; ok {
		return fmt.Errorf("synology file station: %s (code %d)", msg, errBlock.Code)
	}
	return fmt.Errorf("synology file station: api error code %d", errBlock.Code)
}
