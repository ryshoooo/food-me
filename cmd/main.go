package main

import (
	"fmt"
	"net"
	"os"

	foodme "github.com/ryshoooo/food-me/internal"
)

// func copyAndPrint(source string, dst io.Writer, src io.Reader) {
// 	buf := make([]byte, 64000)
// 	// startup [0 0 0 8 4 210 22 47]
// 	for {
// 		n, err := src.Read(buf)
// 		if err != nil {
// 			if err != io.EOF {
// 				log.Printf("Error reading from source: %v\n", err)
// 			}
// 			return
// 		}

// 		// Print the data read from src
// 		fmt.Printf("Data from %s: %v %s\n", source, buf[:n], buf[:n])

// 		// Write the data to the destination
// 		if _, err := dst.Write(buf[:n]); err != nil {
// 			log.Printf("Error writing to destination: %v\n", err)
// 			return
// 		}
// 	}
// }

// func handleConnection(src net.Conn, destAddr string) {
// 	// Establish a connection to the destination address
// 	dest, err := net.Dial("tcp", destAddr)
// 	if err != nil {
// 		log.Printf("Unable to connect to destination: %v\n", err)
// 		src.Close()
// 		return
// 	}
// 	// Close both connections when this function exits
// 	defer fmt.Println("Closing connections")
// 	defer src.Close()
// 	defer dest.Close()

// 	// // Use io.Copy to forward data between the source and destination
// 	go copyAndPrint("PROXY", dest, src)
// 	copyAndPrint("DATABASE", src, dest)
// }

func main() {
	conf, err := foodme.NewConfiguration(os.Args)
	if err != nil {
		fmt.Printf("Error parsing arguments: %v\n", err)
		os.Exit(1)
	}

	logger := foodme.NewLogger(conf)
	server := foodme.NewServer(conf.ServerPort, logger, func(c net.Conn) error { return nil })
	logger.Fatal(server.Start())
}
