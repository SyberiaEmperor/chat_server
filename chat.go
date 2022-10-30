package main

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"time"
)

func main() {

	listener, err := net.Listen("tcp", "localhost:8000")

	if err != nil {
		log.Fatal(err)
	}

	go broadcaster()

	for {
		conn, err := listener.Accept()

		if err != nil {
			log.Print(err)
			continue
		}

		go handleConn(conn)
	}
}

type client struct {
	channel chan<- string
	name    string
}

var (
	entering = make(chan client)
	leaving  = make(chan client)
	messages = make(chan string)
)

func broadcaster() {
	clients := make(map[client]bool)

	for {
		select {
		case msg := <-messages:
			for cli := range clients {
				select {
				case cli.channel <- msg:
				default:
					continue
				}
			}
		case cli := <-entering:
			clients[cli] = true
			cli.channel <- "List of connected users:"
			for cl := range clients {
				if cl != cli {
					select {
					case cli.channel <- cl.name:
					default:
						continue
					}
				}
			}
		case cli := <-leaving:
			delete(clients, cli)
			close(cli.channel)
		}
	}
}

func handleConn(conn net.Conn) {
	ch := make(chan string, 10)
	ch2 := make(chan string, 10)

	go clientWriter(conn, ch)
	who := conn.RemoteAddr().String()
	cli := client{channel: ch, name: who}

	cli.channel <- "You " + cli.name
	messages <- cli.name + " connected"
	entering <- cli

	go clientReader(conn, ch2)

	ticker := time.NewTicker(10 * time.Second)

loop:
	for {
		select {
		case <-ticker.C:
			disconnect(conn, cli)
			break loop
		case text := <-ch2:
			ticker.Reset(10 * time.Second)
			messages <- cli.name + ": " + text
		}
	}
}

func clientWriter(conn net.Conn, ch <-chan string) {
	for msg := range ch {
		fmt.Fprintln(conn, msg+"\n")
	}
}

func clientReader(conn net.Conn, ch chan<- string) {
	input := bufio.NewScanner(conn)
	for input.Scan() {
		ch <- input.Text()
	}
}

func disconnect(conn net.Conn, cli client) {
	leaving <- cli
	messages <- cli.name + " disconnected"
	conn.Close()
}
