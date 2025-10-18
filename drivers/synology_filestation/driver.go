package synology_filestation

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/OpenListTeam/OpenList/v4/internal/driver"
	"github.com/OpenListTeam/OpenList/v4/internal/errs"
	"github.com/OpenListTeam/OpenList/v4/internal/model"
	"github.com/OpenListTeam/OpenList/v4/pkg/utils"
	"github.com/go-resty/resty/v2"
)

type Driver struct {
	model.Storage
	Addition

	client  *resty.Client
	baseURL string
	sid     string
	token   string
}

func (d *Driver) Config() driver.Config {
	return config
}

func (d *Driver) GetAddition() driver.Additional {
	return &d.Addition
}

func (d *Driver) Init(ctx context.Context) error {
	address := strings.TrimSpace(d.Address)
	if address == "" {
		return fmt.Errorf("synology file station: address is required")
	}
	lower := strings.ToLower(address)
	if !strings.HasPrefix(lower, "http://") && !strings.HasPrefix(lower, "https://") {
		address = "https://" + address
	}
	d.baseURL = strings.TrimRight(address, "/")
	d.RootFolderPath = utils.FixAndCleanPath(d.RootFolderPath)
	if d.RootFolderPath == "" {
		d.RootFolderPath = "/"
	}
	mode := strings.ToLower(strings.TrimSpace(d.Mode))
	switch mode {
	case "open":
		d.Mode = "open"
	default:
		d.Mode = "download"
	}
	d.client = nil
	d.sid = ""
	d.token = ""
	return d.login(ctx)
}

func (d *Driver) Drop(ctx context.Context) error {
	return d.logout(ctx)
}

func (d *Driver) List(ctx context.Context, dir model.Obj, args model.ListArgs) ([]model.Obj, error) {
	if err := d.ensureLoggedIn(ctx); err != nil {
		return nil, err
	}
	path := dir.GetPath()
	if path == "" {
		path = d.RootFolderPath
	}
	if path == "" {
		path = "/"
	}
	path = utils.FixAndCleanPath(path)
	entries, err := d.listDirectory(ctx, path)
	if err != nil {
		return nil, err
	}
	return utils.SliceConvert(entries, func(e entry) (model.Obj, error) {
		return entryToObject(e, path), nil
	})
}

func (d *Driver) Link(ctx context.Context, file model.Obj, args model.LinkArgs) (*model.Link, error) {
	path := file.GetPath()
	if path == "" {
		path = file.GetID()
	}
	if path == "" {
		return nil, fmt.Errorf("synology file station: cannot build link without file path")
	}
	url, err := d.buildDownloadURL(ctx, path)
	if err != nil {
		return nil, err
	}
	return &model.Link{URL: url}, nil
}

func (d *Driver) MakeDir(ctx context.Context, parentDir model.Obj, dirName string) (model.Obj, error) {
	return nil, errs.NotImplement
}

func (d *Driver) Move(ctx context.Context, srcObj, dstDir model.Obj) (model.Obj, error) {
	return nil, errs.NotImplement
}

func (d *Driver) Rename(ctx context.Context, srcObj model.Obj, newName string) (model.Obj, error) {
	return nil, errs.NotImplement
}

func (d *Driver) Copy(ctx context.Context, srcObj, dstDir model.Obj) (model.Obj, error) {
	return nil, errs.NotImplement
}

func (d *Driver) Remove(ctx context.Context, obj model.Obj) error {
	return errs.NotImplement
}

func (d *Driver) Put(ctx context.Context, dstDir model.Obj, file model.FileStreamer, up driver.UpdateProgress) (model.Obj, error) {
	return nil, errs.NotImplement
}

func (d *Driver) GetArchiveMeta(ctx context.Context, obj model.Obj, args model.ArchiveArgs) (model.ArchiveMeta, error) {
	return nil, errs.NotImplement
}

func (d *Driver) ListArchive(ctx context.Context, obj model.Obj, args model.ArchiveInnerArgs) ([]model.Obj, error) {
	return nil, errs.NotImplement
}

func (d *Driver) Extract(ctx context.Context, obj model.Obj, args model.ArchiveInnerArgs) (*model.Link, error) {
	return nil, errs.NotImplement
}

func (d *Driver) ArchiveDecompress(ctx context.Context, srcObj, dstDir model.Obj, args model.ArchiveDecompressArgs) ([]model.Obj, error) {
	return nil, errs.NotImplement
}

func (d *Driver) GetDetails(ctx context.Context) (*model.StorageDetails, error) {
	return nil, errs.NotImplement
}

var _ driver.Driver = (*Driver)(nil)

func entryToObject(src entry, parentPath string) *model.Object {
	objPath := strings.TrimSpace(src.Path)
	if objPath == "" {
		if joined, err := utils.JoinBasePath(parentPath, src.Name); err == nil {
			objPath = joined
		} else {
			objPath = utils.FixAndCleanPath(parentPath + "/" + src.Name)
		}
	}
	objPath = utils.FixAndCleanPath(objPath)
	obj := &model.Object{
		ID:       chooseID(src.ID, objPath),
		Path:     objPath,
		Name:     src.Name,
		IsFolder: src.IsDir,
	}
	if src.Additional != nil {
		obj.Size = src.Additional.Size
		if src.Additional.Time != nil {
			if src.Additional.Time.Mtime > 0 {
				obj.Modified = time.Unix(src.Additional.Time.Mtime, 0)
			}
			if src.Additional.Time.Ctime > 0 {
				obj.Ctime = time.Unix(src.Additional.Time.Ctime, 0)
			}
		}
	}
	if !obj.IsFolder && obj.Modified.IsZero() && src.Additional != nil && src.Additional.Time != nil && src.Additional.Time.Atime > 0 {
		obj.Modified = time.Unix(src.Additional.Time.Atime, 0)
	}
	return obj
}

func chooseID(id, fallback string) string {
	if strings.TrimSpace(id) != "" {
		return id
	}
	return fallback
}
