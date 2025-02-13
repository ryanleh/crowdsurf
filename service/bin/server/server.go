package main

import (
    "bufio"
    "log"
    "os"

    "github.com/ryanleh/secure-inference/service"
)

func main() {
    serverType := os.Args[1]
    switch serverType {
        case "pir":
            server := service.StartServer()
            defer server.StopServer()
        case "hint":
            server := service.StartHCServer()
            defer server.StopServer()
        default:
            panic("Invalid server type")
    }
	
    buf := bufio.NewReader(os.Stdin)
	log.Println("Press any button to kill server...")
	buf.ReadBytes('\n')
}
