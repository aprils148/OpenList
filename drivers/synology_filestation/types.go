package synology_filestation

type baseResponse struct {
	Success bool        `json:"success"`
	Error   *errorBlock `json:"error"`
}

type errorBlock struct {
	Code int `json:"code"`
}

type apiResponse interface {
	base() *baseResponse
}

type loginResponse struct {
	baseResponse
	Data struct {
		Sid       string `json:"sid"`
		SynoToken string `json:"synotoken"`
	} `json:"data"`
}

type listResponse struct {
	baseResponse
	Data struct {
		Total   int     `json:"total"`
		Offset  int     `json:"offset"`
		Files   []entry `json:"files"`
		Folders []entry `json:"folders"`
	} `json:"data"`
}

type entry struct {
	ID         string          `json:"id"`
	Path       string          `json:"path"`
	Name       string          `json:"name"`
	IsDir      bool            `json:"isdir"`
	Additional *entryExtraInfo `json:"additional"`
}

type entryExtraInfo struct {
	RealPath string     `json:"real_path"`
	Size     int64      `json:"size"`
	Time     *entryTime `json:"time"`
}

type entryTime struct {
	Atime int64 `json:"atime"`
	Mtime int64 `json:"mtime"`
	Ctime int64 `json:"ctime"`
}

type baseOnlyResponse struct {
	baseResponse
}

func (r *loginResponse) base() *baseResponse {
	return &r.baseResponse
}

func (r *listResponse) base() *baseResponse {
	return &r.baseResponse
}

func (r *baseOnlyResponse) base() *baseResponse {
	return &r.baseResponse
}
