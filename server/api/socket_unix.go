//go:build !windows

package api

import (
	"fmt"
	"net"
	"os"
	"os/user"
	"strconv"

	"github.com/docker/go-connections/sockets"
	"github.com/pkg/errors"
	"github.com/reeveci/reeve/server/config"
	"github.com/reeveci/reeve/server/log"
)

func getSocketListener() (net.Listener, string, error) {
	uid := os.Getuid()
	gid, err := lookupGID(config.Config.Group)
	if err != nil {
		if config.Config.Group != "" {
			log.Error.Printf("could not change group for %s to %s - %s\n", config.Config.Socket, config.Config.Group, err)
		}
		gid = os.Getgid()
	}
	l, err := sockets.NewUnixSocketWithOpts(config.Config.Socket, sockets.WithChown(uid, gid), sockets.WithChmod(0o660))
	if err != nil {
		return nil, "", errors.Wrapf(err, "could not create unix socket %s", config.Config.Socket)
	}
	return l, "unix://" + config.Config.Socket, nil
}

func lookupGID(name string) (int, error) {
	group, err := user.LookupGroup(name)
	if err == nil {
		name = group.Gid
	}
	gid, err := strconv.Atoi(name)
	if err == nil {
		return gid, nil
	}
	return -1, fmt.Errorf("group %s not found", name)
}
