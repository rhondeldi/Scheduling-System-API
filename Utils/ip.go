package Utils

import (
	"fmt"
	"net"
)

// Get preferred outbound ip of the current machine
func DisplayOutboundIP(port string) {
	conn, err := net.Dial("udp", "8.8.8.8:80")

	if err != nil {
		fmt.Print("\n\nNo network connection detected\n\n")
		return
	}
	defer conn.Close()

	localAddr := conn.LocalAddr().(*net.UDPAddr)
	fmt.Printf("\n\nRunning on %v:%s\n\n", localAddr.IP, port)
}
