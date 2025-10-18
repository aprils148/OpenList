package synology_filestation

import (
	"github.com/OpenListTeam/OpenList/v4/internal/driver"
	"github.com/OpenListTeam/OpenList/v4/internal/op"
)

type Addition struct {
	driver.RootPath
	driver.RootID

	Address  string `json:"address" required:"true"`
	Username string `json:"username" required:"true"`
	Password string `json:"password" required:"true"`
	Mode     string `json:"mode" type:"select" options:"download,open" default:"download" help:"Download mode to use when generating direct links"`
}

var config = driver.Config{
	Name:        "Synology File Station",
	DefaultRoot: "/",
}

func init() {
	op.RegisterDriver(func() driver.Driver {
		return &Driver{}
	})
}
