//go:build windows

package api

import (
	"fmt"
	"net"
	"strings"

	winio "github.com/Microsoft/go-winio"
	"github.com/reeveci/reeve/server/config"
)

func getSocketListener() (net.Listener, string, error) {
	// allow Administrators and SYSTEM, plus whatever additional users or groups were specified
	sddl := "D:P(A;;GA;;;BA)(A;;GA;;;SY)"
	if config.Config.Group != "" {
		for _, g := range strings.Split(config.Config.Group, ",") {
			sid, err := winio.LookupSidByName(g)
			if err != nil {
				return nil, "", err
			}
			sddl += fmt.Sprintf("(A;;GRGW;;;%s)", sid)
		}
	}
	c := winio.PipeConfig{
		SecurityDescriptor: sddl,
		MessageMode:        true,  // Use message mode so that CloseWrite() is supported
		InputBufferSize:    65536, // Use 64KB buffers to improve performance
		OutputBufferSize:   65536,
	}
	l, err := winio.ListenPipe(config.Config.Pipe, &c)
	return l, "npipe://" + config.Config.Pipe, err
}
