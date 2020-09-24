package trace

import (
	"fmt"
	"net"
	"os"
	"os/signal"
	"strings"
	"time"

	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv4"
)

// Tracer struct
type Tracer struct {
	interval time.Duration

	localAddr   string
	localIPAddr *net.IPAddr

	targetAddr   string
	targetIPAddr *net.IPAddr

	conn *icmp.PacketConn

	body       *icmp.Echo
	message    *icmp.Message
	binMessage *[]byte

	cancelChannel chan os.Signal
}

// Host struct contain information about each host
type Host struct {
	Addr   string
	IPAddr *net.IPAddr
}

// NewTracer - create new tracer struct
func NewTracer(targetAddr string) (*Tracer, error) {
	tracer := &Tracer{}

	if err := tracer.setMessage(); err != nil {
		return nil, err
	}

	if err := tracer.setLocalAddr(); err != nil {
		return nil, err
	}

	if err := tracer.setTargetAddr(targetAddr); err != nil {
		return nil, err
	}

	tracer.interval = time.Second

	return tracer, nil
}

func (tracer *Tracer) setMessage() error {
	tracer.body = &icmp.Echo{
		ID:   os.Getpid() & 0xffff,
		Seq:  1,
		Data: []byte("DEFAULT-MESSAGE"),
	}

	tracer.message = &icmp.Message{
		Type: ipv4.ICMPTypeEcho,
		Code: 0,
		Body: tracer.body,
	}

	binMessage, err := tracer.message.Marshal(nil)
	if err != nil {
		return err
	}

	tracer.binMessage = &binMessage

	return nil
}

// Start - start trace
func (tracer *Tracer) Start() error {
	fmt.Printf("traceroute to %s (%v)\n",
		tracer.targetAddr,
		tracer.targetIPAddr)

	tracer.cancelChannel = make(chan os.Signal, 1)
	signal.Notify(tracer.cancelChannel, os.Interrupt)

	go func() {
		switch <-tracer.cancelChannel {
		case os.Interrupt:
			fmt.Printf("\nexit\n")
			os.Exit(0)
		}
	}()

	conn, err := icmp.ListenPacket("ip4:icmp", tracer.localIPAddr.String())
	if err != nil {
		return err
	}
	tracer.conn = conn
	defer tracer.conn.Close()

	// go tracer.receivePackets()
	// for i := 2; i < 5; i++ {
	// 	tracer.sendPacketWithTTL(i)

	// 	time.Sleep(tracer.interval)
	// }

	go tracer.receivePackets()
	ttl := 2
	for {
		tracer.sendPacketWithTTL(ttl)

		time.Sleep(tracer.interval)

		ttl++
	}
}

func (tracer *Tracer) sendPacketWithTTL(ttl int) error {
	tracer.conn.IPv4PacketConn().SetTTL(ttl)
	tracer.conn.IPv4PacketConn().SetControlMessage(ipv4.FlagTTL, true)

	if _, err := tracer.conn.WriteTo(*tracer.binMessage, tracer.targetIPAddr); err != nil {
		return err
	}

	return nil
}

func (tracer *Tracer) sendPacketsWithTTL(ttl int) error {
	tracer.conn.IPv4PacketConn().SetTTL(ttl)
	tracer.conn.IPv4PacketConn().SetControlMessage(ipv4.FlagTTL, true)

	for {
		select {
		case <-tracer.cancelChannel:
			return nil
		default:
			if _, err := tracer.conn.WriteTo(*tracer.binMessage, tracer.targetIPAddr); err != nil {
				return err
			}

			time.Sleep(tracer.interval)
		}
	}

}

func (tracer *Tracer) receivePackets() error {
	buffer := make([]byte, 512)

	for {
		select {
		case <-tracer.cancelChannel:
			return nil
		default:
			// n, peer, err := tracer.conn.ReadFrom(buffer)
			// if err != nil {
			// 	return err
			// }
			_, controlMessage, peer, err := tracer.conn.IPv4PacketConn().ReadFrom(buffer)
			if err != nil {
				return err
			}

			// replyMessage, err := icmp.ParseMessage(protocolICMP, buffer[:n])
			// if err != nil {
			// 	return err
			// }

			// switch replyMessage.Type {
			// case ipv4.ICMPTypeEcho:
			// 	fmt.Println(replyMessage)
			// }

			fmt.Printf("packet from (%v) with ttl = %v\n\n", peer, controlMessage.TTL)
		}
	}
}

func (tracer *Tracer) setTargetAddr(targetAddr string) error {
	addr, err := net.ResolveIPAddr("ip", targetAddr)
	if err != nil {
		return err
	}
	tracer.targetIPAddr = addr

	return nil
}

// setLocalAddr ...
func (tracer *Tracer) setLocalAddr() error {
	netInterfaces, err := net.Interfaces()
	if err != nil {
		return err
	}

	for _, i := range netInterfaces {
		if strings.Contains(i.Flags.String(), "up") &&
			strings.Contains(i.Flags.String(), "broadcast") &&
			strings.Contains(i.Flags.String(), "multicast") {

			ipaddrs, err := i.Addrs()
			if err != nil {
				return err
			}

			for _, addr := range ipaddrs {
				switch v := addr.(type) {
				case *net.IPNet:
					ip := v.IP
					tracer.localIPAddr = &net.IPAddr{
						IP:   ip,
						Zone: "",
					}
					return nil
				case *net.IPAddr:
					//ip := v.IP
					//fmt.Println(ip)
				}
			}
			return nil
		}
	}

	return errLocalAddrNotFound
}
