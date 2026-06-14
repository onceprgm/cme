package preflight

import (
	"fmt"
	"net"
	"time"
)

const dialTimeout = 4 * time.Second

func Online() bool {
	hosts := []string{
		"piston-meta.mojang.com:443",
		"resources.download.minecraft.net:443",
	}
	for _, h := range hosts {
		conn, err := net.DialTimeout("tcp", h, dialTimeout)
		if err == nil {
			conn.Close()
			return true
		}
	}
	return false
}

func RequireOnline() error {
	if !Online() {
		return fmt.Errorf("no connection to Mojang servers; check your internet and try again")
	}
	return nil
}
