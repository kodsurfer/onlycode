package executor

import (
	"github.com/kodsurfer/onlycode/internal/config"

	_ "github.com/docker/docker/api/types/container"
	_ "github.com/docker/docker/client"
)

type CodeExecutor struct {
	docker *docker.Client
	cfg    config.Config
}

func (e *CodeExecutor) RunCode() {

}
