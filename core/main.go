package main

import (
	"bufio"
	"os/signal"
	"syscall"
	// "context"
	"eqlcore/eqlcore"
	"fmt"
	"os"
)

type handleMessages struct {}

func (m *handleMessages) OnMessageReceived(from string, msg string){
	fmt.Printf("from %s: msg: %s\n", from, msg)
}

func main(){
	id := eqlcore.StartNode("/ip4/0.0.0.0/tcp/0", "chat-room")
	eqlcore.SetMessageReceiver(&handleMessages{})
	defer eqlcore.StopNode()
	fmt.Printf("id = %s:\n", id)

	addrs := eqlcore.GetNodeInfo()
	fmt.Printf("addrs = %s:\n", addrs)

	if (len(os.Args) > 1){
		eqlcore.ConnectPeer(os.Args[1])
	}

	reader := bufio.NewScanner(os.Stdin)
	fmt.Println("Введите сообщения и нажимайте Enter:")
	go func() {
		for reader.Scan() {
			text := reader.Text()
			if text != "" {
				eqlcore.SendMessage(text, "chat-room")
			}
		}
	}()
	// graceful shutdown
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
	<-ch
	fmt.Println("Shutting down...")
}
