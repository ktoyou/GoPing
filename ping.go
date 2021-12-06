package main

import (
	"flag"
	"log"
	"net"
	"os"
	"strconv"

	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv4"
)

type Ping struct {
	listener *icmp.PacketConn

	ErrorHandler                  func()
	ReplyHandler                  func(header *ipv4.Header)
	DestinationUnreachableHandler func()
}

func (p *Ping) ConfigureIcmpPacket(count int) icmp.Message {
	var messsage icmp.Message
	messsage.Body = &icmp.Echo{
		ID:   os.Getpid(),
		Seq:  count,
		Data: []byte("ping"),
	}
	messsage.Type = ipv4.ICMPTypeEcho
	messsage.Code = 0
	return messsage
}

func (p *Ping) GetListener() *icmp.PacketConn {
	listener, err := icmp.ListenPacket("ip4:icmp", "0.0.0.0")
	if err != nil {
		log.Fatal(err.Error())
	}
	return listener
}

func (p *Ping) WaitPacket() {
	buff := make([]byte, 1500)
	readed, _, err := p.listener.ReadFrom(buff)
	if err != nil {
		log.Fatal(err.Error())
	}

	if message, err := icmp.ParseMessage(ipv4.ICMPTypeEchoReply.Protocol(), buff[:readed]); err == nil {
		ipv4header, _ := ipv4.ParseHeader(buff)

		switch message.Type {
		case ipv4.ICMPTypeEchoReply:
			if p.ReplyHandler != nil {
				p.ReplyHandler(ipv4header)
			}
			return
		case ipv4.ICMPTypeDestinationUnreachable:
			if p.DestinationUnreachableHandler != nil {
				p.DestinationUnreachableHandler()
			}
			return
		}
	}

	if p.ErrorHandler != nil {
		p.ErrorHandler()
	}
}

func (p *Ping) Ping(ip string, count int) {
	listener := p.GetListener()
	p.listener = listener

	for i := 0; i < count; i++ {
		msg := p.ConfigureIcmpPacket(i)
		buff, _ := msg.Marshal(nil)
		_, err := p.listener.WriteTo(buff, &net.IPAddr{IP: net.ParseIP(ip)})
		if err != nil {
			log.Fatal(err.Error())
		}
		p.WaitPacket()
	}

}

func main() {
	var ip string
	var count int
	var ping Ping

	ping.ReplyHandler = func(header *ipv4.Header) {
		log.Printf("%s bytes from %s; ttl %s; ver %s; id %s; tos %s",
			strconv.Itoa(header.TotalLen/1000),
			header.Src.String(),
			strconv.Itoa(header.TTL),
			strconv.Itoa(header.Version),
			strconv.Itoa(header.ID),
			strconv.Itoa(header.TOS))
	}

	ping.ErrorHandler = func() {
		log.Print("error ping")
	}

	ping.DestinationUnreachableHandler = func() {
		log.Printf("destination unreachable")
	}

	flag.StringVar(&ip, "ip", "8.8.8.8", "-ip 127.0.0.1")
	flag.IntVar(&count, "c", 1, "-c 4")
	flag.Parse()

	log.Printf("ping to %s", ip)
	ping.Ping(ip, count)
}
