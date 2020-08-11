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

	stringLocalAddr string
	localAddr       *net.IPAddr

	stringTargetAddr string
	targetAddr       *net.IPAddr

	conn *icmp.PacketConn

	signalChannel chan os.Signal
}

// NewTracer - create new tracer struct
func NewTracer(targetAddr string) (*Tracer, error) {

	tracer := &Tracer{}

	err := tracer.setLocalAddr()
	if err != nil {
		return nil, err
	}

	err = tracer.setTargetAddr(targetAddr)
	if err != nil {
		return nil, err
	}

	tracer.setSignalChan()
	tracer.SetInterval(1 * time.Second)

	return tracer, nil
}

// Start - start trace
func (tracer *Tracer) Start() error {
	fmt.Printf("traceroute to %s (%v)\n",
		tracer.stringTargetAddr,
		tracer.targetAddr)

	go tracer.catchExitSignal()

	err := tracer.start()
	if err != nil {
		return err
	}

	return nil
}

func (tracer *Tracer) start() error {
	conn, err := icmp.ListenPacket("ip4:icmp", tracer.localAddr.String())
	if err != nil {
		return err
	}
	tracer.conn = conn
	defer tracer.conn.Close()

	go tracer.receivePackets()
	tracer.sendPackets()

	return nil
}

func (tracer *Tracer) sendPackets() error {
	body := &icmp.Echo{
		ID:   os.Getpid() & 0xffff,
		Seq:  1,
		Data: []byte("DEFAULT-MESSAGE"),
	}

	message := icmp.Message{
		Type: ipv4.ICMPTypeEcho,
		Code: 0,
		Body: body,
	}

	binMessage, err := message.Marshal(nil)
	if err != nil {
		return err
	}

	ttl := 2

	tracer.conn.IPv4PacketConn().SetTTL(ttl)
	tracer.conn.IPv4PacketConn().SetControlMessage(ipv4.FlagTTL, true)

	for {
		select {
		case <-tracer.signalChannel:
			return nil
		default:
			_, err := tracer.conn.WriteTo(binMessage, tracer.targetAddr)
			if err != nil {
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
		case <-tracer.signalChannel:
			return nil
		default:

			// n, peer, err := tracer.conn.ReadFrom(buffer)
			// if err != nil {
			// 	return err
			// }

			n, controlMessage, peer, err := tracer.conn.IPv4PacketConn().ReadFrom(buffer)
			if err != nil {
				return err
			}
			if controlMessage != nil {
				incomeTTL := controlMessage.TTL
				fmt.Println(incomeTTL)
			}

			replyMessage, err := icmp.ParseMessage(protocolICMP, buffer[:n])
			if err != nil {
				return err
			}

			// switch replyMessage.Type {
			// case ipv4.ICMPTypeEcho:
			// 	fmt.Println(replyMessage)

			// }

			fmt.Println(peer)
			fmt.Println(replyMessage)
		}
	}
}

func (tracer *Tracer) setSignalChan() {
	tracer.signalChannel = make(chan os.Signal, 1)
	signal.Notify(tracer.signalChannel, os.Interrupt)
}

func (tracer *Tracer) catchExitSignal() {
	switch <-tracer.signalChannel {
	case os.Interrupt:
		fmt.Printf("exit\n")
		os.Exit(0)
	}
}

func (tracer *Tracer) setTargetAddr(targetAddr string) error {
	tracer.stringTargetAddr = targetAddr
	addr, err := net.ResolveIPAddr("ip", targetAddr)
	if err != nil {
		return err
	}
	tracer.targetAddr = addr

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
					tracer.localAddr = &net.IPAddr{
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

// SetInterval ...
func (tracer *Tracer) SetInterval(interval time.Duration) {
	tracer.interval = interval
}
