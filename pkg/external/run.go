package external

import (
	// stdlib

	"os"
	"os/exec"

	// external
	"github.com/golang/glog"
	"github.com/pkg/errors"

	// internal
	"github.com/mccurdyc/neighbor/pkg/github"
	"github.com/mccurdyc/neighbor/pkg/neighbor"
)

// Run runs an arbitrary command specified in the Ctx on each project
// that is sent through the pipeline.
//
// The number of workers is defined by the number of logical CPUs that read
// from the pipeline and then run the external command on each project.
//
// When each worker has recieved the empty channel signal from the pipeline, we
// are finished.
func Run(ctx *neighbor.Ctx, in <-chan github.ExternalProject) <-chan github.ExternalProject {
	out := make(chan github.ExternalProject) // an unbuffered, synchronous channel for guaranteed delivery

	go func() {
		for {
			select {
			case project, ok := <-in:
				if !ok {
					close(out)
					return
				}

				if err := run(ctx, project); err != nil {
					glog.Error(err)
				}

				out <- project
			}
		}
	}()

	return out
}

func run(ctx *neighbor.Ctx, p github.ExternalProject) error {
	glog.V(2).Infof("running external command on %s", p.Name)
	err := os.Chdir(p.Directory)
	if err != nil {
		return errors.Wrap(err, "error changing into project working directory")
	}

	// we can't parse the command outside of this loop because exec.Command creates
	// a pointer to a Cmd and if you call Run() on that command, it will say
	// that it is already processing.
	var cmd *exec.Cmd
	if len(ctx.ExternalCmd) == 1 {
		cmd = exec.Command(ctx.ExternalCmd[0])
	} else {
		cmd = exec.Command(ctx.ExternalCmd[0], ctx.ExternalCmd[1:]...)
	}

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return errors.Wrap(err, "failed to run external command with error")
	}

	return nil
}
